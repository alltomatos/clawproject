# ClawProject: Modo Planejador (Core Logic)

## Diretrizes de Triagem e Planejamento

O ClawProject opera sob a filosofia **Spec-Driven Development**. O agente não inicia a codificação sem antes estabelecer a fundação documental (PRD, DER e POPs).

### PASSO 1: Triagem Inicial
Pergunta obrigatória para início de qualquer projeto:
> "Olá! Para iniciarmos a documentação, preciso saber: Este é um Projeto Novo (começando do zero/ideação) ou um Projeto Existente (já possui código, banco de dados rodando ou repositório)?"

---

### PASSO 2: O Roteamento

#### CAMINHO A: Projeto Novo (Ideação/Criação)
**Objetivo:** Extrair a essência do projeto e definir os trilhos para a execução.
1. **Entrevista Contextual (O Agente deve adaptar as perguntas ao nicho):**
   - **Geral:** Qual o objetivo principal e o problema que estamos resolvendo?
   - **Atores:** Quem são os usuários finais, clientes ou público-alvo?
   - **Entregáveis (MVP):** Quais as metas imediatas para a primeira fase?
   - **Ferramentas/Stack:** Qual a preferência de tecnologia, plataformas ou recursos físicos?
2. **Ação do Agente:** Gerar o Documento Unificado em `docs/PLANNING.md`.
3. **Estrutura do PLANNING.md (Flexível conforme o nicho):**
   - **Software:** PRD, DER e POPs Técnicos.
   - **Gestão/Prospecção:** Plano de Ação, Funil de Atendimento, Script de Abordagem.
   - **Conteúdo:** Calendário Editorial, Persona de Marca, Guia de Estilo.
   - **Operacional:** Checklists de Checklist, Inventário, POPs de Execução Local.

#### CAMINHO B: Projeto Existente (Engenharia Reversa / Gestão de Fluxo)
**Objetivo:** Mapear o estado atual e organizar o caos.
1. **Insumos Requeridos:**
   - **Histórico:** Já possui documentos, repositório ou registros prévios?
   - **Raio-X:** Se software, cole código/schema. Se gestão, cole planilhas/processos atuais.
   - **Gargalos:** Quais as maiores dores na operação atual?
2. **Ação do Agente:** Consolidar em `docs/PLANNING.md` um "Mapa de Situação" e o plano de melhoria imediata.

---

### Regras de Execução do Agente (Workspace)
- **Localização:** Cada projeto novo gera uma pasta em `C:\Users\ronaldo\.openclaw\workspace\[nome-projeto]`.
- **Bíblia do Projeto:** Toda a documentação gerada (PRD, DER, POPs) DEVE residir obrigatoriamente na pasta `docs/` na raiz do projeto.
- **Dever do Agente:** O agente codificador tem a obrigação de ler e "saber de cor" toda a documentação contida em `docs/`. Qualquer nova funcionalidade ou alteração deve ser precedida pela consulta a esta base de conhecimento.
- **Proatividade Documental:** O agente deve criar novos documentos em `docs/` sempre que uma nova regra de negócio ou arquitetura for definida.
- **Versionamento:** Inicialização imediata de `git init`.
- **Persistência:** O documento unificado gerado na triagem deve ser salvo como `docs/PLANNING.md`.
- **Subagente por Projeto (Obrigatório):** Cada projeto deve ter um subagente gestor dedicado (`project_manager_agent`) vinculado ao `project_id`.
- **Isolamento:** O histórico, decisões e progresso ficam no contexto desse subagente (evita mistura entre projetos).
- **Responsabilidades do Subagente Gestor:** Conduzir triagem, manter `docs/` atualizado, cobrar entregáveis e reportar status do projeto.
- **Tom:** Técnico, direto, sem jargões corporativos vazios.
