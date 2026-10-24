# Deploy na Railway a partir de imagens Docker

Serve quando a integração com o GitHub está desligada na Railway: você monta as imagens na sua máquina, envia para um registry público, e a Railway puxa as imagens de lá.

## 1. Publicar as imagens

**Docker Hub** (mais simples: os repositórios são públicos por padrão):

```bash
docker login                                   # usuário + access token do Docker Hub
make publicar REGISTRO=docker.io/SEU_USUARIO TAG=v1
```

**Ou GHCR** (registry do GitHub; funciona mesmo com a integração desligada):

```bash
echo "$GITHUB_TOKEN" | docker login ghcr.io -u SEU_USUARIO --password-stdin   # token com write:packages
make publicar REGISTRO=ghcr.io/SEU_USUARIO TAG=v1
```

No GHCR, os pacotes nascem **privados**. Depois do primeiro envio, deixe cada um público em *GitHub → Packages → antecipe-… → Package settings → Change visibility*. Registry privado na Railway só funciona no plano Pro.

O `make publicar` monta para `linux/amd64` e envia três imagens:

| Imagem | O que tem |
|---|---|
| `antecipe-api` | API Go + CLI, com `data/` (estoque, validadores, situação cadastral, gabarito) e `config/antecipe.yaml` |
| `antecipe-web` | nginx com o front, repassando `/api` e `/ai` |
| `antecipe-ai` | assistente de IA (AgentOS) |

Nenhuma imagem leva segredo: o `.env` fica fora do contexto de build (`.dockerignore`). Tudo que é segredo entra como variável na Railway.

## 2. Montar o projeto na Railway

Crie os serviços **nesta ordem**, com **estes nomes**. Os nomes viram os endereços internos (`api.railway.internal`, `ai.railway.internal`).

### 2.1 Postgres
*New → Database → PostgreSQL.* Deixe o nome `Postgres`.

### 2.2 `api`
*New → Docker Image →* `docker.io/SEU_USUARIO/antecipe-api:v1`. Renomeie o serviço para `api`.

| Variável | Valor |
|---|---|
| `DATABASE_URL` | `${{Postgres.DATABASE_URL}}` |
| `GESTOR_SENHA` | uma senha forte (é a senha da tela do gestor) |
| `LOG_LEVEL` | `info` |

Na primeira subida, a API cria as tabelas e carrega o estoque (100.554 notas, alguns segundos). Domínio público para a API é opcional: só gere se os jurados forem chamar a API direto. A porta 8080 é detectada sozinha.

> Memória: um lote de 500 mil linhas usa cerca de 750 MB na API. Confira se o plano da Railway permite (o trial tem pouca memória).

### 2.3 `ai` (opcional: o resto funciona sem ele)
*New → Docker Image →* `docker.io/SEU_USUARIO/antecipe-ai:v1`. Renomeie para `ai`.

| Variável | Valor |
|---|---|
| `OS_SECURITY_KEY` | uma chave aleatória longa (ex.: `openssl rand -hex 24`) |
| `OPENROUTER_API_KEY` | a chave do OpenRouter (sem ela, o assistente aparece como indisponível) |
| `ANTECIPE_API_URL` | `http://${{api.RAILWAY_PRIVATE_DOMAIN}}:8080` |
| `DB_HOST` | `${{Postgres.PGHOST}}` |
| `DB_PORT` | `${{Postgres.PGPORT}}` |
| `DB_USER` | `${{Postgres.PGUSER}}` |
| `DB_PASS` | `${{Postgres.PGPASSWORD}}` |
| `DB_DATABASE` | `${{Postgres.PGDATABASE}}` |
| `AGNO_TELEMETRY` | `false` |

**Não** gere domínio público para o `ai`: ele só deve ser acessado pelo `web`. Se o projeto for antigo (criado antes de 16/10/2025, com rede interna só IPv6), acrescente `AI_HOST=::`.

### 2.4 `web` (o que os jurados abrem)
*New → Docker Image →* `docker.io/SEU_USUARIO/antecipe-web:v1`. Renomeie para `web`.

| Variável | Valor |
|---|---|
| `API_UPSTREAM` | `${{api.RAILWAY_PRIVATE_DOMAIN}}:8080` |
| `AI_UPSTREAM` | `${{ai.RAILWAY_PRIVATE_DOMAIN}}:7777` |
| `AI_SECURITY_KEY` | `${{ai.OS_SECURITY_KEY}}` |

Depois: *Settings → Networking → Generate Domain*. A porta 80 é detectada sozinha, e essa é a URL para entregar. Sem o serviço `ai`, defina `AI_UPSTREAM=127.0.0.1:1` e `AI_SECURITY_KEY=x`: o resto funciona, e o painel do assistente aparece como indisponível.

## 3. Conferir

```bash
URL=https://SEU-WEB.up.railway.app
curl $URL/api/saude                                  # {"status":"ok"}
curl -F arquivo=@data/exemplo.csv $URL/api/lotes     # 202 {"lote_id": …}
curl $URL/ai/assistente/status                       # 200 (com o serviço ai)
curl -o /dev/null -w '%{http_code}\n' $URL/ai/sessions   # 404 (rotas internas do assistente ficam fechadas)
```

## 4. Atualizar depois de mudar o código

- **Mesma tag** (`TAG=latest`): rode `make publicar REGISTRO=… TAG=latest` de novo. Na Railway, o serviço mostra o botão de atualização da imagem (*Settings → Source*), ou use *Redeploy*: a Railway puxa o digest mais novo da tag.
- **Tag nova** (`TAG=v2`, mais previsível numa demo): publique e troque a imagem em *Settings → Source* de cada serviço.

Para voltar a um banco novo (por exemplo, para reproduzir o gabarito depois de vender notas), apague e recrie o serviço Postgres. A API recarrega a semente sozinha.

## Alternativa sem registry: Railway CLI

`railway up` envia a pasta local e a Railway monta o Dockerfile lá, sem GitHub e sem registry:

```bash
npm i -g @railway/cli && railway login && railway link
# na raiz do repositório, um serviço por vez:
RAILWAY_DOCKERFILE_PATH=backend/Dockerfile railway up --service api
cd frontend && railway up --service web
cd ../ai && railway up --service ai
```

As variáveis da seção 2 continuam valendo. Para a API, a variável `RAILWAY_DOCKERFILE_PATH=backend/Dockerfile` também pode ficar salva nas variáveis do serviço.
