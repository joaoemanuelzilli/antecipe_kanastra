# API do Antecipe — contrato v1

Contrato entre o backend (Go) e o front (React). **Qualquer mudança é combinada entre os dois e anotada no histórico no fim deste arquivo.**

## Convenções

- **Base:** `/api`. Em dev, o Vite faz proxy de `/api` para `http://localhost:8080` (a API também libera CORS para `http://localhost:5173`). No `make up`, o front fica em `http://localhost:3000` e o nginx repassa `/api` para a API.
- **JSON** com chaves em `snake_case`.
- **Datas:** `"AAAA-MM-DD"`. **Data e hora:** ISO 8601 com fuso (`"2026-10-24T14:03:12-03:00"`).
- **Dinheiro: inteiro em centavos**, sempre em campos que terminam com `_centavos`. Ex.: `980000` = R$ 9.800,00. Formate com `Intl.NumberFormat('pt-BR', {style: 'currency', currency: 'BRL'}).format(v / 100)`.
- **Percentuais** (só para exibir): número decimal, ex. `12.61`.
- **Textos para o usuário** (`titulo`, `mensagem`, `acao`) já vêm prontos em pt-BR. Use-os como estão, para tela e API dizerem o mesmo.
- **Erro** (qualquer status 4xx/5xx):

```json
{
  "erro": {
    "codigo": "CABECALHO_INVALIDO",
    "mensagem": "O arquivo não tem as colunas esperadas. Faltou DATA_VENCIMENTO_ORIGINAL.",
    "detalhes": [
      { "linha": 1, "campo": "DATA_VENCIMENTO_ORIGINAL", "problema": "coluna não encontrada" }
    ]
  }
}
```

`detalhes` pode vir vazio. `mensagem` pode ser mostrada direto ao usuário.

---

## Fluxo da tela da empresa

```
1. POST /api/lotes (arquivo)           → 202 { lote_id }
2. GET  /api/lotes/{id}  a cada 500 ms → etapa + progresso … até status "concluido" ou "erro"
3. GET  /api/lotes/{id}/notas?pagina=1 → tabela paginada (filtros por status e motivo)
4. POST /api/lotes/{id}/cessao         → confirma a venda das aprovadas
```

---

## `GET /metrics`

Métricas do OpenTelemetry no formato Prometheus (lotes, notas avaliadas, duração por etapa, notas cedidas). Não faz parte da tela.

## `GET /api/saude`

`200 {"status": "ok"}`, ou `503` se o banco estiver fora.

## `GET /api/config`

Dados para montar a tela inicial.

O método de preço padrão é **juros compostos** (`DESCONTO_COMPOSTO`) e pode virar `DESCONTO_SIMPLES` editando `config/antecipe.yaml`, sem reiniciar. Mostre sempre `precificacao.descricao` perto do valor (ex. "Juros compostos · 2% ao mês"), sem fixar o texto no front.

```json
{
  "fundo": { "nome": "ANTECIPE FIDC", "cnpj": "92.234.400/0001-80" },
  "data_base_padrao": "2026-10-22",
  "regras_versao": 1,
  "precificacao": {
    "metodo": "DESCONTO_COMPOSTO",
    "taxa_mensal_percentual": 2.0,
    "base_dias": 30,
    "descricao": "Juros compostos · 2% ao mês",
    "fonte_metodo": "config/antecipe.yaml"
  },
  "upload": {
    "tamanho_maximo_bytes": 314572800,
    "separador": ";",
    "colunas": ["NOME_FUNDO","DOC_FUNDO","NOME_CEDENTE","DOC_CEDENTE","NOME_SACADO","DOC_SACADO","NU_DOCUMENTO","TIPO_RECEBIVEL","VALOR_NOMINAL","DATA_VENCIMENTO_ORIGINAL","DATA_AQUISICAO"]
  }
}
```

---

## `POST /api/lotes`: enviar lote

`multipart/form-data`:

| Campo | Obrigatório | Descrição |
|---|---|---|
| `arquivo` | sim | O CSV (`;`, decimal com vírgula, `dd/mm/aaaa`, UTF-8, cabeçalho) |
| `data_base` | não | `"AAAA-MM-DD"`. Padrão: `data_base_padrao` |

