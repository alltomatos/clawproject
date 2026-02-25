package api

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	if parts[1] == "planner" && r.Method == http.MethodGet {
		st, _ := s.store.GetPlannerState(r.Context(), projectID)
		niche := ""
		stage := "triage_type"
		projectType := ""
		if st != nil {
			niche = st.Niche
			stage = st.Stage
			projectType = st.ProjectType
		}
		deliverables := expectedDeliverablesForNiche(niche)
		docsDir := filepath.Join(project.Path, "docs")
		existing := []string{}
		for _, f := range deliverables {
			if _, err := os.Stat(filepath.Join(docsDir, f)); err == nil {
				existing = append(existing, f)
			}
		}
		lastCheckpoint := ""
		cmd := exec.Command("git", "-C", project.Path, "log", "--oneline", "-n", "1", "--", "docs")
		if out, err := cmd.CombinedOutput(); err == nil {
			lastCheckpoint = strings.TrimSpace(string(out))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"project_id":       projectID,
			"stage":            stage,
			"project_type":     projectType,
			"niche":            niche,
			"deliverables":     deliverables,
			"deliverables_done": existing,
			"last_checkpoint":  lastCheckpoint,
		})
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
			"project_id":          project.ID,
			"manager_session_key": project.ManagerSessionKey,
			"manager_agent_id":    project.ManagerAgentID,
			"manager_status":      project.ManagerStatus,
			"manager_enabled":     s.managerEnabled,
			"api_calls":           calls,
			"daily_calls":         dailyCalls,
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

	dailyLimit := resolveDailyManagerLimit()
	today := time.Now().Format("2006-01-02")
	dailyCalls, err := s.store.GetDailyManagerUsage(r.Context(), project.ID, today)
	if err == nil && dailyCalls >= dailyLimit {
		s.managerRuntime.mu.Lock()
		s.managerRuntime.busy[projectID] = false
		s.managerRuntime.mu.Unlock()
		http.Error(w, fmt.Sprintf("daily manager limit reached (%d/%d)", dailyCalls, dailyLimit), http.StatusTooManyRequests)
		return
	}
	if _, err := s.store.IncrementDailyManagerUsage(r.Context(), project.ID, today); err != nil {
		s.managerRuntime.mu.Lock()
		s.managerRuntime.busy[projectID] = false
		s.managerRuntime.mu.Unlock()
		http.Error(w, "failed to register manager usage", http.StatusInternalServerError)
		return
	}

	_ = s.store.AddProjectMessage(r.Context(), project.ID, "user", req.Message)

	reply := ""
	mode := "local-fallback"
	if s.managerEnabled {
		mode = "openclaw-bridge"
		bridgeReply, err := s.runOpenClawManagerTurn(project, req.Message)
		if err != nil {
			mode = "bridge-fallback"
			reply, _ = s.nextPlannerReply(r.Context(), project, req.Message)
			reply = fmt.Sprintf("%s\n\n[Bridge fallback] %v", reply, err)
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

	if action == "restart" && s.managerEnabled {
		go s.warmupManagerSession(project.ID, sessionKey, agentID)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":                  true,
		"action":              action,
		"manager_session_key": sessionKey,
		"manager_status":      status,
	})
}

func (s *Server) nextPlannerReply(ctx context.Context, project *core.Project, userMessage string) (string, string) {
	st, _ := s.store.GetPlannerState(ctx, project.ID)
	if st == nil {
		st = &db.PlannerState{ProjectID: project.ID, Stage: "triage_type"}
	}

	msg := strings.ToLower(strings.TrimSpace(userMessage))
	normalized := strings.Join(strings.Fields(msg), " ")

	reply := "Recebido."
	switch st.Stage {
	case "triage_type":
		if strings.Contains(normalized, "novo") {
			st.ProjectType = "novo"
			st.Stage = "triage_niche"
			reply = "Perfeito. Projeto Novo confirmado. Qual o nicho principal? (software, conteúdo, prospecção/vendas, gestão/operacional)"
		} else if strings.Contains(normalized, "existente") {
			st.ProjectType = "existente"
			st.Stage = "triage_niche"
			reply = "Perfeito. Projeto Existente confirmado. Qual o nicho principal para engenharia reversa? (software, conteúdo, prospecção/vendas, gestão/operacional)"
		} else {
			reply = "Para iniciar a triagem: este projeto é Novo ou Existente?"
		}
	case "triage_niche":
		st.Niche = detectNiche(normalized)
		if st.Niche == "" {
			reply = "Me diga o nicho principal: software, conteúdo, prospecção/vendas, gestão ou operacional."
		} else {
			st.Stage = "objective"
			reply = fmt.Sprintf("Nicho '%s' definido. Agora descreva o objetivo principal em 1-2 frases.", st.Niche)
		}
	case "objective":
		st.Objective = strings.TrimSpace(userMessage)
		st.Stage = "deliverables"
		reply = "Objetivo registrado. Quais entregáveis imediatos você espera nesta fase (MVP/plano/checklists/scripts/docs)?"
	case "deliverables":
		st.Deliverables = strings.TrimSpace(userMessage)
		st.Stage = "active"
		reply = "Excelente. Triagem concluída. Vou manter o PLANNING.md atualizado conforme o avanço do projeto."
	default:
		reply = "Contexto recebido. Vou atualizar os marcos e próximos passos no PLANNING.md."
	}

	_ = s.store.UpsertPlannerState(ctx, st)
	_ = s.updatePlanningFromState(project, st)
	return reply, "local"
}

