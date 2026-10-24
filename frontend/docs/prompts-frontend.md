# Registro de prompts — frontend

24/10/2026 · horário de Brasília. Ferramenta: Codex. Esta é a contribuição do frontend para ser incorporada ao `docs/prompts.md` do grupo, preservando o histórico já escrito. Nenhuma credencial real ou linha do estoque foi incluída.

## Pedido inicial

> Desenvolve esse arquivo, melhores stacks para vercel e supabase.

Esclarecimento:

> Dados mockados

Foi desenvolvido um protótipo independente com interface, dados fictícios e motor em TypeScript, a partir do PDF original. Esse backend não faz parte do frontend integrado entregue ao grupo.

## Mudança de direção e contrato do grupo

> seguinte, meu grupo aq ja fez todo o plano
> e o contrato da api...
>
> ta aqui https://github.com/joaoemanuelzilli/antecipe_kanastra.git
>
> preciso que vc deixe o nosso perfeitamente funcionando com isso.. ok?

Resposta ao esclarecimento sobre disponibilidade da API:

> Ainda em desenvolvimento; seguir o contrato com mocks

### O que foi feito

- Lidos `docs/api.md`, `docs/plano.md`, o JSON oficial das regras e o histórico de decisões do grupo. Contrato v1.1; última revisão consultada do repositório: `23cdd440e73355e73085817755296ff0de0e8f86`. O contrato não mudou entre essa revisão e a primeira consultada, `2ada0d9`.
- Preservada a identidade visual do protótipo e migrado o frontend para React + Vite. Retiradas dependências de Next.js, banco, IA e motor de regras da interface integrada.
- Criados os tipos e o cliente HTTP com nomes snake_case, centavos inteiros, upload XHR, autenticação por header, paginação no servidor e cessão idempotente.
- Mocks HTTP explícitos do mesmo contrato via MSW, sem fallback silencioso quando uma API real falha.
- Gestor gerado pelo catálogo, respeitando os campos definidos pelo YAML e as rotas de prévia/publicação/histórico.
- Preparados Dockerfile, nginx e instruções de incorporação na pasta `frontend/` do grupo.

### Delegações do Codex

1. Interface da empresa: adaptar envio/andamento/resultado/detalhes/cessão e histórico aos tipos e endpoints oficiais; preservar o visual; nunca carregar todas as notas; usar textos e cálculos do backend sem reescrever.
2. Gestor/regras: reconstruir formulário pelos catálogos; autenticar em `/gestor/verificar`; manter método e parâmetros sobrescritos YAML somente leitura; retirar o assistente de IA; usar os corpos exatos de prévia e publicação.
3. Mocks: simular todos os endpoints do contrato, com estados de fila/processamento, centavos, erros por linha, paginação, CSV, idempotência e versões; testar por HTTP interceptado; identificar os dados como fictícios.

### Validação

Testes de cliente e mocks, compilação estrita, lint e builds. No navegador: envio → etapas → resultado → filtro por motivo → detalhe com campos nulos → confirmação com nota excluída; gestor → prévia → publicação → histórico. Verificação de layout responsivo em360px. Nenhum resultado de carga do backend Go foi inventado.

### Uso da resposta pelo time

Frontend autônomo preparado para copiar para `frontend/`. O backend Go ainda não estava disponível para um teste real nesta etapa; a integração foi validada contra o contrato e seus mocks.


## Adaptação ao Figma

Pedido do usuário:

> ta.. eu ja to desenhando aq no figma
> tenta encaixar dentor do layout ai
> https://www.figma.com/design/EFdeJ9T6mnBayCJcbh60aT/Sem-t%C3%ADtulo?node-id=1-913
> se liga...

Resposta sobre a entrada do app:

> Incluir login de demonstração

Foram lidos os frames Login (1:364) e Dashboard (1:427), com contexto de código de alta fidelidade e screenshots pelo plugin Figma. Assets originais foram baixados para `public/figma/`, com fonte Parkinsans local. O dashboard usa até três resumos de lotes deste navegador; nenhum total de conta ou endpoint não previsto foi inventado. O login fictício descarta e-mail/senha e guarda apenas um marcador booleano de sessão, exclusivamente no modo demo.

Delegações: dashboard e cartões; auditoria das rotas/compatibilidade e ciclo de vida dos mocks; login e responsividade. O root integrou shell, tema e navegação, conferiu os screenshots e os fluxos no navegador. O botão do mascote abre orientação local, respeitando a decisão do plano de não incluir IA. Nenhuma mudança foi escrita no arquivo Figma.


## Novo lote

