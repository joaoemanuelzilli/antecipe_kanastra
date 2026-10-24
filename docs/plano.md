# Antecipe — plano de construção

Hackathon Kanastra · 24/10/2026 · entrega até 18:00 · documento vivo

> Fontes: edital (`context/Edital Hackathon Kanastra.pdf`), página do Notion do desafio (texto, tabela do CSV e anexos), vídeo do desafio (narração e telas), `estoque_antecipe.csv` e `validadores_antecipe.json`. O plano anterior (`context/Antecipe-Claro-Projeto-Completo.pdf`, em Next.js + Supabase) foi usado como base de UX e de regras, mas foi pensado para 1.000 notas e não cobre o que pesa mais na nota.

---

## 1. Por que e para quem

Uma pequena empresa (a "Ana" do vídeo) vendeu a prazo e quer o dinheiro hoje. Ela sobe um CSV de notas, e o portal diz quais o fundo ANTECIPE FIDC compra, quanto ela recebe e, para cada recusa, o motivo e o que fazer. O gestor do fundo muda as regras sem deploy.

Slogan do vídeo: **Importar. Decidir. Explicar. Simular.**

### Onde estão os pontos (edital, item 5.2)

| Critério | Pontos | O que precisamos mostrar |
|---|---|---|
| Escalabilidade | 35 | Concluir **todas as etapas** com lotes de **100 mil a 500 mil linhas**, rápido, e **mostrando o andamento** (vídeo) |
| Design e UX | 25 | Mensagens de recusa claras, design, criatividade, interface |
| Organização (filosofia Unix) | 20 | Cada parte faz uma coisa só, código limpo |
| Logs e credenciais | 15 | Logs estruturados, **nenhum dado do CSV nos logs**, nenhum segredo no repositório |
| Registro de prompts | 5 | `docs/prompts.md` em ordem cronológica, com o que foi feito de cada resposta |

**Desclassifica:** app que não sobe seguindo o README. Por isso a regra é que `make up` funcione numa máquina limpa.

### Requisitos mínimos (Notion)

1. Upload de CSV no formato do exemplo (`;`, decimal com vírgula, `dd/mm/aaaa`, UTF-8, cabeçalho).
2. Regras lidas do `validadores_antecipe.json`, **sem valores fixos no código**.
3. Status de cada nota e, se recusada, o motivo explicado de um jeito que a empresa entenda.
4. Valor líquido de cada nota aprovada e o total a receber.
5. A mesma validação pela tela e por uma API.
6. Relatório curto (pode ir no README), histórico de prompts e apresentação.

### O vídeo acrescenta

- "trata reenvios **sem repetir operações**" → idempotência.
- "impede vendas duplicadas, **mesmo simultâneas**" → trava de concorrência na venda.
- "em lotes de **cem mil a quinhentas mil linhas**, **sempre mostrando o andamento**" → processamento assíncrono com progresso por etapa.
- Exemplo de preço: R$ 10.000 a 30 dias com 2% → desconto R$ 200 → líquido R$ 9.800.
- Exemplo de recusa: "Prazo acima do limite: vence em 90 dias; o limite configurado é 60."

---

## 2. O que os dados mostram

- `estoque_antecipe.csv`: **100.554 linhas, 18 fundos**, 10.621 do ANTECIPE FIDC (CNPJ 92.234.400/0001-80, R$ 53.074.028,00 de face). UTF-8 sem BOM, LF, sem aspas, 11 colunas em todas as linhas. Todos os CNPJs têm dígito verificador válido e todas as datas e valores são válidos. Ou seja, o estoque é limpo, e os erros vão vir nos arquivos que os jurados subirem.
- **Data-base dos dados: 22/10/2026** (está no Notion).
- **Parcelas:** 3.018 grupos (12.675 linhas) têm o mesmo cedente + sacado + número com vencimentos mensais diferentes, e a última parcela leva a diferença de centavos. **A chave cedente + número + vencimento é única no estoque inteiro.** Uma chave sem o vencimento recusaria parcelas legítimas.
- **Armadilha de concentração:** o sacado de raiz **58206002** (SERRALHERIA VIOLETA S.A.) já tem **12,50%** do estoque do ANTECIPE, acima do limite de 10%. Toda nota nova dele será recusada. O cedente 10.062.050/0001-57 tem 22,6% (só importa se o gestor ligar `CONCENTRACAO_MAXIMA_CEDENTE`).
- No ANTECIPE: 203 notas já vencidas na data-base, só Duplicata e Duplicata de Serviço, nenhuma abaixo de R$ 100. Os outros fundos têm CCB, Cheque e Nota Comercial.
- O CSV não traz a situação cadastral do sacado, então ela precisa de uma fonte simulada (seção 7).
- 500 mil linhas ≈ 97 MB (média de 200 bytes por linha).

