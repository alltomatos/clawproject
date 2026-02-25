package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/alltomatos/clawproject/internal/core"
	"github.com/alltomatos/clawproject/internal/db"
)

type Server struct {
	store          *db.Store
	managerEnabled bool
	managerRuntime *managerRuntime
}

type managerRuntime struct {
	mu          sync.Mutex
	lastMessage map[string]time.Time
	calls       map[string]int
	busy        map[string]bool
}

func NewServer(store *db.Store) *Server {
	enabled := strings.EqualFold(os.Getenv("OPENCLAW_MANAGER_ENABLED"), "true")
	return &Server{
		store:          store,
		managerEnabled: enabled,
		managerRuntime: &managerRuntime{
			lastMessage: map[string]time.Time{},
			calls:       map[string]int{},
			busy:        map[string]bool{},
		},
	}
}

func (s *Server) RegisterHandlers() {
	http.HandleFunc("/api/projects", s.handleProjects)
	http.HandleFunc("/api/projects/", s.handleProjectManagerRoutes)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/version", s.handleVersion)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": core.Version})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "ok",
		"time":            time.Now(),
		"gateway":         "stable",
		"manager_enabled": s.managerEnabled,
	})
}

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		projects, err := s.store.ListProjects(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if projects == nil {
			projects = []*core.Project{}
		}
		json.NewEncoder(w).Encode(projects)

	case http.MethodPost:
		var p core.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		p.ID = uuid.New().String()
		p.CreatedAt = time.Now()
		p.Status = "active"
		p.ManagerAgentID = "main"
		p.ManagerStatus = "offline"

		if p.Path == "" {
			workspace := os.Getenv("CLAWPROJECT_WORKSPACE")
			if workspace == "" {
				workspace = "C:/Users/ronaldo/.openclaw/workspace"
			}
			p.Path = fmt.Sprintf("%s/%s", strings.TrimRight(workspace, "/"), sanitizePathName(p.Name))
		}

		if p.ManagerSessionKey == "" {
			if s.managerEnabled {
				p.ManagerSessionKey = fmt.Sprintf("pm-%s", p.ID)
				p.ManagerStatus = "active"
			} else {
				p.ManagerSessionKey = fmt.Sprintf("local-pm-%s", p.ID)
				p.ManagerStatus = "offline"
			}
		}

		if err := s.ensureProjectWorkspace(&p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := s.store.CreateProject(r.Context(), &p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleProjectManagerRoutes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}

	projectID := parts[0]
	project, err := s.store.GetProjectByID(r.Context(), projectID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if project == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	if parts[1] == "messages" && r.Method == http.MethodGet {
		messages, err := s.store.ListProjectMessages(r.Context(), projectID, 200)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(messages)
		return
	}

	if parts[1] == "summary" && r.Method == http.MethodGet {
		summary, err := s.store.GetProjectSummary(r.Context(), projectID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if summary == nil {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"project_id": projectID,
				"summary":    "Sem resumo ainda. Ele será atualizado automaticamente conforme o andamento.",
				"version":    0,
			})
			return
		}
		json.NewEncoder(w).Encode(summary)
		return
	}

	if parts[1] != "manager" {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 2 && r.Method == http.MethodGet {
		s.managerRuntime.mu.Lock()
		calls := s.managerRuntime.calls[projectID]
		s.managerRuntime.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"project_id":          project.ID,
			"manager_session_key": project.ManagerSessionKey,
			"manager_agent_id":    project.ManagerAgentID,
			"manager_status":      project.ManagerStatus,
			"manager_enabled":     s.managerEnabled,
			"api_calls":           calls,
		})
		return
	}

	if len(parts) == 3 && parts[2] == "message" && r.Method == http.MethodPost {
		s.handleManagerMessage(w, r, project)
		return
	}

	if len(parts) == 3 && parts[2] == "control" && r.Method == http.MethodPost {
		s.handleManagerControl(w, r, project)
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

type managerMessageRequest struct {
	Message string `json:"message"`
}

type managerControlRequest struct {
	Action string `json:"action"`
}

func (s *Server) handleManagerMessage(w http.ResponseWriter, r *http.Request, project *core.Project) {
	var req managerMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	s.managerRuntime.mu.Lock()
	projectID := project.ID
	if s.managerRuntime.busy[projectID] {
		s.managerRuntime.mu.Unlock()
		http.Error(w, "manager busy, try again shortly", http.StatusTooManyRequests)
		return
	}
	if last, ok := s.managerRuntime.lastMessage[projectID]; ok {
		if time.Since(last) < 800*time.Millisecond {
			s.managerRuntime.mu.Unlock()
			http.Error(w, "too many requests, slow down", http.StatusTooManyRequests)
			return
		}
	}
	s.managerRuntime.busy[projectID] = true
	s.managerRuntime.lastMessage[projectID] = time.Now()
	s.managerRuntime.calls[projectID]++
	calls := s.managerRuntime.calls[projectID]
	s.managerRuntime.mu.Unlock()

	_ = s.store.AddProjectMessage(r.Context(), project.ID, "user", req.Message)

	reply := "Integração com sessions_send habilitada para próxima etapa."
	mode := "openclaw-enabled"
	if !s.managerEnabled {
		reply = "Gestor do projeto ativo em modo local (API econômica). Mensagem registrada."
		mode = "local-fallback"
	}
	_ = s.store.AddProjectMessage(r.Context(), project.ID, "agent", reply)
	_ = s.refreshProjectSummary(r.Context(), project.ID)

	s.managerRuntime.mu.Lock()
	s.managerRuntime.busy[projectID] = false
	s.managerRuntime.mu.Unlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":    mode,
		"reply":   reply,
		"calls":   calls,
		"project": project.ID,
	})
}