Referência enviada pelo usuário: https://www.figma.com/design/EFdeJ9T6mnBayCJcbh60aT/Sem-t%C3%ADtulo?node-id=1-1365

> novo lote...

Lidos a seção e os três frames por contexto de alta fidelidade e screenshot. A tela de envio virou modal sobre o dashboard com dois estados, instruções expansíveis e novo estado vazio. Delegações: modal/upload; nome como metadado local e downloads; estado vazio e transporte de mocks. O contrato do grupo não foi ampliado: o nome não entra no multipart nem renomeia o arquivo.

A conferência revelou que o transporte anterior de mocks podia deixar chamadas passarem ao Go indisponível depois de tempo sem interação. A demonstração passou a usar os mesmos handlers MSW com Fetch/XHR interceptados na página, e os downloads passaram a baixar o Blob do endpoint no modo demo. O modo conectado ao Go continua explícito e não muda para mocks em falhas. Foram executados40testes, lint, TypeScript e builds, além de envio real de fixtures fictícias pela UI, erro422, resultado, download filtrado, foco de teclado e responsividade.


## Vídeo do mascote de ajuda

Comentário do usuário sobre o botão “Abrir ajuda sobre a antecipação”:

> fi aq precisa ter o videozinho la...

O MP4 fornecido pelo usuário substituiu o avatar estático no botão flutuante. Criada uma versão web menor, sem áudio, em loop, com reprodução inline e poster; mantidos o nome acessível, o recorte circular e a abertura da ajuda. A reprodução pausa com a ajuda aberta, aba oculta ou preferência por movimento reduzido. Revisão independente do enquadramento e acessibilidade; compilação estrita, lint e build aprovados. Na UI: vídeo carregado e reproduzindo, clique abre a ajuda e fechamento restaura o foco e a reprodução; viewport de360px sem overflow horizontal.


## Loading da análise do documento

Referência do usuário: Figma1:1631, com o MP4 `Robot_reviewing_papers_20261024124729.mp4`.

> faz o loading do documento.. e se clicar para usar um documento de exemplo simula ai o loading tbm..

Implementado o estado de análise no próprio modal, mantendo o dashboard com blur. Vídeo otimizado, progresso de upload XHR e depois progresso/linhas fornecidos pela API; polling500ms até conclusão/erro, consulta novamente sem reenviar em falhasHTTP e fechamento preservando o histórico. O mesmo painel aparece ao reabrir um lote pendente. O dashboard passou a ler os metadados recentes ao renderizar para exibir o lote depois do fechamento da análise.

Delegações: painel/vídeo responsivos; timings e testes dos mocks; revisão do contrato/idempotência e teste do cliente. Cada exemplo novo no modo mock usa a chave opcional documentada para iniciar uma nova simulação, estável em tentativas da mesma abertura/data; arquivo e dedupe do envio normal preservados. Simulação ajustada para9,5s com timestamps e progresso coerentes.

Validação:42testes, TypeScript/lint/build, vídeo reproduzindo no navegador, exemplo repetido, CSV válido com uma nota/R$9.803,92, cancelar e reabrir pelo histórico com22/30linhas e75,2%, viewport360×800 sem overflow. Nenhum push/deploy foi realizado. A revisão encontrou uma limitação anterior de sincronização dos mocks em múltiplas abas, fora desta alteração; não foram apagados dados do usuário para ocultá-la.


## Mascote com loop de ida e volta

Comentário do usuário no botão de ajuda:

> deixa aq em loop.. so que tipo terminou o video ele roda ao contrario sabe? pra parecer um loop melhor...

Foi criada a sequência normal seguida do reverso no próprio MP4, sem duplicar os quadros extremos. O botão mantém vídeo sem som, clique para a ajuda e respeito a movimento reduzido/aba oculta. Sequência de478quadros validada contra o vídeo fornecido, incluindo os238quadros em ordem inversa; reprodução e duração conferidas no navegador.


## Tamanho padrão da grade

Pedido do usuário:

> precisamos ter um tamanho maximo para a grid do sistema... em meus lotes, resultado do lote e a dash cada um ta com uma regra diferente... precisamos de um tamanho padrao

Criada uma única regra de1280px e gutters64/40/20px, compartilhada por Dashboard, listas, resultado e demais páginas. Rodapé, perfil e confirmação foram alinhados à mesma grade. Removidas as larguras concorrentes, preservadas a sidebar e as dimensões dos modais.

Delegação: auditoria independente dos seletores e dos wrappers, incluindo a contenção de um rótulo acessível absoluto que provocava overflow na tabela. Root implementou a regra única e verificou no navegador as três páginas em1920,1440,1024,700 e360px, além do Resultado em768px. Corrigidos overflow intermediário, busca comprimida e deslocamento de7,5px ao abrir modal com scrollbar reservada.


