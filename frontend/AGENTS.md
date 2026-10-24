# Frontend do grupo — referência do contrato v1.4
- A fonte da verdade é `docs/api.md` na raiz do repositório do grupo. A cópia integral consultada em `main` (`e5592fe`) fica em `docs/api-grupo-v1.4.md`; a v1.1 é histórica.
- React + Vite, API Go externa. Não adicionar motor financeiro, Supabase, Next.js ou IA ao frontend.
- O backend Go já está implementado no repositório. Os mocks exercitam cenários da UI; não atribuir a eles paridade financeira/gabarito v1.4 nem resultados de testes reais do Go.
- JSON snake_case; dinheiro inteiro em centavos; datas civis ISO.
- Textos de decisão/calculo vêm do backend, exibir sem reescrever.
- Upload XMLHttpRequest com progresso; polling a cada500ms até concluir/erro; nunca carregar todas as notas de um lote.
- Tabela paginada e filtros/ordem no servidor. Download CSV pelo endpoint.
- Gestor X-Gestor-Senha em sessionStorage; nunca colocar senha real em código/logs/URL.
- Cessão com UUID por abertura do diálogo, reutilizado em tentativas.
- Mock apenas quando VITE_MOCK_API=true. Sem fallback automático do backend para mock.
- Teste local a partir da raiz: `cd frontend`, `npm ci`, `npm run demo` (Node >=22.12.0, porta padrão5173). `.env.demo` é versionado; nenhum segredo, Supabase, Go, Docker ou Vercel é necessário. `npm run dev` e `npm run build` usam o Go; build isolado de demonstração é `npm run build:demo`.

- Visual atual: Figma EFdeJ9T6mnBayCJcbh60aT / seção 1:913; login demo e dashboard verde com Parkinsans. Assets originais em public/figma.
- Landing pública em /; Dashboard em /dashboard; envio em /enviar. Login /login é somente demo, sem credenciais reais; nunca transformar o marcador local em autenticação de produção.

- Grade horizontal única em `src/app-layout.css`: conteúdo de até1280px, gutter64/40/20px e centralização dentro de `.app-workspace`. Todas as páginas usam `.container`, inclusive Dashboard e o conteúdo interno do Resultado. Não definir larguras/margens horizontais próprias por página nem somar a sidebar novamente. Rodapé, perfil e confirmação seguem a mesma grade; modais mantêm suas dimensões próprias.

- Cabeçalho único do portal: `PageHeader` em `src/components/page-header.tsx`, com título/descrição à esquerda. O seletor de perfil aparece na Dashboard e na área do gestor, via `showRoleSwitch`; no celular pode ocupar a linha seguinte. Usar também nos estados de loading/erro e no404. Ações, caminho da página, versão e sessão ficam abaixo do cabeçalho. Padding superior64/48/36px vem de `src/app-layout.css`; não criar faixas flutuantes de perfil ou um segundo h1. Login de demonstração e títulos dos modais mantêm seus layouts próprios.

- Meus lotes reutiliza `LotCard` e `NewLotLink` da Dashboard; estilos compartilhados em lot-card.css/new-lot-link.css. Busca/situação/data-base/ordem filtram somente os últimos100 IDs locais e seus resumos, usando src/lib/lot-filters.ts. Não criar endpoint de listagem nem carregar notas para esses filtros. Campos seguem o login Figma:60px, raio24px, Parkinsans.

- Meu perfil em `/perfil` substitui o acesso público às regras; `/regras` redireciona para o perfil. Regras são apresentadas na gestão, com entrada direta na demonstração e transição verde de troca de perfil. No modo Go, o acesso por senha permanece conforme o contrato. Isto muda a UI, não a autorização dos endpoints Go. Perfil é demonstrativo: dados/preferências em sessionStorage, nenhuma senha persistida ou enviada e nenhuma autenticação real inventada. Logout também limpa a senha do gestor da sessão.

- Assistente: robô abre `AssistantChat`, painel com sugestões por contexto, conversa e nova sessão. Conexão real em `/ai`, inclusive quando lotes usam mocks. Consulta status antes do envio; nunca substitui falha por resposta fictícia. POST multipart com session_id, dependencies.lote_id e stream=true (SSE com RunContent/RunCompleted/RunError). Tratar ERROR em HTTP200 sem expor conteúdo técnico. Markdown sem HTML. Chaves apenas no nginx. Proxy Vite AI_PROXY_TARGET padrão localhost:3000 (nginx do grupo).
- Resultado: result-refinement.css amplia leitura e controles, preservando grade/Parkinsans/verde. Confirmação inline sem sobrepor robô; nota prioriza motivo e ação, depois dados e atalho contextual ao chat.

- Mobile: `src/mobile.css` é importado após App para consolidar os ajustes até760px. Navegação inferior com rótulos e safe-area, áreas de toque de44px+, inputs16px para evitar zoom no iOS, cartões de notas, paginação API, chat e envio em tela cheia. Não ocultar controles para corrigir overflow; JSON/tabelas técnicas mantêm rolagem local. Validar360px e767px. Markdown do chat carregado sob demanda.

- Loading: upload/preparação/análise usam indicador simples (`showMedia=false`), sem vídeo ou espera artificial na UI. Mock de análise700ms. O robô aparece na confirmação da cessão via CessionModal, que reutiliza o padrão visual de Novo lote. Durante a requisição de cessão não permitir fechar/cancelar nem submeter novamente. A mesma chave de idempotência é mantida em retries. O mock simula1.200ms de resposta; backend real sem atraso artificial.