Header opcional `Idempotency-Key`. **O front não precisa mandar:** sem ele, o servidor usa o hash do arquivo + data-base + versão das regras.

Use `XMLHttpRequest` com `xhr.upload.onprogress` para mostrar o progresso do **envio** (o `fetch` não informa o progresso de upload).

**Respostas**

- `202 Accepted`: lote novo. Header `Location: /api/lotes/{id}`.
  ```json
  { "lote_id": "8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42", "status": "na_fila", "replay": false }
  ```
- `200 OK`: esse arquivo já foi enviado com a mesma data-base e as mesmas regras. Header `Idempotent-Replay: true`. Sugestão de texto: "Esse arquivo já foi enviado. Mostrando o resultado anterior."
  ```json
  { "lote_id": "8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42", "status": "concluido", "replay": true }
  ```
- `400 ARQUIVO_AUSENTE` · `400 DATA_BASE_INVALIDA` · `400 UPLOAD_INVALIDO` · `409 IDEMPOTENCY_KEY_REUTILIZADA` (mesma chave com outro arquivo ou outra data-base) · `413 ARQUIVO_GRANDE_DEMAIS` · `415 ARQUIVO_NAO_CSV` · `422 CABECALHO_INVALIDO` ou `ARQUIVO_VAZIO` (sem nenhuma nota)

Erros **dentro** das linhas (data inválida, valor ilegível) **não** derrubam o lote: a linha sai como recusada com o código `FORMATO_INVALIDO`.

---

## `GET /api/lotes/{id}`: status, andamento e resumo

Consulte a cada **500 ms** enquanto `status` for `na_fila` ou `processando`. Pare em `concluido` ou `erro`.

**Durante o processamento:**

```json
{
  "id": "8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42",
  "status": "processando",
  "etapa": "validando",
  "etapas": [
    { "codigo": "lendo",        "titulo": "Lendo arquivo",           "status": "concluida" },
    { "codigo": "validando",    "titulo": "Validando regras",        "status": "em_andamento" },
    { "codigo": "concentracao", "titulo": "Calculando concentração", "status": "pendente" },
    { "codigo": "precificando", "titulo": "Calculando valores",      "status": "pendente" },
    { "codigo": "salvando",     "titulo": "Salvando resultado",      "status": "pendente" }
  ],
  "progresso": { "linhas_processadas": 312000, "linhas_total": 500000, "percentual": 62.4 },
  "posicao_fila": null,
  "data_base": "2026-10-22",
  "regras_versao": 1,
  "precificacao": { "metodo": "DESCONTO_COMPOSTO", "taxa_mensal_percentual": 2.0, "base_dias": 30, "descricao": "Juros compostos · 2% ao mês" },
  "criado_em": "2026-10-24T14:03:10-03:00",
  "iniciado_em": "2026-10-24T14:03:11-03:00",
  "concluido_em": null,
  "duracao_ms": null,
  "totais": null,
  "motivos": [],
  "erros_configuracao": [],
  "cessao": null,
  "erro": null
}
```

- `status`: `na_fila` | `processando` | `concluido` | `erro`
- `etapa`: `lendo` | `validando` | `concentracao` | `precificando` | `salvando` | `null`
- `etapas[].status`: `pendente` | `em_andamento` | `concluida`
- `posicao_fila`: número quando `na_fila` (ex. "Seu lote é o 2º da fila"), senão `null`

**Concluído:**

```json
{
  "id": "8f1c2a9e-0d4b-4c51-9a77-2f3e6b1d0c42",
  "status": "concluido",
  "etapa": null,
  "etapas": [ "… todas com status concluida …" ],
  "progresso": { "linhas_processadas": 500000, "linhas_total": 500000, "percentual": 100.0 },
  "data_base": "2026-10-22",
  "regras_versao": 1,
  "criado_em": "2026-10-24T14:03:10-03:00",
  "iniciado_em": "2026-10-24T14:03:11-03:00",
  "concluido_em": "2026-10-24T14:03:19-03:00",
  "duracao_ms": 8120,
  "totais": {
    "linhas": 500000,
    "aprovadas": 412345,
    "recusadas": 87655,
    "face_centavos": 98765432100,
    "desconto_centavos": 2345678900,
    "liquido_centavos": 96419753200
  },
  "motivos": [
    { "codigo": "PRAZO_MAXIMO", "titulo": "Prazo acima do limite", "quantidade": 40211, "acao": "Antecipe essas notas mais perto do vencimento." },
    { "codigo": "DUPLICIDADE",  "titulo": "Nota já vendida",       "quantidade": 21000, "acao": "Não é preciso enviar de novo." }
  ],
  "erros_configuracao": [],
  "cessao": null,
  "erro": null
}
```

