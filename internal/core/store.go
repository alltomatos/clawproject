package core

import (
	"context"
)

// ProjectStore define as operações de banco para projetos
type ProjectStore interface {
	Create(ctx context.Context, p *Project) error
	GetByID(ctx context.Context, id string) (*Project, error)
	List(ctx context.Context) ([]*Project, error)
	Update(ctx context.Context, p *Project) error
	Delete(ctx context.Context, id string) error
}

// POPStore define as operações para POPs
type POPStore interface {
	Create(ctx context.Context, p *POP) error
	GetByProjectID(ctx context.Context, projectID string) ([]*POP, error)
	Update(ctx context.Context, p *POP) error
}
