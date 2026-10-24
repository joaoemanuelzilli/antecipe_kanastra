# Registro de prompts (parte 2: assistente de IA)

Continuação de [`prompts.md`](prompts.md), com os prompts feitos **em paralelo noutra máquina do time** a partir de 24/10/2026 12:07. O `prompts.md` segue com o backend. Os dois usam o horário de Brasília (UTC−3), então dá para intercalar as entradas em ordem cronológica.

- **Ferramenta:** Claude Code (CLI), modelo Claude Opus 5.5 (`claude-opus-5-5`)
- **Regra:** nenhum segredo e nenhuma linha do CSV entram neste arquivo.

Cada entrada tem o prompt exato, o que a IA fez e o que o time fez com a resposta. Mensagens que o time mandou enquanto a IA ainda trabalhava aparecem dentro da entrada em andamento, na ordem em que chegaram.

---

## #1 · 24/10/2026 ~12:07 · Entender o projeto nesta máquina

**Prompt (time → Claude Code):**

> entenda toda a estrutura e explore todo o contexto do projeto

**O que a IA fez:**

1. Leu os documentos: `docs/plano.md`, `docs/api.md` (contrato v1.1), `docs/prompts.md` e o plano de 14 tarefas em `docs/superpowers/plans/`.
2. Leu o edital (critérios e pesos) e o PDF "Antecipe Claro", que é a referência de UX para o front.
3. Leu todo o código Go: domínio e portas, motor, validadores, preço, textos de recusa, lotes, cessão, gestor, Postgres, REST, CSV e observabilidade.
4. Viu que o `origin/main` tinha 4 commits a mais (cessão e gestor) e leu o código deles depois do `pull`.
5. Conferiu as ferramentas da máquina: Node 24 e pnpm instalados, sem Go, Docker parado.
6. Entregou um resumo com o que estava pronto, o que faltava (README, front, teste de logs, relatório) e pontos de atenção para o front:
   - o progresso é por etapa e volta a zero a cada etapa;
   - há códigos de erro que não estavam no `api.md`.

