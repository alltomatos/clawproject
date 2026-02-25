package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	if len(parts) < 2 || parts[1] != "manager" {
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
	defer s.managerRuntime.mu.Unlock()

	projectID := project.ID
	if s.managerRuntime.busy[projectID] {
		http.Error(w, "manager busy, try again shortly", http.StatusTooManyRequests)
		return
	}

	if last, ok := s.managerRuntime.lastMessage[projectID]; ok {
		if time.Since(last) < 800*time.Millisecond {
			http.Error(w, "too many requests, slow down", http.StatusTooManyRequests)
			return
		}
	}

	s.managerRuntime.busy[projectID] = true
	s.managerRuntime.lastMessage[projectID] = time.Now()
	s.managerRuntime.calls[projectID]++
	calls := s.managerRuntime.calls[projectID]
	s.managerRuntime.busy[projectID] = false

	if !s.managerEnabled {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"mode":    "local-fallback",
			"reply":   "Gestor do projeto ativo em modo local (API econômica). Mensagem registrada.",
			"calls":   calls,
			"project": project.ID,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"mode":    "openclaw-enabled",
		"reply":   "Integração com sessions_send habilitada para próxima etapa.",
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

func sanitizePathName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "novo-projeto"
	}
	replacer := strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-")
	return replacer.Replace(name)
}