- `precificacao`: o método, a taxa e a base **usados neste lote** (ficam gravados, mesmo que o YAML mude depois).
- `totais.face_centavos`, `desconto_centavos` e `liquido_centavos` somam **só as aprovadas**. Hero: "Você recebe hoje R$ X" = `liquido_centavos`, com "Valor de face · Desconto · `precificacao.descricao`" abaixo.
- `motivos`: contagem pelo **motivo principal** de cada recusada, em ordem decrescente. A soma das quantidades = `recusadas`.
- `erros_configuracao`: ex. `[{"codigo": "REGRA_X", "mensagem": "Regra REGRA_X não existe e foi ignorada."}]`. Mostre um aviso discreto.
- `cessao`: `null` até confirmar. Depois, igual ao objeto `cessao` da resposta da cessão.
- **Erro:** `status: "erro"` e `"erro": {"codigo": "FALHA_PROCESSAMENTO", "mensagem": "Não conseguimos processar o arquivo. Nada foi gravado. Tente de novo."}`.

`404 LOTE_NAO_ENCONTRADO` se o id não existir.

---

## `GET /api/lotes/{id}/notas`: tabela paginada

**Nunca busque todas as notas.** Um lote pode ter 500 mil.

| Parâmetro | Padrão | Valores |
|---|---|---|
| `pagina` | `1` | ≥ 1 |
| `por_pagina` | `50` | 1–200 |
| `status` | todos | `aprovada` \| `recusada` |
| `motivo` | todos | um código de recusa (filtra pelo motivo principal) |
| `busca` | — | texto: número da nota, nome ou documento do sacado ou do cedente |
| `ordenar` | `linha` | `linha` \| `vencimento` \| `dias` \| `face` \| `liquido` \| `sacado` |
| `direcao` | `asc` | `asc` \| `desc` |

```json
{
  "pagina": 1,
  "por_pagina": 50,
  "total": 87655,
  "total_paginas": 1754,
  "itens": [
    {
      "linha": 1204,
      "numero": "725714",
      "tipo": "Duplicata",
      "cedente": { "nome": "NLR INDÚSTRIA DE ESTOFADOS LTDA ME", "documento": "10.062.050/0001-57" },
      "sacado":  { "nome": "TECIDOS SANTARÉM LTDA ME", "documento": "22.144.961/0001-02" },
      "vencimento": "2027-06-19",
      "dias": 240,
      "status": "recusada",
      "face_centavos": 227808,
      "desconto_centavos": null,
      "liquido_centavos": null,
      "recusas": [
        {
          "codigo": "PRAZO_MAXIMO",
          "titulo": "Prazo acima do limite",
          "mensagem": "Esta nota vence em 240 dias e o fundo compra notas com até 180 dias.",
          "acao": "Você pode antecipá-la a partir de 21/12/2026.",
          "dados": { "dias": 240, "maximo": 180, "antecipavel_em": "2026-12-21" }
        }
      ],
      "calculo": null
    },
    {
      "linha": 1205,
      "numero": "1-759172",
      "tipo": "Duplicata",
      "cedente": { "nome": "CANAÃ INDÚSTRIA DE ETIQUETAS LTDA", "documento": "57.543.525/0001-90" },
      "sacado":  { "nome": "LOJA AURORA LTDA", "documento": "44.911.681/0001-15" },
      "vencimento": "2026-11-21",
      "dias": 30,
      "status": "aprovada",
      "face_centavos": 1000000,
      "desconto_centavos": 19608,
      "liquido_centavos": 980392,
      "recusas": [],
      "calculo": {
        "metodo": "DESCONTO_COMPOSTO",
        "taxa_mensal_percentual": 2.0,
        "base_dias": 30,
        "dias": 30,
        "texto": "R$ 10.000,00 ÷ (1 + 2%)^(30 ÷ 30) = R$ 9.803,92 → desconto de R$ 196,08"
      }
    }
  ]
}
```