**O que o time fez com a resposta:** decidiu abrir este arquivo e construir o assistente de IA (entrada #2).

---

## #2 · 24/10/2026 ~12:15 · Assistente de IA com Agno e AgentOS

**Prompt (time → Claude Code):**

> vamos criar um prompts2.md para ir continuando as entregas
>
> o que vamos entregar agora é um agente de IA que vai ficar no canto direito da plataforma, a ideia é o usuário conseguir interagir com esse LLM e ter respostas em linguagem natural ao invez de ter que procurar um documento ou etc.
>
> como foi o processamento do lote x?
> pq o documento y deu erro?
> quantos documentos foram recusados?
>
> perguntas desse tipo, vamos construir um agente de ia básico usando o framework agno e estruturando de forma organizada aqui no repo em um modulo ai por exmeplo, referencia: https://github.com/agno-agi/agentos-docker

**Mensagens que o time mandou durante o trabalho, em ordem:**

> a ideia é ser em docker tb

> vamos utilizar api do deepseek

> isso a ideia é usar a api do agentos por debaixo, acredito que seria um novo serviço

> vamos usar na vdd o openrouter

**O que a IA fez:**

1. Leu o template `agentos-docker` (clone local) e a documentação do Agno pelo MCP oficial. Conferiu no código instalado do Agno 3.1.3:
   - a classe `OpenRouter`;
   - `Toolkit` com ferramentas assíncronas;
   - o `RunContext` com `dependencies`;
   - o endpoint `POST /agents/{id}/runs`, que aceita `dependencies` em JSON;
   - a telemetria.
2. Trocou o provedor do modelo duas vezes, conforme as mensagens (Claude → DeepSeek → OpenRouter). O modelo ficou isolado em `criar_modelo()`, com o id em `AI_MODELO` e padrão `deepseek/deepseek-v4-flash`. O id foi conferido no catálogo público do OpenRouter: aceita ferramentas e custa US$ 0,09 / 0,18 por milhão de tokens.
3. Criou o serviço `ai/` no estilo do template: `app/` (AgentOS), `agents/`, `tools/` e `db/`, com pastas em inglês e nomes em português, como no backend.
4. **Desenho:** o agente só consulta a API Go (4 ferramentas de leitura) e não acessa as tabelas do backend.
   - O front manda o lote aberto na tela em `dependencies.lote_id`.
   - As recusas usam os textos oficiais da API.
   - Valores e datas já saem formatados.
   - As sessões do chat ficam no Postgres, no schema `ai`.
5. **Docker:** serviço `ai` no `docker-compose.yml` (porta 7777) e `/ai/` no nginx, sem buffer para o streaming. No `.env.example` entrou `OPENROUTER_API_KEY` vazia. Sem a chave, o `make up` continua funcionando, avisa que o assistente está desligado, e `/assistente/status` responde `disponivel: false`.
6. **Dependências:** Agno fixado em 3.1.3 (12/10). Todas as dependências foram resolvidas com `uv pip compile --exclude-newer 2026-10-14`, para não puxar pacotes publicados na véspera.
7. **Logs:** JSON só com metadados (rota como padrão, status, duração e `request_id`), telemetria do Agno desligada e logs do Agno em WARNING. O teste de logs achou um vazamento: o `httpx` logava a URL com a busca do usuário (nome e CNPJ). A IA corrigiu subindo `httpx`, `httpcore` e `openai` para WARNING, e o teste ficou cobrindo esse caso.
8. **Testes:** 18 testes, sem rede e sem chave (`make test-ai`), com API simulada. Cobrem ferramentas, formatação, o app sem chave, o log de acesso e logs sem dados. O ruff passa sem avisos.
9. **Documentação:** `ai/README.md` (contrato do painel para o front, com exemplo de SSE, decisões e limitações), seção "Assistente de IA" no `README.md`, `make logs-ai` e `make test-ai`.
10. **Validação com Docker:** abriu o Docker Desktop e rodou o `make up` numa pasta sem `.env`. Os 5 serviços subiram, o que também cumpre a pendência do backend ("validar o `make up` com Docker"). Depois:
    - enviou o `data/exemplo.csv` e rodou as 4 ferramentas de dentro do container `ai` contra a API real: totais, motivos, recusa por prazo com o texto oficial, cálculo do preço, regras, formato e lote inexistente;
    - conferiu o nginx (`/ai/`), o schema `ai` separado do `public` no mesmo Postgres e os logs em JSON sem dados.
11. **Achado na validação:** sem chave, o AgentOS responde HTTP 200 com `"status": "ERROR"` e uma mensagem técnica em inglês no `content` (no streaming, o evento `RunError`). O contrato do painel no `ai/README.md` passou a pedir que o front confira o `status` e mostre uma mensagem amigável.
12. **Pergunta do time durante o trabalho** ("Como que ficou a parte do agente em questao de infra? Vamos utilizar o mesmo postgres do resto da aplicação para persistencia das sessoes?"): a IA explicou a infra. É o mesmo Postgres e o mesmo usuário, com as sessões no schema `ai`, separadas das tabelas do backend no `public`. O serviço `ai` depende do Postgres saudável e da API, e o nginx depende do `ai`.
13. **Decisão registrada:** o plano original (`plano.md`, seções 4 e 17.8) dizia "sem IA no app". O time mudou isso agora. A IA continua fora da decisão (aprovar, recusar ou precificar continua 100% determinístico no Go) e só explica o que a API já calculou.

**O que o time fez com a resposta:** aprovou o assistente e, com o backend já sem commits pendentes, pediu para sincronizar com a origin, atualizar o plano e os docs e fazer commit e push (entrada #3).

---

## #3 · 24/10/2026 ~12:38 · Sincronizar, atualizar os docs e publicar

**Prompt (time → Claude Code):**

> o backend ja finalizou todos os commits, de git pull para ficar atualizado com a origin, atualize o plano e docs e commit + push

**O que a IA fez:**

1. **Sincronizou com a origin.** Guardou as mudanças locais no stash, puxou 5 commits do backend (revisão final, scripts de carga e concorrência, `.dockerignore`, ajustes de healthcheck e `restart`) e reaplicou as mudanças sem conflito. Conferiu que o backend não mudou nenhum campo de JSON que as ferramentas do assistente leem (só textos de recusa). Pôs `restart: unless-stopped` também no serviço `ai`, como o backend fez em `api` e `web`.
2. **`docs/plano.md`:**
   - decisão 6 (assistente de IA);
   - não-meta revista ("IA na decisão", não "IA no app");
   - stack, arquitetura com a seção "Assistente de IA", schema `ai` no modelo de dados, rotas do assistente, política de logs, painel do front, entregáveis e risco com plano B;
   - decisão 8 revista ("IA só para explicar, nunca para decidir") e decisão 14 (mesmo Postgres; dados enviados ao provedor do modelo).
3. **`docs/api.md` v1.3:** seção do assistente para o front (status, runs, `dependencies.lote_id`, erro com HTTP 200 e `status: "ERROR"`), apontando para o `ai/README.md`.
4. **`docs/relatorio.md`:** o assistente entra em "O que construímos", "Tecnologias", decisão 9 e "O que ficou de fora".
5. **`docs/prompts.md`:** aviso no topo apontando para este arquivo, porque o formulário pede um só caminho.
6. Rodou `make test-ai` e o ruff de novo, fez o commit e o push na `main`.
7. **Push recusado (403)** porque a conta desta máquina ainda não era colaboradora. A IA achou um convite pendente do dono do repositório e perguntou antes de aceitar. Com o ok do time, aceitou só esse convite pelo `gh` e o push passou (`34f92b6..606b2b9`).

**O que o time fez com a resposta:** colou a visão de produto (cedente, sacado e fundo; por que existem as regras; os dois papéis, empresa e gestor) e perguntou se o agente estava alinhado (entrada #4).

---

## #4 · 24/10/2026 ~12:48 · O agente está alinhado com a visão do produto?

**Prompt (time → Claude Code):** "A ideia da plataforma é basicamente pro cedente e pro fundo:", seguido de um texto colado com a visão do produto:
- quem são cedente, sacado e fundo, com o exemplo da Ana;
- por que cada regra existe (risco do fundo, concentração, duplicidade como fraude);
- o fluxo de subir o CSV, simular e confirmar a cessão, com o desafio de UX na explicação da recusa;
- os dois papéis: empresa (sem login) e gestor do fundo (senha no header).

O prompt termina com: "O agente ficou alinhado?"

**O que a IA fez:** comparou o agente (instruções, ferramentas e rotas da API que ele alcança) com a visão colada. A conclusão foi "parcialmente": o fluxo do cedente está coberto, mas faltam o "porquê" das regras e do desconto, o tratamento por papel e a parte do gestor (histórico de versões e prévia de impacto). Propôs as mudanças em ordem de prioridade, sem alterar código.

**O que o time fez com a resposta:** recusou as quatro propostas, cada uma com um motivo (entrada #5).

---

## #5 · 24/10/2026 ~12:52 · Decisões sobre as propostas de alinhamento

**Prompt (time → Claude Code):**

> 1. acredito que é regra de negocio da empresa/fundo não entraria no prompt
> 2. ainda vamos criar junto com autentificação
> 3. acho q n precisa
> 4. n entra agora depois pensamos

**O que a IA fez:** não mexeu no código do agente e registrou as decisões:
1. O porquê das regras é regra de negócio do fundo e **não entra no prompt**. O agente continua explicando as recusas só com os textos oficiais da API.
2. A distinção entre empresa e gestor vem **junto com a autenticação**, mais tarde. Entrou em "Limitações" no `ai/README.md`.
3. Histórico de versões e prévia de impacto para o gestor: **não serão feitos**.
4. Perguntas sobre a carteira inteira do fundo: **fora do escopo agora**.

**O que o time fez com a resposta:** aceitou o registro e pediu para sincronizar com a origin (entrada #6).

---

## #6 · 24/10/2026 ~14:33 · Sincronizar e publicar as imagens no Docker Hub

**Prompt (time → Claude Code):**

> vamos sincronizar o repo com a origin

**Mensagem que o time mandou durante o trabalho:**

> vamos enviar as imagens para o dockerhub pra mim fazer deploy pela railway, o login por github ta desabilitado

**O que a IA fez:**

1. **Sincronizou com a origin.** Guardou as mudanças locais no stash, puxou 8 commits do backend e reaplicou as mudanças sem conflito. Os commits trazem o motor alinhado ao gabarito oficial, a rota `resultado.json`, o deploy na Railway e o assistente protegido por chave interna (só o nginx a injeta, sem porta 7777 no host e só duas rotas liberadas em `/ai/`).
2. Conferiu que nenhum campo de JSON lido pelas ferramentas do assistente mudou (só os textos de concentração) e rodou `make test-ai`: 20 testes passam, contando os 2 novos de segurança.
3. **Docker Hub:** leu o `docs/deploy-railway.md` e o alvo `make publicar` (monta as 3 imagens em `linux/amd64` e as envia). Conferiu que nenhuma imagem leva segredo (`.env` fora do contexto de build) e que o repositório é público, então as imagens públicas não expõem nada a mais. A máquina não tinha login no Docker Hub, então a IA pediu ao time o `docker login` e adiantou a montagem das imagens para aquecer o cache.
4. O `! docker login` falhou, porque roda sem terminal interativo. A IA pediu o login num terminal comum ou no Docker Desktop, e não pelo `!` com o token, para o token não ficar no histórico.

**Mensagens que o time mandou depois, em ordem:**

> loguei, pode publicar

> enviou?

> o de front ainda vai ser subido pro repo, é certo a gente ja enviar agora?

> mergeamos o frontend

**O que a IA fez em seguida:**

5. Rodou o `make publicar` com `leosaraiva11/…:v1`. O primeiro `docker push` travou por 16 minutos sem progresso. A IA cancelou e enviou cada imagem separada, e as três chegaram: `api`, `web` (página de espera) e `ai`, todas em `linux/amd64`.
6. Sobre enviar antes do front: sim. `api` e `ai` não dependem dele. A `web` provisória já serve para montar e testar a Railway antes da entrega, e quando o front entrar basta publicar só a `web` com tag nova.
7. Com o front mergeado, viu que o merge só mudou `frontend/` (as `api` e `ai` publicadas continuam valendo) e que o build de produção usa a API real, não os mocks. Tentou montar a `web:v2` duas vezes, e o `npm ci` falhou nas duas, sempre no mesmo pacote (`UNABLE_TO_VERIFY_LEAF_SIGNATURE`).
8. **Causa:** a rede do evento tem um firewall Fortinet que intercepta o TLS só do `registry.npmjs.org`, com um certificado próprio. PyPI, Docker Hub, GitHub e Railway passam limpos. A IA **não** desligou a verificação de TLS nem confiou no certificado do firewall, porque isso abriria espaço para pacote adulterado entrar na imagem. Propôs montar a `web` noutra rede (ex.: hotspot do celular).

**O que o time fez com a resposta:** desistiu da imagem da `web` e decidiu publicar o front na Vercel (entrada #7).

---

## #7 · 24/10/2026 ~15:10 · Deploy do front na Vercel

**Prompt (time → Claude Code):**

> vou fazer deploy na vercel, por onde seguir? esquece o caminho de docker image

**O que a IA fez:** leu o `frontend/vercel.json`, a seção "Vercel" do `frontend/README.md`, o cliente HTTP do front (`VITE_API_BASE_URL`) e o CORS do Go (`CORS_ORIGINS`, origem exata). Explicou a divisão:
- **Front na Vercel**, com Root Directory `frontend` e `VITE_API_BASE_URL` apontando para a API pública. O build roda nos servidores da Vercel, então o firewall da rede do evento deixa de atrapalhar.
- **API e Postgres continuam fora da Vercel.** A função da Vercel aceita no máximo 4,5 MB por requisição e não mantém processamento em segundo plano, e os lotes chegam a 100 MB e são processados de forma assíncrona. A imagem `antecipe-api:v1` já publicada continua servindo para a Railway.
- O assistente fica de fora por enquanto, porque o front ainda não chama `/ai/`.

**Mensagens que o time mandou depois, em ordem:**

> parece q a vercel suporta a api go e fastapi

> (texto colado pela Vercel pedindo para configurar o repositório como um projeto com vários serviços: `vercel.json` com `services` para `ai`, `backend` e `frontend`, bindings entre eles e confirmação antes de terminar)

> (resposta à pergunta da IA) na versao paga do vercel da?

> vamos subir a api e a ai por imagem docker mesmo entao

> pela railway

**O que a IA fez em seguida:**

1. Leu a documentação dos Services, do runtime Go e dos limites das funções da Vercel. Cada serviço roda como uma Vercel Function, com os mesmos limites: corpo de no máximo **4,5 MB** em todos os planos, e 300 s (Hobby) ou 800 s (Pro) de duração.
2. Mediu o `exemplo.csv` (~166 bytes por linha): cabem cerca de 28 mil linhas em 4,5 MB, então os lotes de 100 mil e 500 mil linhas receberiam `413`. A fila e o andamento do lote ficam em memória (`batch.go`), o que também não combina com funções que podem ser congeladas depois de responder.
3. **Não** criou o `vercel.json` com services. Perguntou o caminho e respondeu que o plano pago só aumenta a duração e a memória, sem mudar o limite de 4,5 MB. Contornar isso exigiria envio direto ao Blob, filas e andamento no banco, e não ficaria pronto antes das 18:00.
4. **Decisão do time:** front na Vercel, e API, assistente e Postgres na Railway com as imagens `v1`, que já estão no Docker Hub e não mudaram com o merge do front. A IA passou o roteiro dos dois painéis. Conferiu que a API escuta em `:8080` fixo (o domínio da Railway precisa apontar para essa porta) e que o repositório importado na Vercel (`leo-saraiva11/antecipe_kanastra`, privado) já tem o front mergeado e não tem `.env`.

**O que o time fez com a resposta:** subiu os serviços na Railway e foi trazendo os erros para a IA (entrada #8).

---

## #8 · 24/10/2026 ~15:40–16:36 · Deploy na Railway, erro por erro

**Mensagens do time, em ordem (resumidas quando eram logs ou prints):**

> (curl de `/api/saude` na Railway voltando 502)

> 3. mande as variaveis do db e agno telemtry para mim colar por raw editor

> (print da aba Data do Postgres: "Attempting to connect to the database...")

> (logs do Postgres pronto, depois logs da API: "api encerrada com erro")

> deu certo / as variaveis do postgres precisa ficar entre "" / sem ser por ref

> (traceback do `ai`: `invalid literal for int() with base 10: ''`) / host é url?

> (erro da API: `lookup postgres.railway.internal ... no such host`) / qual region deixo os containers?

> tava colocando em projeto diferente vou juntar td no msm projeto

> qual a docker image do front? / qual region? / ai tem q gerar domain?

> vai ser tudo na railway, o pessoal ja subiu tudo pra main, vamos dar pull aqui e fazer o deploy

> o chat ta com stream né?

> (print do deploy do `web` falhando: imagem `v2` não encontrada) / O chatbot da plataforma já está funcionandao com o serviço de ai usando o endpoint de send?

> (logs do `ai` no ar, variáveis do `web`) / nao to conseguindo enviar mensagem no chat

> eu usei o chat aqui falhou 2x e so na terceira respondeu

**O que a IA fez:**

1. **502 na API.** Rodou a mesma imagem `antecipe-api:v1` localmente: subiu, carregou o estoque e respondeu `ok`. Logo, o problema estava na configuração da Railway. Os logs da API mostravam `banco indisponível` depois de 30 tentativas (`postgres.go`), com o host `postgres.railway.internal` dando `no such host`. **Causa:** a API e o Postgres estavam em projetos diferentes, e a rede privada e as referências `${{…}}` só valem dentro do mesmo projeto e ambiente. O time juntou tudo num projeto só, em US East.
2. **Variáveis.** Passou os blocos do Raw Editor (entre aspas), explicou como separar a `DATABASE_URL` em `DB_HOST`, `DB_PORT` etc., e diagnosticou o traceback do `ai` (`DB_PORT` vazio). Indicou `PORT="8080"` para o domínio público da API.
3. **Pull e imagens `v2`.** Puxou 44 commits (front completo, situação cadastral final, migração `0003`). Conferiu que o novo `frontend/Dockerfile` cai para o espelho `registry.yarnpkg.com` quando a rede intercepta o npm, mantendo o TLS verificado e o hash do lockfile, e que o front chama `/api` e `/ai` na mesma origem, com streaming. Montou e publicou `antecipe-api:v2`, `antecipe-web:v2` (com o React) e `antecipe-ai:v2`.
4. **Chat sem resposta (504).** Testou pelo domínio do `web`: front, `/api/saude` e o bloqueio de `/ai/sessions` (404) funcionavam, mas `/ai/assistente/status` dava 504 depois de 60 s. Pediu `AI_HOST="::"` no `ai`, para escutar também em IPv6. Como o 504 continuou, pediu as variáveis do `web`. **Causa:** o `AI_UPSTREAM` apontava para o domínio público do `ai` na porta 7777, que a borda da Railway não atende. Trocado para `antecipe-ai.railway.internal:7777`.
5. **Conferência final (19:35 UTC):** status 200 três vezes e três perguntas com streaming pelo site ("Qual o prazo máximo que o fundo aceita?"). As três vieram completas, sem `RunError`, e o `ai` consultou as regras na API pela rede interna.

**O que o time fez com a resposta:** tirou dúvidas sobre onde ficam `AI_UPSTREAM` e `AI_HOST` e pediu uma revisão completa do agente (entrada #9).

---

## #9 · 24/10/2026 ~16:45 · Revisão do agente em produção

**Prompt (time → Claude Code):**

> revise todas as partes do agente se estão funcionais, no caso da tool consultar_Lote se pega o lote q esta aberto por exemplo e se todas as outras tools estão alinhadas

**O que a IA fez:**

1. **Front.** O chat (`assistant-chat.tsx`) lê o id da rota `/lote/:id` com `useLocation`, que se atualiza a cada mudança de página, e o envia em `dependencies.lote_id`. Fora dessa página, não manda lote.
2. **Contrato.** Enviou o `data/exemplo.csv` pelo site em produção e comparou o JSON real de `/api/lotes/{id}`, `/notas`, `/regras` e `/config` com os campos que as ferramentas leem: todos batem. Testou os filtros `status`, `motivo`, `busca` e `pagina` direto na API.
3. **Agente de ponta a ponta**, com 10 perguntas pelo site, conferindo as ferramentas chamadas nos eventos SSE:
   - lote aberto: `consultar_lote` usou o lote da tela (23 linhas, 15 aprovadas, 8 recusadas, R$ 73.512,68 líquido, iguais à API);
   - sem lote aberto: pediu o id;
   - id citado na pergunta: teve prioridade, e com lote inexistente o agente avisou;
   - "por que o documento 900019 deu erro?": `buscar_notas` com o texto oficial de PRAZO_MAXIMO;
   - cliente por nome, aprovadas em duas páginas, regras, formato e pergunta fora do tema (recusou).
4. **Achado:** perguntado sobre a coluna `DATA_AQUISICAO`, o agente inventou que era "a data em que a empresa adquiriu o título". No motor (`motor.go:183`), é a data em que o fundo adquire a nota e vira a data-base dela. A ferramenta não trazia essa explicação. A IA a incluiu em `consultar_formato_do_arquivo` com o texto do contrato v1.4, criou um teste (21 passam, ruff limpo) e publicou `antecipe-ai:v3`.

**O que o time fez com a resposta:** _(preencher)_
