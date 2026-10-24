# Antecipe Claro — frontend do grupo

Interface React + Vite para o backend Go de [`joaoemanuelzilli/antecipe_kanastra`](https://github.com/joaoemanuelzilli/antecipe_kanastra). Referência atual: **contrato v1.4**, copiado de `docs/api.md` na revisão `e5592fe` da `main`.

O backend Go já está implementado no repositório. O frontend tem dois modos explícitos: demonstração com dados sintéticos e conexão à API Go. A ajuda do mascote é um guia local; esta interface não chama o serviço de IA do repositório. Não depende de Supabase.

A entrega para testes está na branch `codex/frontend-contract-v1.1` do fork [`matteuszu/antecipe_kanastra`](https://github.com/matteuszu/antecipe_kanastra), para revisão pelo grupo em pull request. O nome da branch foi mantido; a referência documental atual é v1.4.

## Testar agora com mocks

Requer Node.js **22.12 ou superior** e npm. A partir da raiz de um clone que contenha a branch da contribuição:

```sh
git switch codex/frontend-contract-v1.1
cd frontend
npm ci
npm run demo
```

Abra [http://localhost:5173](http://localhost:5173), ou a porta informada pelo Vite se 5173 estiver ocupada. Não é necessário criar `.env.local`, fornecer segredos, iniciar o Go, instalar Docker ou configurar Vercel. O arquivo versionado `.env.demo` já ativa `VITE_MOCK_API=true`. Se você já estiver em `frontend/`, execute somente os dois comandos npm.

Na landing, entre na demonstração e use dados fictícios ou clique em **Explorar demonstração**. O login só grava um marcador de acesso visual; não autentica contas nem envia ou guarda e-mail/senha. O portal começa em `/dashboard`.

Use o lote de exemplo para acompanhar o envio, as cinco etapas de processamento, as notas paginadas e as explicações. Os valores são fictícios e nenhuma cessão movimenta dinheiro.

No gestor, a senha pública de demonstração é `demo-gestor`. Ela vale somente no mock, nunca deve ser usada no backend real.

## Conectar ao Go

Depois de instalar as dependências em `frontend/`:

```sh
npm run dev
```

O Vite atende em `http://localhost:5173` e encaminha `/api` para `http://localhost:8080`. Para iniciar o backend e seus serviços, siga o [README do repositório](../README.md). Para mudar o destino local, crie `.env.local` a partir de `.env.example` e ajuste `API_PROXY_TARGET`. O modo normal não inicia o mock e não o usa como fallback se o Go estiver indisponível. Não coloque segredos em variáveis `VITE_*`, pois elas chegam ao navegador.

A senha de gestor é validada em `GET /api/gestor/verificar`, guardada em `sessionStorage` e enviada em `X-Gestor-Senha`, exatamente como combinado pelo grupo. Nenhuma senha real vai para código, URL ou logs.

Com a API rodando, um check de leitura confirma as principais estruturas do contrato:

```sh
npm run verificar:api -- http://localhost:8080
```

Esse check não envia notas nem confirma cessões. Os testes desta contribuição usam mocks; não houve validação desta entrega contra o Go/Postgres real nem execução do Compose. Os testes e resultados do backend registrados pelo grupo são independentes dessa validação de interface.

## Docker/Compose do repositório

Docker não é necessário para testar a demonstração. Para executar a stack completa, use a configuração já mantida pelo grupo e o comando `make up` da raiz; não é preciso copiar pastas nem criar outro serviço `web`.

O Dockerfile do frontend compila com `node:24-alpine` e serve `dist` em `nginx:1.27-alpine`. O template em `templates/default.conf.template` recebe `API_UPSTREAM` e `AI_UPSTREAM` e conserva a integração interna de `AI_SECURITY_KEY` do repositório. Essa chave pertence ao nginx, nunca ao bundle ou às variáveis `VITE_*`.

O nginx encaminha `/api` ao Go, preserva os headers de autenticação/idempotência e aceita uploads de até 300 MiB. O limite apresentado pela interface vem de `/api/config`, e a API é a autoridade final. As rotas `/ai/` existentes são preservadas pela infraestrutura; a UI desta contribuição oferece apenas ajuda local. A imagem não foi executada nesta validação.

## Contrato seguido

Uma cópia integral da referência atual está em [`docs/api-grupo-v1.4.md`](docs/api-grupo-v1.4.md). A fonte da verdade continua sendo `docs/api.md` na raiz do repositório do grupo. A cópia v1.1 fica preservada como histórico.

- JSON em snake_case e valores inteiros em centavos.
- Upload por `XMLHttpRequest` com progresso real de envio, campo `data_base`, resposta 202 e replay 200.
- Consulta de andamento a cada 500 ms, encerrada em `concluido`/`erro`. Exibe etapas, linhas, percentual, fila e duração.
- Notas buscadas por páginas de 50, com filtros, pesquisa e ordenação na API. O frontend nunca percorre todas as páginas para montar uma tabela ou exportação.
- Download CSV pelo endpoint, com os filtros atuais. Linhas `FORMATO_INVALIDO` exibem campos nulos com “—”.
- Textos de recusa (`titulo`, `mensagem`, `acao`) e cálculo (`calculo.texto`) exibidos como chegam da API.
- Confirmação com UUID por abertura do diálogo e reutilização da chave nas tentativas; mostra notas excluídas e o valor final da cessão.
- Formulário do gestor gerado pelo catálogo. O método efetivo vem do YAML e é somente leitura; os parâmetros sobrescritos pelo YAML também ficam bloqueados.
- Prévia/publicação/histórico com os corpos e endpoints combinados; o catálogo pode marcar parâmetros como opcionais.
- A v1.4 define a data-base por nota quando há `DATA_AQUISICAO`, a concentração estática por fundo e a duplicidade contra cessões confirmadas na análise. Essas decisões são do Go; a tela apresenta os resultados recebidos.

O contrato também documenta `resultado.json`, métricas e um serviço separado de IA. Esta interface não oferece ações para essas rotas. A presença delas na referência não significa cobertura de tela ou do mock.

O contrato não define uma rota para listar lotes. Por isso **Meus lotes** reúne até 100 IDs enviados neste navegador, com nome opcional, arquivo original e data do envio, e busca cada resumo pelo endpoint existente. Esse histórico de metadados não guarda notas nem conteúdo de CSV; os snapshots do mock usam armazenamento separado, descrito abaixo.

## Mocks

Ativados explicitamente por `npm run demo` (`.env.demo`) ou por `VITE_MOCK_API=true` no build. Os handlers MSW respondem às chamadas Fetch e XHR na própria página, com `@mswjs/interceptors`; o cliente mantém as mesmas rotas da API real. Isso dispensa um Service Worker ativo para a demonstração. Os links CSV buscam o endpoint e baixam o Blob original no modo demo; o modo Go mantém o download nativo.

No modo conectado, o progresso vem do envio XHR para a API. No modo demo, o transporte, as etapas e as respostas financeiras são simulações para exercitar a UI. Os mocks não reproduzem o estoque oficial, todos os erros e as decisões financeiras da v1.4, nem validam o gabarito ou o desempenho de 100 mil/500 mil linhas. Detalhes em [`src/mocks/README.md`](src/mocks/README.md).

O mock pode guardar snapshots com dados do CSV no `localStorage` do navegador para reabrir a demonstração; por isso use somente dados fictícios. Snapshots entre abas podem divergir. Isso é separado do histórico local de metadados e não representa sincronização ou persistência do Go.

## Verificar e compilar

```sh
npm test
npm run typecheck
npm run lint
```

Build conectado ao Go (exige API acessível para usar o portal):

```sh
npm run build
npm run preview
```

Build de demonstração, sem backend:

```sh
npm run build:demo
npm run preview
```

## Vercel, se necessário

Vercel é opcional e não faz parte do teste local. Se quiser hospedar apenas o frontend lá, selecione a pasta `frontend` como Root Directory. A configuração incluída usa Vite, saída `dist` e fallback de SPA. Para publicar uma demonstração, use `npm run build:demo`; para conectar ao Go, use o build normal.

Defina `VITE_API_BASE_URL` como a origem pública do backend Go, sem `/api` no final. O Go precisa aceitar CORS da origem do frontend e os headers `X-Gestor-Senha` e `Idempotency-Key`. O upload segue diretamente para o Go, evitando passar grandes CSVs por uma função da Vercel. `API_PROXY_TARGET` só funciona no desenvolvimento local; não é proxy de produção.

## Stack

React 19, Vite 8, TypeScript estrito, Tailwind 4, componentes Radix no padrão shadcn, TanStack Query, Parkinsans local e Vitest. A interface segue os frames de login e dashboard do [Figma do time](https://www.figma.com/design/EFdeJ9T6mnBayCJcbh60aT/Sem-t%C3%ADtulo?node-id=1-913): menu lateral, cores verdes, mascotes e cartões. O envio fica em `/enviar`; os fluxos existentes continuam no mesmo contrato.

O login de demonstração existe somente em `VITE_MOCK_API=true`. No modo conectado ao Go, o portal continua público como define o contrato e a gestão mantém sua verificação por senha. O botão do mascote abre um guia local, sem simular um serviço de IA.

## Novo lote no Figma

O frame `1:1365` foi incorporado em `/enviar`: janela sobre o dashboard, nome do lote, data-base, área de CSV, progresso, erros do arquivo e painel “Como funciona” expansível. Escape e fechar retornam ao dashboard com o foco em “Novo envio”. O layout se adapta a360px. O estado sem lotes usa o mascote do frame `1:1584`.

O nome é um metadado do histórico deste navegador (até80 caracteres). O multipart do contrato permanece com `arquivo` e `data_base`, e o arquivo original não é renomeado. Os cartões, a busca de lotes e o contexto do resultado exibem o nome escolhido, com compatibilidade com envios anteriores.


## Análise do documento

O loading segue o frame Figma1:1631: vídeo do robô, progresso e contador de linhas dentro do modal, com o dashboard desfocado. O upload usa o progresso real do XHR; depois de aceito, o frontend consulta o lote a cada500ms e só abre o resultado ao terminar. Fechar/Cancelar interrompe o acompanhamento no navegador; um lote já enviado continua no histórico. Falhas de consulta podem ser tentadas novamente sem reenviar o arquivo.

No modo demo, cada nova abertura e uso do exemplo inicia uma simulação de9,5segundos (500ms na fila e cinco etapas de1,8s). É usada a chave opcional de idempotência do contrato, estável nas tentativas da mesma ação/data. Os bytes do CSV e o envio normal não mudam; no modo Go o exemplo conserva o comportamento anterior. O prazo de demonstração não atrasa nem simula respostas da API real. Movimento reduzido usa poster estático.


## Grade compartilhada do portal

Dashboard, Meus lotes, Resultado e demais páginas usam `.container`, com largura máxima de **1280px**. A grade se centraliza no espaço após a navegação lateral e usa margens mínimas de64px no desktop,40px até1100px e20px até700px. Rodapé, acesso de perfil e conteúdo da barra de confirmação seguem os mesmos limites.

A fonte dessas medidas é `src/app-layout.css`; altere os tokens uma vez para atualizar o sistema inteiro. O Resultado mantém a superfície da barra de confirmação ampla, com seu conteúdo alinhado à grade. A reserva estável da barra de rolagem e a compensação dos modais evitam deslocamentos laterais ao navegar ou abrir diálogos.

Todas as páginas do portal usam o componente `PageHeader` (`src/components/page-header.tsx` e `.css`), seguindo a Dashboard: título e descrição à esquerda e seletor Usuário/Gestor à direita. Ações e informações específicas ficam abaixo. O espaço superior é compartilhado em64/48/36px; em telas pequenas o seletor quebra para uma linha própria. Loading, erros e404 preservam o mesmo cabeçalho. O login de demonstração e os títulos dos modais mantêm sua composição própria.


## Landing page

A apresentação pública fica em `/`, com base no Figma LP `6:1335`. O portal começa em `/dashboard`; `/enviar`, `/lotes`, `/regras` e `/gestor` mantêm seus fluxos. No modo demo, o acesso preserva o destino de envio ou consulta após entrar. O marcador de demonstração continua sem representar autenticação de produção.

O conteúdo resume envio CSV, análise, motivos de recusa e confirmação da cessão. Fred apresenta dúvidas em respostas expansíveis, sem serviço de IA. Os números da prévia são rotulados como ilustrativos. Assets locais em `public/figma/landing/`; fotografias WebP com carregamento sob demanda, exceto a imagem principal.


## Cards e filtros de Meus lotes

A página reutiliza os cards e o botão verde Novo envio da Dashboard. Busca por nome, arquivo, ID ou data-base; situação; data-base exata e ordenação por envio ou maior valor líquido. Os campos seguem os inputs do login Figma, com60px e raio24px. Filtros complementares abrem pelo botão Filtros, sem perder valores ao recolher.

A coleção continua sendo o histórico dos últimos100 envios deste navegador; os filtros operam sobre os resumos já consultados, sem buscar notas nem inventar uma rota de listagem. Resumos indisponíveis mantêm nova tentativa; dados em cache preservam o último valor. A grade tem3/2/1colunas e os cards crescem para acomodar contagens maiores.
