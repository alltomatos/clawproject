package db

import (
	"context"
	"github.com/alltomatos/clawproject/internal/core"
)

func (s *Store) CreateProject(ctx context.Context, p *core.Project) error {
	query := `INSERT INTO projects (id, name, description, path, git_url, status, created_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := s.DB.ExecContext(ctx, query, p.ID, p.Name, p.Description, p.Path, p.GitURL, p.Status, p.CreatedAt)
	return err
}

func (s *Store) ListProjects(ctx context.Context) ([]*core.Project, error) {
	query := `SELECT id, name, description, path, git_url, status, created_at FROM projects ORDER BY created_at DESC`
	rows, err := s.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []*core.Project
	for rows.Next() {
		p := &core.Project{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Path, &p.GitURL, &p.Status, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}
