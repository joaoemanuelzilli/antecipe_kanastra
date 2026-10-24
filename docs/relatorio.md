# Relatório · Antecipe

Hackathon Kanastra · 24/10/2026

## Problema

Uma pequena empresa vendeu a prazo e precisa do dinheiro hoje. Ela pode vender as notas a um fundo (o ANTECIPE FIDC), mas o fundo só compra o que cumpre as regras dele, e a empresa raramente entende por que uma nota foi recusada. Do outro lado, o gestor do fundo depende de deploy para mudar uma regra.

## O que construímos

- **Upload de lotes de 100 mil a 500 mil linhas** com processamento assíncrono em 5 etapas visíveis (ler, validar regras, concentração, preço, gravar). 500 mil linhas terminam em ~5 s.
- **Motor de regras dirigido pelo JSON do fundo.** As 9 regras estão implementadas, só as ativas valem e nenhum número de regra está no código. Código desconhecido no JSON gera um aviso e não derruba o lote.
- **Recusa explicada.** Cada recusa diz qual regra falhou, o que isso significa e o que fazer, com o mesmo texto na tela, na API e no CSV.
- **Preço por juros compostos** (padrão, combinado com um jurado) ou simples, trocável no `config/antecipe.yaml` sem reiniciar.
- **Simular e depois vender.** A cessão é idempotente e segura contra venda dupla, inclusive com confirmações simultâneas (teste ponta a ponta e `make demo-concorrencia`).
- **Regras sem deploy.** O gestor vê o catálogo, testa o impacto de uma mudança no último lote (prévia) e publica uma nova versão. Cada lote guarda a versão e o preço que usou.
- **A mesma validação pela tela e pela API**, mais uma CLI no estilo Unix que usa o mesmo motor.
- **Assistente de IA.** Um painel onde a empresa pergunta em linguagem natural ("como foi o lote?", "por que a nota X deu erro?", "quantas foram recusadas?").
  - É um serviço à parte (Agno + AgentOS, modelo via OpenRouter).
  - Só consulta a API e responde com os textos oficiais das recusas.
  - É opcional: sem chave, todo o resto funciona.

## Tecnologias

Go 1.27 (arquitetura hexagonal), PostgreSQL 18, OpenTelemetry (Jaeger + métricas Prometheus), React 19 + Vite, Python 3.13 com Agno + AgentOS (assistente de IA, modelo via OpenRouter), Docker Compose e Makefile.

## Principais decisões

1. **Hexagonal.** `domain` (entidades e portas), `service` (regras de negócio), `handler` (HTTP e CSV), `repository` (Postgres e arquivos) e `platform` (erros, observabilidade, config). O motor é puro e testado sem banco.
2. **Escala.**
   - O CSV é lido em fluxo.
   - As regras rodam em paralelo por blocos.
   - A concentração faz uma única passada ordenada.
   - O resultado é gravado com `COPY`.
   - A tabela da tela é paginada no servidor, e o navegador nunca recebe 500 mil linhas.
3. **Dinheiro em centavos inteiros.** A potência dos juros compostos usa decimal com 20 casas e fica memorizada por prazo. Nenhum `float` toca em valores.
4. **Duplicidade.**
   - A chave é cedente + número da nota + vencimento. O estoque tem 3.018 notas parceladas, que uma chave sem o vencimento recusaria por engano.
   - O número é canonizado (sem zeros à esquerda nem separadores), para a mesma nota não escapar com outra grafia.
   - A comparação é contra o estoque dos 18 fundos.
5. **Idempotência.**
   - No upload, a chave é o hash do arquivo + data-base + versão das regras + preço. Reenviar o mesmo arquivo devolve o mesmo lote, e mudar a regra gera um resultado novo.
   - Uma `Idempotency-Key` reutilizada com outro arquivo responde 409.
6. **Concorrência na venda.** `FOR UPDATE` no lote, advisory lock por fundo, concentração recalculada sobre o estoque atual e `ON CONFLICT DO NOTHING`.
7. **Uma linha ruim não derruba o lote.** Ela vira a recusa `FORMATO_INVALIDO`, com coluna e problema. O arquivo inteiro só é recusado por problema estrutural.
8. **Logs só com metadados**, com um teste que falha se aparecer qualquer dado do CSV. Segredos ficam só no `.env` gerado pelo `make up`.
9. **IA para explicar, nunca para decidir.**
   - Aprovar, recusar e precificar continuam determinísticos no Go.
   - O assistente é mais um cliente da API: 4 ferramentas só de leitura, sem acesso às tabelas do backend. Usa o mesmo Postgres só para as sessões do chat, no schema `ai`.
   - Ele repete os textos oficiais das recusas e recebe valores já formatados, então não inventa motivo nem erra conta.
   - O teste de logs do serviço pegou o cliente HTTP logando a busca do usuário, e isso foi corrigido.

## Conferência com o gabarito oficial

O estoque de 100 mil linhas enviado como lote bate **100%** com o `resultado_esperado_100k.json` publicado pelos organizadores: 100.554 de 100.554 linhas com o mesmo status e os mesmos motivos, na mesma ordem. A conferência foi feita pela API e só com SQL no banco (view `resultado_gabarito`). O lote ampliado de 1 milhão de linhas também bate 100%.

Para chegar lá:
- a data-base de cada nota é a `DATA_AQUISICAO` dela;
- a concentração é estática e por fundo;
- a duplicidade na simulação vale contra as cessões confirmadas, e a cessão bloqueia qualquer nota que já está no estoque;
- a situação cadastral vem do `cnpj_situacao_mock.csv` final publicado no Notion.

## Resultados

| Lote | Tempo total | Etapa mais lenta |
|---|---|---|
| 100 mil | 1,0 s | gravação (COPY) |
| 250 mil | 2,5 s | gravação (COPY) |
| 500 mil | 4,9 s | gravação: 3,6 s · leitura 0,5 s · regras 0,6 s · preço 0,1 s |

Pela CLI, sem banco, o mesmo motor faz `gerar | validar` de 500 mil linhas em 1,9 s.

## O que ficou de fora

- Consulta real à Receita: a situação cadastral é simulada (`data/situacao_cadastral.csv`).
- Login de várias empresas, assinatura da cessão, registradora e pagamento.
- Autenticação do gestor mais forte que uma senha em header. O assistente de IA também está sem autenticação (o AgentOS aceita JWT).
- Os dados das notas que o assistente consulta vão para o OpenRouter e para o provedor do modelo. No desafio os dados são simulados.
- Deploy público.

## Próximos passos

- Checar a duplicidade no banco com uma tabela temporária, em vez de carregar as chaves em memória, para estoques de milhões de notas.
- Ligar o span do upload ao span do processamento (trace link).
- Eventos (SSE) no lugar do polling de andamento.
- Consulta real de situação cadastral, com cache.