### Com que arquivos os jurados vão testar?

O único arquivo grande público é o próprio estoque (o zip se chama `estoque_antecipe_100k`, o que sugere que os jurados têm versões maiores). Se subirem o estoque, ou um arquivo de 500 mil feito a partir dele, as linhas que já estão no estoque saem todas como **"Nota já vendida"**. Isso é o comportamento certo, e é justamente o teste de "sem venda duplicada". O processamento passa por todas as etapas do mesmo jeito.

Para a demo e para medir o desempenho, precisamos de lotes com notas novas. Por isso o gerador (`antecipe gerar`, exposto como **`make gerar`**) está no caminho crítico, não na lista de corte. Ele gera 100k, 250k e 500k linhas em `amostras/` (fora do git) com mistura realista. O pequeno `data/exemplo.csv` (≈ 40 linhas, uma recusa de cada tipo) é commitado. O README explica os dois usos.

---

## 3. Decisões do time (24/10, 10:45)

1. **Venda em dois passos:** o upload **simula** e "Confirmar cessão" **vende**, com trava no banco contra venda dupla, inclusive simultânea.
2. **Tela do gestor:** editar parâmetros, ligar e desligar regras, ver a prévia do impacto no último lote, publicar uma nova versão e ver o histórico.
3. **Postgres + Docker**, e tudo sobe com **`make up`** para facilitar ao máximo para o jurado.
4. **Divisão:** uma pessoa faz o **backend Go nesta máquina** (com o Claude Code), outra já está fazendo o **front React** em paralelo, e no fim integramos. O contrato entre os dois é o [`docs/api.md`](api.md).
5. **Juros compostos por padrão** (alinhado com um jurado), configurável em **YAML** (`config/antecipe.yaml`) para voltar a simples sem mexer no código (seção 7).

---

## 4. Metas, não-metas e anti-metas

**Metas**
- Lote de 500 mil linhas processado de ponta a ponta (upload → resultado salvo) em **≤ 15 s** num notebook, com progresso visível por etapa. Meta interna: parse + regras em ≤ 2 s.
- Todas as 9 regras do JSON implementadas por um registro de validadores, ligadas e parametrizadas só pelo JSON publicado.
- Toda recusa com título, mensagem e ação, geradas pelo backend (tela e API mostram o mesmo texto).
- Reenvio do mesmo arquivo não cria nada novo. Duas confirmações simultâneas nunca vendem a mesma nota duas vezes.
- `make up` numa pasta limpa sobe tudo. Nenhum segredo no git e nenhum dado do CSV nos logs.

**Não-metas (ficam de fora e entram no relatório)**
- Consulta real à Receita, login de várias empresas, assinatura da cessão, registradora, pagamento e deploy público.
- IA dentro do app. O texto das recusas é determinístico, o que é mais rápido em 500 mil linhas, sem chave de API e sem risco.

**Anti-metas (seria fracasso mesmo funcionando)**
- Carregar 500 mil linhas no navegador ou numa única resposta JSON.
- Logar nome, CNPJ, número, valor ou data de qualquer nota.
- Qualquer número de regra (180, 10%, 2%, R$ 100…) escrito no código.
- Uma linha ruim derrubar um lote de 500 mil.
- README que só funciona na máquina de quem escreveu.

---

## 5. Stack

| Camada | Escolha | Por quê |
|---|---|---|
| Backend | **Go 1.27** (só biblioteca padrão onde der) | Lê CSV em fluxo com pouca memória, paralelismo simples com goroutines, binário único. `net/http` já roteia por método e caminho, e `log/slog` dá logs JSON estruturados sem dependência. |
| Banco | **PostgreSQL 18** + **pgx v5** (`pgxpool`, `CopyFrom`) | `COPY` grava centenas de milhares de linhas em segundos, e índice único com `ON CONFLICT` garante "sem venda duplicada" mesmo com requisições simultâneas. |
| Dinheiro | **centavos em `int64`** + **shopspring/decimal v1.6** só para taxa e fator composto (memorizado por prazo). Parâmetros do JSON lidos com `json.Number` | Exato e rápido em 500 mil linhas. Nada de `float64` em valor. |
| Configuração | **YAML** (`go.yaml.in/yaml/v3` v3.0.5) em `config/antecipe.yaml` para o comportamento; **env** (`.env`) só para segredos e infra | Arquivo legível e versionado. Segredos fora do git. |
| Front (outra pessoa) | **React 19 + Vite 8 + TypeScript**, Tailwind 4, shadcn/ui, TanStack Query (progresso por polling) e TanStack Table (paginação no servidor) | SPA simples, build estático servido por nginx. Não precisa de Next.js. |
| Infra | **Docker Compose** (postgres, api, web) + **Makefile** | `make up` é o único comando. Jurado não precisa de Go nem de Node. |
| Qualidade | `go test`, benchmarks `testing.B`, `go vet`, **gitleaks** (varredura de segredos) | Testes do motor, carga medida e repositório limpo. |

