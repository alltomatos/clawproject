-- Schema Inicial para o ClawFlow (SQLite)

-- Tabela de Projetos
CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    path TEXT NOT NULL,          -- Caminho absoluto no workspace
    git_url TEXT,
    status TEXT DEFAULT 'active', -- active, archived, completed
    manager_session_key TEXT,
    manager_agent_id TEXT,
    manager_status TEXT,
    leader_name TEXT,
    leader_email TEXT,
    location TEXT,
    vibe TEXT,
    project_type TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de POPs (Procedimento Operacional Padro)
-- Um projeto pode ter mltiplos POPs conforme evolui
CREATE TABLE IF NOT EXISTS pops (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    title TEXT NOT NULL,
    content_markdown TEXT,       -- O corpo flexvel do POP
    structure_type TEXT,         -- software, infra, business, etc.
    status TEXT DEFAULT 'draft', -- draft, approved, deprecated
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
);

-- Tabela de Cards (A execuo do POP)
CREATE TABLE IF NOT EXISTS cards (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    pop_id TEXT,                 -- Opcional: Card vinculado a um POP especfico
    title TEXT NOT NULL,
    description TEXT,
    status TEXT DEFAULT 'todo',  -- todo, doing, review, done
    priority INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (pop_id) REFERENCES pops(id) ON DELETE SET NULL
);

-- Tabela de Logs de Atividade do Agente
CREATE TABLE IF NOT EXISTS agent_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id TEXT,
    agent_id TEXT,               -- tinker, designer, main
    message TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (card_id) REFERENCES cards(id) ON DELETE CASCADE
);
