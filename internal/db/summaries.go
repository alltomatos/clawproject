package db

import (
	"context"
	"database/sql"
	"time"
)

type ProjectSummary struct {
	ProjectID string    `json:"project_id"`
	Summary   string    `json:"summary"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) GetProjectSummary(ctx context.Context, projectID string) (*ProjectSummary, error) {
	query := `SELECT project_id, summary, version, updated_at FROM project_summaries WHERE project_id = ? LIMIT 1`
	row := s.DB.QueryRowContext(ctx, query, projectID)
	ps := &ProjectSummary{}
	if err := row.Scan(&ps.ProjectID, &ps.Summary, &ps.Version, &ps.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return ps, nil
}

func (s *Store) UpsertProjectSummary(ctx context.Context, projectID, summary string) error {
	current, err := s.GetProjectSummary(ctx, projectID)
	if err != nil {
		return err
	}
	if current == nil {
		_, err = s.DB.ExecContext(ctx, `INSERT INTO project_summaries (project_id, summary, version, updated_at) VALUES (?, ?, 1, ?)`, projectID, summary, time.Now())
		return err
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE project_summaries SET summary = ?, version = ?, updated_at = ? WHERE project_id = ?`, summary, current.Version+1, time.Now(), projectID)
	return err
}
