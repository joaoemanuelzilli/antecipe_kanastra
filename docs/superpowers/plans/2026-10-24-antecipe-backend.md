# Antecipe Backend — Plano de implementação

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Backend Go + Postgres que recebe lotes CSV de 100k a 500k linhas, valida cada nota pelas regras publicadas, precifica (juros compostos por padrão), mostra o andamento, explica cada recusa e vende sem duplicar, tudo subindo com `make up`.

**Architecture:** Motor puro (`internal/motor` e pacotes folha) sem I/O. As bordas (`store` para o Postgres, `httpapi` para HTTP, `cmd/antecipe` para a CLI) só ligam as peças. O lote é processado por um job assíncrono com etapas e progresso em memória. A venda é uma transação com trava por fundo e `ON CONFLICT`.

**Tech Stack:** Go 1.27.3, pgx v5.11.0, go.yaml.in/yaml/v3 v3.0.5, shopspring/decimal v1.6.0, PostgreSQL 18 (imagem `postgres:18-alpine`), Docker Compose, Make.

**Spec:** [`docs/plano.md`](../../plano.md) (design) e [`docs/api.md`](../../api.md) (contrato v1.1 com o front: nomes de campos, status e códigos de erro valem como especificação).

## Global Constraints

- Módulo Go `antecipe` em `backend/` (`go 1.27`). Dependências: **só** pgx/v5, go.yaml.in/yaml/v3 e shopspring/decimal.
- **Nenhum número de regra no código** (180, 10%, 2%, R$ 100, 30). Tudo vem de `regras.Regras` (versão publicada) e de `config/antecipe.yaml`.
- Dinheiro **sempre `int64` em centavos**. `float64` só em percentuais de exibição. `decimal.MarshalJSONWithoutQuotes = true` (em `regras`, `init`).
- Logs: `log/slog` JSON. Campos permitidos: `request_id, metodo, rota, status, duracao_ms, lote_id, etapa, linhas, aprovadas, recusadas, codigo_regra, versao_regras, erro` (este só com o tipo do erro, sem conteúdo do CSV). **Proibido** logar qualquer campo do CSV, nome de arquivo, corpo de requisição, `GESTOR_SENHA` ou `DATABASE_URL`.
- Textos em pt-BR. Moeda `R$ 1.234,56`, datas `dd/mm/aaaa`, percentuais `12,6%`.
- JSON da API em `snake_case`, dinheiro em `*_centavos`, exatamente como em `docs/api.md`.
- Data-base padrão vem do YAML (`2026-10-22`). Dias = vencimento − data-base em dias corridos.
- Chave da nota: `documento.Normalizar(DocCedente) + "|" + strings.TrimSpace(Numero) + "|" + Vencimento.Format("2006-01-02")`.
- Env: `HTTP_ADDR` (`:8080`), `DATABASE_URL`, `LOG_LEVEL` (`info`), `CONFIG_PATH` (`config/antecipe.yaml`), `DATA_DIR` (`data`), `GESTOR_SENHA`, `CORS_ORIGINS` (`http://localhost:5173`), `TMP_DIR` (`os.TempDir()`).
- Commit ao fim de cada tarefa, com `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`. Cada prompt do time entra em `docs/prompts.md`.

## Review Focus

1. **CSV salvo pelo Excel em Windows-1252** (não UTF-8): os acentos precisam sair certos (`SANTARÉM`), e não como `SANTAR�M`. Teste em T3.
2. **Cabeçalho em outra ordem, minúsculo, com espaços ou com coluna extra:** mapear pelo nome e ignorar as extras. Teste em T3.
3. **API reiniciada no meio de um lote:** o lote não pode ficar "processando" para sempre. Na subida ele vira `erro` (`PROCESSAMENTO_INTERROMPIDO`), e reenviar o mesmo arquivo reprocessa. Teste em T10.
4. **Fronteiras:** vencimento = data-base é recusado (`NAO_VENCIDA`). Valor exatamente no mínimo passa. Dias exatamente no máximo passa. Valor ≤ 0 é `FORMATO_INVALIDO`. Testes em T3 e T6.
5. **O mesmo arquivo enviado duas vezes ao mesmo tempo** gera um único lote. Teste em T10.

## Ordem e paralelismo

T1 → (T2, T3, T4, T5, T6 independentes; T5 e T6 usam T2) → T7 → T8 → T9 → T10 → (T11, T12) → T13. T14 entra quando o código do front chegar.
Cronograma alvo: T1–T8 até 12:45 · T9–T10 até 14:15 · T11–T12 até 15:15 · integração do front 15:15–16:15 · T13 e carga até 16:45 (congelamento).

---

### Task 1: Esqueleto, logs e infraestrutura (`make up`)

**Files:**
- Create: `backend/go.mod`, `backend/cmd/api/main.go`, `backend/internal/obs/log.go`, `backend/internal/obs/log_test.go`
- Create: `backend/Dockerfile`, `docker-compose.yml`, `Makefile`, `.env.example`, `config/antecipe.yaml`
- Create (provisório até T14): `frontend/Dockerfile`, `frontend/nginx.conf`, `frontend/public/index.html`

**Interfaces:**
- Produces: `obs.NovoLogger(w io.Writer, nivel string) *slog.Logger`; `obs.Middleware(log *slog.Logger, next http.Handler) http.Handler` (gera ou propaga `X-Request-Id`, loga `metodo`, `rota` = `r.Pattern`, `status`, `duracao_ms`, `request_id`); `obs.RequestID(ctx) string`.

- [ ] **Step 1: Testes do logger**

```go
func TestNovoLoggerJSON(t *testing.T) // linha = JSON válido com "level":"INFO" e "msg":"oi"
func TestMiddlewareCampos(t *testing.T) {
    // mux.HandleFunc("GET /api/saude", ...) envolvido pelo Middleware
    // GET /api/saude → resposta tem header X-Request-Id != ""
    // log tem rota == "GET /api/saude", status == 200, request_id == header, duracao_ms presente
}
```

- [ ] **Step 2:** `cd backend && go test ./internal/obs/` → FAIL (pacote não existe)
- [ ] **Step 3:** Implementar `obs`. Em `cmd/api/main.go`, ler as env e servir `GET /api/saude` → `{"status":"ok"}` (o banco entra em T9), com shutdown gracioso em SIGTERM.
- [ ] **Step 4:** `go test ./internal/obs/` → PASS
- [ ] **Step 5: Infra.**
  - `config/antecipe.yaml`: exatamente o bloco da seção 7 do plano (`metodo: DESCONTO_COMPOSTO`, `data_base_padrao: "2026-10-22"`, `situacao_cadastral_padrao: ATIVA`, `jobs_simultaneos: 2`, `tamanho_maximo_upload_mb: 300`).
  - `.env.example`: `POSTGRES_PASSWORD=troque-me`, `GESTOR_SENHA=troque-me`, `LOG_LEVEL=info`.
  - `backend/Dockerfile`: build com contexto na raiz do repositório. Estágio `golang:1.27-alpine` compila `./cmd/api` e `./cmd/antecipe` (`CGO_ENABLED=0`). Estágio final `alpine:3.22`, com `/app/bin/{api,antecipe}`, `/app/data` copiado de `data/` e `WORKDIR /app`, `PATH=/app/bin:$PATH`, usuário não-root.
  - `docker-compose.yml`: três serviços.
    - `postgres`: `postgres:18-alpine`, `POSTGRES_USER/DB=antecipe`, senha de `${POSTGRES_PASSWORD}`, volume `pgdata:/var/lib/postgresql` (**no PG18 o volume é montado em `/var/lib/postgresql`, não em `/data`**), healthcheck `pg_isready`, porta `5433:5432`.
    - `api`: `DATABASE_URL=postgres://antecipe:${POSTGRES_PASSWORD}@postgres:5432/antecipe?sslmode=disable`, `GESTOR_SENHA`, `LOG_LEVEL`, `CONFIG_PATH=/app/config/antecipe.yaml`, `DATA_DIR=/app/data`, volume `./config:/app/config:ro`, porta `8080:8080`, `depends_on` postgres saudável.
    - `web`: build `./frontend`, porta `3000:80`, `depends_on` api.
  - `frontend/` provisório: nginx servindo um `index.html` ("Antecipe: front em integração") e fazendo proxy de `/api/` → `http://api:8080`, com `client_max_body_size 300m`, `proxy_request_buffering off` e `proxy_read_timeout 300s`.
  - `Makefile`:
    - `up`: se não existir `.env`, cria a partir do `.env.example` trocando as duas senhas por `head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n'`. Depois roda `docker compose up --build -d`, espera `curl -fsS localhost:8080/api/saude` por até 180 s e imprime "Front: http://localhost:3000 · API: http://localhost:8080 · Senha do gestor: …".
    - `down`, `logs` (`docker compose logs -f api`) e `test` (`cd backend && go test ./...`).
- [ ] **Step 6: Verificar.** `make up` e depois `curl -s localhost:8080/api/saude` → `{"status":"ok"}`. Em `http://localhost:3000` aparece o placeholder, e `curl -s localhost:3000/api/saude` → ok. `git status` não mostra `.env`. (Exige o usuário no grupo `docker`.)
- [ ] **Step 7:** Commit `chore: esqueleto Go, logs estruturados e make up`

