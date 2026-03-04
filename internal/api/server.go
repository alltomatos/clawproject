package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	mu           sync.Mutex
	lastMessage  map[string]time.Time
	lastWarmup   map[string]time.Time
	calls        map[string]int
	busy         map[string]bool
}

type managerMessageRequest struct {
	Message string `json:"message"`
}

type managerControlRequest struct {
	Action string `json:"action"`
}

type deliverProjectRequest struct {
	ApprovedBy string `json:"approved_by"`
	Notes      string `json:"notes"`
	Force      bool   `json:"force"`
}

type openclawAgentResult struct {
	Status string `json:"status"`
	Result struct {
		Payloads []struct {
			Text string `json:"text"`
		} `json:"payloads"`
	} `json:"result"`
}

func NewServer(store *db.Store) *Server {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv("OPENCLAW_MANAGER_ENABLED")))
	enabled := raw != "false" && raw != "0" && raw != "off"
	return &Server{
		store:          store,
		managerEnabled: enabled,
		managerRuntime: &managerRuntime{
			lastMessage: map[string]time.Time{},
			lastWarmup:  map[string]time.Time{},
			calls:       map[string]int{},
			busy:        map[string]bool{},
		},
	}
}

func (s *Server) RegisterHandlers() {
	http.HandleFunc("/api/projects", s.handleProjects)
	http.HandleFunc("/api/projects/", s.handleProjectManagerRoutes)
	http.HandleFunc("/api/projects/delete/", s.handleDeleteProject)
	http.HandleFunc("/api/status", s.handleStatus)
	http.HandleFunc("/api/version", s.handleVersion)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	projectID := strings.TrimPrefix(r.URL.Path, "/api/projects/delete/")
	project, err := s.store.GetProjectByID(r.Context(), projectID)
	if err != nil || project == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	// 1. Destruição do Agente no OpenClaw
	if project.ManagerAgentID != "" && project.ManagerAgentID != "main" {
		fmt.Printf("[destroy] removing agent id=%s\n", project.ManagerAgentID)
		// Alterado para --force para automatizar
		_ = exec.Command("openclaw", "agents", "delete", project.ManagerAgentID, "--force").Run()
		
		// 1.1 Limpeza manual da pasta do agente caso o CLI falhe em deletar tudo
		home, _ := os.UserHomeDir()
		agentPath := filepath.Join(home, ".openclaw", "agents", project.ManagerAgentID)
		if _, err := os.Stat(agentPath); err == nil {
			fmt.Printf("[destroy] manual cleanup of agent folder: %s\n", agentPath)
			_ = os.RemoveAll(agentPath)
		}

		// Garante que o OpenClaw Gateway recarregue a configuração após a remoção
		fmt.Printf("[destroy] restarting gateway to apply changes\n")
		_ = exec.Command("openclaw", "gateway", "restart").Run()
	}

	// 2. Remoção de Arquivos (Cuidado extremo)
	if project.Path != "" && strings.Contains(project.Path, ".openclaw/workspace") {
		fmt.Printf("[destroy] removing path=%s\n", project.Path)
		_ = os.RemoveAll(project.Path)
	}

	// 3. Remoção do Banco de Dados
	_ = s.store.DeleteProject(r.Context(), projectID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "Destruição total concluída."})
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
			log.Printf("[api] error listing projects: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if projects == nil {
			projects = []*core.Project{}
		}
		log.Printf("[api] handleProjects GET: returning %d projects", len(projects))
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
		p.ManagerAgentID = sanitizePathName(p.Name)
		p.ManagerStatus = "offline"

		if p.Path == "" {
			workspace := os.Getenv("CLAWPROJECT_WORKSPACE")
			if workspace == "" {
				workspace = "C:/Users/ronaldo/.openclaw/workspace"
			}
			p.Path = fmt.Sprintf("%s/%s", strings.TrimRight(workspace, "/"), sanitizePathName(p.Name))
		}

		if p.ManagerSessionKey == "" {
			p.ManagerSessionKey = "main" 
			if s.managerEnabled {
				p.ManagerStatus = "active"
			}
		}

		if err := s.ensureProjectWorkspace(&p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if p.ManagerAgentID != "main" {
			go func(agentID, agentName, workspace string, p core.Project) {
				absWorkspace, _ := filepath.Abs(workspace)
				_ = exec.Command("openclaw", "agents", "add", agentID, "--workspace", absWorkspace, "--non-interactive").Run()
				
				// DIRETRIZ MASTER DO SUBAGENTE - Definida como instrução de base (system prompt) via CLI
				soulInstructions := fmt.Sprintf(`Você é o Gestor deste projeto. 
Identidade: %s.
Líder do Projeto: %s (%s).
Localização: %s.
Tipo de Projeto: %s.
Objetivo: %s.

Diretrizes:
1. Sua Bíblia reside em docs/. SEMPRE consulte docs/ antes de agir.
2. Quebre o objetivo em um ROADMAP.md em docs/.
3. Para cada item do Roadmap, crie um card no pipeline.
4. Reporte-se ao %s de forma %s.`, 
					p.Vibe, p.LeaderName, p.LeaderEmail, p.Location, p.ProjectType, p.Description, p.LeaderName, p.Vibe)
				
				if p.ProjectType == "existing" && p.GitURL != "" {
					soulInstructions += "\n\nIMPORTANTE: Este é um projeto existente. Sua primeira tarefa é analisar o repositório GIT: " + p.GitURL
					// Clone em background (exemplo simplificado)
					_ = exec.Command("git", "clone", p.GitURL, absWorkspace).Run()
				}

				_ = exec.Command("openclaw", "agents", "set-identity", "--agent", agentID, "--name", "Gestor "+agentName, "--emoji", "🦞", "--theme", soulInstructions).Run()
				_ = exec.Command("openclaw", "config", "set", fmt.Sprintf("agents.models.%s.model", agentID), "google-antigravity/gemini-3-flash").Run()
				_ = exec.Command("openclaw", "gateway", "restart").Run()
			}(p.ManagerAgentID, p.Name, p.Path, p)
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
			json.NewEncoder(w).Encode(map[string]interface{}{ "project_id": projectID, "summary": "Sem resumo.", "version": 0 })
			return
		}
		json.NewEncoder(w).Encode(summary)
		return
	}

	if parts[1] == "planner" && r.Method == http.MethodGet {
		st, _ := s.store.GetPlannerState(r.Context(), projectID)
		niche, stage, projectType := "", "triage_type", ""
		if st != nil {
			niche, stage, projectType = st.Niche, st.Stage, st.ProjectType
		}
		deliverables := expectedDeliverablesForNiche(niche)
		docsDir := filepath.Join(project.Path, "docs")
		existing := []string{}
		for _, f := range deliverables {
			if _, err := os.Stat(filepath.Join(docsDir, f)); err == nil { existing = append(existing, f) }
		}
		lastCheckpoint := ""
		cmd := exec.Command("git", "-C", project.Path, "log", "--oneline", "-n", "1", "--", "docs")
		if out, err := cmd.CombinedOutput(); err == nil { lastCheckpoint = strings.TrimSpace(string(out)) }
		json.NewEncoder(w).Encode(map[string]interface{}{
			"project_id": projectID, "stage": stage, "project_type": projectType, "niche": niche,
			"deliverables": deliverables, "deliverables_done": existing, "last_checkpoint": lastCheckpoint,
		})
		return
	}

	if parts[1] == "deliver" && r.Method == http.MethodPost {
		s.handleProjectDeliver(w, r, project)
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
		dailyCalls, _ := s.store.GetDailyManagerUsage(r.Context(), projectID, time.Now().Format("2006-01-02"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"project_id": project.ID, "manager_session_key": project.ManagerSessionKey,
			"manager_agent_id": project.ManagerAgentID, "manager_status": project.ManagerStatus,
			"manager_enabled": s.managerEnabled, "api_calls": calls, "daily_calls": dailyCalls,
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
	if s.managerRuntime.busy[project.ID] {
		s.managerRuntime.mu.Unlock()
		http.Error(w, "manager busy", http.StatusTooManyRequests)
		return
	}
	s.managerRuntime.busy[project.ID] = true
	s.managerRuntime.mu.Unlock()
	defer func() {
		s.managerRuntime.mu.Lock()
		s.managerRuntime.busy[project.ID] = false
		s.managerRuntime.mu.Unlock()
	}()

	_ = s.store.AddProjectMessage(r.Context(), project.ID, "user", req.Message)

	reply := ""
	if s.managerEnabled {
		bridgeReply, err := s.runOpenClawManagerTurn(project, req.Message)
		if err != nil {
			reply = fmt.Sprintf("Erro no Gestor OpenClaw: %v", err)
		} else {
			reply = bridgeReply
			_ = s.advancePlannerFromMessage(r.Context(), project, req.Message)
		}
	} else {
		reply, _ = s.nextPlannerReply(r.Context(), project, req.Message)
	}

	_ = s.store.AddProjectMessage(r.Context(), project.ID, "agent", reply)
	_ = s.refreshProjectSummary(r.Context(), project.ID)
	_ = s.syncIncrementalDeliverables(r.Context(), project, req.Message, reply)

	json.NewEncoder(w).Encode(map[string]interface{}{ "reply": reply, "project": project.ID })
}

func (s *Server) handleManagerControl(w http.ResponseWriter, r *http.Request, project *core.Project) {
	var req managerControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action := strings.ToLower(strings.TrimSpace(req.Action))
	sessionKey, agentID, status := project.ManagerSessionKey, project.ManagerAgentID, project.ManagerStatus

	switch action {
	case "pause": status = "paused"
	case "resume": status = "active"
	case "restart": 
		sessionKey = "main"
		status = "active"
	case "start-execution":
		// Envia diretriz master e salva a resposta NO CHAT do ClawProject
		directive := "ESTADO: Triagem concluída. DIRETRIZ MASTER ATIVADA: Você é o Gestor deste projeto. Sua Bíblia reside em docs/. Regra #1: SEMPRE consulte docs/ antes de agir. Regra #2: Quebre o objetivo em um ROADMAP.md em docs/. Regra #3: Para cada item do Roadmap, crie um card no pipeline. Seja pragmático. Confirme que entendeu lendo docs/."
		reply, err := s.runOpenClawManagerTurn(project, directive)
		if err != nil {
			http.Error(w, fmt.Sprintf("erro ao iniciar execução: %v", err), http.StatusInternalServerError)
			return
		}
		_ = s.store.AddProjectMessage(r.Context(), project.ID, "agent", reply)
		_ = s.refreshProjectSummary(r.Context(), project.ID)
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "message": "Diretrizes enviadas ao gestor.", "reply": reply})
		return
	default:
		http.Error(w, "invalid action", http.StatusBadRequest)
		return
	}

	if err := s.store.UpdateProjectManager(r.Context(), project.ID, sessionKey, agentID, status); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if action == "restart" && s.managerEnabled { go s.warmupManagerSession(project.ID, sessionKey, agentID) }

	json.NewEncoder(w).Encode(map[string]interface{}{ "ok": true, "manager_session_key": sessionKey, "manager_status": status })
}