func (s *Server) handleManagerControl(w http.ResponseWriter, r *http.Request, project *core.Project) {
	var req managerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action == "" {
		http.Error(w, "action is required", http.StatusBadRequest)
		return
	}

	sessionKey := project.ManagerSessionKey
	agentID := project.ManagerAgentID
	status := project.ManagerStatus

	switch action {
	case "pause":
		status = "paused"
	case "resume":
		if s.managerEnabled {
			status = "active"
		} else {
			status = "offline"
		}
	case "restart":
		sessionKey = fmt.Sprintf("pm-%s-%d", project.ID, time.Now().Unix())
		if s.managerEnabled {
			status = "active"
		} else {
			status = "offline"
		}
	default:
		http.Error(w, "invalid action", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateProjectManager(r.Context(), project.ID, sessionKey, agentID, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":                  true,
		"action":              action,
		"manager_session_key": sessionKey,
		"manager_status":      status,
	})
}

func (s *Server) refreshProjectSummary(ctx context.Context, projectID string) error {
	messages, err := s.store.ListProjectMessages(ctx, projectID, 120)
	if err != nil {
		return err
	}
	if len(messages) == 0 {
		return nil
	}

	previous, _ := s.store.GetProjectSummary(ctx, projectID)
	previousText := ""
	if previous != nil {
		previousText = previous.Summary
	}

	userMsgs := []string{}
	agentMsgs := []string{}
	for _, m := range messages {
		if m.Sender == "user" {
			userMsgs = append(userMsgs, m.Message)
		} else {
			agentMsgs = append(agentMsgs, m.Message)
		}
	}

	latestUser := "(sem entrada)"
	if len(userMsgs) > 0 {
		latestUser = compactText(userMsgs[len(userMsgs)-1], 180)
	}
	latestAgent := "(sem resposta)"
	if len(agentMsgs) > 0 {
		latestAgent = compactText(agentMsgs[len(agentMsgs)-1], 180)
	}

	objectives := collectMilestones(userMsgs, []string{"objetivo", "meta", "resultado", "mvp", "entreg"}, 3)
	decisions := collectMilestones(userMsgs, []string{"decid", "escolh", "vamos", "usar", "padrão", "stack"}, 4)
	blockers := collectMilestones(userMsgs, []string{"bloque", "erro", "falha", "problema", "risco", "imped"}, 3)
	nextSteps := collectMilestones(userMsgs, []string{"próximo", "next", "fazer", "etapa", "seguir", "depois"}, 4)

	if len(objectives) == 0 {
		objectives = []string{"Objetivo ainda não explicitado claramente no chat."}
	}
	if len(nextSteps) == 0 {
		nextSteps = []string{"Definir próximo passo acionável com prazo curto."}
	}

	carry := compactPreviousSummary(previousText, 380)
	summary := fmt.Sprintf("Resumo estratégico do projeto (auto-update)\n\n[Estado]\n- Interações: %d (user: %d | gestor: %d)\n- Última solicitação: %s\n- Última resposta do gestor: %s\n\n[Objetivo]\n%s\n\n[Decisões]\n%s\n\n[Bloqueios/Riscos]\n%s\n\n[Próximos passos]\n%s\n\n[Memória compactada]\n%s\n\n[Atualizado em]\n- %s",
		len(messages), len(userMsgs), len(agentMsgs), latestUser, latestAgent,
		toBullets(objectives), toBullets(decisions), toBullets(blockers), toBullets(nextSteps), carry,
		time.Now().Format(time.RFC3339))

	return s.store.UpsertProjectSummary(ctx, projectID, summary)
}

func (s *Server) ensureProjectWorkspace(p *core.Project) error {
	if p == nil {
		return fmt.Errorf("project is nil")
	}
	root := filepath.Clean(p.Path)
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return fmt.Errorf("falha ao criar estrutura do projeto: %w", err)
	}

	planningPath := filepath.Join(docsDir, "PLANNING.md")
	if _, err := os.Stat(planningPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		bootstrap := fmt.Sprintf("# PLANNING.md\n\nProjeto: %s\n\n## Triagem Inicial\n- Tipo: pendente (Novo/Existente)\n- Nicho: pendente\n\n## Objetivo\n- Definir objetivo principal\n\n## Entregáveis\n- docs/PLANNING.md (este arquivo)\n- PRD/DER/POPs conforme nicho\n\n## Próximos passos\n- Conduzir triagem no chat com o gestor dedicado.\n", p.Name)
		if err := os.WriteFile(planningPath, []byte(bootstrap), 0644); err != nil {
			return fmt.Errorf("falha ao criar PLANNING.md: %w", err)
		}
	}

	if _, err := os.Stat(filepath.Join(root, ".git")); os.IsNotExist(err) {
		_ = exec.Command("git", "-C", root, "init").Run()
	}

	return nil
}