func (s *Server) advancePlannerFromMessage(ctx context.Context, project *core.Project, userMessage string) error {
	_, _ = s.nextPlannerReply(ctx, project, userMessage)
	return nil
}

type openclawAgentResult struct {
	Status string `json:"status"`
	Result struct {
		Payloads []struct {
			Text string `json:"text"`
		} `json:"payloads"`
	} `json:"result"`
}

func (s *Server) runOpenClawManagerTurn(project *core.Project, userMessage string) (string, error) {
	if project == nil {
		return "", fmt.Errorf("project nil")
	}
	sessionID := strings.TrimSpace(project.ManagerSessionKey)
	if sessionID == "" {
		sessionID = fmt.Sprintf("pm-%s", project.ID)
	}
	agentID := strings.TrimSpace(project.ManagerAgentID)
	if agentID == "" {
		agentID = "main"
	}
	st, _ := s.store.GetPlannerState(context.Background(), project.ID)
	niche := "geral"
	if st != nil && strings.TrimSpace(st.Niche) != "" {
		niche = st.Niche
	}
	summary, _ := s.store.GetProjectSummary(context.Background(), project.ID)
	summaryText := ""
	if summary != nil {
		summaryText = compactText(summary.Summary, 500)
	}
	prompt := fmt.Sprintf("%s\n\nProjeto: %s\nSessão: %s\nContexto resumido: %s\nMensagem do usuário: %s", baseManagerPromptByNiche(niche), project.Name, sessionID, summaryText, userMessage)
	cmd := exec.Command("openclaw", "agent", "--agent", agentID, "--session-id", sessionID, "--message", prompt, "--json", "--timeout", "90")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("openclaw agent failed: %v | %s", err, strings.TrimSpace(string(out)))
	}
	var parsed openclawAgentResult
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", fmt.Errorf("parse agent json failed: %v", err)
	}
	if len(parsed.Result.Payloads) == 0 || strings.TrimSpace(parsed.Result.Payloads[0].Text) == "" {
		return "Sem resposta textual do gestor OpenClaw.", nil
	}
	return parsed.Result.Payloads[0].Text, nil
}

func detectNiche(msg string) string {
	switch {
	case strings.Contains(msg, "software"), strings.Contains(msg, "sistema"), strings.Contains(msg, "api"):
		return "software"
	case strings.Contains(msg, "conteúdo"), strings.Contains(msg, "conteudo"), strings.Contains(msg, "editorial"):
		return "conteudo"
	case strings.Contains(msg, "prospec"), strings.Contains(msg, "vendas"), strings.Contains(msg, "funil"):
		return "prospeccao"
	case strings.Contains(msg, "gest"):
		return "gestao"
	case strings.Contains(msg, "operac"), strings.Contains(msg, "checklist"), strings.Contains(msg, "invent"):
		return "operacional"
	default:
		return ""
	}
}

func expectedDeliverablesForNiche(niche string) []string {
	switch strings.ToLower(strings.TrimSpace(niche)) {
	case "software":
		return []string{"PLANNING.md", "PRD.md", "DER.md", "POPS.md"}
	case "prospeccao":
		return []string{"PLANNING.md", "FUNIL.md", "SCRIPT_ABORDAGEM.md", "METRICAS.md"}
	case "conteudo", "conteúdo":
		return []string{"PLANNING.md", "CALENDARIO_EDITORIAL.md", "PERSONA.md", "GUIA_ESTILO.md"}
	case "gestao", "gestão", "operacional":
		return []string{"PLANNING.md", "PLANO_ACAO.md", "CHECKLISTS.md", "POPS.md"}
	default:
		return []string{"PLANNING.md", "ENTREGAVEIS.md"}
	}
}