func (s *Server) handleProjectDeliver(w http.ResponseWriter, r *http.Request, project *core.Project) {
	var req deliverProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.ApprovedBy = strings.TrimSpace(req.ApprovedBy)
	if req.ApprovedBy == "" {
		http.Error(w, "approved_by is required", http.StatusBadRequest)
		return
	}

	if project.Status != "ready_for_delivery" && project.Status != "active" && !req.Force {
		http.Error(w, "project status must be ready_for_delivery or active", http.StatusConflict)
		return
	}

	var runningJobs int
	if err := s.store.DB.QueryRowContext(r.Context(), `SELECT COUNT(1) FROM project_jobs WHERE project_id = ? AND status = 'running'`, project.ID).Scan(&runningJobs); err != nil {
		runningJobs = 0
	}
	if runningJobs > 0 && !req.Force {
		http.Error(w, "there are running jobs", http.StatusConflict)
		return
	}

	docsDir := filepath.Join(project.Path, "docs")
	planningPath := filepath.Join(docsDir, "PLANNING.md")
	roadmapPath := filepath.Join(docsDir, "ROADMAP.md")
	if _, err := os.Stat(planningPath); err != nil && !req.Force {
		http.Error(w, "docs/PLANNING.md not found", http.StatusConflict)
		return
	}
	if _, err := os.Stat(roadmapPath); err != nil && !req.Force {
		http.Error(w, "docs/ROADMAP.md not found", http.StatusConflict)
		return
	}

	var doneCards int
	if err := s.store.DB.QueryRowContext(r.Context(), `SELECT COUNT(1) FROM cards WHERE project_id = ? AND status = 'done'`, project.ID).Scan(&doneCards); err != nil {
		doneCards = 0
	}
	if doneCards < 1 && !req.Force {
		http.Error(w, "at least one done card is required", http.StatusConflict)
		return
	}

	if err := os.MkdirAll(docsDir, 0755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	now := time.Now().UTC()
	deliveryPath := filepath.Join(docsDir, "DELIVERY.md")
	deliveryContent := fmt.Sprintf("# DELIVERY - %s\n\n## Resumo Executivo\n- Aprovado por: %s\n- Data: %s\n- Notas: %s\n\n## Entregáveis\n- docs/PLANNING.md\n- docs/ROADMAP.md\n- docs/DELIVERY.md\n\n## Execução\n- Cards concluídos: %d\n\n## Próximos passos (V2)\n- Refinar backlog e integrações\n", project.Name, req.ApprovedBy, now.Format(time.RFC3339), req.Notes, doneCards)
	if err := os.WriteFile(deliveryPath, []byte(deliveryContent), 0644); err != nil {
		http.Error(w, "failed to write DELIVERY.md", http.StatusInternalServerError)
		return
	}

	summaryJSON, _ := json.Marshal(map[string]any{
		"approved_by": req.ApprovedBy,
		"notes":       req.Notes,
		"done_cards":  doneCards,
		"artifacts":   []string{"docs/PLANNING.md", "docs/ROADMAP.md", "docs/DELIVERY.md"},
	})

	_, err := s.store.DB.ExecContext(r.Context(), `
		UPDATE projects
		SET status = 'delivered', delivered_at = CURRENT_TIMESTAMP, delivery_summary = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, string(summaryJSON), project.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"ok":           true,
		"project_id":   project.ID,
		"state":        "delivered",
		"delivered_at": now.Format(time.RFC3339),
		"artifacts":    []string{"docs/PLANNING.md", "docs/ROADMAP.md", "docs/DELIVERY.md"},
	})
}

func (s *Server) runOpenClawManagerTurn(project *core.Project, userMessage string) (string, error) {
	agentID := strings.TrimSpace(project.ManagerAgentID)
	if agentID == "" { agentID = "main" }
	sessionID := "main"

	// AGORA ENVIAMOS APENAS A MENSAGEM PURA, SEM INJEÇÃO DE CONTEXTO OCULTA EM CADA TURNO
	cmd := exec.Command("openclaw", "agent", "--agent", agentID, "--session-id", sessionID, "--message", userMessage, "--json", "--timeout", "90", "--verbose", "off")
	
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	outStr := stdout.String()
	
	if err != nil { return "", fmt.Errorf("openclaw failed: %v | %s", err, stderr.String()) }
	
	var parsed openclawAgentResult
	start, end := strings.Index(outStr, "{"), strings.LastIndex(outStr, "}")
	if start >= 0 && end > start {
		if err := json.Unmarshal([]byte(outStr[start:end+1]), &parsed); err != nil {
			return strings.TrimSpace(outStr), nil
		}
	} else {
		return strings.TrimSpace(outStr), nil
	}
	
	if len(parsed.Result.Payloads) > 0 && strings.TrimSpace(parsed.Result.Payloads[0].Text) != "" {
		return parsed.Result.Payloads[0].Text, nil
	}
	return "Processado.", nil
}

func (s *Server) nextPlannerReply(ctx context.Context, project *core.Project, userMessage string) (string, string) {
	st, _ := s.store.GetPlannerState(ctx, project.ID)
	if st == nil { st = &db.PlannerState{ProjectID: project.ID, Stage: "triage_type"} }
	msg := strings.ToLower(strings.TrimSpace(userMessage))
	reply := "Recebido."
	switch st.Stage {
	case "triage_type":
		if strings.Contains(msg, "novo") { st.ProjectType, st.Stage, reply = "novo", "triage_niche", "Qual o nicho principal?" } else { reply = "Este projeto é Novo ou Existente?" }
	case "triage_niche":
		st.Niche = detectNiche(msg)
		if st.Niche != "" { st.Stage, reply = "objective", fmt.Sprintf("Nicho '%s' definido. Qual o objetivo principal?", st.Niche) }
	case "objective":
		st.Objective, st.Stage, reply = userMessage, "deliverables", "Objetivo registrado. Quais os entregáveis esperados?"
	case "deliverables":
		st.Deliverables, st.Stage, reply = userMessage, "active", "Triagem finalizada. Use o painel lateral para ver os docs e inicie a execução quando pronto."
	default: reply = "Contexto recebido."
	}
	_ = s.store.UpsertPlannerState(ctx, st)
	_ = s.updatePlanningFromState(project, st)
	return reply, "local"
}

func (s *Server) advancePlannerFromMessage(ctx context.Context, project *core.Project, userMessage string) error {
	_, _ = s.nextPlannerReply(ctx, project, userMessage)
	return nil
}

func (s *Server) ensureProjectWorkspace(p *core.Project) error {
	root := filepath.Clean(p.Path)
	docsDir := filepath.Join(root, "docs")
	_ = os.MkdirAll(docsDir, 0755)
	planningPath := filepath.Join(docsDir, "PLANNING.md")
	if _, err := os.Stat(planningPath); os.IsNotExist(err) {
		_ = os.WriteFile(planningPath, []byte("# PLANNING.md\n\nProjeto: "+p.Name+"\n"), 0644)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); os.IsNotExist(err) { _ = exec.Command("git", "-C", root, "init").Run() }
	return nil
}

func detectNiche(msg string) string {
	if strings.Contains(msg, "soft") { return "software" }
	if strings.Contains(msg, "cont") { return "conteudo" }
	return "geral"
}

func expectedDeliverablesForNiche(niche string) []string { return []string{"PLANNING.md", "ROADMAP.md", "DELIVERY.md"} }

func (s *Server) updatePlanningFromState(project *core.Project, st *db.PlannerState) error {
	path := filepath.Join(project.Path, "docs", "PLANNING.md")
	content := fmt.Sprintf("# PLANNING.md\n\nNicho: %s\nEtapa: %s\nObjetivo: %s\n", st.Niche, st.Stage, st.Objective)
	return os.WriteFile(path, []byte(content), 0644)
}

func (s *Server) refreshProjectSummary(ctx context.Context, projectID string) error {
	messages, _ := s.store.ListProjectMessages(ctx, projectID, 50)
	if len(messages) == 0 { return nil }
	summary := "Estado Atual: " + messages[len(messages)-1].Message
	return s.store.UpsertProjectSummary(ctx, projectID, summary)
}

func (s *Server) syncIncrementalDeliverables(ctx context.Context, project *core.Project, userMsg, agentReply string) error {
	path := filepath.Join(project.Path, "docs", "PLANNING.md")
	entry := fmt.Sprintf("\n## Log %s\n- U: %s\n- A: %s\n", time.Now().Format("15:04"), userMsg, agentReply)
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if f != nil { defer f.Close(); _, _ = f.WriteString(entry) }
	
	_ = exec.Command("git", "-C", project.Path, "add", ".").Run()
	_ = exec.Command("git", "-C", project.Path, "commit", "-m", "chore: sync docs").Run()
	return nil
}

func (s *Server) warmupManagerSession(projectID, sessionID, agentID string) {
	_ = exec.Command("openclaw", "agent", "--agent", agentID, "--session-id", sessionID, "--message", "GESTOR_PRONTO", "--timeout", "30").Run()
}

func sanitizePathName(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "-")
}