---

### Task 2: Utilitários folha (`documento`, `formato`)

**Files:** Create `backend/internal/documento/cnpj.go`, `cnpj_test.go`, `backend/internal/formato/formato.go`, `formato_test.go`

**Interfaces (Produces):**
- `documento.Normalizar(s string) string`: tira `.`, `/`, `-` e espaços, e põe em maiúsculas.
- `documento.CNPJValido(norm string) bool`: 14 caracteres `[0-9A-Z]{12}[0-9]{2}`, não todos iguais, DVs por módulo 11 com valor = `rune − '0'`, pesos `5,4,3,2,9,8,7,6,5,4,3,2` e `6,5,4,3,2,9,8,7,6,5,4,3,2`, resto < 2 → 0, senão 11 − resto.
- `documento.Raiz(norm string) string` (os 8 primeiros caracteres) · `documento.Formatar(norm string) string` (`12.ABC.345/01DE-35`; se `len != 14`, devolve a entrada).
- `formato.Reais(centavos int64) string` (`R$ 1.234,56`, `-R$ 5,00`) · `formato.Data(t time.Time) string` (`dd/mm/aaaa`) · `formato.DataISO(s string) string` (`"2026-12-21"` → `21/12/2026`) · `formato.Inteiro(n int) string` (`1.204`) · `formato.Percentual(f float64, casas int) string` (`12,6%`) · `formato.Lista(xs []string) string` (`A, B e C`).

- [ ] **Step 1: Testes**

```go
// CNPJValido(Normalizar(x)):
//   "92.234.400/0001-80" true · "12.ABC.345/01DE-35" true · "12.abc.345/01de-35" true
//   "77.788.899/0001-80" false · "11.111.111/1111-11" false · "123" false
// Raiz("92234400000180") == "92234400" · Formatar("12ABC34501DE35") == "12.ABC.345/01DE-35"
// Reais(123456) == "R$ 1.234,56" · Reais(980392) == "R$ 9.803,92" · Reais(5) == "R$ 0,05"
// Percentual(12.612, 1) == "12,6%" · Percentual(10, 0) == "10%" · Inteiro(500000) == "500.000"
// Lista([]string{"Duplicata","Duplicata de Serviço"}) == "Duplicata e Duplicata de Serviço"
```

- [ ] **Step 2:** `go test ./internal/documento/ ./internal/formato/` → FAIL
- [ ] **Step 3:** Implementar.
- [ ] **Step 4:** → PASS
- [ ] **Step 5:** Commit `feat: CNPJ (inclusive alfanumérico) e formatação pt-BR`

---

### Task 3: Domínio e leitura do CSV (`dominio`, `csvnota`)

**Files:** Create `backend/internal/dominio/dominio.go`, `backend/internal/csvnota/ler.go`, `backend/internal/csvnota/ler_test.go`

**Interfaces (Produces):**

```go
package dominio
type Nota struct {
    Linha int // linha física: cabeçalho = 1, primeira nota = 2
    NomeFundo, DocFundo, NomeCedente, DocCedente, NomeSacado, DocSacado string // como no arquivo, com TrimSpace
    Numero, Tipo  string
    ValorCentavos int64
    Vencimento    time.Time // UTC 00:00; zero se inválida
    Aquisicao     time.Time // zero se vazia
    ErroFormato   *ErroCampo
}
type ErroCampo struct{ Campo, Valor, Problema string }
type Status uint8
const (Aprovada Status = 1; Recusada Status = 2)
type Recusa struct { Codigo string `json:"codigo"`; Dados map[string]any `json:"dados"` }
type Resultado struct {
    Nota *Nota; Chave string; Dias int; Status Status
    DescontoCentavos, LiquidoCentavos int64
    Recusas []Recusa // [0] = motivo principal
}
```

```go
package csvnota
var Colunas = []string{"NOME_FUNDO","DOC_FUNDO","NOME_CEDENTE","DOC_CEDENTE","NOME_SACADO","DOC_SACADO","NU_DOCUMENTO","TIPO_RECEBIVEL","VALOR_NOMINAL","DATA_VENCIMENTO_ORIGINAL","DATA_AQUISICAO"}
type ErroArquivo struct { Codigo, Mensagem string; Detalhes []Detalhe } // Codigo: ARQUIVO_VAZIO | CABECALHO_INVALIDO | ARQUIVO_NAO_CSV
type Detalhe struct { Linha int `json:"linha"`; Campo string `json:"campo"`; Problema string `json:"problema"` }
func (e *ErroArquivo) Error() string
func Ler(r io.Reader, aoAvancar func(lidas int)) ([]dominio.Nota, error) // erro só *ErroArquivo; aoAvancar a cada 10.000 linhas (pode ser nil)
func ValidarCabecalho(r io.Reader) *ErroArquivo // lê só o começo; NUL nos primeiros 512 bytes → ARQUIVO_NAO_CSV
func ContarLinhas(r io.Reader) (int, error)     // linhas de dados (sem cabeçalho e sem linhas vazias)
```

Regras do parser: tira o BOM UTF-8. Linha com UTF-8 inválido é decodificada como Windows-1252/Latin-1 (byte 0xA0–0xFF → rune igual; 0x80–0x9F pela tabela do cp1252). `encoding/csv` com `Comma=';'`, `LazyQuotes`, `FieldsPerRecord=-1` e `ReuseRecord`. O cabeçalho é mapeado por nome (TrimSpace e maiúsculas), em qualquer ordem, ignorando colunas extras. Todas as colunas são obrigatórias, menos `DATA_AQUISICAO`. A primeira falha de cada linha vai para `ErroFormato` e a linha continua na lista:

| Situação | `Campo` | `Problema` |
|---|---|---|
| nº de colunas errado | `linha` | `tem N colunas, esperávamos 11` |
| valor ilegível | `VALOR_NOMINAL` | `não é um valor válido (use 1234,56)` |
| valor ≤ 0 | `VALOR_NOMINAL` | `precisa ser maior que zero` |
| data inválida | nome da coluna | `não é uma data válida (use dd/mm/aaaa)` |
| campo obrigatório vazio | nome da coluna | `está vazio` |

Valor aceita `1234,56`, `1.234,56`, `1234` e prefixo `R$`. Mais de 2 casas decimais é erro. Data por `time.Parse("02/01/2006")` (estrito: `31/02/2026` falha).

- [ ] **Step 1: Testes** (CSV em strings no próprio teste)

```go
func TestLerValido(t *testing.T)            // linha do estoque "…;725714;Duplicata;2278,08;07/10/2026;27/04/2026" → Linha 2, ValorCentavos 227808, Vencimento 2026-10-07, Aquisicao 2026-04-27
func TestLerBOMeCRLF(t *testing.T)          // "\xEF\xBB\xBF" + linhas com \r\n → 2 notas sem lixo no NomeFundo
func TestLerAspas(t *testing.T)             // NOME_SACADO "\"MÓVEIS; SUL LTDA\"" → "MÓVEIS; SUL LTDA"
func TestLerMilhar(t *testing.T)            // "1.234,56" → 123456; "R$ 10.000,00" → 1000000
func TestLerDataInvalida(t *testing.T)      // 31/02/2026 → ErroFormato{Campo:"DATA_VENCIMENTO_ORIGINAL", Valor:"31/02/2026"}; a linha seguinte é lida normalmente
func TestLerValorNaoPositivo(t *testing.T)  // "0,00" e "-5,00" → ErroFormato.Campo == "VALOR_NOMINAL", Problema "precisa ser maior que zero"
func TestLerCabecalhoFaltando(t *testing.T) // sem DATA_VENCIMENTO_ORIGINAL → *ErroArquivo{Codigo:"CABECALHO_INVALIDO"}, Detalhes[0].Campo == "DATA_VENCIMENTO_ORIGINAL"
func TestLerVazio(t *testing.T)             // "" → ARQUIVO_VAZIO
func TestLerCabecalhoForaDeOrdem(t *testing.T) // " nu_documento ;NOME_FUNDO;…;OBS" em outra ordem → campos certos, OBS ignorada (Review Focus 2)
func TestLerWindows1252(t *testing.T)       // bytes "TECIDOS SANTAR\xC9M LTDA" → NomeSacado "TECIDOS SANTARÉM LTDA" (Review Focus 1)
func TestLerAquisicaoVazia(t *testing.T)    // DATA_AQUISICAO "" → Aquisicao zero, ErroFormato nil
func TestValidarCabecalhoBinario(t *testing.T) // "\x00\x01PNG" → ARQUIVO_NAO_CSV
func TestContarLinhas(t *testing.T)         // cabeçalho + 3 linhas + "\n\n" → 3
func BenchmarkLer500k(b *testing.B)         // 500.000 linhas sintéticas em memória
```

- [ ] **Step 2:** `go test ./internal/csvnota/` → FAIL
- [ ] **Step 3:** Implementar.
- [ ] **Step 4:** → PASS. Rodar `go test -bench Ler500k -benchmem ./internal/csvnota/` e anotar o tempo (meta < 1,5 s).
- [ ] **Step 5:** Commit `feat: leitura de CSV pt-BR tolerante (BOM, CRLF, cp1252, cabeçalho por nome)`