func (s *Server) updatePlanningFromState(project *core.Project, st *db.PlannerState) error {
	if project == nil || st == nil {
		return nil
	}
	planningPath := filepath.Join(project.Path, "docs", "PLANNING.md")
	base := fmt.Sprintf("# PLANNING.md\n\nProjeto: %s\n\n## Triagem Inicial\n- Tipo: %s\n- Nicho: %s\n- Etapa: %s\n\n## Objetivo\n%s\n\n## Entregáveis\n%s\n\n## Próximos passos\n- Continuar execução pelo chat do gestor dedicado.\n- Atualizar este documento a cada marco relevante.\n", project.Name, emptyOrPending(st.ProjectType), emptyOrPending(st.Niche), st.Stage, emptyOrPending(st.Objective), emptyOrPending(st.Deliverables))
	if err := os.WriteFile(planningPath, []byte(base), 0644); err != nil {
		return err
	}
	if st.Stage == "active" {
		_ = s.ensureNicheDeliverables(project, st)
	}
	return nil
}

func emptyOrPending(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "pendente"
	}
	return v
}

func (s *Server) ensureNicheDeliverables(project *core.Project, st *db.PlannerState) error {
	docsDir := filepath.Join(project.Path, "docs")
	_ = os.MkdirAll(docsDir, 0755)

	writeIfMissing := func(name, content string) {
		path := filepath.Join(docsDir, name)
		if _, err := os.Stat(path); err == nil {
			return
		}
		_ = os.WriteFile(path, []byte(content), 0644)
	}

	objective := emptyOrPending(st.Objective)
	deliverables := emptyOrPending(st.Deliverables)
	niche := strings.ToLower(strings.TrimSpace(st.Niche))

	switch niche {
	case "software":
		writeIfMissing("PRD.md", fmt.Sprintf("# PRD\n\n## Objetivo\n%s\n\n## Escopo MVP\n%s\n\n## Histórias de Usuário\n- Definir histórias principais\n", objective, deliverables))
		writeIfMissing("DER.md", "# DER\n\n## Entidades\n- Definir entidades e relacionamentos\n\n## Modelo\n```mermaid\nerDiagram\n  ENTITY ||--o{ OTHER : relates\n```\n")
		writeIfMissing("POPS.md", "# POPs Técnicos\n\n## POP: Setup local\n- Passos de ambiente\n\n## POP: Deploy\n- Checklist de release\n")
	case "prospeccao":
		writeIfMissing("FUNIL.md", fmt.Sprintf("# Funil de Prospecção\n\n## Objetivo\n%s\n\n## Entregáveis esperados\n%s\n\n## Etapas\n- ICP\n- Lista de leads\n- Cadência de contato\n", objective, deliverables))
		writeIfMissing("SCRIPT_ABORDAGEM.md", "# Script de Abordagem\n\n## Primeiro contato\n- Mensagem inicial\n\n## Follow-up\n- Cadência e objeções\n")
		writeIfMissing("METRICAS.md", "# Métricas\n\n- Taxa de resposta\n- Taxa de reunião\n- Taxa de conversão\n")
	case "conteudo", "conteúdo":
		writeIfMissing("CALENDARIO_EDITORIAL.md", fmt.Sprintf("# Calendário Editorial\n\n## Objetivo\n%s\n\n## Entregáveis\n%s\n\n## Plano semanal\n- Segunda: ...\n- Quarta: ...\n- Sexta: ...\n", objective, deliverables))
		writeIfMissing("PERSONA.md", "# Persona\n\n## Público-alvo\n- Perfil\n\n## Dores\n- ...\n")
		writeIfMissing("GUIA_ESTILO.md", "# Guia de Estilo\n\n## Tom de voz\n- ...\n\n## Formatos\n- ...\n")
	case "gestao", "gestão", "operacional":
		writeIfMissing("PLANO_ACAO.md", fmt.Sprintf("# Plano de Ação\n\n## Objetivo\n%s\n\n## Entregáveis\n%s\n\n## Frentes\n- Frente 1\n- Frente 2\n", objective, deliverables))
		writeIfMissing("CHECKLISTS.md", "# Checklists Operacionais\n\n## Rotina diária\n- [ ] ...\n\n## Rotina semanal\n- [ ] ...\n")
		writeIfMissing("POPS.md", "# POPs Operacionais\n\n## POP 01\n- Procedimento\n")
	default:
		writeIfMissing("ENTREGAVEIS.md", fmt.Sprintf("# Entregáveis\n\n## Objetivo\n%s\n\n## Entregáveis esperados\n%s\n", objective, deliverables))
	}

	return nil
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

func (s *Server) syncIncrementalDeliverables(ctx context.Context, project *core.Project, userMessage, agentReply string) error {
	if project == nil {
		return nil
	}
	st, err := s.store.GetPlannerState(ctx, project.ID)
	if err != nil || st == nil {
		return err
	}
	if st.Stage != "active" {
		return nil
	}

	docsDir := filepath.Join(project.Path, "docs")
	niche := strings.ToLower(strings.TrimSpace(st.Niche))
	targets := []string{"PLANNING.md"}
	switch niche {
	case "software":
		targets = append(targets, "PRD.md", "DER.md", "POPS.md")
	case "prospeccao":
		targets = append(targets, "FUNIL.md", "SCRIPT_ABORDAGEM.md", "METRICAS.md")
	case "conteudo", "conteúdo":
		targets = append(targets, "CALENDARIO_EDITORIAL.md", "PERSONA.md", "GUIA_ESTILO.md")
	case "gestao", "gestão", "operacional":
		targets = append(targets, "PLANO_ACAO.md", "CHECKLISTS.md", "POPS.md")
	default:
		targets = append(targets, "ENTREGAVEIS.md")
	}

	entry := fmt.Sprintf("\n\n## Marco %s\n- Input usuário: %s\n- Resposta gestor: %s\n", time.Now().Format("2006-01-02 15:04"), compactText(userMessage, 220), compactText(agentReply, 260))
	updated := false
	for _, name := range targets {
		path := filepath.Join(docsDir, name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if err := appendWithRotation(path, entry, 180_000); err == nil {
			updated = true
		}
	}
	if updated {
		_ = checkpointProjectDocs(project.Path, st.Niche, userMessage, agentReply)
	}
	return nil
}

func appendWithRotation(path, entry string, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 180_000
	}
	if info, err := os.Stat(path); err == nil && info.Size() >= maxBytes {
		stamp := time.Now().Format("20060102-150405")
		rotated := fmt.Sprintf("%s.%s.bak", path, stamp)
		if err := os.Rename(path, rotated); err == nil {
			header := fmt.Sprintf("# Arquivo rotacionado\n\nOrigem: %s\nData: %s\n\n", filepath.Base(path), time.Now().Format(time.RFC3339))
			_ = os.WriteFile(path, []byte(header), 0644)
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

func checkpointProjectDocs(projectPath, niche, userMessage, agentReply string) error {
	if strings.TrimSpace(projectPath) == "" {
		return nil
	}
	h := sha1.Sum([]byte(compactText(userMessage, 120) + "|" + compactText(agentReply, 120)))
	hash := hex.EncodeToString(h[:])[:8]
	msg := fmt.Sprintf("chore(docs): marco %s [%s]", strings.ToLower(strings.TrimSpace(niche)), hash)

	cmdAdd := exec.Command("git", "-C", projectPath, "add", "docs")
	if out, err := cmdAdd.CombinedOutput(); err != nil {
		_ = out
		return nil // não quebrar fluxo do MVP
	}
	cmdCommit := exec.Command("git", "-C", projectPath, "commit", "-m", msg)
	out, err := cmdCommit.CombinedOutput()
	if err != nil {
		// Sem mudanças para commit é esperado em alguns ciclos
		if strings.Contains(strings.ToLower(string(out)), "nothing to commit") {
			return nil
		}
		return nil
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

func resolveDailyManagerLimit() int {
	raw := strings.TrimSpace(os.Getenv("OPENCLAW_MANAGER_DAILY_LIMIT"))
	if raw == "" {
		return 120
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 120
	}
	return n
}

func baseManagerPromptByNiche(niche string) string {
	switch strings.ToLower(strings.TrimSpace(niche)) {
	case "software":
		return "Você é gestor de projeto de software. Seja técnico, pragmático, entregue próximos passos acionáveis e mantenha docs/PLANNING.md atualizado por marcos."
	case "prospeccao":
		return "Você é gestor de prospecção/vendas. Foque em funil, ICP, scripts, cadência e métricas de conversão com ações objetivas."
	case "conteudo", "conteúdo":
		return "Você é gestor de conteúdo. Foque em persona, calendário editorial, formatos, distribuição e métricas de performance."
	case "gestao", "gestão", "operacional":
		return "Você é gestor operacional. Foque em processos, checklists, responsabilidades, riscos e melhoria contínua."
	default:
		return "Você é gestor especialista de projeto multi-nicho. Entregue decisões claras, riscos e próximos passos práticos."
	}
}

func (s *Server) warmupManagerSession(projectID, sessionID, agentID string) {
	msg := "Inicie a sessão do gestor deste projeto. Responda apenas: GESTOR_PRONTO."
	cmd := exec.Command("openclaw", "agent", "--agent", agentID, "--session-id", sessionID, "--message", msg, "--json", "--timeout", "30")
	_ = cmd.Run()
	_ = s.store.AddProjectMessage(context.Background(), projectID, "agent", "[manager-restart] sessão reinicializada")
}

func sanitizePathName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "novo-projeto"
	}
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(name)
}