Dependências Go previstas: `pgx/v5`, `go.yaml.in/yaml/v3` e `shopspring/decimal`, e mais nada essencial. Migrations em SQL puro embutidas no binário (`embed`) e aplicadas na subida.

---

## 6. Arquitetura (hexagonal + filosofia Unix)

**Decisão do time (24/10, 11:42): arquitetura hexagonal com pastas em inglês.** `handler` para dependências externas, `service` para regras de negócio, `repository` para o banco, testes ponta a ponta, erros pelo `apperrors` e observabilidade com OpenTelemetry. Os nomes de tipos e funções ficam em português, porque são a linguagem do negócio (nota, cedente, sacado).

Cada pacote faz uma coisa só. O domínio e os serviços não importam adaptadores, e os adaptadores implementam as portas definidas em `domain`. Dados vão para stdout e logs para stderr.

```
backend/
  cmd/api/                 servidor HTTP (só liga as peças)
  cmd/antecipe/            CLI: gerar · exemplo · validar (CSV stdin → CSV stdout)
  internal/
    domain/                entidades (Nota, Resultado, Lote, Cessao…) e portas (interfaces dos repositórios)
      rules/               regras do fundo: JSON, catálogo, validação de schema, resumo de mudanças
    service/               regras de negócio
      engine/              motor puro: regras em paralelo → duplicidade → concentração → preço
      validator/           uma função pura por regra
      pricing/             juros compostos (padrão) e simples, em centavos
      message/             texto de cada recusa
      batch/               caso de uso: lotes assíncronos (fila, etapas, progresso)
      assignment/          caso de uso: cessão segura contra concorrência
      rulebook/            caso de uso: versões das regras e prévia
    handler/               adaptadores de entrada
      rest/                HTTP (contrato docs/api.md)
      csvfile/             leitura do CSV pt-BR
    repository/            adaptadores de saída
      postgres/            pgx, migrations embutidas, COPY, transação da cessão
      filesystem/          estoque em zip e situação cadastral (CLI e semente)
    platform/              apperrors · observability (slog + OpenTelemetry) · config (YAML) · document (CNPJ) · format (pt-BR) · generator
    bootstrap/             raiz de composição (cmd/api e testes ponta a ponta)
  test/e2e/                testes ponta a ponta: aplicação inteira + Postgres
config/antecipe.yaml · data/ · frontend/ · docs/ · docker-compose.yml · Makefile · .env.example · README.md
```

**Observabilidade:**
- traces OTLP para o Jaeger (`http://localhost:16686`), com um span por requisição e um por etapa do lote;
- métricas em `/metrics` (lotes, notas, duração por etapa, notas cedidas);
- `trace_id` e `span_id` em cada linha de log.

A CLI mostra a filosofia Unix na prática, com o mesmo motor da API:

```bash
antecipe gerar --linhas 500000 --semente 42 \
  | antecipe validar --regras data/validadores_antecipe.json --estoque data/estoque_antecipe.zip \
  | grep ';RECUSADA;' | cut -d';' -f14 | sort | uniq -c
```

### Fluxo de um lote

```
POST /api/lotes (multipart)
  │  grava em arquivo temporário calculando o sha256 (sem guardar 100 MB em memória)
  │  confere o cabeçalho → 422 se faltar coluna ou se não for CSV
  │  chave de idempotência = header Idempotency-Key ou sha256(arquivo + data-base + versão das regras)
  │  INSERT lote ... ON CONFLICT (idempotency_key) → se já existe, devolve o mesmo lote (200, Idempotent-Replay: true)
  └─ 202 {lote_id}  →  job entra na fila

Job (worker)                                           etapa mostrada na tela
  1. conta as linhas (para o percentual)               "Lendo arquivo"
  2. parse em fluxo → blocos de 4.096 linhas → N workers validam as regras por nota    "Validando regras"
  3. duplicidade: estoque (todos os fundos) + cessões + repetidas no arquivo
  4. concentração: só notas sem recusa, ordenadas por (vencimento, número, linha)       "Calculando concentração"
  5. preço das aprovadas                               "Precificando"
  6. COPY dos resultados em lote_notas + totais        "Salvando resultados"
  7. status = concluído (duração registrada)

GET /api/lotes/{id}  ← o front consulta a cada 500 ms: etapa, linhas processadas/total, % e, no fim, totais e resumo dos motivos
GET /api/lotes/{id}/notas?status=&motivo=&busca=&pagina=  ← tabela paginada no servidor
POST /api/lotes/{id}/cessao  ← venda (seção 8)
```

