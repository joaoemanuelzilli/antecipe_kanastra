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