## Cabeçalho padrão da Dashboard

Comentário do usuário sobre o título/descrição e seletor Usuário/Gestor da Dashboard:

> esse deve ser o padrão de header para todas as paginas..

Criado `PageHeader`, adotado em Dashboard, Meus lotes, Resultado, Regras, Gestor e404. O envio e o processamento continuam sobre a Dashboard com seu cabeçalho. Removidos o perfil flutuante duplicado e as faixas antigas acima do título. Ações, breadcrumb, versão e sessão ficam abaixo. Padding superior64/48/36px centralizado; login demo e modais conservam seus títulos próprios. Gestor mantém um único h1 no gate e no editor.

Delegações: componente compartilhado e migração da Dashboard; Regras e Gestor; auditoria independente da cascata e importações. Root integrou Meus lotes, Resultado/estados,404 e espaçamentos, preservando o contrato e a grade de1280px.

Validação: lint, TypeScript/build e navegação real com mocks. Desktop1440px: os cinco cabeçalhos emx170/y64, largura1191, título32px, descrição16px e seletor alinhado no topo. Em1024px o topo é48px; em700/360px,36px. Conferidos os cinco em360px, login/editor do gestor e abrir/fechar Novo lote. Corrigido um overflow do histórico de regras no celular com quebra dos cartões. Sem alteração nos endpoints, publicação de regras ou push/deploy.


## Meus lotes com o padrão da home

Pedido do usuário: botão de enviar lote no padrão da Dashboard, filtros melhores junto da busca com inputs do login Figma e substituir a listagem pelos cards da home.

Consultado o login Figma1:364: campos60px, raio24px, borda10%, Parkinsans16px. Reutilizados os componentes e assets existentes dos cards da home, extraindo seu CSS para evitar dependência do wrapper Dashboard. Criado NewLotLink compartilhado; posição absoluta mobile permanece restrita ao hero.

Meus lotes agora tem grid3/2/1, busca sem acentos, situação, data-base exata, ordenação recentes/antigos/maior valor, contador e limpeza de filtros com foco devolvido à busca. Consulta pendente sem resumo não é confundida com processamento; concluído sem aprovadas fica em Analisados. Erros não são mascarados como zero, há aviso para resumos indisponíveis em filtros e nova tentativa; crescimento dos cards suporta quebra dos badges.

Delegações: extração dos componentes; módulo puro de filtros e11testes; auditoria de semântica e acessibilidade. Root integrou a UI e conferiu os fluxos no navegador.

Validação:53testes, TypeScript/lint e builds normal/demo. Preview mock isolada em3088 para evitar resets por alterações concorrentes: histórico vazio → Novo envio → exemplo/loading → resultado → Meus lotes; data-base24/10/2026 encontra o lote, outra data não; limpar e ordenar. Na prévia3086: busca sem acento, Analisados9/14, Cedidos sem resultado com aviso dos5resumos antigos indisponíveis, limpeza com foco na busca. Desktop1440, tablet1024 e celular360 sem overflow nas verificações; cards normais163px. Preservados os dados locais e o contrato Go. Alterações de landing/rotas feitas em paralelo por outro trabalho foram preservadas.


## Publicação para o grupo

Pedido do usuário: “Sobe para o repositorio.. os meninos precisam testar la”. Snapshot completo, incluindo a landing final, reaplicado sobre a main e5592fe. A conta conectada tem somente leitura no upstream; preparada contribuição pelo fork matteuszu/antecipe_kanastra e pull request ao grupo, sem merge/deploy.

Preservados backend, contrato e template nginx atuais. Build React incorporado ao Dockerfile existente e placeholder removido. Corrigido suporte aos parâmetros opcionais do catálogo v1.4. Instalação limpa no clone,65testes, TypeScript, lint e builds normal/demo aprovados; Docker e Go não executados nesta rodada. Guia de teste no README.


## Perfil do usuário

Pedido: usuário não vê regras; criar perfil, logout e troca de senha seguindo o visual existente. Implementado Meu perfil com identidade fictícia, formulário nome/e-mail/empresa/cargo, preferências e sessão. A troca de senha é explicitamente demonstrativa, sem gravação nem envio de credenciais. O logout limpa o acesso demo e a senha do gestor. Regras ficam na gestão; a rota antiga redireciona para perfil.

Verificados salvamento após reload, efeitos reais da redução de animações, erro e sucesso da simulação de senha, logout/reentrada, mobile360px e rota antiga.69testes aprovados; verificação de tipos/lint/build. Atualização da mesma branch/PR de testes autorizado anteriormente.