O progresso fica num registro em memória do job (lido pelo GET) e é gravado no banco a cada troca de etapa. Por padrão rodam dois jobs ao mesmo tempo, e os outros esperam com status `na_fila`.

---

## 7. Regras

O motor lê a **versão publicada** em `regras_versoes`. A versão 1 é criada na primeira subida a partir de `data/validadores_antecipe.json`. Só rodam validadores com `ativo: true`. Cada nota passa por **todas** as regras ativas e guarda **todos** os motivos. O principal é o primeiro na ordem em que aparecem no JSON.

| Código | Regra (parâmetros sempre do JSON) |
|---|---|
| `CNPJ_SACADO_VALIDO` | Normaliza (tira `. / -` e espaços, maiúsculas). Exige 14 caracteres `[0-9A-Z]{12}[0-9]{2}`, recusa todos iguais e confere os 2 DVs por módulo 11, com o valor de cada caractere = ASCII − 48. Isso aceita o **CNPJ alfanumérico** emitido desde julho/2026. |
| `SITUACAO_CADASTRAL_SACADO` | Situação vinda da fonte simulada (`situacao_cadastral`), conferida contra `situacoes_aceitas`. CNPJ fora da fonte usa `SITUACAO_CADASTRAL_PADRAO` (env, padrão `ATIVA`). A semente tem alguns `INAPTA`, `BAIXADA` e `SUSPENSA`, documentados no README. |
| `TIPO_RECEBIVEL` | `tipos_aceitos`, comparando sem diferenciar maiúsculas, acentos e espaços nas pontas. |
| `VALOR_MINIMO` | valor ≥ `valor_minimo`. |
| `NAO_VENCIDA` | vencimento > data-base. |
| `PRAZO_MAXIMO` | dias ≤ `prazo_maximo_dias`. Calcula também "dá para antecipar a partir de" = vencimento − prazo máximo. |
| `PRAZO_MINIMO` | dias ≥ `prazo_minimo_dias` (desligada no JSON). |
| `CONCENTRACAO_MAXIMA_SACADO` | Agrupa por `agrupamento` (`RAIZ_CNPJ` = 8 primeiros caracteres; `CNPJ` = completo). Recusa se (estoque do grupo + aceitas do grupo no lote + face) ÷ (estoque do fundo + aceitas no lote + face) > `percentual_maximo`. |
| `CONCENTRACAO_MAXIMA_CEDENTE` | Mesma conta por cedente (CNPJ completo, ou o `agrupamento` se vier). Desligada no JSON. |

- **Dias** = vencimento − data-base, em dias corridos. A data-base vem do upload e tem como padrão `DATA_BASE_PADRAO` (env, padrão `2026-10-22`, a data-base dos dados).
- **Estoque do fundo** para a concentração = notas do estoque com o CNPJ de `fundo.cnpj` do JSON, mais as cessões já confirmadas. Somado em uma query agrupada no início do job.
- **Código desconhecido** no JSON vira um erro de configuração listado no resultado do lote e no log (só o código), e as outras regras continuam. Parâmetro com tipo errado: o `PUT` recusa. Se mesmo assim chegar ao motor, vira erro de configuração daquela regra.
- **Regras de integridade sempre ligadas** (não são política do fundo, então não estão no JSON):
  - `DUPLICIDADE`: nota já vendida, ou seja, no estoque de qualquer um dos 18 fundos ou numa cessão confirmada.
  - `DUPLICADA_NO_ARQUIVO`: mesma chave numa linha anterior do mesmo arquivo.
  - `FORMATO_INVALIDO`: a linha não pôde ser lida, com campo e problema ("data 31/02/2026 não existe"). Só essa linha é recusada e o resto do lote segue.
- **Chave da nota** = CNPJ do cedente normalizado + `NU_DOCUMENTO` sem espaços nas pontas + vencimento (ISO). O valor fica fora da chave de propósito: a mesma nota reenviada com outro valor continua sendo duplicata, e a mensagem mostra o valor original.
- **Erro do arquivo inteiro (422):** arquivo vazio, não-texto, sem cabeçalho ou com coluna obrigatória faltando. O parser aceita BOM, CRLF, linha em branco no fim, campos entre aspas e valor com separador de milhar (`1.234,56`).

### Preço: juros compostos por padrão, trocável em YAML

Decisão do time (24/10, ~11:05, depois de conversar com um jurado): **juros compostos**, com a opção de voltar para simples depois **editando um arquivo YAML**.

**`config/antecipe.yaml`** (commitado, sem segredos):

```yaml
# Configuração do comportamento do Antecipe. Segredos ficam no .env, nunca aqui.
precificacao:
  metodo: DESCONTO_COMPOSTO        # DESCONTO_COMPOSTO | DESCONTO_SIMPLES
  # taxa_mensal_percentual e base_dias vêm das regras publicadas (validadores_antecipe.json).
  # Descomente para sobrescrever:
  # taxa_mensal_percentual: 2.0
  # base_dias: 30

processamento:
  data_base_padrao: "2026-10-22"
  situacao_cadastral_padrao: ATIVA
  jobs_simultaneos: 2
  tamanho_maximo_upload_mb: 300
```

