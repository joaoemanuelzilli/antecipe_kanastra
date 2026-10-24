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

**O que o time fez com a resposta:** _(preencher)_