func collectMilestones(messages []string, keywords []string, max int) []string {
	out := []string{}
	seen := map[string]bool{}
	for i := len(messages) - 1; i >= 0; i-- {
		m := strings.ToLower(strings.TrimSpace(messages[i]))
		if m == "" {
			continue
		}
		matched := false
		for _, k := range keywords {
			if strings.Contains(m, k) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		item := compactText(messages[i], 150)
		if !seen[item] {
			out = append(out, item)
			seen[item] = true
		}
		if len(out) >= max {
			break
		}
	}
	// reverse para manter ordem mais natural (antigo -> recente)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func compactText(v string, max int) string {
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	if v == "" {
		return "(vazio)"
	}
	if len(v) > max {
		return v[:max] + "..."
	}
	return v
}

func compactPreviousSummary(summary string, max int) string {
	summary = compactText(summary, max)
	if summary == "(vazio)" {
		return "Sem memória anterior consolidada."
	}
	return summary
}

func toBullets(items []string) string {
	if len(items) == 0 {
		return "- (sem itens)"
	}
	lines := make([]string, 0, len(items))
	for _, it := range items {
		lines = append(lines, "- "+it)
	}
	return strings.Join(lines, "\n")
}

func sanitizePathName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "novo-projeto"
	}
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(name)
}