**Precedência (de cima para baixo):**
1. `precificacao.metodo` (e `taxa_mensal_percentual`/`base_dias`, se descomentados) do **YAML**.
2. `precificacao` da **versão publicada das regras** (semente: o JSON, que diz `DESCONTO_SIMPLES`, 2%, 30).

**Sem reinício:** a API relê o YAML quando o arquivo muda (checa a data de modificação a cada 2 s, sem dependência extra). No compose, `./config` é montado como volume, então editar o arquivo no host vale para o próximo lote, sem rebuild. Um YAML inválido é recusado com log de erro, e a API continua com a última configuração válida. Cada lote grava o método, a taxa e a base que usou, e o resultado mostra isso ("Juros compostos · 2% ao mês").

**Fórmulas** (dias corridos, `i = taxa_mensal_percentual ÷ 100`, `n = dias ÷ base_dias`):

```
DESCONTO_COMPOSTO (padrão; desconto racional composto, o padrão de mercado em FIDC)
  líquido  = face ÷ (1 + i)^n
  desconto = face − líquido

DESCONTO_SIMPLES (o que o JSON original define)
  desconto = face × i × n
  líquido  = face − desconto
```

- Valores sempre em **centavos inteiros**, arredondados meio-para-cima **por nota**. Totais = soma das notas arredondadas.
- O fator `(1 + i)^n` usa `shopspring/decimal` (`PowWithPrecision`, 20 casas) e é **memorizado por número de dias**. Um lote de 500 mil linhas tem só algumas centenas de prazos diferentes, então são poucas centenas de potências por lote. Nenhum `float64` toca em dinheiro.
- Os métodos ficam num registro (`código → função`), como os validadores. Método desconhecido no YAML ou no `PUT` é recusado.

**Conferência com R$ 10.000,00 a 2% ao mês:**

| Prazo | Composto (padrão) | Simples |
|---|---|---|
| 30 dias | líquido **R$ 9.803,92** (desconto R$ 196,08) | R$ 9.800,00 (desconto R$ 200,00, o exemplo do vídeo) |
| 60 dias | **R$ 9.611,69** | R$ 9.600,00 |
| 180 dias | **R$ 8.879,71** | R$ 8.800,00 |

Esses três valores viram testes dos dois métodos.

### Textos de recusa (formato decisão → regra → consequência → ação)

Gerados pelo backend a partir de `(codigo, dados)` na leitura. Não guardamos 500 mil strings no banco. Exemplos:

- **PRAZO_MAXIMO:** "Prazo acima do limite · Esta nota vence em 240 dias e o fundo compra notas com até 180 dias. · Você pode antecipá-la a partir de 21/12/2026." (vencimento 19/06/2027, data-base 22/10/2026)
- **CONCENTRACAO_MAXIMA_SACADO:** "Limite por cliente atingido · Com esta nota, SERRALHERIA VIOLETA ficaria com 12,6% da carteira do fundo, e o limite é 10%. · Tente com notas de outros clientes ou aguarde o vencimento das notas atuais desse cliente."
- **DUPLICIDADE:** "Nota já vendida · Esta nota (nº 725714, vencimento 07/10/2026) já foi cedida a um fundo em 27/04/2026. · Não é preciso enviar de novo."
- **SITUACAO_CADASTRAL_SACADO:** "CNPJ do cliente com pendência · O CNPJ de quem paga a nota está INAPTA na Receita. · Confirme os dados com seu cliente."
- **TIPO_RECEBIVEL:** "Tipo de nota não aceito · O fundo compra só Duplicata e Duplicata de Serviço, e esta nota é Cheque. · Envie esta nota para outro tipo de operação."
- **FORMATO_INVALIDO:** "Linha com erro · Na linha 1.204, o vencimento '31/02/2026' não é uma data válida. · Corrija a linha e envie de novo."

---

## 8. Idempotência, duplicidade e concorrência

- **Upload idempotente:** `lotes.idempotency_key` é `UNIQUE`. Dois envios iguais, mesmo ao mesmo tempo, resultam num só lote e o segundo recebe o mesmo `lote_id`. A versão das regras faz parte da chave derivada, então reenviar depois de mudar a regra gera um resultado novo (é o roteiro do pitch).
- **Cessão (`POST /api/lotes/{id}/cessao`):** numa transação, nesta ordem:
  1. `SELECT … FOR UPDATE` no lote. Se já foi cedido, devolve a **mesma** cessão (200, `Idempotent-Replay: true`). O duplo clique não repete nada.
  2. `pg_advisory_xact_lock(fundo)` serializa as cessões do mesmo fundo, porque a concentração depende do estoque.
  3. Recalcula a concentração das aprovadas sobre o estoque **atual** (outra cessão pode ter entrado depois da simulação).
  4. `COPY` das aprovadas para uma tabela temporária e `INSERT INTO estoque … ON CONFLICT (chave) DO NOTHING RETURNING chave`. O que não entrou foi vendido por outra requisição simultânea e vira `DUPLICIDADE` na resposta.
  5. Grava `cessoes` com os totais do que entrou de fato.