---

### Task 4: Regras (`regras`)

**Files:** Create `backend/internal/regras/regras.go`, `parametros.go`, `catalogo.go`, `validar.go`, `resumo.go`, `regras_test.go`

**Interfaces (Produces):**

```go
type Regras struct { Fundo Fundo `json:"fundo"`; Validadores []Validador `json:"validadores"`; Precificacao Precificacao `json:"precificacao"` }
type Fundo struct { Nome string `json:"nome"`; CNPJ string `json:"cnpj"` }
type Validador struct { Codigo string `json:"codigo"`; Ativo bool `json:"ativo"`; Parametros Parametros `json:"parametros"` }
type Parametros map[string]json.RawMessage
type Precificacao struct { Metodo string `json:"metodo"`; TaxaMensalPercentual decimal.Decimal `json:"taxa_mensal_percentual"`; BaseDias int `json:"base_dias"` }
func Ler(data []byte) (Regras, error)
func (p Parametros) Inteiro(nome string) (int, error)
func (p Parametros) Decimal(nome string) (decimal.Decimal, error)
func (p Parametros) ListaTexto(nome string) ([]string, error)
func (p Parametros) Texto(nome string) (string, error)
type ParametroCatalogo struct { Nome, Rotulo, Tipo string; Unidade string `json:",omitempty"`; Minimo, Maximo *float64; Opcoes []string } // Tipo: inteiro|decimal|lista_texto|opcao; tags json snake_case
type ItemCatalogo struct { Codigo, Titulo, Descricao string; Parametros []ParametroCatalogo }
func Catalogo() []ItemCatalogo            // as 9 regras; títulos e opções iguais a docs/api.md
func CatalogoPrecificacao() []ParametroCatalogo
type ErroValidacao struct { Campo string `json:"campo"`; Problema string `json:"problema"` }
func ValidarSchema(r Regras, metodos []string) []ErroValidacao // usada no PUT
func Resumo(antes, depois Regras) []string
```

O catálogo tem as **opções** `tipos_aceitos`: `Duplicata, Duplicata de Serviço, Cheque, CCB, Nota Comercial`; `situacoes_aceitas`: `ATIVA, SUSPENSA, INAPTA, BAIXADA, NULA`; `agrupamento`: `RAIZ_CNPJ, CNPJ`. `ValidarSchema` recusa: código fora do catálogo, parâmetro ausente ou de tipo errado, inteiro ≤ 0, percentual fora de 0–100, lista vazia, opção fora das opções, CNPJ do fundo inválido, método fora de `metodos`, taxa < 0 e base < 1. O campo vem no formato `validadores[5].parametros.prazo_maximo_dias`.

- [ ] **Step 1: Testes**

```go
func TestLerJSONOriginal(t *testing.T)   // os.ReadFile("../../../data/validadores_antecipe.json") → 9 validadores, 7 ativos, Metodo "DESCONTO_SIMPLES", Taxa.Equal(decimal.NewFromInt(2)), BaseDias 30
func TestParametros(t *testing.T)        // PRAZO_MAXIMO.Inteiro("prazo_maximo_dias") == 180; VALOR_MINIMO.Decimal == 100; TIPO_RECEBIVEL.ListaTexto len 2; Inteiro em lista → erro
func TestValidarSchemaRecusa(t *testing.T) // prazo_maximo_dias = -1 → ErroValidacao{Campo:"validadores[5].parametros.prazo_maximo_dias"}; JSON original → nenhum erro
func TestResumo(t *testing.T)            // 180→120 → []string{"PRAZO_MAXIMO: 180 → 120 dias"}; PRAZO_MINIMO ativo false→true → "PRAZO_MINIMO: ligada"
func TestJSONTaxaComoNumero(t *testing.T) // json.Marshal(Precificacao) contém `"taxa_mensal_percentual":2`
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** → PASS
- [ ] **Step 5:** Commit `feat: regras do fundo, catálogo, validação de schema e resumo de mudanças`

---

### Task 5: Preço e configuração YAML (`precificacao`, `config`)

**Files:** Create `backend/internal/precificacao/precificacao.go`, `precificacao_test.go`, `backend/internal/config/config.go`, `config_test.go`

**Interfaces (Produces):**

```go
package precificacao
const (Composto = "DESCONTO_COMPOSTO"; Simples = "DESCONTO_SIMPLES")
type Config struct { Metodo string `json:"metodo"`; TaxaMensalPercentual decimal.Decimal `json:"taxa_mensal_percentual"`; BaseDias int `json:"base_dias"` }
func Metodos() []string
func (c Config) Descricao() string // "Juros compostos · 2% ao mês" | "Juros simples · 2% ao mês"
func Efetiva(base Config, metodo *string, taxa *decimal.Decimal, baseDias *int) (Config, []string) // sobrescritos: "metodo","taxa_mensal_percentual","base_dias"
type Calculadora struct{ /* config + memo por dias, seguro para concorrência */ }
func Nova(c Config) (*Calculadora, error) // método desconhecido ou BaseDias < 1 → erro
func (c *Calculadora) Calcular(faceCentavos int64, dias int) (descontoCentavos, liquidoCentavos int64) // dias <= 0 → (0, face)
func (c *Calculadora) Texto(faceCentavos int64, dias int) string
```

- Composto: `liquido = Round0(face ÷ (1+i)^(dias/base))`, `desconto = face − liquido`. O fator usa `decimal.NewFromInt(1).Add(i).PowWithPrecision(decimal.NewFromInt(dias).Div(decimal.NewFromInt(base)), 20)` e fica memorizado por `dias`.
- Simples: `desconto = Round0(face × i × dias ÷ base)`.
- `i = taxa ÷ 100`. `Round(0)` do decimal arredonda meio-para-cima nos positivos.
- `Texto` composto: `R$ 10.000,00 ÷ (1 + 2%)^(30 ÷ 30) = R$ 9.803,92 → desconto de R$ 196,08`. Simples: `R$ 10.000,00 × 2% × (30 ÷ 30) = R$ 200,00 de desconto → você recebe R$ 9.800,00`.

```go
package config
type Precificacao struct { Metodo *string `yaml:"metodo"`; TaxaMensalPercentual *decimal.Decimal `yaml:"taxa_mensal_percentual"`; BaseDias *int `yaml:"base_dias"` }
type Processamento struct { DataBasePadrao string `yaml:"data_base_padrao"`; SituacaoCadastralPadrao string `yaml:"situacao_cadastral_padrao"`; JobsSimultaneos int `yaml:"jobs_simultaneos"`; TamanhoMaximoUploadMB int `yaml:"tamanho_maximo_upload_mb"` }
type Config struct { Precificacao Precificacao `yaml:"precificacao"`; Processamento Processamento `yaml:"processamento"` }
func Carregar(caminho string) (Config, error) // valida: método ∈ precificacao.Metodos(), data "2006-01-02" obrigatória, jobs ≥ 1, tamanho ≥ 1, situação não vazia
func (c Config) DataBasePadrao() time.Time
type Atual struct{ /* atomic.Pointer[Config] */ }
func NovoAtual(caminho string, log *slog.Logger) (*Atual, error)
func (a *Atual) Obter() Config
func (a *Atual) Observar(ctx context.Context, intervalo time.Duration) // confere o ModTime; se Carregar falhar, loga "config inválida, mantendo a anterior" (campo erro só com o tipo) e mantém a anterior
```

Se o yaml.v3 não decodificar `*decimal.Decimal` como TextUnmarshaler, use `*string` e converta em `Carregar`.

- [ ] **Step 1: Testes**

```go
func TestComposto(t *testing.T) // Calcular(1000000,30)==(19608,980392); (1000000,60) liquido 961169; (1000000,180) liquido 887971
func TestSimples(t *testing.T)  // (1000000,30)==(20000,980000); 60 → 960000; 180 → 880000
func TestMetodoDesconhecido(t *testing.T) // Nova(Config{Metodo:"X",...}) erro
func TestTexto(t *testing.T)    // composto (1000000,30) == "R$ 10.000,00 ÷ (1 + 2%)^(30 ÷ 30) = R$ 9.803,92 → desconto de R$ 196,08"
func TestEfetiva(t *testing.T)  // base Simples + metodo "DESCONTO_COMPOSTO" → Metodo Composto, sobrescritos == []string{"metodo"}, taxa/base da base
func TestCalcularConcorrente(t *testing.T) // 32 goroutines × 1000 chamadas; rodar com -race
func TestCarregarRepo(t *testing.T) // Carregar("../../../config/antecipe.yaml") → *Metodo == "DESCONTO_COMPOSTO", DataBasePadrao 2026-10-22, Jobs 2, Tamanho 300
func TestCarregarMetodoInvalido(t *testing.T) // metodo: X → erro
func TestObservarTroca(t *testing.T)   // yaml temp composto → NovoAtual → Observar(ctx, 20ms) → reescreve simples (e muda o ModTime) → em até 2 s Obter() mostra "DESCONTO_SIMPLES"
func TestObservarInvalidoMantem(t *testing.T) // reescreve com ": : lixo" → Obter() continua com o anterior
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** `go test -race ./internal/precificacao/ ./internal/config/` → PASS
- [ ] **Step 5:** Commit `feat: juros compostos (padrão) e simples, configuráveis em config/antecipe.yaml`

