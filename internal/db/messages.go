package db

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/alltomatos/clawproject/internal/core"
)

func (s *Store) AddProjectMessage(ctx context.Context, projectID, sender, message string) error {
	query := `INSERT INTO project_messages (id, project_id, sender, message, created_at) VALUES (?, ?, ?, ?, ?)`
	_, err := s.DB.ExecContext(ctx, query, uuid.NewString(), projectID, sender, message, time.Now())
	return err
}

func (s *Store) ListProjectMessages(ctx context.Context, projectID string, limit int) ([]*core.ProjectMessage, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	query := `SELECT id, project_id, sender, message, created_at FROM project_messages WHERE project_id = ? ORDER BY created_at ASC LIMIT ?`
	rows, err := s.DB.QueryContext(ctx, query, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*core.ProjectMessage{}
	for rows.Next() {
		m := &core.ProjectMessage{}
		if err := rows.Scan(&m.ID, &m.ProjectID, &m.Sender, &m.Message, &m.CreatedAt); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, nil
}