- **Prova:** `make demo-concorrencia` dispara duas cessões simultâneas de lotes com notas em comum e mostra que cada nota foi vendida uma vez só. Há também um teste de integração do mesmo cenário.

---

## 9. Modelo de dados (Postgres)

| Tabela | Campos principais | Índices |
|---|---|---|
| `estoque` | id, fundo_doc, fundo_nome, cedente_doc, cedente_nome, sacado_doc, sacado_nome, sacado_raiz, numero, tipo, valor_centavos, vencimento, aquisicao, origem (`seed`/`cessao`), cessao_id | `UNIQUE(chave)`, `(fundo_doc, sacado_raiz)`, `(fundo_doc, cedente_doc)` |
| `regras_versoes` | versao, regras (jsonb), autor, publicada_em | PK versao (a maior é a publicada) |
| `situacao_cadastral` | cnpj, situacao | PK cnpj |
| `lotes` | id (uuid), idempotency_key, hash_arquivo, data_base, regras_versao, precificacao (jsonb: método, taxa e base usados), status, etapa, linhas_total, linhas_processadas, totais, erros_configuracao (jsonb), criado/iniciado/concluido_em, duracao_ms | `UNIQUE(idempotency_key)` |
| `lote_notas` | lote_id, linha, campos da nota, chave, status, motivo_principal, dias, face/desconto/liquido em centavos, recusas (jsonb `[{codigo,dados}]`) | PK (lote_id, linha), `(lote_id, status)`, `(lote_id, motivo_principal)` |
| `cessoes` | id, lote_id, idempotency_key, confirmada_em, notas_cedidas, face/desconto/liquido | `UNIQUE(lote_id)`, `UNIQUE(idempotency_key)` |

Na primeira subida, a API aplica as migrations e, se o estoque estiver vazio, carrega `data/estoque_antecipe.zip` (100.554 linhas via `COPY`), a versão 1 das regras e a situação cadastral simulada. Também dá para rodar à mão com `antecipe importar-estoque`.

---

## 10. API

Contrato completo em [`docs/api.md`](api.md). Esse arquivo é o que o dev do front usa.

| Método | Rota | O que faz |
|---|---|---|
| GET | `/api/saude` | Saúde (API e banco) |
| GET | `/api/config` | Fundo, data-base padrão, versão publicada das regras, limites do upload |
| POST | `/api/lotes` | Upload multipart (`arquivo`, `data_base`), header `Idempotency-Key` opcional → 202 |
| GET | `/api/lotes/{id}` | Status, etapa, progresso, totais e resumo dos motivos |
| GET | `/api/lotes/{id}/notas` | Notas paginadas com filtro (status, motivo, busca) e ordenação |
| GET | `/api/lotes/{id}/notas.csv` | Download do resultado (ou só das recusadas) com motivo e ação |
| POST | `/api/lotes/{id}/cessao` | Confirma a venda das aprovadas (idempotente e segura com concorrência) |
| POST | `/api/validar` | Validação síncrona de notas em JSON (até 10 mil), mesmo motor, sem gravar |
| GET | `/api/regras` | Versão publicada + catálogo de regras (rótulos e tipos de parâmetro para a tela) |
| GET | `/api/regras/versoes` | Histórico |
| POST | `/api/regras/previa` | Roda o último lote (ou um `lote_id`) com o JSON proposto e devolve antes e depois. Exige `X-Gestor-Senha`. |
| PUT | `/api/regras` | Publica uma nova versão (valida o schema). Exige `X-Gestor-Senha`. |

---

## 11. Logs e credenciais (15 pts)

- `log/slog` com handler JSON em stdout (container) e nível via `LOG_LEVEL`.
- **Campos permitidos:** `request_id`, `metodo`, `rota` (o padrão, ex. `/api/lotes/{id}`), `status`, `duracao_ms`, `lote_id`, `etapa`, `linhas`, `aprovadas`, `recusadas`, `codigo_regra`, `versao_regras`.
- **Proibido:** qualquer campo do CSV (nomes, CNPJs, números, valores, datas), nome do arquivo, corpo da requisição, senha do gestor e URL do banco.
- **Garantia por teste:** `TestLogsNaoVazamDados` processa um arquivo de exemplo capturando os logs e falha se aparecer qualquer CNPJ, nome ou número do arquivo.
- **Segredos:** só em variáveis de ambiente. `.env` fica no `.gitignore` e o repositório só tem o `.env.example`. **`make up` gera o `.env` com senhas aleatórias** (Postgres e gestor) se ele não existir, e mostra a senha do gestor no terminal, então o jurado não configura nada e nada secreto vai para o git. `make check` roda o gitleaks.