---

### Task 6: Validadores por nota e textos de recusa (`validadores`, `textos`)

**Files:** Create `backend/internal/validadores/validadores.go` (registro e as 7 regras por nota), `validadores_test.go`, `backend/internal/textos/textos.go`, `textos_test.go`

**Interfaces:**
- Consumes: `regras.Parametros` (T4), `documento` e `formato` (T2), `dominio.Nota` (T3)
- Produces:

```go
package validadores
type Ctx struct { DataBase time.Time; Situacao func(cnpjNorm string) string } // Situacao já aplica o padrão do YAML
type Item struct { Nota *dominio.Nota; Dias int; SacadoNorm string }
type Func func(it Item, p regras.Parametros, c Ctx) (ok bool, dados map[string]any, err error) // err = parâmetro inválido
var PorNota map[string]Func // CNPJ_SACADO_VALIDO, SITUACAO_CADASTRAL_SACADO, TIPO_RECEBIVEL, VALOR_MINIMO, NAO_VENCIDA, PRAZO_MAXIMO, PRAZO_MINIMO
var DeLote = map[string]bool{"CONCENTRACAO_MAXIMA_SACADO": true, "CONCENTRACAO_MAXIMA_CEDENTE": true}
func Conhecido(codigo string) bool
```

```go
package textos
type Texto struct { Titulo, Mensagem, Acao string }
func Recusa(codigo string, dados map[string]any) Texto // aceita dados vindos do Go (int, int64, []string) OU do jsonb (float64, []any)
func Titulo(codigo string) string
func AcaoResumo(codigo string) string // ação sem placeholders, para a lista de motivos
```

**`dados` por código** (chaves exatas, datas como `"2006-01-02"`):
- `CNPJ_SACADO_VALIDO`: `documento`
- `SITUACAO_CADASTRAL_SACADO`: `situacao`, `aceitas`
- `TIPO_RECEBIVEL`: `tipo`, `aceitos`
- `VALOR_MINIMO`: `valor_centavos`, `minimo_centavos` (parâmetro em reais × 100)
- `NAO_VENCIDA`: `vencimento`, `data_base`
- `PRAZO_MAXIMO`: `dias`, `maximo`, `antecipavel_em` (vencimento − máximo dias)
- `PRAZO_MINIMO`: `dias`, `minimo`

`TIPO_RECEBIVEL` compara sem acento, sem diferenciar maiúsculas e com TrimSpace.

**Textos exatos** (`{x}` = dado formatado; dinheiro com `formato.Reais`, datas com `formato.DataISO`, listas com `formato.Lista`, percentuais com 1 casa e limite com 0):

| Código | Título | Mensagem | Ação |
|---|---|---|---|
| CNPJ_SACADO_VALIDO | CNPJ do cliente inválido | O CNPJ de quem paga a nota ({documento}) não é válido: os dígitos verificadores não conferem. | Confira o CNPJ com seu cliente e envie a nota de novo. |
| SITUACAO_CADASTRAL_SACADO | CNPJ do cliente com pendência | O CNPJ de quem paga a nota está {situacao} na Receita, e o fundo só compra de clientes com situação {aceitas}. | Confirme os dados com seu cliente antes de antecipar esta nota. |
| TIPO_RECEBIVEL | Tipo de nota não aceito | O fundo compra só {aceitos}, e esta nota é {tipo}. | Esta nota não pode ser antecipada neste fundo. |
| VALOR_MINIMO | Valor abaixo do mínimo | Esta nota vale {valor} e o fundo compra notas a partir de {minimo}. | Notas de valor menor não podem ser antecipadas aqui. |
| NAO_VENCIDA | Nota vencida | Esta nota venceu em {vencimento} (data-base {data_base}) e o fundo só compra notas a vencer. | Cobre esta nota diretamente do seu cliente. |
| PRAZO_MAXIMO | Prazo acima do limite | Esta nota vence em {dias} dias e o fundo compra notas com até {maximo} dias. | Você pode antecipá-la a partir de {antecipavel_em}. |
| PRAZO_MINIMO | Prazo abaixo do mínimo | Esta nota vence em {dias} dias e o fundo compra notas com pelo menos {minimo} dias. | Esta nota está perto demais do vencimento para ser antecipada. |
| CONCENTRACAO_MAXIMA_SACADO | Limite por cliente atingido | Com esta nota, {grupo_nome} ficaria com {percentual_resultante} da carteira do fundo, e o limite é {limite}. | Tente com notas de outros clientes ou aguarde o vencimento das notas atuais desse cliente. |
| CONCENTRACAO_MAXIMA_CEDENTE | Limite por empresa atingido | Com esta nota, sua empresa ficaria com {percentual_resultante} da carteira do fundo, e o limite é {limite}. | Aguarde o vencimento de parte das notas já vendidas antes de antecipar mais. |
| DUPLICIDADE | Nota já vendida | Esta nota (nº {numero}, vencimento {vencimento}) já foi cedida a um fundo em {vendida_em}. | Não é preciso enviar de novo. |
| DUPLICADA_NO_ARQUIVO | Nota repetida no arquivo | Esta nota já aparece na linha {linha_original} deste arquivo. | Remova a linha repetida. A primeira ocorrência foi avaliada normalmente. |
| FORMATO_INVALIDO | Linha com erro | Na coluna {campo}, o valor "{valor}" {problema}. | Corrija a linha e envie o arquivo de novo. |

Em `DUPLICIDADE`, se `valor_original_centavos` ≠ `valor_centavos` (os dois vêm em `dados`), a mensagem ganha " O valor registrado na venda foi {valor_original}.". `AcaoResumo("PRAZO_MAXIMO")` = "Antecipe essas notas mais perto do vencimento." Para os outros códigos, é a própria Ação.

- [ ] **Step 1: Testes (tabela, com dataBase 2026-10-22)**

