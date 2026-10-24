# Antecipe

Portal de antecipação de recebíveis do **ANTECIPE FIDC** (Hackathon Kanastra · 24/10/2026).

A empresa sobe um CSV de notas e vê, em segundos, quais notas o fundo compra, quanto recebe e, para cada recusa, o motivo e o que fazer. O gestor do fundo muda as regras sem deploy. Uma nota nunca é vendida duas vezes, nem com duas confirmações simultâneas.

| Lote | Tamanho | Processamento completo (upload → resultado gravado) |
|---|---|---|
| 100 mil linhas | 16 MB | **1,0 s** |
| 250 mil linhas | 39 MB | **2,5 s** |
| 500 mil linhas | 77 MB | **4,9 s** |
| **1 milhão de linhas** (1.005.540) | 196 MB | **14,1 s** (no Docker, pelo front; pico de 1,3 GB na API) |

_Notebook com 16 núcleos, Postgres 18. Medido com `make carga`._

---

## Testar somente a interface com dados fictícios

A contribuição do frontend está em publicação na branch `codex/frontend-contract-v1.1` do fork [`matteuszu/antecipe_kanastra`](https://github.com/matteuszu/antecipe_kanastra), para PR a este repositório. A partir de um clone que contenha essa branch, use Node.js >=22.12.0:

```bash
git switch codex/frontend-contract-v1.1
cd frontend
npm ci
npm run demo
```

Abra `http://localhost:5173` (ou a porta informada pelo Vite). O arquivo `.env.demo` é versionado: não precisa criar `.env.local`, fornecer segredos ou iniciar Go, Supabase, Docker ou Vercel. Entre com dados fictícios ou use **Explorar demonstração**; a senha pública do gestor mock é `demo-gestor`. Valores, etapas e cessões são simulados, sem movimentação financeira. O mascote desta interface oferece um guia local, sem chamadas de IA.

Para conectar ao backend já implementado, use `npm run dev` com a API disponível; para a stack completa, siga os passos abaixo. O build normal conecta ao Go; `npm run build:demo` prepara a demonstração isolada. Detalhes e limites dos mocks estão no [README do frontend](frontend/README.md). Os testes de UI desta contribuição não substituem nem constituem execução dos testes de Go descritos neste documento.

## Como rodar

**Pré-requisitos:** Docker 24+ com Docker Compose v2 e `make`. Não precisa de Go nem de Node.

```bash
make up
```

Na primeira vez leva alguns minutos (build das imagens). O comando faz tudo:
- cria o `.env` com senhas aleatórias (ele não vai para o git);
- sobe Postgres, API, assistente de IA, front e Jaeger;
- aplica as migrations e carrega o estoque do desafio (100.554 notas).

No fim ele mostra:

```
  Front:            http://localhost:3000
  API:              http://localhost:8080/api/saude
  Traces (Jaeger):  http://localhost:16686
  Métricas:         http://localhost:8080/metrics
  Assistente (IA):  http://localhost:3000/ai/assistente/status  (só via nginx)
  Senha do gestor:  <gerada no .env>
```

O assistente de IA é opcional: sem `OPENROUTER_API_KEY` no `.env`, todo o resto funciona e o `make up` avisa que ele está desligado (veja [Assistente de IA](#assistente-de-ia)).

Outros comandos (`make` sozinho lista todos):

| Comando | O que faz |
|---|---|
| `make down` | Para tudo (os dados ficam no volume) |
| `make limpar` | Para tudo e apaga o banco local (volta ao estoque original) |
| `make logs` / `make logs-ai` | Acompanha os logs da API / do assistente de IA |
| `make gerar N=500000` | Gera um lote de teste em `amostras/` |
| `make carga` | Mede lotes de 100k, 250k e 500k linhas |
| `make demo-concorrencia` | Duas cessões simultâneas com notas em comum, sem venda duplicada |
| `make check` | Procura segredos no repositório (gitleaks) |
| `make publicar REGISTRO=…` | Monta e envia as imagens para um registry (deploy na Railway) |
| `make test` / `make test-integracao` | Testes de unidade / todos, inclusive banco e ponta a ponta (precisam de Go 1.27) |
| `make test-ai` | Testes do assistente de IA (precisa do `uv`; não usa rede nem chave) |

### Arquivos para testar

- `data/exemplo.csv`: 23 notas, 13 aprovadas e uma recusa de cada tipo. É bom para ver as mensagens. A última nota já está no estoque: é aprovada na simulação, mas fica fora da cessão, porque não se vende a mesma nota duas vezes.
- **O próprio estoque** (`data/estoque_antecipe.zip`, o lote de 100 mil dos jurados). O resultado bate com o gabarito oficial (veja abaixo) e pode ser baixado no mesmo formato em `GET /api/lotes/{id}/resultado.json`.
- **1 milhão de linhas com resposta conhecida:**
  - gere com `scripts/lote-ampliado.sh > amostras/lote_1m.csv`: são 10 cópias do estoque, e nas cópias 2 a 10 o número da nota ganha `-C1` a `-C9`;
  - a concentração por fundo não muda, então cada linha tem a resposta do gabarito para a linha original;
  - depois de processar, baixe o `resultado.json` e confira com `python3 scripts/comparar-gabarito.py resultado.json`;
  - resultado medido: 974.600 linhas iguais ao gabarito, 30.940 diferentes só na situação cadastral e 0 em outras regras.
- `make gerar N=500000`: lote grande com mistura realista, determinístico pela semente:
  - ~80% aprováveis;
  - casos de cada regra;
  - linhas repetidas e linhas quebradas.

### Conferência com o gabarito oficial

O Notion publicou o `resultado_esperado_100k.json` (o estoque avaliado como lote). O teste `backend/test/gabarito` e o ponta a ponta `TestGabaritoPelaAPI` comparam o nosso resultado com ele, linha a linha:

| | Linhas |
|---|---|
| Idênticas ao gabarito (status e motivos, na mesma ordem) | **97.460** |
| Diferem **só** na situação cadastral | 3.094 |
| Diferem em qualquer outra regra | **0** |

A situação cadastral vem do `data/cnpj_situacao_mock.csv` publicado no Notion. O gabarito foi gerado com outra fonte de situação:
- ele barra 87 sacados que o CSV diz estarem `ATIVA` ou que nem estão no CSV;
- ele aprova 42 sacados que o CSV marca como `INAPTA`, `SUSPENSA` ou `BAIXADA`.

Com a situação do próprio gabarito, o nosso motor reproduz o resultado esperado sem nenhuma diferença.

> As cessões confirmadas entram na carteira do fundo e mudam a concentração. Para reproduzir o gabarito depois de vender notas (por exemplo, depois do `make demo-concorrencia`), rode `make limpar && make up` para voltar a um banco novo.

**Conferência só com SQL, no banco:** a view `resultado_gabarito` devolve cada nota no formato do gabarito.

```sql
SELECT linha, id, status, barrado_por FROM resultado_gabarito WHERE lote_id = '<lote_id>' ORDER BY linha;
```

`make conferir-gabarito LOTE=<lote_id>` carrega o gabarito no Postgres e cruza com a view. Resultado medido com banco novo:
- 100.554 linhas;
- 97.460 idênticas;
- 3.094 diferentes só na situação cadastral;
- 0 diferenças em outras regras;
- aprovadas: 86.713 no banco contra 85.215 no gabarito. A diferença vem toda da situação cadastral.

Também dá para conferir pela linha de comando:

```bash
docker compose run --rm --no-deps -T api sh -c \
  'unzip -p data/estoque_antecipe.zip | antecipe validar --formato json' > nosso.json
```

---

## Usando pela API

Contrato completo: [`docs/api.md`](docs/api.md).

```bash
# 1. envia o lote (responde 202 com o lote_id)
curl -F arquivo=@data/exemplo.csv http://localhost:8080/api/lotes

# 2. acompanha o andamento (etapa, linhas processadas, %) e, no fim, os totais e os motivos
curl http://localhost:8080/api/lotes/<lote_id>

# 3. notas paginadas, com filtro por status, motivo e busca
curl "http://localhost:8080/api/lotes/<lote_id>/notas?status=recusada&motivo=PRAZO_MAXIMO"

# 4. resultado em CSV (com motivo e o que fazer)
curl -o resultado.csv http://localhost:8080/api/lotes/<lote_id>/notas.csv

# resultado no formato do gabarito do desafio (linha, id, status, barrado_por)
curl -o resultado.json http://localhost:8080/api/lotes/<lote_id>/resultado.json

# 5. confirma a venda das aprovadas (idempotente)
curl -X POST -H "Idempotency-Key: $(uuidgen)" http://localhost:8080/api/lotes/<lote_id>/cessao

# validação síncrona, sem gravar (JSON ou o próprio CSV, até 10 mil notas)
curl -H "Content-Type: text/csv" --data-binary @data/exemplo.csv http://localhost:8080/api/validar

# regras: ver, testar o impacto e publicar uma nova versão (gestor)
curl http://localhost:8080/api/regras
curl -X POST -H "X-Gestor-Senha: <senha>" -d '{"regras": {...}}' http://localhost:8080/api/regras/previa
curl -X PUT  -H "X-Gestor-Senha: <senha>" -d '{"regras": {...}, "autor": "Rafael"}' http://localhost:8080/api/regras
```

### Pela linha de comando (filosofia Unix)

O mesmo motor também roda como CLI. Ela lê o CSV no stdin, escreve o resultado no stdout e manda os logs para o stderr:

```bash
docker compose run --rm --no-deps -T api sh -c \
  'antecipe gerar --linhas 500000 | antecipe validar | grep -c ";RECUSADA;"'
```

---

## Juros compostos ou simples

O preço padrão é **juros compostos** (desconto racional composto): `líquido = face ÷ (1 + taxa)^(dias ÷ base)`. Para voltar a juros simples (`desconto = face × taxa × dias ÷ base`), edite [`config/antecipe.yaml`](config/antecipe.yaml):

```yaml
precificacao:
  metodo: DESCONTO_SIMPLES   # ou DESCONTO_COMPOSTO
```

A API relê o arquivo sozinha e o próximo lote já usa o método novo, sem reiniciar. Taxa e base vêm das regras publicadas (2% ao mês, 30 dias no JSON original) e também podem ser sobrescritas no YAML.

| R$ 10.000,00 a 2% ao mês | 30 dias | 60 dias | 180 dias |
|---|---|---|---|
| Composto (padrão) | R$ 9.803,92 | R$ 9.611,69 | R$ 8.879,71 |
| Simples | R$ 9.800,00 | R$ 9.600,00 | R$ 8.800,00 |

---

## Assistente de IA

Um painel no canto direito da plataforma onde a empresa pergunta em linguagem natural, em vez de procurar na tabela:

> "Como foi o processamento deste lote?" · "Por que a nota 725714 foi recusada?" · "Quantas notas foram recusadas por prazo?"

É um serviço à parte (`ai/`), feito com [Agno](https://docs.agno.com) e servido pelo **AgentOS**. O AgentOS **não fica exposto no host**: só o nginx alcança o serviço, pela rede interna do compose, e ele exige a chave `AI_SECURITY_KEY` (gerada pelo `make up` no `.env`). O nginx injeta a chave **só nas duas rotas que o front usa** (`/ai/assistente/status` e `POST /ai/agents/antecipe-assistente/runs`). Ela nunca chega ao navegador, e o resto do AgentOS (`/sessions`, `/config`, `/docs`…) responde 404. O modelo vem do **OpenRouter** (padrão `deepseek/deepseek-v4-flash`). O assistente **só consulta a API do Antecipe**, como o front: não lê as tabelas do backend, não envia lotes, não confirma cessões e não muda regras. Ele repete os textos oficiais de recusa da API, então tela, API e assistente dizem a mesma coisa.

Para ligar, preencha a linha `OPENROUTER_API_KEY=` do `.env` com a sua chave e rode `make up` de novo.

```bash
# pergunta sobre o lote aberto na tela (o front manda o lote_id em dependencies)
curl -F 'message=Quantas notas foram recusadas e por quê?' \
     -F 'session_id=demo' -F 'stream=false' \
     -F 'dependencies={"lote_id":"<id do lote>"}' \
     http://localhost:3000/ai/agents/antecipe-assistente/runs
```

Contrato para o front, ferramentas e decisões: [`ai/README.md`](ai/README.md).

---

## Arquitetura

**Backend em Go 1.27, arquitetura hexagonal.** O domínio e as regras de negócio não dependem de banco, HTTP nem arquivos.

```
backend/
  cmd/api/                 servidor HTTP (só liga as peças)
  cmd/antecipe/            CLI: gerar · exemplo · validar
  internal/
    domain/                entidades (nota, resultado, lote, cessão) e portas (interfaces)
      rules/               regras do fundo: JSON, catálogo, validação de schema, resumo de mudanças
    service/               regras de negócio
      engine/              motor: regras em paralelo → duplicidade → concentração → preço
      validator/           uma função pura por regra do JSON
      pricing/             juros compostos e simples, em centavos
      message/             texto de cada recusa (decisão → regra → consequência → ação)
      batch/               caso de uso: lotes assíncronos com etapas e progresso
      assignment/          caso de uso: cessão (venda) segura contra concorrência
      rulebook/            caso de uso: versões das regras e prévia do impacto
    handler/               adaptadores de entrada (dependências externas)
      rest/                API HTTP (contrato em docs/api.md)
      csvfile/             leitura do CSV pt-BR (BOM, CRLF, Windows-1252, aspas, milhar)
    repository/            adaptadores de saída
      postgres/            pgx + migrations embutidas + COPY
      filesystem/          estoque em zip e situação cadastral (CLI e semente)
    platform/              apperrors · observability (OpenTelemetry) · config (YAML) · document (CNPJ) · format (pt-BR)
    bootstrap/             raiz de composição (usada pela API e pelos testes ponta a ponta)
  test/e2e/                testes ponta a ponta (API + Postgres de verdade)
ai/                        assistente de IA: Agno + AgentOS, ferramentas que consultam a API (Python)
frontend/                  React + Vite (servido por nginx, que repassa /api e /ai)
config/antecipe.yaml       comportamento ajustável (método de preço, data-base, fila)
data/                      estoque, validadores, exemplo e situação cadastral simulada
```

### O caminho de um lote

```
POST /api/lotes ─► grava o arquivo em disco calculando o sha256 (sem guardar 100 MB em memória)
                ─► confere o cabeçalho (422/415 na hora) e a idempotência
                ─► 202 { lote_id }  →  fila (2 lotes em paralelo por padrão)

job:  lendo ──► validando ──► concentracao ──► precificando ──► salvando ──► concluido
      parse em    regras por     ordem fixa      composto ou     COPY no
      fluxo       nota, em       (vencimento,    simples, por    Postgres
                  paralelo       número, linha)  nota, centavos

GET /api/lotes/{id}  ◄─ o front consulta a cada 500 ms: etapa, linhas processadas, %
```

**Venda sem duplicidade (`POST /api/lotes/{id}/cessao`), numa transação:**
1. `FOR UPDATE` no lote: a confirmação repetida devolve a mesma cessão.
2. Advisory lock por fundo: cessões do mesmo fundo entram em fila.
3. A concentração é recalculada sobre o estoque **atual**.
4. `INSERT … ON CONFLICT (chave) DO NOTHING`: nota vendida por outra requisição no mesmo instante sai da cessão com o motivo.

### Stack

| Camada | Escolha |
|---|---|
| Backend | Go 1.27, `net/http`, `log/slog`, pgx v5, shopspring/decimal, go.yaml.in/yaml/v3 |
| Banco | PostgreSQL 18 |
| Observabilidade | OpenTelemetry: traces OTLP no Jaeger, métricas em `/metrics`, `trace_id` em cada log |
| Front | React 19 + Vite 8 + TypeScript (por outra pessoa do time, contra `docs/api.md`) |
| Assistente de IA | Python 3.13, Agno 3.1 + AgentOS, modelo via OpenRouter, sessões no Postgres (schema `ai`) |
| Infra | Docker Compose + Makefile |

---

## Regras

As regras são lidas da **versão publicada** (a versão 1 vem de `data/validadores_antecipe.json`). Só valem as que estão com `ativo: true`, e nenhum número de regra está no código.

| Código | Regra |
|---|---|
| `CNPJ_SACADO_VALIDO` | Dígitos verificadores, inclusive do **CNPJ alfanumérico** (desde jul/2026) |
| `SITUACAO_CADASTRAL_SACADO` | Duas checagens: primeiro o `CNPJ_SACADO_VALIDO` confere se o CNPJ existe de fato (DV); depois a situação vem do `data/cnpj_situacao_mock.csv` oficial e tem que estar nas `situacoes_aceitas`. CNPJ fora do CSV = `ATIVA` (configurável no YAML), como no gabarito |
| `TIPO_RECEBIVEL` | Tipos aceitos (sem diferenciar acento nem maiúsculas) |
| `VALOR_MINIMO` | Valor ≥ mínimo |
| `NAO_VENCIDA` | Vencimento depois da **data-base da nota**: a `DATA_AQUISICAO` dela. Sem essa coluna, vale a data-base do lote (padrão 22/10/2026). |
| `PRAZO_MAXIMO` / `PRAZO_MINIMO` | Dias corridos da data-base da nota até o vencimento (os mesmos dias do preço) |
| `CONCENTRACAO_MAXIMA_SACADO` / `_CEDENTE` | Estática e **por fundo** (o `DOC_FUNDO` de cada linha). A carteira de cada fundo = estoque do fundo, sem as notas que vieram no lote, + notas do lote que passaram nas outras regras. Se um grupo (raiz do CNPJ) passa do limite, todas as notas dele no lote são recusadas. |

Sempre ligadas, porque são integridade e não política do fundo:
- `DUPLICIDADE`: na simulação, nota já vendida pelo portal (cessão confirmada). Na cessão, qualquer nota que já está no estoque de algum fundo fica de fora (`INSERT … ON CONFLICT`), então a mesma nota nunca é vendida duas vezes.
- `DUPLICADA_NO_ARQUIVO`: a mesma nota repetida no arquivo enviado.
- `FORMATO_INVALIDO`: linha ilegível. Só essa linha é recusada, e o resto do lote segue.

Toda recusa vem com título, mensagem e ação, geradas pelo backend. Por isso tela e API dizem a mesma coisa. Exemplo:

> **Prazo acima do limite** · Esta nota vence em 240 dias e o fundo compra notas com até 180 dias. · Você pode antecipá-la a partir de 21/12/2026.

---

## Logs, segredos e erros

- **Logs estruturados** (JSON, `log/slog`) só com metadados: rota, status, duração, `lote_id`, etapa e contagens. **Nenhum dado do CSV** aparece neles, e o teste `TestLogsNaoVazamDados` garante isso.
- **Segredos** ficam só no `.env`, gerado pelo `make up` e ignorado pelo git. O repositório tem só o `.env.example`, e `make check` roda o gitleaks.
- **Erros** passam pelo `apperrors`: código estável, mensagem em pt-BR e status HTTP. Erro inesperado vira `ERRO_INTERNO` sem expor detalhes.
- **CSV de saída** com as células neutralizadas contra fórmulas (`=`, `+`, `-`, `@`).

---

## Testes

```bash
make test               # unidade (motor, regras, preço, CSV, textos…)
make test-integracao    # tudo, inclusive repositório Postgres e ponta a ponta (backend/test/e2e)
```

Os testes ponta a ponta sobem a aplicação inteira contra um Postgres de verdade. Eles cobrem:
- upload e andamento até o resultado;
- reenvio idempotente e envios simultâneos;
- erros de arquivo;
- API reiniciada no meio de um lote;
- `/validar` e CSV de download;
- cessões simultâneas sem venda dupla;
- regras publicadas sem deploy e prévia;
- senha do gestor;
- logs sem dados.

---

## Deploy na Railway (por imagens Docker)

```bash
docker login
make publicar REGISTRO=docker.io/SEU_USUARIO TAG=v1   # monta e envia antecipe-api, antecipe-web e antecipe-ai
```

Depois, crie na Railway os serviços `Postgres`, `api`, `ai` e `web` a partir das imagens. O passo a passo, com todas as variáveis, está em [`docs/deploy-railway.md`](docs/deploy-railway.md). O nginx recebe os endereços internos por variável (`API_UPSTREAM`, `AI_UPSTREAM`) e resolve o DNS a cada 10 s, porque na Railway o IP interno muda a cada deploy.

## Sem Docker (alternativa)

Precisa de Go 1.27 e Postgres 16+:

```bash
cd backend
DATABASE_URL="postgres://usuario:senha@localhost:5432/antecipe?sslmode=disable" \
GESTOR_SENHA=uma-senha CONFIG_PATH=../config/antecipe.yaml DATA_DIR=../data \
go run ./cmd/api
```

---

## Relatório

O que construímos, as decisões e o que ficou de fora estão em [`docs/relatorio.md`](docs/relatorio.md). O plano está em [`docs/plano.md`](docs/plano.md).

## Registro de prompts

Todos os prompts de IA, em ordem cronológica e com o que foi feito de cada resposta: [`docs/prompts.md`](docs/prompts.md) (backend) e [`docs/prompts2.md`](docs/prompts2.md) (assistente de IA, feito em paralelo noutra máquina). Os horários permitem intercalar os dois.
