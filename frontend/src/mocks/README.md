# Demonstração da interface — referência v1.4

Este diretório simula os fluxos usados pela interface. A referência HTTP atual é [`docs/api-grupo-v1.4.md`](../../docs/api-grupo-v1.4.md), cópia integral de `docs/api.md` do repositório do grupo. O mock é independente do Go: não importa seu motor, banco, API, estoque ou segredos. `regras.json` conserva o JSON fornecido pelo grupo. O método efetivo de preço simula a precedência de `config/antecipe.yaml`: juros compostos, apesar de o JSON original dizer desconto simples.

Para executar, na raiz do repositório use `cd frontend`, `npm ci` e `npm run demo` (Node >=22.12.0). Abra a porta informada pelo Vite, normalmente `http://localhost:5173`. `.env.demo` é versionado e não contém segredos; nenhum serviço Go, Supabase, Docker ou Vercel é necessário. Para uma prévia compilada, use `npm run build:demo` e `npm run preview`.

O mock mantém cenários controlados de UI, não um segundo motor de produção. A concentração depende das empresas marcadas nas fixtures, a data-base de análise simulada é a do lote, e não há estoque oficial. Algumas respostas de erro/publicação também preservam comportamentos anteriores à v1.4. Portanto não use os resultados mockados para validar o gabarito, a política financeira ou a equivalência completa com o backend implementado.

Ative os mocks explicitamente no bootstrap de demonstração, aguardando `startMockWorker()` antes de renderizar. No modo conectado, não inicie os interceptadores. Falha de uma API real não ativa esta simulação automaticamente. Endpoints `/api` não previstos retornam erro 501 aninhado; nenhum endpoint de IA foi acrescentado.

- `handlers.ts`: exporta `handlers`, `resetMockState()` e `advanceMockTime(ms)`.
- `browser.ts`: mantém uma instância por página de `BatchInterceptor` (Fetch e XHR), com `getResponse` dos handlers MSW. `startMockWorker()` conserva o nome do bootstrap anterior, mas não registra nem inicia Service Worker. O HMR troca apenas os handlers. Uma migração remove somente a antiga inscrição `/mockServiceWorker.js` do mesmo origin; inscrições alheias são preservadas.
- `server.ts`: exporta `server` para testes Node com `setupServer`.
- `fixtures.ts`: exporta `gerarCSVExemplo(dataBase)`, `DEMO_PASSWORD`, catálogo e configuração.
- Senha pública e fictícia do gestor: `demo-gestor`. Enviada em `X-Gestor-Senha`; não é uma credencial de backend.
- Exemplo: `/demo/exemplo.csv`. Em modo demo, `?data_base=AAAA-MM-DD` adapta os vencimentos. O arquivo público estático usa 24/10/2026.

A fixture sintética possui 30 linhas: 19 aprovadas e 11 recusadas nas regras iniciais, com exemplos de todos os motivos ativos, duplicidade e uma linha com `FORMATO_INVALIDO`. Essa contagem pertence somente à demonstração, não ao `data/exemplo.csv` ou ao estoque do grupo. As linhas de prazo mínimo e concentração do cedente passam inicialmente, porque as duas regras vêm desligadas no JSON original; ligá-las no gestor demonstra os dois motivos adicionais. Notas podem ter várias recusas; os contadores do resumo usam somente a principal.

Os cenários cadastrais e de concentração são marcados nos nomes das empresas fictícias. Isso exercita os componentes e o contrato, sem representar consulta cadastral real ou cálculo do estoque. O mock lê o arquivo enviado, conserva número, nomes e documentos, e trata problemas de linhas como recusas. Um CSV inteiro com cabeçalho inválido é rejeitado antes da criação do lote.

O processamento fica 500 ms na fila e passa por cinco etapas de 1.800 ms, totalizando 9,5 segundos de demonstração. As constantes `DEMO_QUEUE_MS`, `DEMO_STAGE_MS`, `DEMO_PROCESSING_MS` e `DEMO_TOTAL_MS` concentram esses tempos. `iniciado_em` ocorre 500 ms após `criado_em`; `concluido_em` ocorre aos 9.500 ms; `duracao_ms` informa os 9.000 ms de processamento ativo, sem o tempo da fila. O progresso fica em zero enquanto está na fila e só chega a 100% na conclusão. Polling repetido não acelera o relógio nem altera os timestamps. A situação deriva do tempo decorrido, permitindo polling e reabertura do lote. Snapshots com dados das linhas enviadas podem ser conservados no `localStorage` do navegador; use somente dados fictícios. Se a quota for excedida, ficam apenas na memória da sessão de mocks. Abas distintas podem conservar snapshots divergentes; isso não representa a persistência do Go. Os testes avançam o relógio com `advanceMockTime`.

O documento `DEMO-PARALELA` aparece aprovado na simulação e é excluído na confirmação, exercitando `notas_excluidas`. A cessão retorna os valores efetivamente cedidos e mantém o mesmo resultado quando repetida; a tela pode comparar o resumo original com o resultado confirmado. A nota de R$ 10.000,00 a 30 dias produz líquido de **980392 centavos**, com juros compostos de 2% ao mês, como no contrato.

Execute `npm test -- src/mocks/handlers.test.ts`. Os testes verificam respostas e formatos HTTP reais por `fetch` interceptado, sem navegador ou chamadas ao backend.

Os downloads CSV do modo demo usam o endpoint pelo Fetch interceptado e baixam os bytes recebidos em Blob; não reconstroem CSV a partir da tabela. No modo conectado, os links continuam navegando para a API.

O envio normal continua idempotente: mesmos bytes, data-base e versão de regras devolvem o lote anterior. Nome de arquivo ou parâmetros de cache da URL não alteram essa decisão. Para demonstrar as etapas novamente, cada nova abertura/ação de **Usar lote de exemplo**, somente no modo mock, fornece uma chave explícita opcional de idempotência; tentativas da mesma ação/data reutilizam a chave. Os bytes da fixture não mudam. O modo Go e o dedupe do envio normal são preservados.