```go
func TestCNPJ(t *testing.T)          // "77.788.899/0001-80" falha com dados{"documento":"77.788.899/0001-80"}; "12.ABC.345/01DE-35" passa
func TestSituacao(t *testing.T)      // Situacao devolve "INAPTA" → falha dados{"situacao":"INAPTA","aceitas":["ATIVA"]}
func TestTipo(t *testing.T)          // "duplicata de servico" passa; "Cheque" falha dados{"tipo":"Cheque","aceitos":[...]}
func TestValorMinimo(t *testing.T)   // 10000 centavos com minimo 100 → passa (fronteira); 9999 → falha {valor_centavos:9999, minimo_centavos:10000}
func TestNaoVencida(t *testing.T)    // vencimento == data-base → falha (fronteira); +1 dia → passa
func TestPrazoMaximo(t *testing.T)   // dias 180 → passa (fronteira); dias 240, vencimento 2027-06-19 → falha {dias:240, maximo:180, antecipavel_em:"2026-12-21"}
func TestPrazoMinimo(t *testing.T)   // dias 29 com minimo 30 → falha
func TestParametroInvalido(t *testing.T) // prazo_maximo_dias:"abc" → err != nil
func TestTextoPrazoMaximo(t *testing.T)  // Mensagem "Esta nota vence em 240 dias e o fundo compra notas com até 180 dias."; Acao "Você pode antecipá-la a partir de 21/12/2026."
func TestTextoDadosDeJSON(t *testing.T)  // dados após json round-trip (float64, []any) → mesmo texto de TIPO_RECEBIVEL
func TestTodosCodigosTemTexto(t *testing.T) // os 12 códigos: Titulo != "" e Recusa(...).Mensagem sem "{"
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** → PASS
- [ ] **Step 5:** Commit `feat: validadores por nota e textos de recusa`

---

### Task 7: Motor (`motor`)

**Files:** Create `backend/internal/motor/motor.go`, `duplicidade.go`, `concentracao.go`, `motor_test.go`, `bench_test.go`

**Interfaces:**
- Consumes: T2–T6
- Produces:

```go
type Vendida struct { Em time.Time; ValorCentavos int64 }
type EstoqueFundo struct { TotalCentavos int64; PorSacado, PorCedente map[string]int64 } // chave: documento normalizado completo
type Contexto struct {
    DataBase  time.Time
    Situacao  func(cnpjNorm string) string
    Vendidas  map[string]Vendida
    Estoque   EstoqueFundo
    Preco     *precificacao.Calculadora
    AoAvancar func(etapa string, feitas int) // etapas "validando","concentracao","precificando"; pode ser nil
}
type Totais struct { Linhas, Aprovadas, Recusadas int; FaceCentavos, DescontoCentavos, LiquidoCentavos int64 } // tags json snake_case como docs/api.md
type ErroConfig struct { Codigo string `json:"codigo"`; Mensagem string `json:"mensagem"` }
type Saida struct { Resultados []dominio.Resultado; Totais Totais; Motivos map[string]int; ErrosConfig []ErroConfig }
func Validar(notas []dominio.Nota, r regras.Regras, c Contexto) Saida // Resultados[i] ↔ notas[i]
func Chave(n *dominio.Nota) string
func Concentrar(cands []*dominio.Resultado, r regras.Regras, est EstoqueFundo) []ErroConfig // reaproveitado na cessão (T11)
```

**Algoritmo (ordem fixa):**
1. `ErroFormato != nil` → só `FORMATO_INVALIDO` `{campo, valor, problema, linha}`, sem outras regras.
2. Por nota, em paralelo (blocos de 4.096, `runtime.GOMAXPROCS(0)` workers): `Dias`, `Chave` e os validadores **ativos** de `PorNota`, na ordem do JSON. Código desconhecido → `ErroConfig{Codigo, "Regra X não existe e foi ignorada."}` (uma vez por código). `err` de parâmetro → `ErroConfig` com "Regra X tem parâmetro inválido e foi ignorada." (uma vez).
3. Duplicidade, sequencial na ordem das linhas (só notas sem erro de formato): chave em `Vendidas` → `DUPLICIDADE {vendida_em, valor_original_centavos, valor_centavos, numero, vencimento}`. Senão, chave já vista no arquivo → `DUPLICADA_NO_ARQUIVO {linha_original}`. Senão, marca como vista. **Recusas de integridade vão para a frente da lista** (viram o motivo principal).
4. `Concentrar`: para cada validador ativo de `DeLote`, na ordem do JSON, candidatos = notas ainda sem recusa, ordenadas de forma estável por (Vencimento, Numero, Linha). Grupo = `documento.Raiz` se `agrupamento == "RAIZ_CNPJ"`, senão o documento completo (cedente sem `agrupamento` → completo). Base do grupo = soma de `PorSacado`/`PorCedente` agregada pelo grupo. Recusa se `(baseGrupo + aceitoGrupo + face) × 10000 > limiteBP × (Total + aceitoTotal + face)`, com `limiteBP = percentual_maximo × 100` (inteiro). Senão, acumula. `dados {percentual_resultante (float, 2 casas), limite (float), grupo, grupo_nome}`.
5. Sem recusa → `Aprovada` + `Preco.Calcular`. Com recusa → `Recusada`.
6. `Totais` (face/desconto/líquido só das aprovadas) e `Motivos[Recusas[0].Codigo]++`.

- [ ] **Step 1: Testes** (regras = JSON original, Preço composto 2%/30, dataBase 2026-10-22)

```go
func TestAprovadaComposto(t *testing.T)        // 1000000 a 30 dias → Aprovada, Desconto 19608, Liquido 980392, Totais.LiquidoCentavos 980392
func TestRegraInativaIgnorada(t *testing.T)    // nota de 5 dias → Aprovada (PRAZO_MINIMO inativa)
func TestCodigoDesconhecido(t *testing.T)      // +{"codigo":"REGRA_X","ativo":true} → ErrosConfig[0].Codigo=="REGRA_X" e PRAZO_MAXIMO continua recusando
func TestDuplicidadeEstoque(t *testing.T)      // Vendidas[chave]={2026-04-27, 227808} → Recusas[0].Codigo=="DUPLICIDADE", dados["vendida_em"]=="2026-04-27"
func TestDuplicadaNoArquivo(t *testing.T)      // linhas 2 e 3 iguais → linha 3 DUPLICADA_NO_ARQUIVO {linha_original:2}; linha 2 aprovada
func TestParcelasNaoSaoDuplicadas(t *testing.T) // mesmo cedente+número, vencimentos 2026-11-15 e 2026-12-15 → as duas Aprovadas
func TestDuplicidadeVemPrimeiro(t *testing.T)  // nota vendida E com 240 dias → Recusas[0] DUPLICIDADE, Recusas[1] PRAZO_MAXIMO
func TestConcentracaoDeterministica(t *testing.T) {
    // Estoque Total 1.000.000, PorSacado{grupo A: 90.000}; 3 notas do A de 5.000 com vencimentos d1<d2<d3, entrada embaralhada
    // → d1 e d2 aprovadas (9,45% e 9,90%), d3 recusada com percentual_resultante 10.34; mesma saída em 5 ordens de entrada
}
func TestConcentracaoJaEstourada(t *testing.T) // grupo com 12,5% do estoque → qualquer nota nova do grupo recusada (caso do sacado 58206002)
func TestAgrupamentoRaiz(t *testing.T)         // dois CNPJs com a mesma raiz (filiais 0001 e 0002) somam no mesmo grupo
func TestProgresso(t *testing.T)               // AoAvancar chamado com "validando", "concentracao" e "precificando"
func BenchmarkValidar500k(b *testing.B)        // 500.000 notas sintéticas, 2.000 sacados; meta < 1 s
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** `go test -race ./internal/motor/` → PASS e `go test -bench Validar500k ./internal/motor/` (anotar o tempo)
- [ ] **Step 5:** Commit `feat: motor de regras paralelo com duplicidade e concentração determinística`

---

### Task 8: Fontes de dados, gerador e CLI (`fontes`, `gerador`, `cmd/antecipe`)

**Files:**
- Create: `backend/internal/fontes/fontes.go`, `fontes_test.go`, `backend/internal/gerador/gerador.go`, `gerador_test.go`
- Create: `backend/cmd/antecipe/main.go`, `data/situacao_cadastral.csv`, `data/exemplo.csv` (gerado pelo próprio comando)

**Interfaces (Produces):**

```go
package fontes
func LerEstoqueZip(caminho string) ([]dominio.Nota, error) // primeiro .csv do zip, via csvnota.Ler
func LerSituacoes(caminho string) (map[string]string, error) // chave: CNPJ normalizado
func ContextoDeEstoque(estoque []dominio.Nota, fundoCNPJ string) (map[string]motor.Vendida, motor.EstoqueFundo) // Vendidas = todos os fundos; EstoqueFundo = só fundoCNPJ
func SituacaoCom(m map[string]string, padrao string) func(string) string

package gerador
type Opcoes struct { Linhas int; Semente int64; DataBase time.Time; Estoque []dominio.Nota; FundoNome, FundoCNPJ string }
func Gerar(w io.Writer, o Opcoes) error   // cabeçalho + o.Linhas linhas, determinístico pela semente (math/rand/v2 PCG)
func Exemplo(w io.Writer, dataBase time.Time, estoque []dominio.Nota) error // ~30 linhas fixas
```

`data/situacao_cadastral.csv` (exato):

```
CNPJ;SITUACAO
11.122.233/0001-83;INAPTA
22.233.344/0001-83;BAIXADA
33.344.455/0001-83;SUSPENSA
44.455.566/0001-83;INAPTA
55.566.677/0001-83;ATIVA
66.677.788/0001-83;ATIVA
```

**Mistura do `Gerar`** (por linha, sorteio com a semente): 80% válidas novas; 3% prazo > 180 dias; 2% vencidas; 2% valor < R$ 100; 2% tipo `Cheque`/`CCB`; 2% sacado `77.788.899/0001-80` (DV errado); 2% sacado com situação do mock (`11.122.233/0001-83`, `22.233.344/0001-83` ou `33.344.455/0001-83`); 3% cópia exata de uma linha do estoque (se `Estoque != nil`); 2% repetição de uma linha já gerada; 1% formato inválido (`31/02/2026` ou valor `abc`); 1% sacado `58.206.002/0001-11`. As válidas usam 2.000 sacados e 50 cedentes sintéticos (CNPJs com DV válido), tipo Duplicata ou Duplicata de Serviço, valor entre R$ 150 e R$ 50.000, vencimento entre data-base + 5 e + 170 dias e número `G-%08d` único.

**`Exemplo`:** ≥ 10 aprovadas (incluindo R$ 10.000,00 a 30 dias, sacado `LOJA AURORA LTDA` `55.566.677/0001-83`) e pelo menos uma de cada: `CNPJ_SACADO_VALIDO`, `SITUACAO_CADASTRAL_SACADO`, `TIPO_RECEBIVEL`, `VALOR_MINIMO`, `NAO_VENCIDA`, `PRAZO_MAXIMO`, `CONCENTRACAO_MAXIMA_SACADO`, `DUPLICIDADE`, `DUPLICADA_NO_ARQUIVO` e `FORMATO_INVALIDO`.

**CLI (`antecipe <subcomando>`):** dados em stdout, logs e resumo (só contagens) em stderr, saída 0 = ok, 1 = uso, 2 = arquivo.
- `gerar --linhas N --semente S [--estoque data/estoque_antecipe.zip] [--data-base 2026-10-22]`
- `exemplo [--estoque …]`
- `validar --regras data/validadores_antecipe.json --estoque data/estoque_antecipe.zip [--config config/antecipe.yaml] [--situacao data/situacao_cadastral.csv] [--data-base AAAA-MM-DD]` lê CSV no stdin e escreve CSV no stdout: colunas de entrada + `STATUS;MOTIVO;O_QUE_FAZER;DIAS;DESCONTO;VALOR_LIQUIDO`.

- [ ] **Step 1: Testes**