---

## 12. Front (feito por outra pessoa): o que precisa saber

- **Telas no escopo do front:** início/upload, andamento, resultado (hero, chips, resumo dos motivos, tabela paginada, painel de detalhe, download das recusadas), diálogo de cessão, e o **gestor** (login por senha, lista de regras editável a partir do `catalogo`, prévia do impacto, publicar, histórico com "Ver JSON").
- Usar só a API do [`docs/api.md`](api.md). Dinheiro chega em **centavos inteiros** e é formatado com `Intl.NumberFormat('pt-BR', {style: 'currency', currency: 'BRL'})`.
- **Nunca** baixar todas as notas: a tabela é paginada no servidor (50 por página), com filtros por status e por motivo, e busca.
- **Andamento:** depois do upload, consultar `GET /api/lotes/{id}` a cada 500 ms e mostrar a etapa atual (as 5 etapas), a barra de progresso, "312.000 de 500.000 linhas" e o tempo. O progresso do envio do arquivo vem do `XMLHttpRequest.upload.onprogress`.
- **Textos de recusa:** usar `titulo`, `mensagem` e `acao` que vêm da API, para tela e API dizerem o mesmo.
- **Mandar um `Idempotency-Key`** (ex. `crypto.randomUUID()` por clique) na cessão, e travar o botão durante o envio.
- **Dev:** proxy do Vite `/api` → `http://localhost:8080`. A API também libera CORS para `http://localhost:5173`.
- **Integração no compose:** `frontend/Dockerfile` com build Node → nginx servindo `dist/` e fazendo proxy de `/api` para `api:8080` (`client_max_body_size 300m`). O backend deixa esse Dockerfile e o `nginx.conf` prontos.
- **Identidade visual sugerida:** a do vídeo (verde-escuro com destaque verde-limão, cards escuros, status com ícone e texto). As boas regras de UX e acessibilidade do PDF anterior continuam valendo: hero "Você recebe hoje R$ X", chips de aprovadas e recusadas, painel lateral com o detalhe, nunca status só por cor, teclado e `aria-live`.

---

## 13. Testes e medição

- **Motor:** um teste por validador (passa e falha), parse (data inválida, decimal, milhar, BOM, CRLF, aspas, coluna faltando), preço composto (R$ 9.803,92 / 9.611,69 / 8.879,71) e simples (R$ 9.800 / 9.600 / 8.800), troca de método pelo YAML sem reiniciar, YAML inválido mantendo a última configuração, concentração determinística, regra inativa ignorada, código desconhecido, duplicidade no arquivo e no estoque, parcelas que **não** são duplicadas e CNPJ alfanumérico.
- **Integração (Postgres no compose):** upload → progresso → resultado, reenvio idempotente, cessão dupla simultânea, mudança de regra sem deploy.
- **Carga:** `antecipe gerar` cria lotes de 100k, 250k e 500k linhas com mistura realista (maioria aprovável, casos de cada regra, duplicatas do estoque, repetidas no arquivo e linhas quebradas). `make carga` sobe os três e imprime o tempo de cada etapa. Os números vão para o README e para o pitch.

---

## 14. Entregáveis

- **README.md:** o que é, pré-requisitos (Docker, Docker Compose, make), `make up`, URLs, senha do gestor, exemplos de `curl`, como testar, como gerar lotes grandes, decisões, limitações e o relatório curto.
- **docs/relatorio.md:** problema, solução, arquitetura, regras, resultados medidos e próximos passos.
- **docs/prompts.md:** atualizado a cada prompt. O dev do front registra os dele no mesmo arquivo, com horário, para a ordem cronológica fechar.
- **Formulário de entrega** do Notion, até as 18:00. Ele pede: nome da equipe, integrantes (nome completo e e-mail de cada um), link do repositório no GitHub, caminho do arquivo de prompts (`docs/prompts.md`) e a confirmação de que o repositório está acessível. Não pede vídeo, slides nem URL pública. O repositório `joaoemanuelzilli/antecipe_kanastra` já está público.

---

## 15. Cronograma de hoje (backend nesta máquina)

