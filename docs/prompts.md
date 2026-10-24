# Registro de prompts

Histórico de todos os prompts de IA usados no projeto, em ordem cronológica, com o que foi feito com cada resposta (item 4.1.2 do edital).

- **Ferramenta:** Claude Code (extensão do VS Code), modelo Claude Opus 5.5 (`claude-opus-5-5`)
- **Fuso:** horário de Brasília (UTC−3)
- **Regra:** nenhum segredo e nenhuma linha do CSV entram neste arquivo.

Cada entrada tem o prompt exato, o que a IA fez e o que o time fez com a resposta. Prompts que a IA mandou para subagentes aparecem dentro da entrada que os originou.

> **Continua em [`prompts2.md`](prompts2.md):** os prompts do assistente de IA (serviço `ai/`), feitos em paralelo noutra máquina a partir das 12:07. Os dois arquivos usam o horário de Brasília, então as entradas se intercalam em ordem cronológica.

---

## #1 · 24/10/2026 10:14 · Exploração do contexto e plano

**Prompt (time → Claude Code):**

> https://app.notion.com/p/kanastra/Hackathon-Antecipe-dataset-para-os-participantes-4037e0475c338109906dee93fbf9cebb
>
> estou participando de um hackaton, todas as informações estão em /home/joaoemanuel/git/kanastra-challenge/antecipe_kanastra/context e no link do notion acima, pegue os arquivos dentro do notion e assista ao vídeo
>
> explore todo o contexto a fundo e monte um plano
>
> para aderir ao que pedem de transparencia com os prompts utilizados, salve tudo em um .md
>
> estude a melhor stack, pensamos em go e react - vite

**O que a IA fez:**