```go
func TestLerEstoqueZip(t *testing.T)        // ../../../data/estoque_antecipe.zip → 100554 notas, 10621 do fundo "92234400000180"
func TestContextoDeEstoque(t *testing.T)    // len(Vendidas)==100554; Estoque.TotalCentavos==5307402800; raiz 58206002 / total entre 12,49% e 12,51%
func TestGerarDeterministico(t *testing.T)  // mesma semente → mesmos bytes; 1000 linhas → 1001 linhas de texto
func TestGerarParseavel(t *testing.T)       // csvnota.Ler da saída → 1000 notas
func TestExemploCobreTodosOsCodigos(t *testing.T) // motor.Validar no Exemplo (JSON original + composto + situações do mock) → Motivos tem os 10 códigos acima e Aprovadas >= 10
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** → PASS
- [ ] **Step 5:** `cd backend && go run ./cmd/antecipe exemplo --estoque ../data/estoque_antecipe.zip > ../data/exemplo.csv`
- [ ] **Step 6: Medir:** `time (go run ./cmd/antecipe gerar --linhas 500000 --semente 42 --estoque ../data/estoque_antecipe.zip | go run ./cmd/antecipe validar --regras ../data/validadores_antecipe.json --estoque ../data/estoque_antecipe.zip --config ../config/antecipe.yaml --situacao ../data/situacao_cadastral.csv > /dev/null)`. Meta: < 6 s. Anotar.
- [ ] **Step 7:** Commit `feat: CLI antecipe (gerar, exemplo, validar), fontes e dados de demonstração`

---

### Task 9: Banco, migrations e semente (`store`)

**Files:** Create `backend/migrations/0001_init.sql`, `backend/migrations/embed.go` (`package migrations; //go:embed *.sql`), `backend/internal/store/store.go`, `semente.go`, `contexto.go`, `store_test.go`. Modify `backend/cmd/api/main.go` (abre o banco, migra, semeia e faz `/api/saude` checar `Ping`).

**Interfaces (Produces):**

```go
type Store struct{ Pool *pgxpool.Pool }
var ErrNaoEncontrado = errors.New("não encontrado")
func Abrir(ctx context.Context, url string) (*Store, error)
func (s *Store) Migrar(ctx context.Context) error // todos os .sql em ordem, sob pg_advisory_lock(424242), idempotente
func (s *Store) SemearSeVazio(ctx context.Context, dataDir string, log *slog.Logger) error // estoque (COPY, origem 'seed', docs normalizados), regras v1 (autor "semente", resumo ["Versão inicial"]), situações
func (s *Store) ContextoMotor(ctx context.Context, fundoCNPJ string) (map[string]motor.Vendida, motor.EstoqueFundo, error) // 3 queries agrupadas
func (s *Store) Situacoes(ctx context.Context) (map[string]string, error)
func (s *Store) Ping(ctx context.Context) error
```

`0001_init.sql`:

```sql
CREATE TABLE IF NOT EXISTS estoque (
  id BIGSERIAL PRIMARY KEY, chave TEXT NOT NULL UNIQUE,
  fundo_doc TEXT NOT NULL, fundo_nome TEXT NOT NULL, cedente_doc TEXT NOT NULL, cedente_nome TEXT NOT NULL,
  sacado_doc TEXT NOT NULL, sacado_nome TEXT NOT NULL, numero TEXT NOT NULL, tipo TEXT NOT NULL,
  valor_centavos BIGINT NOT NULL, vencimento DATE NOT NULL, aquisicao DATE,
  origem TEXT NOT NULL CHECK (origem IN ('seed','cessao')), cessao_id UUID);
CREATE INDEX IF NOT EXISTS estoque_fundo_sacado ON estoque (fundo_doc, sacado_doc);
CREATE INDEX IF NOT EXISTS estoque_fundo_cedente ON estoque (fundo_doc, cedente_doc);
CREATE TABLE IF NOT EXISTS regras_versoes (versao INT PRIMARY KEY, regras JSONB NOT NULL, autor TEXT NOT NULL,
  resumo JSONB NOT NULL DEFAULT '[]', publicada_em TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS situacao_cadastral (cnpj TEXT PRIMARY KEY, situacao TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS lotes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), idempotency_key TEXT NOT NULL UNIQUE, hash_arquivo TEXT NOT NULL,
  data_base DATE NOT NULL, regras_versao INT NOT NULL REFERENCES regras_versoes(versao), precificacao JSONB NOT NULL,
  status TEXT NOT NULL, etapa TEXT, linhas_total INT NOT NULL DEFAULT 0, linhas_processadas INT NOT NULL DEFAULT 0,
  totais JSONB, motivos JSONB, erros_configuracao JSONB NOT NULL DEFAULT '[]', erro JSONB,
  criado_em TIMESTAMPTZ NOT NULL DEFAULT now(), iniciado_em TIMESTAMPTZ, concluido_em TIMESTAMPTZ, duracao_ms INT);
CREATE TABLE IF NOT EXISTS lote_notas (
  lote_id UUID NOT NULL REFERENCES lotes(id) ON DELETE CASCADE, linha INT NOT NULL,
  nome_fundo TEXT, doc_fundo TEXT, nome_cedente TEXT, doc_cedente TEXT, nome_sacado TEXT, doc_sacado TEXT,
  numero TEXT, tipo TEXT, valor_centavos BIGINT, vencimento DATE, aquisicao DATE, chave TEXT,
  status SMALLINT NOT NULL, motivo_principal TEXT, dias INT, desconto_centavos BIGINT, liquido_centavos BIGINT,
  recusas JSONB NOT NULL DEFAULT '[]', erro_formato JSONB, PRIMARY KEY (lote_id, linha));
CREATE INDEX IF NOT EXISTS lote_notas_filtro ON lote_notas (lote_id, status, motivo_principal);
CREATE TABLE IF NOT EXISTS cessoes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(), lote_id UUID NOT NULL UNIQUE REFERENCES lotes(id),
  idempotency_key TEXT UNIQUE, confirmada_em TIMESTAMPTZ NOT NULL DEFAULT now(), notas_cedidas INT NOT NULL,
  face_centavos BIGINT NOT NULL, desconto_centavos BIGINT NOT NULL, liquido_centavos BIGINT NOT NULL,
  notas_excluidas JSONB NOT NULL DEFAULT '[]');
```

Testes de integração usam `TEST_DATABASE_URL` (com `t.Skip` se vazia). Alvo novo no Makefile: `test-integracao` sobe só o `postgres`, cria o banco `antecipe_test` se não existir e roda `TEST_DATABASE_URL=postgres://antecipe:$$POSTGRES_PASSWORD@localhost:5433/antecipe_test?sslmode=disable go test -p 1 ./...` (lendo a senha do `.env`). Cada teste começa com `TRUNCATE` das tabelas.

- [ ] **Step 1: Testes**

```go
func TestMigrarIdempotente(t *testing.T) // Migrar 2× sem erro
func TestSemente(t *testing.T)           // count(estoque)==100554; regras_versoes v1 com 9 validadores; situacao_cadastral == 6; segunda chamada não duplica
func TestContextoMotor(t *testing.T)     // Total 5307402800; len(vendidas)==100554; soma por raiz 58206002 / total ≈ 12,50%
```

- [ ] **Step 2:** `make test-integracao` → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** → PASS. `make up` → `docker compose logs api` mostra a semente com só contagens.
- [ ] **Step 5:** Commit `feat: schema Postgres, migrations embutidas e semente do estoque`

---

### Task 10: Lotes assíncronos e API de leitura (`lotes`, `httpapi`)

**Files:**
- Create: `backend/internal/lotes/gerente.go`, `backend/internal/lotes/processar.go`
- Create: `backend/internal/store/lotes.go`, `backend/internal/store/notas.go`, `backend/internal/store/regras.go` (só leitura aqui: `RegrasPublicadas`)
- Create: `backend/internal/httpapi/servidor.go`, `erros.go`, `dto.go`, `lotes.go`, `notas.go`, `validar.go`, `config.go`, `cors.go`, `lotes_test.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes: T3 (`Ler`, `ValidarCabecalho`, `ContarLinhas`), T5 (`config.Atual`, `precificacao.Efetiva`/`Nova`), T6 (`textos`), T7 (`motor.Validar`), T9 (`Store`)
- Produces:

```go
package store
type Versao struct { Numero int; Regras regras.Regras; Autor string; PublicadaEm time.Time; Resumo []string }
func (s *Store) RegrasPublicadas(ctx context.Context) (Versao, error)
func (s *Store) ObterVersao(ctx context.Context, n int) (Versao, error) // ErrNaoEncontrado
type NovoLote struct { IdempotencyKey, Hash string; DataBase time.Time; RegrasVersao int; Precificacao precificacao.Config }
func (s *Store) CriarLote(ctx context.Context, n NovoLote) (id string, criado bool, status string, err error) // INSERT … ON CONFLICT (idempotency_key) DO NOTHING; se não criou, SELECT o existente
func (s *Store) ReiniciarLote(ctx context.Context, id string) error  // status na_fila, zera progresso e erro, apaga lote_notas
func (s *Store) MarcarEtapa(ctx context.Context, id, status, etapa string, total int) error
func (s *Store) GravarResultados(ctx context.Context, id string, rs []dominio.Resultado, aoAvancar func(int)) error // CopyFrom em lote_notas
type ResumoLote struct { Totais motor.Totais; Motivos map[string]int; ErrosConfig []motor.ErroConfig; DuracaoMs int }
func (s *Store) ConcluirLote(ctx context.Context, id string, r ResumoLote) error
func (s *Store) FalharLote(ctx context.Context, id, codigo, mensagem string) error
type Lote struct { /* todas as colunas de lotes + Cessao *Cessao */ }
func (s *Store) ObterLote(ctx context.Context, id string) (Lote, error) // ErrNaoEncontrado
type FiltroNotas struct { Status, Motivo, Busca, Ordenar, Direcao string; Pagina, PorPagina int }
type NotaLinha struct { /* colunas de lote_notas */ }
func (s *Store) ListarNotas(ctx context.Context, id string, f FiltroNotas) ([]NotaLinha, int, error) // Ordenar só por lista branca → coluna
func (s *Store) PercorrerNotas(ctx context.Context, id string, f FiltroNotas, fn func(NotaLinha) error) error // para o CSV
func (s *Store) RecuperarInterrompidos(ctx context.Context) (int, error) // na_fila/processando → erro {"codigo":"PROCESSAMENTO_INTERROMPIDO","mensagem":"O processamento foi interrompido. Envie o arquivo de novo."}

