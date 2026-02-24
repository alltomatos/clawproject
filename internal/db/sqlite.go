package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

// Store gerencia a conexo com o SQLite
type Store struct {
	DB *sql.DB
}

// NewStore inicializa o banco de dados na pasta .openclaw
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter home: %w", err)
	}

	dbPath := filepath.Join(home, ".openclaw", "clawflow.db")
	
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao abrir sqlite: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("falha no ping do db: %w", err)
	}

	s := &Store{DB: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("falha nas migrations: %w", err)
	}

	return s, nil
}

func (s *Store) migrate() error {
	query := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		path TEXT NOT NULL,
		git_url TEXT,
		status TEXT DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS pops (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		title TEXT NOT NULL,
		content_markdown TEXT,
		status TEXT DEFAULT 'draft',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);

	CREATE TABLE IF NOT EXISTS cards (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		pop_id TEXT,
		title TEXT NOT NULL,
		status TEXT DEFAULT 'todo',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	`
	_, err := s.DB.Exec(query)
	return err
}
