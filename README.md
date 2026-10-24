# Antecipe

Portal de antecipação de recebíveis do **ANTECIPE FIDC** (Hackathon Kanastra · 24/10/2026).

A empresa sobe um CSV de notas e vê, em segundos, quais notas o fundo compra, quanto recebe e, para cada recusa, o motivo e o que fazer. O gestor do fundo muda as regras sem deploy. Uma nota nunca é vendida duas vezes, nem com duas confirmações simultâneas.

| Lote | Tamanho | Processamento completo (upload → resultado gravado) |
|---|---|---|
| 100 mil linhas | 16 MB | **1,0 s** |
| 250 mil linhas | 39 MB | **2,5 s** |
| 500 mil linhas | 77 MB | **4,9 s** |

_Notebook com 16 núcleos, Postgres 18. Medido com `make carga`._

---

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
  Assistente (IA):  http://localhost:7777/docs
  Senha do gestor:  <gerada no .env>
```

O assistente de IA é opcional: sem `OPENROUTER_API_KEY` no `.env`, todo o resto funciona e o `make up` avisa que ele está desligado (veja [Assistente de IA](#assistente-de-ia)).

Outros comandos (`make` sozinho lista todos):

| Comando | O que faz |
|---|---|
| `make down` | Para tudo (os dados ficam no volume) |
| `make logs` / `make logs-ai` | Acompanha os logs da API / do assistente de IA |
| `make gerar N=500000` | Gera um lote de teste em `amostras/` |
| `make carga` | Mede lotes de 100k, 250k e 500k linhas |
| `make demo-concorrencia` | Duas cessões simultâneas com notas em comum, sem venda duplicada |
| `make check` | Procura segredos no repositório (gitleaks) |
| `make test` / `make test-integracao` | Testes de unidade / todos, inclusive banco e ponta a ponta (precisam de Go 1.27) |
| `make test-ai` | Testes do assistente de IA (precisa do `uv`; não usa rede nem chave) |

### Arquivos para testar

- `data/exemplo.csv`: 23 notas, 12 aprovadas e uma recusa de cada tipo. É bom para ver as mensagens.
- `make gerar N=500000`: lote grande com mistura realista, determinístico pela semente:
  - ~80% aprováveis;
  - casos de cada regra;
  - notas já vendidas (copiadas do estoque);
  - linhas repetidas e linhas quebradas.
- O próprio estoque (`data/estoque_antecipe.zip`) também pode ser enviado: todas as notas voltam como **"Nota já vendida"**, que é o comportamento correto.

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

É um serviço à parte (`ai/`), feito com [Agno](https://docs.agno.com) e servido pelo **AgentOS** na porta 7777. O modelo vem do **OpenRouter** (padrão `deepseek/deepseek-v4-flash`). O assistente **só consulta a API do Antecipe**, como o front: não lê as tabelas do backend, não envia lotes, não confirma cessões e não muda regras. Ele repete os textos oficiais de recusa da API, então tela, API e assistente dizem a mesma coisa.

Para ligar, preencha a linha `OPENROUTER_API_KEY=` do `.env` com a sua chave e rode `make up` de novo.

```bash
# pergunta sobre o lote aberto na tela (o front manda o lote_id em dependencies)
curl -F 'message=Quantas notas foram recusadas e por quê?' \
     -F 'session_id=demo' -F 'stream=false' \
     -F 'dependencies={"lote_id":"<id do lote>"}' \
     http://localhost:7777/agents/antecipe-assistente/runs
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
| `SITUACAO_CADASTRAL_SACADO` | Situação da fonte simulada `data/situacao_cadastral.csv`. CNPJ fora dela = `ATIVA` (configurável no YAML) |
| `TIPO_RECEBIVEL` | Tipos aceitos (sem diferenciar acento nem maiúsculas) |
| `VALOR_MINIMO` | Valor ≥ mínimo |
| `NAO_VENCIDA` | Vencimento depois da data-base (padrão 22/10/2026, a data-base dos dados) |
| `PRAZO_MAXIMO` / `PRAZO_MINIMO` | Dias corridos até o vencimento |
| `CONCENTRACAO_MAXIMA_SACADO` / `_CEDENTE` | (estoque do grupo + aceitas + nota) ÷ (estoque do fundo + aceitas + nota) ≤ limite, em ordem determinística |

Sempre ligadas, porque são integridade e não política do fundo:
- `DUPLICIDADE`: nota já vendida a qualquer um dos 18 fundos ou numa cessão.
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