- `recusas`: **todas** as regras que a nota não cumpriu. A primeira é a principal.
- `calculo.texto` já vem pronto conforme o método. No simples, ele fica assim: "R$ 10.000,00 × 2% × (30 ÷ 30) = R$ 200,00 de desconto → você recebe R$ 9.800,00".
- Em linhas com `FORMATO_INVALIDO`, os campos que não puderam ser lidos vêm `null` (`vencimento`, `dias`, `face_centavos`). Os textos vêm como estavam no arquivo.

## `GET /api/lotes/{id}/notas.csv`

Download (`text/csv; charset=utf-8`, `;`, com `Content-Disposition: attachment`). Aceita os mesmos filtros (`status`, `motivo`, `busca`). Inclui colunas `STATUS`, `MOTIVO`, `O_QUE_FAZER`, `DESCONTO` e `VALOR_LIQUIDO`. Basta um `<a href="/api/lotes/{id}/notas.csv?status=recusada">Baixar recusadas</a>`.

---

## `GET /api/lotes/{id}/resultado.json`

Resultado no **formato do gabarito do desafio** (`resultado_esperado_100k.json`), em fluxo, na ordem das linhas:

```json
[
{"linha":2,"id":"725714","status":"APROVADO"},
{"linha":3,"id":"1-759172","status":"BARRADO","barrado_por":["CONCENTRACAO_MAXIMA_SACADO"]}
]
```

`barrado_por` traz todas as regras que a nota não cumpriu, na ordem do JSON de regras.

---

## `POST /api/lotes/{id}/cessao`: confirmar a venda

Header **`Idempotency-Key`**: gere um `crypto.randomUUID()` quando o diálogo de confirmação abrir e reutilize o mesmo valor em novas tentativas. Trave o botão durante o envio.

Sem corpo.

- `201 Created`:
  ```json
  {
    "cessao": {
      "id": "c0f7…",
      "lote_id": "8f1c2a9e-…",
      "confirmada_em": "2026-10-24T14:05:40-03:00",
      "notas_cedidas": 412300,
      "face_centavos": 98760000000,
      "desconto_centavos": 2345000000,
      "liquido_centavos": 96415000000
    },
    "notas_excluidas": [
      {
        "linha": 77,
        "codigo": "DUPLICIDADE",
        "titulo": "Nota já vendida",
        "mensagem": "Esta nota foi vendida em outra operação enquanto você revisava o lote."
      }
    ]
  }
  ```
  `notas_excluidas`: aprovadas na simulação que não puderam ser vendidas na confirmação (vendidas em paralelo por outra operação, ou a concentração mudou). Normalmente vem vazio. Se vier com itens, avise: "N notas saíram da cessão desde a simulação."
- `200 OK` com o mesmo corpo e o header `Idempotent-Replay: true`: confirmação repetida, nada foi vendido de novo.
- `409 LOTE_NAO_CONCLUIDO` · `409 SEM_APROVADAS` · `409 IDEMPOTENCY_KEY_REUTILIZADA` (chave já usada em outro lote) · `404 LOTE_NAO_ENCONTRADO`

Texto de erro genérico sugerido: "Não conseguimos confirmar. Nada foi cedido. Tente de novo."

---

## `POST /api/validar`: validação síncrona (API para integração)

Mesmo motor, **sem gravar nada**. Até **10.000 notas**. Acima disso, `413` com a mensagem para usar `POST /api/lotes`.

Aceita JSON **ou** o próprio CSV (`Content-Type: text/csv`, com `?data_base=AAAA-MM-DD` opcional). Os valores vêm no formato do CSV. Corpo inválido → `400 JSON_INVALIDO`.

```json
{
  "data_base": "2026-10-22",
  "notas": [
    {
      "nome_fundo": "ANTECIPE FIDC",
      "doc_fundo": "92.234.400/0001-80",
      "nome_cedente": "NLR INDÚSTRIA DE ESTOFADOS LTDA ME",
      "doc_cedente": "10.062.050/0001-57",
      "nome_sacado": "TECIDOS SANTARÉM LTDA ME",
      "doc_sacado": "22.144.961/0001-02",
      "nu_documento": "900001",
      "tipo_recebivel": "Duplicata",
      "valor_nominal": "10000,00",
      "data_vencimento_original": "21/11/2026",
      "data_aquisicao": ""
    }
  ]
}
```

