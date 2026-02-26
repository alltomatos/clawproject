package db

import (
	"context"
	"database/sql"

	"github.com/alltomatos/clawproject/internal/core"
)

func (s *Store) CreateProject(ctx context.Context, p *core.Project) error {
	query := `INSERT INTO projects (id, name, description, path, git_url, status, manager_session_key, manager_agent_id, manager_status, leader_name, leader_email, location, vibe, project_type, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.DB.ExecContext(ctx, query,
		p.ID, p.Name, p.Description, p.Path, p.GitURL, p.Status,
		p.ManagerSessionKey, p.ManagerAgentID, p.ManagerStatus, 
		p.LeaderName, p.LeaderEmail, p.Location, p.Vibe, p.ProjectType,
		p.CreatedAt,
	)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]*core.Project, error) {
	query := `SELECT id, name, description, path, git_url, status, manager_session_key, manager_agent_id, manager_status, leader_name, leader_email, location, vibe, project_type, created_at FROM projects ORDER BY created_at DESC`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*core.Project
	for rows.Next() {
		p := &core.Project{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.Path, &p.GitURL, &p.Status, 
			&p.ManagerSessionKey, &p.ManagerAgentID, &p.ManagerStatus,
			&p.LeaderName, &p.LeaderEmail, &p.Location, &p.Vibe, &p.ProjectType,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) GetProjectByID(ctx context.Context, id string) (*core.Project, error) {
	query := `SELECT id, name, description, path, git_url, status, manager_session_key, manager_agent_id, manager_status, leader_name, leader_email, location, vibe, project_type, created_at FROM projects WHERE id = ? LIMIT 1`
	p := &core.Project{}
	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.Path, &p.GitURL, &p.Status,
		&p.ManagerSessionKey, &p.ManagerAgentID, &p.ManagerStatus,
		&p.LeaderName, &p.LeaderEmail, &p.Location, &p.Vibe, &p.ProjectType,
		&p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (s *Store) UpdateProjectManager(ctx context.Context, projectID, sessionKey, agentID, status string) error {
	query := `UPDATE projects SET manager_session_key = ?, manager_agent_id = ?, manager_status = ? WHERE id = ?`
	_, err := s.DB.ExecContext(ctx, query, sessionKey, agentID, status, projectID)
	return err
}