package lotes
type Progresso struct { Status, Etapa string; Processadas, Total, PosicaoFila int }
type Envio struct { CaminhoArquivo, Hash, IdempotencyKey string; DataBase time.Time }
func NovoGerente(st *store.Store, cfg *config.Atual, log *slog.Logger) *Gerente
func (g *Gerente) Iniciar(ctx context.Context) error // RecuperarInterrompidos + N workers (jobs_simultaneos)
func (g *Gerente) Enviar(ctx context.Context, e Envio) (id string, replay bool, err error)
func (g *Gerente) Progresso(id string) (Progresso, bool)
func (g *Gerente) PrecificacaoEfetiva(r regras.Regras) (precificacao.Config, []string)

package httpapi
type Deps struct { Store *store.Store; Gerente *lotes.Gerente; Config *config.Atual; Log *slog.Logger; GestorSenha string; CORSOrigens []string; TmpDir string }
func NovoServidor(d Deps) http.Handler
func escreverErro(w http.ResponseWriter, status int, codigo, mensagem string, detalhes any)
```

**Comportamento:**
- Chave de idempotência sem header = `sha256hex(hash|data_base|regras_versao|metodo|taxa|base)`. Em `Enviar`: se o lote existente estiver em `erro`, `ReiniciarLote` e enfileira (`replay=false`). Se estiver em outro status, `replay=true`.
- Job: `lendo` (`ContarLinhas` → total, depois `Ler` com progresso) → contexto (`ContextoMotor`, `Situacoes` + padrão do YAML, regras da versão do lote, `Calculadora` da precificação do lote) → `motor.Validar` (etapas `validando`, `concentracao`, `precificando`) → `salvando` (`GravarResultados`) → `ConcluirLote`. Logs em cada etapa: `lote_id`, `etapa`, `linhas`, `duracao_ms`. Ao final, `recover` → `FalharLote("FALHA_PROCESSAMENTO", "Não conseguimos processar o arquivo. Nada foi gravado. Tente de novo.")`. O arquivo temporário é apagado ao final.
- `POST /api/lotes`: `http.MaxBytesReader` (MB do YAML) + `r.MultipartReader()` em fluxo. Partes `data_base` e `arquivo` → arquivo temporário com `io.MultiWriter(arquivo, sha256)`. `ValidarCabecalho` → 422/415. Sem arquivo → 400 `ARQUIVO_AUSENTE`. Excedeu o tamanho → 413 `ARQUIVO_GRANDE_DEMAIS`. Resposta 202 com `Location`, ou 200 com `Idempotent-Replay: true`.
- `GET /api/lotes/{id}`: progresso da memória se o job estiver ativo, senão do banco. `etapas[]` derivado de `etapa`/`status`. Formato exato de `docs/api.md`.
- `GET /api/lotes/{id}/notas`: renderiza `titulo/mensagem/acao` com `textos.Recusa`, documentos com `documento.Formatar(Normalizar(...))` e `calculo` das aprovadas com a `Calculadora` da precificação do lote. `por_pagina` ≤ 200.
- `GET /api/lotes/{id}/notas.csv`: `PercorrerNotas` em fluxo, `;`, colunas de entrada + `STATUS;MOTIVO;O_QUE_FAZER;DIAS;DESCONTO;VALOR_LIQUIDO`.
- `POST /api/validar`: JSON (`{"data_base","notas":[…]}` com os campos em minúsculas do CSV) ou `text/csv`. Mais de 10.000 → 413 `VALIDACAO_GRANDE_DEMAIS` "Para mais de 10.000 notas, use POST /api/lotes.". Mesmo motor, nada gravado.
- `GET /api/config` conforme o contrato. CORS para `CORSOrigens` (inclui `OPTIONS`).

- [ ] **Step 1: Testes de integração** (`httptest.NewServer(NovoServidor(...))` com banco de teste semeado)

```go
func TestUploadExemploConcluiEMostraProgresso(t *testing.T) // POST data/exemplo.csv → 202; GET até "concluido" (≤ 10 s); totais.aprovadas >= 10; motivos tem DUPLICIDADE; GET /notas?status=recusada&motivo=PRAZO_MAXIMO → itens[0].recusas[0].titulo == "Prazo acima do limite"
func TestReenvioIdempotente(t *testing.T)       // mesmo arquivo 2× → segunda 200, replay true, mesmo lote_id; count(lotes) == 1
func TestEnviosSimultaneos(t *testing.T)        // 2 goroutines com o mesmo arquivo → mesmo lote_id (Review Focus 5)
func TestCabecalhoInvalido(t *testing.T)        // sem DATA_VENCIMENTO_ORIGINAL → 422, erro.codigo == "CABECALHO_INVALIDO"
func TestRecuperarInterrompidos(t *testing.T)   // lote em "processando" no banco → Iniciar → GET status "erro", erro.codigo "PROCESSAMENTO_INTERROMPIDO"; reenviar o mesmo arquivo → 202 e conclui (Review Focus 3)
func TestValidarJSON(t *testing.T)              // 1 nota R$ 10.000,00 a 30 dias → notas[0].liquido_centavos == 980392, precificacao.metodo == "DESCONTO_COMPOSTO"
func TestNotasCSV(t *testing.T)                 // Content-Type text/csv; primeira linha termina com "STATUS;MOTIVO;O_QUE_FAZER;DIAS;DESCONTO;VALOR_LIQUIDO"
func TestLoteInexistente404(t *testing.T)
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** `make test-integracao` → PASS
- [ ] **Step 5: Verificar carga à mão:** `make up`, gerar 500k (`docker compose run --rm api antecipe gerar --linhas 500000 --semente 42 --estoque data/estoque_antecipe.zip > amostras/lote_500k.csv`) e `curl -F arquivo=@amostras/lote_500k.csv localhost:8080/api/lotes`. Consultar até concluir. Anotar `duracao_ms` (meta ≤ 15.000).
- [ ] **Step 6:** Commit `feat: processamento assíncrono de lotes com progresso por etapa e API de consulta`

---

### Task 11: Cessão segura contra venda dupla

**Files:** Create `backend/internal/store/cessao.go`, `backend/internal/httpapi/cessao.go`, `backend/internal/httpapi/cessao_test.go`, `scripts/demo-concorrencia.sh`. Modify `Makefile` (`demo-concorrencia`).

**Interfaces:**
- Consumes: `motor.Concentrar` (T7), `store.ObterVersao`/`ObterLote` (T10)
- Produces:

```go
type Cessao struct { ID, LoteID string; ConfirmadaEm time.Time; NotasCedidas int; FaceCentavos, DescontoCentavos, LiquidoCentavos int64 }
type Excluida struct { Linha int; Codigo, Titulo, Mensagem string }
var ErrLoteNaoConcluido, ErrSemAprovadas, ErrChaveReutilizada error
func (s *Store) Ceder(ctx context.Context, loteID, idemKey string) (c Cessao, excluidas []Excluida, replay bool, err error)
```

**Transação, nesta ordem:**
1. `SELECT … FROM lotes WHERE id=$1 FOR UPDATE`. Se não estiver `concluido` → `ErrLoteNaoConcluido`.
2. Cessão existente para o lote → devolve a mesma, com `replay=true`.
3. Sem aprovadas → `ErrSemAprovadas`.
4. `SELECT pg_advisory_xact_lock(hashtext(fundo_doc))`.
5. Carrega as aprovadas e o `EstoqueFundo` **atual** (agrupado dentro da transação), roda `motor.Concentrar` com as regras da versão do lote, e as que estourarem vão para `excluidas` com o texto de `CONCENTRACAO_*`.
6. `CREATE TEMP TABLE cessao_tmp (LIKE estoque INCLUDING DEFAULTS) ON COMMIT DROP`, `CopyFrom` das restantes (origem `cessao`, `aquisicao = current_date`), e `INSERT INTO estoque (…) SELECT … FROM cessao_tmp ON CONFLICT (chave) DO NOTHING RETURNING chave`. Chave que não voltou → `Excluida{Codigo:"DUPLICIDADE", Titulo:"Nota já vendida", Mensagem:"Esta nota foi vendida em outra operação enquanto você revisava o lote."}`.
7. `INSERT INTO cessoes` com os totais **do que entrou** (somando desconto e líquido gravados em `lote_notas`) e `notas_excluidas`. `idempotency_key` = header ou `"lote:" + id`. Conflito de chave com outro lote → `ErrChaveReutilizada` (409 `IDEMPOTENCY_KEY_REUTILIZADA`).

