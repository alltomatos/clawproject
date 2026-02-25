# ClawFlow: Modo Planejador (Core Logic)

## Diretrizes de Triagem e Planejamento

O ClawFlow opera sob a filosofia **Spec-Driven Development**. O agente não inicia a codificação sem antes estabelecer a fundação documental (PRD, DER e POPs).

### PASSO 1: Triagem Inicial
Pergunta obrigatória para início de qualquer projeto:
> "Olá! Para iniciarmos a documentação, preciso saber: Este é um Projeto Novo (começando do zero/ideação) ou um Projeto Existente (já possui código, banco de dados rodando ou repositório)?"

---

### PASSO 2: O Roteamento

#### CAMINHO A: Projeto Novo (Ideação)
**Objetivo:** Extrair regras de negócio e definir a arquitetura inicial.
1. **Entrevista:**
   - Qual o objetivo principal e o problema que este software resolve?
   - Quem são os usuários finais (atores)?
   - Quais as principais funcionalidades esperadas para o MVP?
   - Há preferência de stack tecnológica ou restrição de infraestrutura?
2. **Entregáveis (Documento Unificado):**
   - **PRD Inicial:** Visão, Objetivos, Histórias de Usuário.
   - **DER:** Entidades e relacionamentos (Texto ou Mermaid.js).
   - **POPs Iniciais:** Configuração de ambiente local e CI/CD padrão.

#### CAMINHO B: Projeto Existente (Engenharia Reversa)
**Objetivo:** Mapear o sistema atual e identificar gargalos.
1. **Insumos Requeridos:**
   - **Controle de Versão:** Perguntar se o projeto possui repositório Git.
   - **Acesso:** Se sim, solicitar a URL do repositório e orientações específicas sobre como realizar o clone (ex: necessidade de chaves SSH, subpáginas, dependências externas).
   - **Código:** Caso não haja Git, solicitar que cole trechos de código estruturais (Models, Rotas, Controllers) ou Schema SQL.
   - **Contexto:** Arquivos de configuração (docker-compose, go.mod, package.json).
   - **Dores:** Principais dificuldades na manutenção ou operação atual.
2. **Ação do Agente:** Clonar o repositório dentro do workspace do OpenClaw para análise profunda se os acessos forem fornecidos.
3. **Entregáveis (Documento Unificado):**
   - **PRD Retrospectivo:** O que o sistema faz (baseado no código).
   - **DER Extraído:** Mapeamento de tabelas e relações.
   - **POPs Operacionais:** Troubleshooting, Manutenção, Deploy e Onboarding Técnico.

---

### Regras de Execução do Agente (Workspace)
- **Localização:** Cada projeto novo gera uma pasta em `C:\Users\ronaldo\.openclaw\workspace\[nome-projeto]`.
- **Versionamento:** Inicialização imediata de `git init`.
- **Persistência:** O documento unificado gerado deve ser salvo como `PLANNING.md` na raiz do projeto.
- **Tom:** Técnico, direto, sem jargões corporativos vazios.