1. Leu os dois PDFs de `context/`: o edital (critérios de nota e entregáveis) e o "Antecipe Claro — Projeto Completo" (plano anterior em Next.js + Supabase).
2. Baixou o conteúdo da página do Notion pela API pública do Notion: texto do desafio, tabela do formato do CSV, data-base dos dados (22/10/2026) e os três anexos (`estoque_antecipe_100k.zip`, `validadores_antecipe.json` e o vídeo).
3. Fez um perfil do `estoque_antecipe.csv` com um script Python local (contagens, tipos, CNPJs, datas, duplicatas, concentração). Nenhum dado do CSV foi para fora da máquina.
4. "Assistiu" ao vídeo: extraiu quadros a cada 6 s (ffmpeg) e transcreveu o áudio localmente (faster-whisper).
5. Checou as ferramentas instaladas e as versões atuais de Go e das libs de front.
6. Analisou a chave de duplicidade no estoque e achou 3.018 grupos de parcelas (mesmo cedente + sacado + número, vencimentos diferentes). Concluiu que a chave precisa incluir o vencimento.
7. Criou este arquivo e fez 4 perguntas de decisão ao time (entrada #2).

**O que o time fez com a resposta:** respondeu às 4 perguntas (entrada #2).

---

## #2 · 24/10/2026 10:45 · Decisões de escopo (respostas às perguntas da IA)

**Perguntas da IA (múltipla escolha) e respostas do time:**

> **Como deve funcionar a venda das notas?** → Simular + confirmar (recomendada)
>
> **Quanto investir em "regras sem deploy"?** → Tela do gestor (recomendada)
>
> **Banco e forma de subir o app?** → "primeira opção de postgres + docker que vc recomendou e use um make up para facilitar ao maximo ao jurado"
>
> **Como o time de 3 vai trabalhar?** → "1 vai fazer o backend aq neste computador e o outro já esta fazendo o front e no final conectamos tudo e fazemos ficar perfeito"

**O que a IA fez:**

1. Escreveu `docs/plano.md`: requisitos e critérios de nota, achados dos dados, decisões, metas e anti-metas, stack, arquitetura no estilo Unix, regras, preço, idempotência e concorrência, modelo de dados, logs, cronograma, riscos e as decisões tomadas pela IA para o time conferir.
2. Escreveu `docs/api.md`, o contrato da API para o dev do front trabalhar em paralelo.
3. Mandou um subagente revisar o plano como se fosse construí-lo sem poder perguntar nada. Prompt enviado ao subagente:

   > You are a talented senior backend engineer (Go, PostgreSQL, high-throughput CSV processing) who also understands Brazilian receivables financing (FIDC, duplicatas, cessão). You've been handed a design document and asked to put together a plan to build what it describes. You will not be able to talk to the person who wrote it.
   >
   > Read /home/joaoemanuel/git/kanastra-challenge/antecipe_kanastra/docs/plano.md (written in Portuguese). For background you may also skim /home/joaoemanuel/git/kanastra-challenge/antecipe_kanastra/context/Edital Hackathon Kanastra.pdf (use `pdftotext -layout <file> -` via Bash). Do not modify any files.
   >
   > Context: this is a one-day hackathon; delivery is today at 18:00 and it is now about 11:00. The backend is built by one person with an AI assistant; the frontend is built in parallel by another person against docs/api.md (which may not exist yet when you read).
   >
   > Report only the questions you'd still need answered before you could plan this well: places where you'd have to guess and a wrong guess would matter. For each, give the question and why the answer changes the plan. Rank them by importance.
   >
   > Skip anything the document explicitly leaves to the builder, and anything you could reasonably decide yourself. Skip nitpicks about wording.
   >
   > If nothing important is left, say "nothing important left". Keep the report under 400 words.

**O que a IA fez com a revisão do subagente:**

- **Arquivos de teste dos jurados:** registrou no plano que subir o estoque devolve tudo como "já vendida" (o que é correto) e tirou o gerador de lotes (`make gerar`) da lista de corte.
- **Escopo do front:** listou no plano as telas do front, incluindo o gestor. Levou ao time a pergunta sobre o que o front já tem pronto.
- **Formulário de entrega:** leu o formulário e anotou os campos que ele pede. Confirmou que o repositório no GitHub está público.
- **Dados de origem:** copiou `estoque_antecipe.zip` e `validadores_antecipe.json` para `data/`.

**O que o time fez com a resposta:** leu o plano e fez duas perguntas (entrada #3).

---

## #3 · 24/10/2026 ~11:00 · Juros composto e containers

**Prompt (time → Claude Code):**

> o juros é composto certo? simples é mt básico
>
> o backend e o front end subirão por container ocm make?

**O que a IA fez:** explicou que o `validadores_antecipe.json`, o Notion e o vídeo definem `DESCONTO_SIMPLES` (R$ 10.000 a 30 dias com 2% → R$ 9.800). Trocar o método no código quebraria o requisito "regras lidas do JSON, sem valores fixos no código" e daria números diferentes dos que os jurados esperam. Propôs manter o simples como padrão e adicionar `DESCONTO_COMPOSTO` como opção que o gestor escolhe na tela. Confirmou que `make up` sobe tudo em containers (Postgres, API Go e front React servido por nginx).

**O que o time fez com a resposta:** conversou com um jurado e decidiu pelos juros compostos (entrada #4).

---

## #4 · 24/10/2026 ~11:05 · Juros compostos configuráveis em YAML

**Prompt (time → Claude Code):**

> vamos de composto mesmo acabei de falar com um jurado, mas deixe configurável caso queira mudar para simples depois, em yaml

**O que a IA fez:**

1. Conferiu a biblioteca YAML mantida para Go (`go.yaml.in/yaml/v3` v3.0.5).
2. Atualizou `docs/plano.md`:
   - Preço padrão `DESCONTO_COMPOSTO` (desconto racional composto, `face ÷ (1 + i)^(dias/base)`), com `DESCONTO_SIMPLES` disponível.
   - Novo `config/antecipe.yaml` com precedência sobre as regras publicadas, relido sem reiniciar e montado como volume no compose.
   - Fator calculado com decimal e memorizado por prazo; tabela de conferência (30/60/180 dias) que vira teste.
   - O método usado fica gravado em cada lote e entra na chave de idempotência.
3. Atualizou `docs/api.md` para v1.1: `precificacao` em `/config`, `/lotes/{id}` e `/validar`, `calculo.metodo`, `precificacao_efetiva` em `/regras` (método somente leitura na tela do gestor) e exemplos recalculados. Corrigiu também um valor de exemplo da prévia do gestor que estava errado por um fator de 10.

**O que o time fez com a resposta:** aprovou o plano e pediu commit, push e início da construção (entrada #5).

---

## #5 · 24/10/2026 ~11:15 · Aprovação e início

**Prompt (time → Claude Code):**

> de commit e push e comece

**O que a IA fez:** criou o `.gitignore` (`.env`, `amostras/` e builds fora do git), fez o primeiro commit com o plano, o contrato da API, este registro, os dados de origem e o contexto, deu push para o GitHub, instalou o Go 1.27.3 em `~/.local/go` (sem sudo) e escreveu o plano de implementação do backend em `docs/superpowers/plans/2026-10-24-antecipe-backend.md`: 14 tarefas com arquivos, assinaturas, valores e testes. Os pontos sensíveis (CSV em Windows-1252, cabeçalho fora de ordem, API reiniciada no meio do lote, fronteiras das regras e envios simultâneos) têm testes próprios.

**O que o time fez com a resposta:** aprovou o plano e escolheu a execução direta (entrada #6).

---

## #6 · 24/10/2026 11:18 · Execução direta do plano

**Prompt (time → Claude Code):**

> pode fazer assim mesmo in line

**O que a IA fez:** passou a implementar as tarefas do plano em sequência nesta sessão (skill superpowers:executing-plans), com testes antes do código e um commit por tarefa.

- Tarefas 1 a 8 prontas às 11:39 (esqueleto e `make up`, CNPJ, leitor de CSV, regras, juros compostos/simples com YAML, validadores e textos, motor, CLI e gerador).
- **Medições:** leitura de 500 mil linhas em 0,44 s, motor em 0,64 s e `gerar | validar` de 500 mil linhas em 1,9 s.
- A revisão de segurança automática do Claude Code apontou 4 problemas, todos corrigidos com teste:
  - parâmetros numéricos extremos;
  - chave de duplicidade burlável com zeros à esquerda;
  - estouro na conta de concentração;
  - injeção de fórmula no CSV de saída.
- Como o Docker estava sem permissão, os testes de integração passaram a usar um Postgres 18 portátil.

**O que o time fez com a resposta:** pediu arquitetura hexagonal (entrada #7).

---

## #7 · 24/10/2026 11:42 · Arquitetura hexagonal, apperrors, OpenTelemetry e pastas em inglês

**Prompt (time → Claude Code):**

> use arquitetura hexagonal
> pensando em arquitetura de pasta:
>
> - handler para dependencias externas
> - service para regras de negocio
> - repositor para comunicar com o banco
> - test end to end
>
> para erros use o app errors
>
> para observalidade em nivel mais alto use o open telemetri
>
> coloque o nome das pastras do projeto em ingles

**O que a IA fez:** reorganizou o backend em arquitetura hexagonal com pastas em inglês:
- `domain`: entidades, regras do fundo e portas (interfaces);
- `service`: motor, validadores, preço, textos e casos de uso;
- `handler`: REST e leitura do CSV;
- `repository`: Postgres e arquivos;
- `platform`: `apperrors`, `observability` com OpenTelemetry, `config`, `document` e `format`;
- `test/e2e`: testes ponta a ponta.

Os nomes de tipos e funções ficaram em português, porque são a linguagem do negócio (nota, cedente, sacado). Atualizou o plano.

Depois disso, a IA seguiu o plano até a Tarefa 13 (12:15):
- **Postgres:** migrations embutidas, semente do estoque e `COPY` dos resultados.
- **Lotes:** fila, 5 etapas com progresso, spans e métricas.
- **API REST** no contrato `docs/api.md` (agora v1.2).
- **Cessão:** transação com trava por fundo e `ON CONFLICT`.
- **Gestor:** regras versionadas com prévia.
- **Testes ponta a ponta:** 22, todos verdes, contra Postgres de verdade.
- **Logs:** teste que garante que nenhum dado do CSV aparece neles.
- **Medições:** `make carga` com 100 mil linhas em 1,0 s, 250 mil em 2,5 s e 500 mil em 4,9 s.
- **Documentos:** README, relatório e gitleaks sem achados.
- **Segurança:** a revisão automática apontou mais 3 problemas, todos corrigidos com teste:
  - injeção de fórmula no CSV;
  - `Idempotency-Key` reutilizada com outro arquivo;
  - chave de cessão reservada.
- **Pendente:**
  - validar o `make up` com Docker (o usuário ainda sem permissão no grupo `docker`);
  - integrar o front do colega.

**O que o time fez com a resposta:** _(preencher)_

---

## #7a · 24/10/2026 12:13 · Revisão final por subagente (sem prompt novo do time)

**Prompt da IA para um subagente revisor (modelo Opus):** revisar o branch inteiro (commits `2ada0d9..2991d34`) como revisor sênior. Usou o template de revisão da skill superpowers:requesting-code-review, preenchido com:
- o resumo do que foi construído e os critérios de nota do edital;
- os caminhos do plano, do contrato da API, do plano de implementação e do registro de decisões;
- os 5 pontos sensíveis do plano (Windows-1252, cabeçalho fora de ordem, API reiniciada, fronteiras das regras, envios simultâneos);
- o pedido de ler com atenção `docker-compose.yml`, Dockerfiles e Makefile, que ainda não puderam ser executados sem Docker;
- o formato de saída (pontos fortes, problemas por gravidade, o que deixou de julgar e veredito).

**O que foi feito com a resposta:** o revisor confirmou que dinheiro, duplicidade, concorrência, idempotência e recuperação após reinício estão corretos, e mediu 500 mil linhas em 4,85 s. Ele apontou 12 problemas, e a IA os reclassificou pelo efeito na demo e nos critérios de nota.
- **10 foram corrigidos numa rodada única**, cada um com teste que falhou antes e passou depois (o `.dockerignore` e o Makefile não têm teste):
  - corrida na primeira subida do `make up`, que podia desclassificar: healthcheck por TCP, `restart` e espera pelo banco;
  - prévia do gestor errada depois da cessão;
  - erro 500 no `/validar` para arquivo inválido;
  - BOM no CSV de download;
  - perda de notas no `/validar` com aspas;
  - textos de recusa mais claros;
  - aviso para CSV separado por vírgula;
  - erros do OpenTelemetry no log JSON;
  - `.dockerignore`;
  - `.env` como pré-requisito no Makefile.
- **2 menores ficaram anotados:** a tabela leva cerca de 0,5 s por página com 500 mil linhas, e o id do span nos logs de etapa.
- **Validação:** a subida com volume novo foi conferida no Docker.

---

## #8 · 24/10/2026 ~12:15 · Docker liberado

**Prompt (time → Claude Code):**

> rodei esses comandos q vc pediu do sudo

**O que a IA fez:** validou o `make up` com Docker e o caminho completo num clone limpo do GitHub, como o jurado faria.
- **Clone limpo:** subiu tudo, gerou o `.env` sozinho e carregou a semente de 100.554 notas. O exemplo processou certo.
- **Front:** o upload pelo nginx do front (`:3000`), o andamento e a cessão funcionaram.
- **Observabilidade:** traces apareceram no Jaeger e as métricas em `/metrics`. Os logs do container saíram sem nenhum dado do CSV.
- **Carga:** `make carga` no Docker fez 500 mil linhas em 7,1 s.
- **Scripts:** corrigiu o `demo-concorrencia.sh`, que caía quando uma cessão não tinha duplicatas e repetia os mesmos lotes a cada execução. Também silenciou as mensagens do Docker no `carga.sh`.

**O que o time fez com a resposta:** _(preencher)_

---

## #9 · 24/10/2026 ~12:30 · Git pull e CSV oficial de situação cadastral

**Prompt (time → Claude Code):**

> git pull no repo
>
> verifique novamente o notion q eles subiram um csv para fazer a verificação do cnpj, atente-se que são dois tipos de verificação, o primeiro pra ver se de fato é um cnpj válido, e depois vc verifica nesse csv a situação do cnpj para adequar-se nas regras padrões do fundo

**O que a IA fez:**

1. Fez o `git pull`. Entraram o assistente de IA do colega (`ai/`, Agno + AgentOS via OpenRouter), o contrato v1.3 e o `prompts2.md`. O merge foi limpo.
2. Releu o Notion e achou dois arquivos novos: `cnpj_situacao_mock.csv` (3.041 CNPJs) e `resultado_esperado_100k.json` (o gabarito: o estoque de 100 mil linhas avaliado como lote).
3. Fez a engenharia reversa do gabarito com scripts locais:
   - `TIPO_RECEBIVEL` e `VALOR_MINIMO` batem 100%.
   - A data-base de cada nota é a `DATA_AQUISICAO` dela. Com isso, `PRAZO_MAXIMO` bate 100% e `NAO_VENCIDA` fica zerada.
   - A concentração é estática e por fundo.
   - A situação do gabarito vem de outra fonte, diferente do CSV publicado.
4. Ajustou o motor com testes:
   - data-base por nota;
   - concentração estática por fundo sobre a carteira (estoque sem as notas do lote + notas elegíveis do lote);
   - duplicidade na simulação só contra cessões, com a cessão bloqueando notas do estoque;
   - situação cadastral pelo CSV oficial, recarregada a cada subida, depois da checagem do DV do CNPJ.
5. Criou a rota `GET /api/lotes/{id}/resultado.json` e o `antecipe validar --formato json`, no formato do gabarito, além dos testes `backend/test/gabarito` e `TestGabaritoPelaAPI`.
6. **Resultado:** 97.460 linhas idênticas ao gabarito, 3.094 diferentes só na situação cadastral (por causa da fonte divergente dos organizadores) e 0 em outras regras. Validado também no Docker com banco novo (100 mil linhas em ~1,3 s).
7. Criou o `make limpar`, que zera o banco local. Atualizou README, contrato (v1.4), plano e relatório.

**O que o time fez com a resposta:** _(preencher)_

---

## #10 · 24/10/2026 ~14:10 · Proteger o assistente de IA

**Contexto:** a revisão de segurança automática apontou que o serviço do assistente (AgentOS) subia sem autenticação, com a porta 7777 aberta. A IA explicou o risco e sugeriu três correções.

**Prompt (time → Claude Code):**

> sim

(aprovação para aplicar as correções 1, fechar a porta 7777, e 2, ligar a autenticação do AgentOS)

**O que a IA fez:**

1. **Porta fechada:** o serviço `ai` deixou de publicar a porta 7777 (`expose` em vez de `ports`) e só é alcançável pela rede interna do compose.
2. **Autenticação ligada:**
   - o AgentOS recebe a `OS_SECURITY_KEY` e passa a exigir `Authorization: Bearer`;
   - a chave `AI_SECURITY_KEY` é gerada aleatoriamente pelo `make up`, que também a acrescenta em `.env` antigos;
   - o nginx do front injeta a chave em `/ai/` via template (`frontend/templates/default.conf.template`), então ela nunca chega ao navegador.
3. **Testes:** criou `ai/tests/test_seguranca.py`, que confere 401 sem chave ou com chave errada e 200 com a chave. Os 20 testes do serviço passam.
4. **Verificação no Docker:**
   - porta 7777 sem conexão no host;
   - 401 direto na rede interna sem a chave;
   - 200 pelo nginx;
   - chave ausente das respostas públicas.
5. **Documentação:** atualizou README, `ai/README.md` e `docs/api.md`. Em dev, o proxy do Vite `/ai` passa a apontar para `http://localhost:3000`.

A correção 3 (separar as conversas por usuário) não foi aplicada, porque não foi pedida.

**Complemento (revisão de segurança do commit):** o nginx injetava a chave em qualquer rota de `/ai/`, então quem alcançasse o front ainda usaria rotas internas como `/ai/sessions`. O nginx passou a liberar só `GET /ai/assistente/status` e `POST /ai/agents/antecipe-assistente/runs`.
- Verificado no Docker: `/ai/sessions`, `/ai/config`, `/ai/docs` e `/ai/agents/...` respondem 404, e o `GET` em `.../runs` responde 403.
- A sugestão de `:?` no compose ficou de fora de propósito: ela quebraria `make down` em `.env` antigos, e sem a chave o assistente continua fora do alcance externo, porque a porta está fechada.

**O que o time fez com a resposta:** _(preencher)_

---

## #11 · 24/10/2026 ~14:25 · Deploy na Railway por imagens Docker

**Prompt (time → Claude Code):**

> Como conseguimos subir as docker image para realizarmos o deploy pela railway puxando delas, pq o github está desativado temporariamente na railway

**O que a IA fez:**

1. **Consultou a documentação da Railway** por um subagente (web-fetch), que leu as páginas de serviços, private networking, variáveis e domínios. Pontos que importam:
   - deploy a partir de imagem pública (Docker Hub ou GHCR);
   - registry privado só no plano Pro;
   - endereço interno `<serviço>.railway.internal`, dual-stack nos ambientes novos;
   - variáveis de referência `${{Serviço.VAR}}`;
   - porta pública detectada sozinha.
2. **Ajustou a infraestrutura para rodar fora do compose:**
   - o nginx passou a receber os endereços dos serviços por variável (`API_UPSTREAM`, `AI_UPSTREAM`), com padrão para o compose;
   - o nginx passou a resolver o DNS a cada 10 s (`resolver` com `NGINX_LOCAL_RESOLVERS`), porque na Railway o IP interno muda a cada deploy;
   - entrou o `listen [::]:80`;
   - o assistente ganhou a variável `AI_HOST` para ambientes Railway antigos, só IPv6. Testar `::` direto quebrou o compose (o asyncio cria o socket só IPv6), então o padrão continua `0.0.0.0`.
3. **Criou o `make publicar REGISTRO=… TAG=…`**, que monta as três imagens para `linux/amd64` e as envia.
4. **Escreveu o `docs/deploy-railway.md`**, com os serviços e todas as variáveis, e uma alternativa sem registry via `railway up`.
5. **Validou:**
   - compose local: API e assistente pelo nginx, inclusive depois de reiniciar a API;
   - build das três imagens sem enviar: nenhuma tem `.env`, e a da API leva os dados e a config.

**O que o time fez com a resposta:** _(preencher)_

---

## #12 · 24/10/2026 ~14:40 · Onde fazer deploy de graça e custo na Railway

**Prompt (time → Claude Code):**

> onde podemos fazer deplo disso de forma gratuita, e qual seria o custo na railway

**O que a IA fez:**

1. **Pesquisou preços e planos gratuitos atuais** (WebSearch e a página oficial da Railway, por subagente web-fetch). Railway:
   - Free Trial: US$ 5 uma vez, 30 dias, sem cartão, até 1 GB e 2 vCPU por serviço;
   - Free: US$ 1/mês e 0,5 GB por serviço;
   - Hobby: US$ 5/mês com US$ 5 de uso incluídos;
   - uso: cerca de US$ 10 por GB de RAM/mês, US$ 20 por vCPU/mês e US$ 0,15 por GB de volume/mês.
2. **Mediu o consumo real no Docker:**
   - parado, cerca de 0,2 GB de RAM somando assistente, banco, front e API;
   - pico de 711 MB na API durante um lote de 500 mil linhas;
   - banco com 217 MB.
3. **Estimou o custo na Railway:** cerca de US$ 3 a 5 de uso por mês, coberto pelo Hobby (US$ 5/mês). O Free Trial cobre o hackathon de graça, e o plano Free (0,5 GB) não aguenta o pico da API.
4. **Comparou as opções gratuitas:** Railway Trial, Oracle Always Free, notebook com Cloudflare Tunnel, Render e outras.

**O que o time fez com a resposta:** _(preencher)_

---

## #13 · 24/10/2026 ~14:50 · Enviar as imagens

**Prompt (time → Claude Code):**

> pode enviar as imagens entao, como vc recomenda?

**O que a IA fez:** conferiu que a máquina não tinha login no Docker Hub (só o nome dos registries, sem ler credenciais). Pediu ao time que criasse um token Read & Write e fizesse o `docker login`, para a senha não passar pelo chat, e que informasse o usuário para rodar `make publicar REGISTRO=docker.io/USUARIO TAG=v1`.

**O que o time fez com a resposta:** decidiu continuar no notebook do colega (entrada #14).

---

## #14 · 24/10/2026 ~14:55 · Commit e push para o colega continuar

**Prompt (time → Claude Code):**

> commit e push para meu colega continuar no notebook dele

**O que a IA fez:** conferiu que não havia commits novos no remoto, registrou estas entradas e fez commit e push de tudo.

Pendências para quem continuar:
1. **Publicar as imagens:** `docker login -u USUARIO` (token Read & Write do Docker Hub) e depois `make publicar REGISTRO=docker.io/USUARIO TAG=v1`. Na Railway, seguir o `docs/deploy-railway.md`.
2. **Integrar o front React** na pasta `frontend/`, mantendo o `frontend/Dockerfile` e o `frontend/templates/default.conf.template`. O contrato está no `docs/api.md` (v1.4).
3. **Antes do pitch,** rodar `make limpar && make up` para começar com banco novo.

**O que o time fez com a resposta:** _(preencher)_



---

## Frontend · 24/10/2026 · Interface React e Figma

Contribuição desenvolvida no Codex por Mateus. O pedido, os esclarecimentos, as delegações e a validação estão registrados em [prompts-frontend.md](../frontend/docs/prompts-frontend.md). A interface foi adaptada ao contrato v1.1 com mocks explícitos, pois a API Go ainda está em desenvolvimento.

**O que o time fez com a resposta:** _(preencher após incorporar e validar contra o Go)_

A contribuição também incorpora os frames de login e dashboard desenhados pelo time no Figma; o login é uma entrada fictícia exclusiva do modo demo. Histórico e validação atualizados no registro do frontend.

Atualização Novo lote (Figma1:1365): modal com nome/data/CSV e instruções expansíveis, estado vazio do dashboard e mocks em página. Nome permanece como metadado local; contrato multipart inalterado. Validação atual:40testes e fluxos de UI.

Vídeo de ajuda: mascote animado fornecido pelo usuário, otimizado para web e integrado ao botão existente; reprodução sem áudio e respeito a movimento reduzido.

Loading do documento (Figma1:1631): vídeo local otimizado, progresso real e contador, modal até conclusão, acompanhamento reaberto pelo histórico e exemplo mock com simulação perceptível. Registro e42testes em frontend/docs/prompts-frontend.md.

Grade única: máximo1280px e margens responsivas64/40/20, compartilhados por Dashboard, Meus lotes, Resultado e elementos do shell. Validação visual e medidas registradas em frontend/docs/integracao.md.

Cabeçalho padrão: título/descrição e seletor da Dashboard compartilhados por todas as páginas do portal, inclusive estados de carregamento/erro. Ações e contexto ficam abaixo; padding superior64/48/36px. Conferidos desktop/celular, acesso/editor do gestor e modal de envio; lint/build aprovados. Registro em frontend/docs/prompts-frontend.md.

Meus lotes: cards e Novo envio compartilhados com a Dashboard; busca, situação, data-base e ordenação com inputs do login Figma.53testes, lint/build e fluxos desktop/celular conferidos. Detalhes em frontend/docs/prompts-frontend.md.

---

## #15 · 24/10/2026 · Pull do front e teste local

**Prompt (time → Claude Code):**

> de git pull q mergeamos o front end
>
> vamos testar local, rode aqui

**O que a IA fez:**

1. **Pull do front** (PR #1 do Mateus: React + Vite, Figma, MSW só com `VITE_MOCK_API=true`). Resolveu o conflito no `docs/prompts.md` mantendo as duas entradas.
2. **Conferiu o front contra o contrato:**
   - o cliente chama exatamente as rotas do `docs/api.md`;
   - os mocks ficam desligados no build de produção;
   - o Dockerfile (Node → nginx) usa o nosso template;
   - os 96 campos do `types.ts` existem nas respostas reais da API.
3. **O `make up` falhou no `npm ci`:** o firewall Fortinet da rede do evento intercepta o HTTPS de `registry.npmjs.org`, que chega com certificado da FortiGate. Os outros registries (yarnpkg, PyPI, Docker Hub, Go) passam normalmente. O `frontend/Dockerfile` passou a tentar o npm oficial e, se falhar, usar o espelho `registry.yarnpkg.com`; o npm troca o host do lock sozinho.
4. **Teste local pelo nginx (`:3000`):**
   - página e arquivos JS/CSS, e as rotas internas da SPA;
   - config e envio do exemplo concluído, com as 5 etapas;
   - notas, CSV e `resultado.json`;
   - cessão com 12 cedidas e 1 excluída por duplicidade;
   - gestor (204) e assistente (200), com `/ai/sessions` em 404.

**O que o time fez com a resposta:** _(preencher)_

---

## #16 · 24/10/2026 · Arquivo de teste com 1 milhão de linhas

**Prompt (time → Claude Code):**

> /home/joaoemanuel/Downloads/estoque_antecipe_100k/
>
> veja esse arquivo de teste e faça outro com 1 milhao de linhas para testarmos

**O que a IA fez:**

1. Conferiu que o arquivo é o mesmo `estoque_antecipe.csv` do Notion (mesmo SHA-256).
2. **Criou o `scripts/lote-ampliado.sh`** (bash + awk), que monta 10 cópias do estoque: a primeira idêntica e, nas outras, `NU_DOCUMENTO` com o sufixo `-C1` a `-C9`. São 1.005.540 linhas e 196 MB, geradas em 1 s, sem chaves duplicadas. Como cada fundo cresce na mesma proporção, a concentração não muda, e o resultado esperado de cada linha é o do gabarito para a linha original.
3. **Criou o `scripts/comparar-gabarito.py`**, que compara o `resultado.json` de um lote com o gabarito (vale para o original e para o ampliado).
4. **Rodou o teste num banco novo, enviando pelo front (`:3000`):**
   - upload em 0,4 s e conclusão em 14,1 s (13,5 s no servidor);
   - pico de memória: 1,32 GB na API e 220 MB no Postgres;
   - **974.600 linhas iguais ao gabarito, 30.940 diferentes só na situação cadastral e 0 em outras regras.**
5. **Observou** que o andamento fica uns 7 s em "salvando 99%" enquanto o Postgres termina de gravar os índices do `COPY`. Ficou anotado como melhoria visual.

**O que o time fez com a resposta:** _(preencher)_

---

## #17 · 24/10/2026 · Onde está o arquivo de 1 milhão

**Prompt (time → Claude Code):**

> cade o arquivo?

**O que a IA fez:** informou o caminho do `amostras/lote_1m.csv`. A pasta `amostras/` fica fora do git (`.gitignore`), por isso não aparece no GitHub e o VS Code a mostra apagada.

**O que o time fez com a resposta:** _(preencher)_

---

## #18 · 24/10/2026 · O 100k bate 100% com o gabarito? E num SELECT no banco?

**Prompt (time → Claude Code):**

> vc testos no de 100k contra o gabarito que tem no notion deu 100%? se der o select no banco vai dar exatamente todos? eles farão este teste

**O que a IA fez:**

1. **Testou mais uma hipótese para a divergência da situação** (consulta pela raiz do CNPJ): 0 de 87. Confirmou que os sacados barrados por situação no gabarito e os não-ATIVA do CSV publicado não têm nenhuma interseção. São fontes de dados diferentes.
2. **Criou a view SQL `resultado_gabarito`** (migration 0003), que devolve `linha, id, status, barrado_por` no formato do gabarito, com o teste ponta a ponta `TestViewResultadoGabarito` (a view bate com o `resultado.json`).
3. **Criou o `scripts/conferir-gabarito-sql.sh`** e o `make conferir-gabarito LOTE=…`, que carregam o gabarito no Postgres e comparam só com SQL.
4. **Rodou com banco novo, enviando o arquivo de 100 mil pelo front:**
   - 100.554 linhas;
   - 97.460 idênticas (96,9%);
   - 3.094 diferentes só na situação cadastral;
   - 0 em outras regras;
   - aprovadas: 86.713 no banco contra 85.215 no gabarito.
5. **Respondeu ao time:** não dá 100%. Toda a diferença vem da fonte de situação. Recomendou perguntar aos organizadores qual fonte vale e não derivar a situação a partir do gabarito sem o aval deles.

**O que o time fez com a resposta:** _(preencher)_

---

## #19 · 24/10/2026 · Novo cnpj_situacao_mock.csv no Notion

**Prompt (time → Claude Code):**

> colocaram um novo csv chamado cnpj_situacao_mock.csv no final da pagina testa ai

**O que a IA fez:**

1. **Baixou o CSV novo** do fim da página do Notion: 4.360 CNPJs, 89 não-ATIVA, cobrindo os 4.209 sacados do estoque. Antes eram 3.041 CNPJs e 42 não-ATIVA.
2. **Rodou o modelo de regras com ele:** 100.554 de 100.554 linhas iguais ao gabarito (com CNPJ ausente valendo como ATIVA ou como recusado, porque nenhum sacado ficou de fora).
3. **Trocou `data/cnpj_situacao_mock.csv` e ajustou o que dependia dele:** os CNPJs não-ATIVA do gerador e do exemplo (os antigos viraram ATIVA), os testes de contagem e os testes de gabarito, que agora exigem 100%.
4. **Verificou:**
   - suíte inteira verde (17 pacotes);
   - no Docker com banco novo, enviando o arquivo de 100 mil pelo front, a comparação só com SQL deu 100.554 idênticas, com 85.215 aprovadas e 15.339 barradas, igual ao gabarito;
   - o lote de 1 milhão deu 1.005.540 de 1.005.540 iguais, em 13,7 s.
5. Atualizou README, relatório e plano.

**O que o time fez com a resposta:** _(preencher)_

---

## #20 · 24/10/2026 · Pull e rodar o front local

**Prompt (time → Claude Code):**

> git pull e roda local o front pra testar

**O que a IA fez:**

1. **Fez o pull** de mais 4 commits do front: landing animada com vídeo, `VITE_AI_BASE_URL` no build e ajustes de paginação. Conferiu que o Dockerfile continua compatível (com a variável vazia, o assistente fica em `/ai` no mesmo endereço).
2. **Subiu o build de produção no Docker** (`make up`, http://localhost:3000): front 200, vídeo da landing 200, API ok, assistente 200.
3. **Subiu o servidor de desenvolvimento do Vite** (http://localhost:5173), com as dependências instaladas pelo espelho `registry.yarnpkg.com` por causa do firewall da rede. O proxy repassa `/api` para a porta 8080 e `/ai` para a 3000, e os dois responderam.

**O que o time fez com a resposta:** _(preencher)_

---

## #21 · 24/10/2026 · Pull e rodar de novo

**Prompt (time → Claude Code):**

> git pull e roda dnv

**O que a IA fez:**

1. **Fez o pull** de mais 3 commits do front: animação de carregamento na confirmação da cessão, vídeo que toca ao entrar na seção e mais contraste na navegação. Não havia dependência nova, então o Vite recarregou sozinho.
2. **Reconstruiu o Docker** (`make up`) e testou: front, API e assistente responderam nas portas 3000 e 5173, e o exemplo concluiu.
3. **Investigou por que o exemplo deu 14 aprovadas e 9 recusadas** (em vez de 13 e 10). O banco tinha 2 cessões confirmadas pelo front, com 849.734 notas, que diluíram a SERRALHERIA de 12,5% para 0,53% da carteira do ANTECIPE, e a concentração deixou de barrar. É o comportamento correto para esse estado do banco, e confirmou que a cessão aguenta 850 mil notas. Para voltar ao estado do gabarito: `make limpar && make up`.

**O que o time fez com a resposta:** _(preencher)_

