# ClawProject - Subagente Gestor por Projeto

## Regra
Todo novo projeto criado no ClawProject deve criar automaticamente um subagente dedicado, que será o gestor especialista daquele projeto.

## Objetivo
Garantir execução com contexto isolado por projeto, sem contaminação de histórico entre iniciativas.

## Modelo de Dados (mínimo)
Adicionar no projeto os campos:
- `manager_session_key` (string, único)
- `manager_agent_id` (string, default: `main` ou perfil futuro dedicado)
- `manager_status` (string: `active|paused|archived`)

## Fluxo Backend
1. `POST /api/projects` cria projeto base.
2. Backend chama `sessions_spawn` com:
   - `mode: "session"`
   - `label: "pm-{project_id}"`
   - `task`: instruções do gestor + contexto do projeto
3. Persistir `sessionKey` retornado em `manager_session_key`.
4. Expor endpoint:
   - `GET /api/projects/:id/manager` (status e sessão)
   - `POST /api/projects/:id/manager/message` (encaminha mensagem com `sessions_send`)

## Fluxo Frontend
Na tela do projeto:
- Badge: `Gestor: Online/Offline`
- Chat conectado ao subagente do projeto
- Botões: `Reiniciar Gestor`, `Pausar`, `Retomar`

## Prompt Base do Gestor (resumo)
- Você é gestor especialista deste projeto.
- Siga `docs/planner_mode.md`.
- Mantenha a Bíblia em `docs/` sempre atualizada.
- Não iniciar execução sem triagem e PLANNING.md.
- Entregáveis variam por nicho (software, conteúdo, prospecção, gestão).

## Regras de Segurança
- Um projeto nunca usa sessão de outro projeto.
- Se `manager_session_key` inválido, recriar gestor automaticamente.
- Toda ação crítica deve gerar log de auditoria no projeto.