HTTP: 201 / 200 + `Idempotent-Replay: true` / 409 `LOTE_NAO_CONCLUIDO` · `SEM_APROVADAS` / 404, com o corpo de `docs/api.md`.

- [ ] **Step 1: Testes**

```go
func TestCessaoReplay(t *testing.T)  // exemplo concluído → POST cessao 201; de novo 200 replay; count(estoque origem cessao) não muda na segunda
func TestCessoesSimultaneasNaoVendemDuasVezes(t *testing.T) {
    // A = gerar(200, semente 1); B = primeiras 20 linhas de A + gerar(200, semente 2); os dois concluídos ANTES de qualquer cessão
    // POST /cessao de A e de B ao mesmo tempo → as 20 chaves comuns aparecem em exatamente uma cessão;
    // a outra lista essas linhas em notas_excluidas com DUPLICIDADE; count(estoque cessao) == |aprovadas(A) ∪ aprovadas(B)|
}
func TestCessaoLoteProcessando409(t *testing.T)
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** `make test-integracao` → PASS
- [ ] **Step 5:** `scripts/demo-concorrencia.sh` (bash + curl + o `antecipe gerar` via `docker compose run`): monta A e B com sobreposição, envia, espera concluir, dispara as duas cessões com `&` e `wait`, e imprime "A: N cedidas, M excluídas · B: …". Alvo `make demo-concorrencia`.
- [ ] **Step 6:** Commit `feat: cessão transacional idempotente e segura contra venda dupla simultânea`

---

### Task 12: Regras pela API (gestor) e prévia de impacto

**Files:** Create `backend/internal/httpapi/regras.go`, `backend/internal/httpapi/gestor.go`, `regras_test.go`. Modify `backend/internal/store/regras.go`, `backend/internal/store/notas.go`.

**Interfaces (Produces):**

```go
func (s *Store) PublicarRegras(ctx context.Context, r regras.Regras, autor string, resumo []string) (Versao, error) // versao = max+1, em transação com lock
func (s *Store) ListarVersoes(ctx context.Context) ([]Versao, error)
func (s *Store) NotasOriginais(ctx context.Context, loteID string) ([]dominio.Nota, error) // reconstrói as Notas de lote_notas (incl. ErroFormato)
func (s *Store) UltimoLoteConcluido(ctx context.Context) (string, error)
```

- Middleware do gestor: `subtle.ConstantTimeCompare` com `X-Gestor-Senha`. `GESTOR_SENHA` vazia → 503 `GESTOR_DESABILITADO`. Errada → 401 `SENHA_INVALIDA`. O valor do header nunca é logado.
- Rotas: `GET /api/gestor/verificar` (204), `GET /api/regras` (com `catalogo`, `precificacao_efetiva`, `catalogo_precificacao`), `GET /api/regras/versoes`, `GET /api/regras/versoes/{versao}`, `PUT /api/regras` (`ValidarSchema` com `precificacao.Metodos()` → 400 `REGRAS_INVALIDAS`; `Resumo` contra a publicada), `POST /api/regras/previa` (sem lote → 409 `SEM_LOTE`; "antes" = totais e motivos gravados do lote; "depois" = `motor.Validar(NotasOriginais, propostas, contexto atual)`).

- [ ] **Step 1: Testes**

```go
func TestPutRegrasCriaVersao(t *testing.T) // PUT com prazo_maximo_dias 120 → 201 versao 2; GET /api/regras versao 2; versoes[0].resumo == ["PRAZO_MAXIMO: 180 → 120 dias"]
func TestPutSemSenha401(t *testing.T)
func TestPutInvalido400(t *testing.T)      // prazo -1 → detalhes[0].campo == "validadores[5].parametros.prazo_maximo_dias"
func TestPreviaMudaAprovadas(t *testing.T) // após o exemplo, prévia com prazo 30 → depois.aprovadas < antes.aprovadas e mudancas == ["PRAZO_MAXIMO: 180 → 30 dias"]
func TestNovoLoteUsaNovaVersao(t *testing.T) // após o PUT, reenviar o exemplo → 202 (lote novo) com regras_versao 2
func TestRegrasMostraMetodoDoYAML(t *testing.T) // precificacao_efetiva.metodo == "DESCONTO_COMPOSTO", sobrescritos_pelo_yaml == ["metodo"]
```

- [ ] **Step 2:** → FAIL · [ ] **Step 3:** Implementar · [ ] **Step 4:** → PASS
- [ ] **Step 5:** Commit `feat: regras versionadas pela API do gestor, com prévia de impacto`

---

### Task 13: Política de logs, carga, segredos e documentação

**Files:** Create `backend/internal/httpapi/logs_test.go`, `scripts/carga.sh`, `README.md`, `docs/relatorio.md`. Modify `Makefile` (`gerar`, `carga`, `check`).

- [ ] **Step 1: Teste**

```go
func TestLogsNaoVazamDados(t *testing.T) {
    // logger num bytes.Buffer; upload de data/exemplo.csv até concluir + GET /notas + POST /validar + PUT /regras com a senha
    // para cada nota do exemplo: o buffer NÃO contém DocSacado, DocCedente (formatado e normalizado), NomeSacado, NomeCedente, nem Numero com len >= 5
    // o buffer NÃO contém a GESTOR_SENHA de teste nem "postgres://"
}
```

- [ ] **Step 2:** `make test-integracao` → PASS. Se falhar, corrigir a origem do log, nunca o teste.
- [ ] **Step 3: Makefile.**
  - `gerar` (`N ?= 500000`, `SEMENTE ?= 42`): `docker compose run --rm -T api antecipe gerar --linhas $(N) --semente $(SEMENTE) --estoque data/estoque_antecipe.zip > amostras/lote_$(N).csv`.
  - `carga`: roda `scripts/carga.sh` para 100000, 250000 e 500000. O script gera o arquivo, envia, consulta até concluir e imprime uma tabela "linhas · duração · aprovadas · recusadas".
  - `check`: `docker run --rm -v "$$PWD:/repo" zricethezav/gitleaks:v8.21.2 detect --source /repo --no-banner`.
- [ ] **Step 4:** `make carga` e anotar os números. `make check` → "no leaks found".
- [ ] **Step 5: `README.md`:**
  - O que é, em 3 linhas, e pré-requisitos (Docker 24+, Docker Compose v2, make).
  - `make up` e as URLs, e onde está a senha do gestor.
  - Como usar o `data/exemplo.csv` e o `make gerar`.
  - `curl` de cada rota principal e a CLI com o pipe Unix.
  - Como trocar juros compostos/simples no `config/antecipe.yaml`.
  - Tabela de carga, decisões (lista do plano seção 17) e limitações.
  - Relatório curto apontando para `docs/relatorio.md`, e caminho alternativo sem Docker.
- [ ] **Step 6: `docs/relatorio.md`** (1–2 páginas): problema, solução, arquitetura (fluxo do lote), regras, preço, números medidos e próximos passos.
- [ ] **Step 7: Clone limpo:** `git clone <repo> "$TMPDIR/antecipe-limpo" && cd "$TMPDIR/antecipe-limpo" && make up` → saúde ok e upload do `data/exemplo.csv` conclui.
- [ ] **Step 8:** Commit `docs: README, relatório, teste de logs, carga e varredura de segredos`

---

### Task 14: Integração do front (quando o código chegar)

**Files:** Replace `frontend/` pelo app do colega, mantendo `frontend/Dockerfile` e `frontend/nginx.conf`.

- [ ] **Step 1:** `frontend/Dockerfile`: estágio `node:22-alpine` com `npm ci && npm run build` (ajustar ao gerenciador dele: pnpm ou npm), e estágio `nginx:1.27-alpine` servindo `dist/` com fallback de SPA (`try_files $uri /index.html`), mais o mesmo proxy `/api/` da Task 1.
- [ ] **Step 2:** Conferir as chamadas do front contra `docs/api.md`: upload com progresso, polling a cada 500 ms, paginação no servidor, `Idempotency-Key` na cessão e `X-Gestor-Senha`. Divergência → combinar e registrar no histórico do contrato.
- [ ] **Step 3:** `make up` e roteiro do pitch no navegador: exemplo → resultado → recusa explicada → reenvio idempotente → cessão → gestor muda o prazo para 120 → prévia → publica → reenvio com resultado novo → 500k com andamento.
- [ ] **Step 4:** Commit `feat: integra o front React ao make up`
