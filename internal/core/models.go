package core

import (
	"time"
)

type Project struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	Path              string    `json:"path"`
	GitURL            string    `json:"git_url"`
	Status            string    `json:"status"`
	ManagerSessionKey string    `json:"manager_session_key"`
	ManagerAgentID    string    `json:"manager_agent_id"`
	ManagerStatus     string    `json:"manager_status"`
	CreatedAt         time.Time `json:"created_at"`
}

type POP struct {
	ID              string    `json:"id"`
	ProjectID       string    `json:"project_id"`
	Title           string    `json:"title"`
	ContentMarkdown string    `json:"content_markdown"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

type Card struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	POPID     string    `json:"pop_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectMessage struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Sender    string    `json:"sender"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ProjectRepository define as operaes de banco para projetos
type ProjectRepository interface {
	Create(p *Project) error
	GetByID(id string) (*Project, error)
	List() ([]*Project, error)
}