Resposta `200`:

```json
{
  "data_base": "2026-10-22",
  "regras_versao": 1,
  "precificacao": { "metodo": "DESCONTO_COMPOSTO", "taxa_mensal_percentual": 2.0, "base_dias": 30, "descricao": "Juros compostos · 2% ao mês" },
  "totais": { "linhas": 1, "aprovadas": 1, "recusadas": 0, "face_centavos": 1000000, "desconto_centavos": 19608, "liquido_centavos": 980392 },
  "motivos": [],
  "notas": [ "… mesmo formato dos itens de /notas …" ]
}
```

---

## Regras e tela do gestor

As rotas que alteram algo exigem o header **`X-Gestor-Senha`**. A senha é mostrada no terminal pelo `make up` e fica no `.env`. Guarde-a no `sessionStorage` depois do login.

### `GET /api/gestor/verificar`

Com o header `X-Gestor-Senha`: `204` se a senha estiver certa, `401 SENHA_INVALIDA` se não. É o que a tela de login usa.

### `GET /api/regras`: versão publicada + catálogo para montar o formulário

```json
{
  "versao": 1,
  "publicada_em": "2026-10-24T10:59:00-03:00",
  "autor": "semente",
  "regras": {
    "fundo": { "nome": "ANTECIPE FIDC", "cnpj": "92.234.400/0001-80" },
    "validadores": [
      { "codigo": "PRAZO_MAXIMO", "ativo": true, "parametros": { "prazo_maximo_dias": 180 } }
    ],
    "precificacao": { "metodo": "DESCONTO_SIMPLES", "taxa_mensal_percentual": 2.0, "base_dias": 30 }
  },
  "catalogo": [
    {
      "codigo": "PRAZO_MAXIMO",
      "titulo": "Prazo máximo",
      "descricao": "Maior número de dias entre a data-base e o vencimento.",
      "parametros": [
        { "nome": "prazo_maximo_dias", "rotulo": "Prazo máximo", "tipo": "inteiro", "unidade": "dias", "minimo": 1 }
      ]
    },
    {
      "codigo": "TIPO_RECEBIVEL",
      "titulo": "Tipos de nota aceitos",
      "descricao": "Só os tipos marcados podem ser comprados.",
      "parametros": [
        { "nome": "tipos_aceitos", "rotulo": "Tipos aceitos", "tipo": "lista_texto",
          "opcoes": ["Duplicata", "Duplicata de Serviço", "Cheque", "CCB", "Nota Comercial"] }
      ]
    },
    {
      "codigo": "CONCENTRACAO_MAXIMA_SACADO",
      "titulo": "Concentração máxima por cliente (sacado)",
      "descricao": "Parte máxima da carteira que um mesmo cliente pode ter.",
      "parametros": [
        { "nome": "percentual_maximo", "rotulo": "Limite", "tipo": "decimal", "unidade": "%", "minimo": 0, "maximo": 100 },
        { "nome": "agrupamento", "rotulo": "Agrupar por", "tipo": "opcao", "opcoes": ["RAIZ_CNPJ", "CNPJ"] }
      ]
    }
  ],
  "precificacao_efetiva": {
    "metodo": "DESCONTO_COMPOSTO",
    "taxa_mensal_percentual": 2.0,
    "base_dias": 30,
    "descricao": "Juros compostos · 2% ao mês",
    "fonte_metodo": "config/antecipe.yaml",
    "sobrescritos_pelo_yaml": ["metodo"]
  },
  "catalogo_precificacao": {
    "parametros": [
      { "nome": "taxa_mensal_percentual", "rotulo": "Taxa", "tipo": "decimal", "unidade": "% ao mês", "minimo": 0 },
      { "nome": "base_dias", "rotulo": "Base", "tipo": "inteiro", "unidade": "dias", "minimo": 1 }
    ]
  }
}
```