| Horário | Bloco | Pronto quando |
|---|---|---|
| 10:50–11:15 | Instalar Go 1.27, liberar Docker (`sudo usermod -aG docker $USER`), esqueleto do repo, compose com Postgres, Makefile, **enviar `docs/api.md` ao front** | `make up` sobe Postgres e uma API "olá" |
| 11:15–12:45 | Motor: csvnota, documento, validadores, duplicidade, concentração, preço, textos + testes + CLI `validar` e `gerar` | `go test ./...` verde e 500k linhas pela CLI em poucos segundos |
| 12:45–14:15 | Banco, seed do estoque, fila de jobs com etapas, `POST/GET /api/lotes`, `/notas`, `/notas.csv`, `/validar` | Upload de 500k pelo `curl` com progresso e resultado paginado |
| 14:15–15:15 | Cessão concorrente, regras versionadas + prévia, política de logs + teste, gitleaks | `make demo-concorrencia` e `TestLogsNaoVazamDados` passando |
| 15:15–16:15 | **Integração com o front** (Dockerfile web, nginx, ajustes de contrato) | Fluxo completo no navegador via `make up` |
| 16:15–16:45 | Carga 100k/250k/500k, ajustes de desempenho | Tempos medidos e anotados |
| **16:45** | **Congelamento de funcionalidades** | |
| 16:45–17:30 | README testado num clone limpo, relatório, prompts | `git clone` + `make up` funciona do zero |
| 17:30–17:50 | Folga, push final, formulário | Entregue |

**Ordem de corte se o tempo apertar:** CSV de download → prévia do gestor (fica o PUT) → `make demo-concorrencia` (fica o teste) → CLI `validar` (fica a API). O `gerar` **não** sai: sem ele não há como medir nem demonstrar 500 mil linhas.

---

## 16. Riscos e plano B

| Risco | Plano B |
|---|---|
| Front e back divergirem no contrato | `docs/api.md` enviado já, com exemplos de JSON. Mudança só combinada e registrada no próprio arquivo. |
| Docker falhar na máquina do jurado | README com caminho alternativo sem Docker (Postgres local + `go run` + `npm run dev`). |
| `COPY` de 500k ficar lento | Medir cedo. Alternativa: gravar só as recusadas + agregados, e reconstruir as aprovadas do arquivo do lote. |
| Memória com 500k linhas | Struct enxuta (~300 B/linha ≈ 150 MB). Se apertar, processar em blocos e gravar por bloco. |
| Go/Docker indisponíveis nesta máquina | Primeiro bloco do cronograma resolve. Instalar o Go não precisa de sudo (tarball em `~/.local/go`). |

---

## 17. Decisões que tomei (confira)

1. Duplicidade compara com o estoque de **todos os 18 fundos**, não só do ANTECIPE: nota vendida a outro fundo também já foi vendida. A mensagem não revela qual fundo.
2. Chave da nota = **cedente + número + vencimento**, sem o valor (protege contra reenvio com valor alterado e respeita as parcelas).
3. **Linha com erro de formato recusa só aquela linha**. O arquivo inteiro só é recusado por problema estrutural (cabeçalho, vazio, não-CSV).
4. Situação cadastral **simulada**, com CNPJ desconhecido = `ATIVA` (configurável). Rejeitar desconhecidos recusaria quase tudo que é novo.
5. Data-base padrão **22/10/2026** (a dos dados), editável no upload.
6. Concentração sobre **todo o estoque do fundo** (inclusive já vencidas, porque não sabemos se foram pagas) mais as cessões confirmadas.
7. Arredondamento **por nota**, meio-para-cima, e total = soma das notas.
7a. "Composto" = **desconto racional composto** (`face ÷ (1 + i)^(dias/base)`), o padrão de mercado. Existe também o desconto comercial composto (`face × (1 − i)^n`), que daria R$ 9.800 a 30 dias, mas é pouco usado.
7b. A chave de idempotência do upload inclui o método de preço em uso. Trocar o YAML e reenviar o arquivo gera um resultado novo.
8. **Sem IA no app.** Textos determinísticos gerados pelo backend.
9. Linhas do upload com outro `NOME_FUNDO` são avaliadas pelas regras do ANTECIPE normalmente (o upload é uma oferta ao fundo do JSON).
10. Autenticação do gestor por header `X-Gestor-Senha`, com a senha em env (simples, documentado como limitação).
11. Front servido por nginx em `http://localhost:3000`, que repassa `/api` para a API em `:8080`.
12. Subir o próprio estoque como lote devolve tudo como "Nota já vendida", e isso fica assim de propósito. Demo e medição usam os lotes do `make gerar`.
13. Arquivo de prompts em `docs/prompts.md` (é esse o caminho que vai no formulário).

## 18. Fica a critério de quem constrói

Estrutura interna dos structs, tamanho dos blocos e número de workers, nomes de funções, formato exato do CSV de saída da CLI, mistura de casos do gerador, copy fina dos textos (mantendo o formato decisão → regra → consequência → ação) e detalhes do nginx e dos Dockerfiles.
