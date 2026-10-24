# Registro de prompts

Histórico de todos os prompts de IA usados no projeto, em ordem cronológica, com o que foi feito com cada resposta (item 4.1.2 do edital).

- **Ferramenta:** Claude Code (extensão do VS Code), modelo Claude Opus 5.5 (`claude-opus-5-5`)
- **Fuso:** horário de Brasília (UTC−3)
- **Regra:** nenhum segredo e nenhuma linha do CSV entram neste arquivo.

Cada entrada tem o prompt exato, o que a IA fez e o que o time fez com a resposta. Prompts que a IA mandou para subagentes aparecem dentro da entrada que os originou.

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

**O que o time fez com a resposta:** _(preencher)_