- `regras` vem **exatamente** no formato do `validadores_antecipe.json` (que diz `DESCONTO_SIMPLES`).
- `precificacao_efetiva` é o que **vale de fato**: o YAML tem precedência sobre as regras. Na tela do gestor, mostre o método como **somente leitura** ("Juros compostos, definido em config/antecipe.yaml"). Taxa e base continuam editáveis pelas regras, a menos que estejam em `sobrescritos_pelo_yaml`; nesse caso, também ficam somente leitura.
- `catalogo` traz as 9 regras. Parâmetros com `"opcional": true` (ex.: `agrupamento`) podem ficar de fora. `tipo` é um de: `inteiro` | `decimal` | `lista_texto` | `opcao`. Regras sem parâmetro (`CNPJ_SACADO_VALIDO`, `NAO_VENCIDA`) vêm com `parametros: []` e só o switch Ativa.

### `GET /api/regras/versoes`

```json
{
  "versoes": [
    { "versao": 2, "autor": "Rafael", "publicada_em": "2026-10-24T15:10:00-03:00", "resumo": ["PRAZO_MAXIMO: 180 → 120 dias"] },
    { "versao": 1, "autor": "semente", "publicada_em": "2026-10-24T10:59:00-03:00", "resumo": ["Versão inicial"] }
  ]
}
```

`GET /api/regras/versoes/{versao}` devolve `{versao, autor, publicada_em, regras}` (para "Ver JSON").

### `POST /api/regras/previa`: impacto antes de publicar

Header `X-Gestor-Senha`. Corpo: `{"regras": { … JSON completo proposto … }, "lote_id": "opcional (padrão: último lote concluído)"}`

```json
{
  "lote_id": "8f1c2a9e-…",
  "linhas": 500000,
  "antes":  { "regras_versao": 1, "aprovadas": 412345, "recusadas": 87655, "liquido_centavos": 96419753200 },
  "depois": { "aprovadas": 371002, "recusadas": 128998, "liquido_centavos": 86800000000 },
  "motivos": [ { "codigo": "PRAZO_MAXIMO", "titulo": "Prazo acima do limite", "antes": 40211, "depois": 81554 } ],
  "mudancas": ["PRAZO_MAXIMO: 180 → 120 dias"]
}
```

Texto sugerido: "Aprovadas 412.345 → 371.002 · Líquido −R$ 96.197.532,00".
`409 SEM_LOTE`: "Envie um lote para ver o impacto." · `422 REGRAS_INVALIDAS` (mesmo formato do PUT).

### `PUT /api/regras`: publicar nova versão

Header `X-Gestor-Senha`. Corpo: `{"regras": { … }, "autor": "Rafael"}`

- `201 {"versao": 2, "publicada_em": "…", "resumo": ["PRAZO_MAXIMO: 180 → 120 dias"]}`. Os próximos lotes já usam a nova versão, sem deploy.
- `422 SEM_MUDANCAS`: nada mudou em relação à versão publicada.
- `422 REGRAS_INVALIDAS`:
  ```json
  { "erro": { "codigo": "REGRAS_INVALIDAS", "mensagem": "Algumas regras têm valores inválidos.",
      "detalhes": [ { "campo": "validadores[5].parametros.prazo_maximo_dias", "problema": "deve ser um número inteiro maior que zero" } ] } }
  ```
- `401 SENHA_INVALIDA`

---

## Códigos de recusa

| Código | `titulo` | `dados` |
|---|---|---|
| `CNPJ_SACADO_VALIDO` | CNPJ do cliente inválido | `documento` |
| `SITUACAO_CADASTRAL_SACADO` | CNPJ do cliente com pendência | `situacao`, `aceitas` |
| `TIPO_RECEBIVEL` | Tipo de nota não aceito | `tipo`, `aceitos` |
| `VALOR_MINIMO` | Valor abaixo do mínimo | `valor_centavos`, `minimo_centavos` |
| `NAO_VENCIDA` | Nota vencida | `vencimento`, `data_base` |
| `PRAZO_MAXIMO` | Prazo acima do limite | `dias`, `maximo`, `antecipavel_em` |
| `PRAZO_MINIMO` | Prazo abaixo do mínimo | `dias`, `minimo` |
| `CONCENTRACAO_MAXIMA_SACADO` | Limite por cliente atingido | `percentual_resultante`, `limite`, `grupo`, `grupo_nome` |
| `CONCENTRACAO_MAXIMA_CEDENTE` | Limite por empresa atingido | `percentual_resultante`, `limite`, `grupo`, `grupo_nome` |
| `DUPLICIDADE` | Nota já vendida | `vendida_em`, `valor_original_centavos`, `valor_centavos`, `numero`, `vencimento` |
| `DUPLICADA_NO_ARQUIVO` | Nota repetida no arquivo | `linha_original` |
| `FORMATO_INVALIDO` | Linha com erro | `campo`, `valor`, `problema` |

