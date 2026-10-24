# Assistente de IA do Antecipe

Agente [Agno](https://docs.agno.com) servido pelo **AgentOS**, que responde em linguagem natural sobre lotes, notas e regras. Fica num painel no canto direito da plataforma:

> "Como foi o processamento do lote?" · "Por que a nota 725714 deu erro?" · "Quantas notas foram recusadas?" · "Qual o prazo máximo do fundo?"

```
painel (React) ──► nginx /ai/ (injeta a chave) ──► AgentOS :7777 (rede interna, exige Bearer) ──► agente ──► OpenRouter (modelo)
                                        │              │
                                        │              └─► ferramentas ──► API do Antecipe :8080 (só GET)
                                        └─► sessões do chat no Postgres (schema "ai")
```

O assistente é **mais um cliente da API**, como o front: não lê as tabelas do backend, não envia lote, não confirma cessão e não muda regra. As recusas são explicadas com os textos oficiais que a API já gera (título, mensagem e ação), então tela, API e assistente dizem a mesma coisa.

## Estrutura

```
ai/
  app/main.py          AgentOS: monta o app HTTP e a rota /assistente/status
  app/settings.py      variáveis de ambiente
  app/logs.py          logs JSON só com metadados + log de acesso por rota
  agents/assistente.py o agente: modelo, instruções, ferramentas, histórico
  tools/antecipe.py    ferramentas que consultam a API (consultar_lote, buscar_notas, consultar_regras, consultar_formato_do_arquivo)
  tools/formato.py     reais, datas e durações em pt-BR (o modelo não converte nada)
  db/session.py        sessões do chat no Postgres, schema "ai"
  tests/               sem rede e sem chave (API simulada com httpx.MockTransport)
```

## Ferramentas

| Ferramenta | API | Para perguntas como |
|---|---|---|
| `consultar_lote` | `GET /api/lotes/{id}` | Como foi o processamento? Quantas recusadas? Quanto recebo? Já foi cedido? |
| `buscar_notas` | `GET /api/lotes/{id}/notas` | Por que a nota X deu erro? Quais notas do cliente Y entraram? Recusadas por prazo? |
| `consultar_regras` | `GET /api/regras` | Qual o prazo máximo? Que tipos o fundo aceita? Como é o preço? |
| `consultar_formato_do_arquivo` | `GET /api/config` | Como monto o CSV? Quais colunas? |

Sem `lote_id` na pergunta, as ferramentas usam o lote aberto na tela (`dependencies.lote_id`, enviado pelo front). Ids fora do formato UUID são recusados antes de chegar à API.

## Configuração

| Variável | Padrão | O que é |
|---|---|---|
| `OPENROUTER_API_KEY` | (vazia) | Chave do OpenRouter. Vazia: o serviço sobe, e `/assistente/status` responde `disponivel: false` |
| `AI_MODELO` | `deepseek/deepseek-v4-flash` | Qualquer modelo do OpenRouter que aceite ferramentas |
| `ANTECIPE_API_URL` | `http://localhost:8080` | API do Antecipe (no compose: `http://api:8080`) |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASS`, `DB_DATABASE` | Postgres local | Banco das sessões do chat |
| `CORS_ORIGINS` | `http://localhost:5173,http://localhost:3000` | Origens liberadas |
| `LOG_LEVEL` | `info` | Nível dos logs do serviço |

No `make up` tudo isso vem do `docker-compose.yml` e do `.env`. Basta preencher `OPENROUTER_API_KEY`.

## Contrato para o painel (front)

Pelo nginx, o front chama `/ai/...` na mesma origem, **sem chave nenhuma**. O nginx acrescenta `Authorization: Bearer <AI_SECURITY_KEY>` e o AgentOS recusa (401) qualquer chamada sem ela. A porta 7777 não é publicada no host. Em dev, aponte o proxy do Vite para o nginx do `make up`:

```ts
// vite.config.ts → server.proxy
"/ai": { target: "http://localhost:3000" },
```

**1. Mostrar ou esconder o painel:** `GET /ai/assistente/status`

```json
{ "disponivel": true, "agente_id": "antecipe-assistente", "modelo": "deepseek/deepseek-v4-flash" }
```

Com `disponivel: false`, mostre o painel desligado ("Assistente indisponível"), sem quebrar a tela.

**2. Perguntar:** `POST /ai/agents/antecipe-assistente/runs` (`multipart/form-data`)

| Campo | Obrigatório | O que mandar |
|---|---|---|
| `message` | sim | A pergunta |
| `session_id` | não | Um id por conversa (ex.: `crypto.randomUUID()` ao abrir o painel). Reutilize-o: o agente lembra as últimas 6 perguntas |
| `dependencies` | não | JSON com o lote aberto na tela: `{"lote_id": "8f1c2a9e-…"}` |
| `stream` | não | `true` (padrão) para receber o texto aos poucos (SSE); `false` para um JSON só |

Com `stream=false`, a resposta tem `content` (o texto, em **markdown**), `session_id`, `run_id` e `status`. **Confira o `status` antes de mostrar o `content`:** quando o modelo falha (ex.: chave ausente ou inválida), o AgentOS responde **HTTP 200** com `"status": "ERROR"`, e o `content` traz a mensagem técnica em inglês. Nesse caso mostre "Não consegui responder agora. Tente de novo."

Com `stream=true`, chegam eventos SSE (`event: …` / `data: {json}`). Os que importam:

| Evento | O que fazer |
|---|---|
| `RunContent` | Acrescentar `data.content` ao texto da resposta |
| `ToolCallStarted` | Mostrar "Consultando o lote…" |
| `RunCompleted` | Fim da resposta |
| `RunError` | Mostrar "Não consegui responder agora. Tente de novo." (o `content` desse evento é técnico, não mostre) |

A sequência é sempre `RunStarted` → … → `RunCompleted` ou `RunError`. Também chegam eventos que o painel pode ignorar (`ModelRequestStarted`, `RunContentCompleted` etc.).

```ts
const form = new FormData();
form.append("message", pergunta);
form.append("session_id", sessionId);
form.append("stream", "true");
if (loteId) form.append("dependencies", JSON.stringify({ lote_id: loteId }));

const r = await fetch("/ai/agents/antecipe-assistente/runs", { method: "POST", body: form });
const leitor = r.body!.pipeThrough(new TextDecoderStream()).getReader();
let resto = "";
for (;;) {
  const { value, done } = await leitor.read();
  if (done) break;
  resto += value;
  const blocos = resto.split("\n\n");
  resto = blocos.pop()!;
  for (const bloco of blocos) {
    const linha = bloco.split("\n").find((l) => l.startsWith("data: "));
    if (!linha) continue;
    const ev = JSON.parse(linha.slice(6));
    if (ev.event === "RunContent" && ev.content) acrescentar(ev.content);
    if (ev.event === "RunError") mostrarErro("Não consegui responder agora. Tente de novo.");
  }
}
```

Pelo nginx só passam `GET /ai/assistente/status` e `POST /ai/agents/antecipe-assistente/runs`. O resto do AgentOS (`/sessions`, `/config`, `/docs`…) responde 404, para ninguém listar conversas alheias. A documentação interativa fica dentro da rede do compose (`http://ai:7777/docs`, exige a chave).

## Decisões

- **Só leitura, pela API.** O agente tem 4 ferramentas de consulta e nenhuma de escrita. Vender, enviar e publicar regras continuam exigindo a ação do usuário na tela.
- **Textos oficiais.** As recusas usam o título, a mensagem e a ação da API. O modelo não inventa motivos.
- **Valores prontos.** Dinheiro em reais e datas em dd/mm/aaaa (horário de Brasília) já saem das ferramentas. O modelo não converte centavos.
- **Logs sem dados.** Uma linha JSON por requisição (rota como padrão, status, duração e `request_id`) e por consulta à API (rota, status e duração). Perguntas, respostas, buscas e dados de notas nunca vão para o log. Os clientes HTTP (`httpx`, `openai`) só logam a partir de WARNING, porque em INFO logam a URL com a busca. Os logs do Agno ficam em WARNING. Há teste para isso.
- **Sem telemetria.** `AGNO_TELEMETRY=false` e `telemetry=False` no agente e no AgentOS.
- **Os dados vão para o provedor do modelo.** As respostas das ferramentas (nomes, CNPJs e valores das notas consultadas) são enviadas ao OpenRouter e ao provedor do modelo escolhido. No desafio os dados são simulados. Em produção, isso exigiria um acordo com o provedor ou um modelo próprio.

## Testes

```bash
make test-ai     # da raiz: cria ai/.venv com o uv, instala requirements-dev.txt e roda o pytest
```

Os testes cobrem as 4 ferramentas (lote da tela e lote citado, id inválido, lote inexistente, API fora do ar, filtros), a formatação pt-BR, o app sem chave e sem banco, o log de acesso com a rota como padrão e a ausência de dados da nota nos logs.

## Limitações

- Sem autenticação, como a área da empresa na API. Em produção, o AgentOS aceita JWT (`authorization=True`).
- A API não lista lotes, então o assistente precisa do id ou do lote aberto na tela.
- As dependências são resolvidas em 14/10/2026 (`uv pip compile --exclude-newer 2026-10-14`), para não puxar pacotes publicados na véspera do evento.
