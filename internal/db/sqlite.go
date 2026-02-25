package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/glebarez/go-sqlite"
)

// Store gerencia a conexão com o SQLite
type Store struct {
	DB *sql.DB
}

// NewStore inicializa o banco de dados na pasta .openclaw
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("falha ao obter home: %w", err)
	}

	dbPath := filepath.Join(home, ".openclaw", "clawproject.db")

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
		manager_session_key TEXT,
		manager_agent_id TEXT DEFAULT 'main',
		manager_status TEXT DEFAULT 'offline',
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

	CREATE TABLE IF NOT EXISTS project_messages (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		sender TEXT NOT NULL,
		message TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);
	`
	if _, err := s.DB.Exec(query); err != nil {
		return err
	}

	// Backward-compatible migration for existing DBs.
	if err := s.ensureColumn("projects", "manager_session_key", "TEXT"); err != nil {
		return err
	}
	if err := s.ensureColumn("projects", "manager_agent_id", "TEXT DEFAULT 'main'"); err != nil {
		return err
	}
	if err := s.ensureColumn("projects", "manager_status", "TEXT DEFAULT 'offline'"); err != nil {
		return err
	}

	return nil
}

func (s *Store) ensureColumn(table, column, definition string) error {
	rows, err := s.DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}

	_, err = s.DB.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition))
	return err
}
