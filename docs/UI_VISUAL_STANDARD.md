# ClawProject - UI Visual Standard

## Regra de Produto
Todo projeto que envolva tela (web/mobile/desktop/painéis) deve seguir um **padrão visual único** do ClawProject.

## Princípios
- Consistência de layout (sidebar, header, cards, ações primárias)
- Hierarquia visual clara (título > contexto > ações > detalhes)
- Estados obrigatórios: loading, vazio, erro, sucesso
- Componentes reutilizáveis antes de criar variações
- Cores e tipografia coerentes (sem “temas por tela” aleatórios)

## Aplicação prática
- Páginas de listagem: suporte a grid/tabela com mesmo estilo de botões/filtros
- Formulários: padrão único de labels, inputs, espaçamento e CTA
- Chat/fluxos guiados: painel lateral para contexto operacional; área central para conversa/execução
- Cards de status: padrão de badges e semântica de cor

## Checklist de entrega UI
- [ ] Seguiu componentes e estilos padrão
- [ ] Tem estados loading/empty/error
- [ ] Acessibilidade mínima (contraste, foco, labels)
- [ ] Responsivo (desktop e largura menor)
- [ ] Não criou variações visuais desnecessárias