Sugestão de UI: ícone + texto em todo status (nunca só cor), filtro por motivo usando `motivos` do resumo e "Detalhes técnicos" recolhido mostrando `codigo` e `dados`.

---

## Assistente de IA (serviço `ai`, fora desta API)

O painel do assistente fala com outro serviço: o AgentOS, que **só o nginx alcança** e que é exposto em `/ai/`. O AgentOS exige uma chave interna, e o nginx a injeta; o front não manda chave nenhuma. O contrato completo, com exemplo de streaming, está em [`ai/README.md`](../ai/README.md). Em resumo:

- `GET /ai/assistente/status` → `{"disponivel": true, "agente_id": "antecipe-assistente", "modelo": "…"}`. Com `disponivel: false` (sem chave configurada), mostre o painel como indisponível.
- `POST /ai/agents/antecipe-assistente/runs` (`multipart/form-data`) com os campos:
  - `message`: a pergunta;
  - `session_id`: um por conversa;
  - `dependencies`: `{"lote_id": "…"}`, o lote aberto na tela;
  - `stream`: `true` para SSE (eventos `RunContent`, `RunCompleted`, `RunError`) ou `false` para JSON.
- **Erro do modelo vem com HTTP 200:** confira `"status": "ERROR"` (ou o evento `RunError`) antes de mostrar o `content`, que nesse caso é técnico.
- A resposta vem em **markdown**. O assistente só consulta esta API, então os textos de recusa que ele cita são os mesmos daqui.
- Em dev, configure o proxy do Vite `/ai` → `http://localhost:3000` (o nginx do `make up`). A porta 7777 não fica mais aberta no host.

---

## Histórico do contrato

- **v1 · 24/10/2026 11:00:** versão inicial.
- **v1.4 · 24/10/2026 13:10:** a regra muda para bater com o gabarito oficial.
  - A data-base de cada nota passa a ser a `DATA_AQUISICAO` dela, quando houver: `dias`, prazo, vencimento e preço contam a partir dela.
  - A concentração fica estática e por fundo, e o texto vira "Com as notas deste lote, …".
  - A `DUPLICIDADE` na simulação passa a valer só contra cessões confirmadas. Notas do estoque são barradas na cessão (`notas_excluidas`).
  - Entra a rota `GET /api/lotes/{id}/resultado.json`.
  - A situação cadastral vem do `data/cnpj_situacao_mock.csv` oficial.
- **v1.3 · 24/10/2026 12:45:** seção do assistente de IA. É um serviço à parte, servido em `/ai/` e detalhado em `ai/README.md`. Esta API não mudou.
- **v1.2 · 24/10/2026 12:15:** alinhado ao backend implementado. Erros de regras viram `422` (`REGRAS_INVALIDAS`, `SEM_MUDANCAS`). Entram `409 IDEMPOTENCY_KEY_REUTILIZADA` no upload e na cessão, `400 DATA_BASE_INVALIDA`/`UPLOAD_INVALIDO`, `?data_base` no `/validar` com CSV, `resumo` na resposta do PUT, `opcional` no catálogo, `grupo_nome` na concentração, campos extras em `DUPLICIDADE` e a rota `/metrics`.
- **v1.1 · 24/10/2026 11:10:** preço padrão passa a ser **juros compostos** (`DESCONTO_COMPOSTO`), trocável em `config/antecipe.yaml`. Entram `precificacao` em `/config`, `/lotes/{id}` e `/validar`, `calculo.metodo` nas notas e `precificacao_efetiva` em `/regras`. Valores dos exemplos recalculados (R$ 10.000 a 30 dias → R$ 9.803,92).
