# Validação da integração — 24/10/2026

## Estado atual da contribuição

Referência atual: contrato **v1.4** em `docs/api.md` do grupo, na `main` consultada em `e5592fe`. A cópia integral está em [`api-grupo-v1.4.md`](api-grupo-v1.4.md); a v1.1 fica como histórico. O backend Go já está implementado no repositório. Esta contribuição não executou testes contra Go/Postgres nem o Compose e não atribui a si os resultados de testes ou de carga publicados pelo grupo.

A contribuição foi reaplicada sobre a `main` atual e utiliza a branch `codex/perfil-usuario` do fork [`matteuszu/antecipe_kanastra`](https://github.com/matteuszu/antecipe_kanastra), para revisão pelo grupo em pull request. Merge e deploy não foram executados.

Teste isolado da UI, a partir da raiz da branch: `cd frontend`, `npm ci`, `npm run demo`, com Node >=22.12.0. O endereço padrão é `http://localhost:5173`. `.env.demo` é versionado; a demonstração usa dados sintéticos e não exige `.env.local`, segredos, Go, Supabase, Docker ou Vercel. `npm run dev` e `npm run build` usam a API Go; para compilar a demo use `npm run build:demo`.

O contrato v1.4 define `DATA_AQUISICAO` como data-base por nota quando preenchida, concentração estática por fundo, duplicidade contra cessões confirmadas na análise e o download `resultado.json`. A UI deixa essas decisões com o Go e mostra os textos/valores recebidos. Nem todas as rotas documentadas têm uma ação de tela: métricas, `resultado.json` e o serviço separado de IA não são usados por esta interface. A ajuda do mascote permanece local.

Os mocks validam fluxos de UI, sem prometer paridade financeira com a v1.4. Concentração por empresas marcadas, data-base única do lote e algumas respostas de erro/publicação são simplificações conhecidas. Não são um oráculo do gabarito nem medição de desempenho. O Dockerfile atual usa Node24 → nginx1.27 com `templates/default.conf.template`, preservando `API_UPSTREAM`, `AI_UPSTREAM` e a chave interna do nginx; nenhuma chave entra no frontend.

## Verificação do snapshot para publicação

Instalação limpa no clone com `npm ci`:280pacotes e0vulnerabilidades. Passaram65testes em7arquivos, TypeScript, lint e builds normal/demo. Não havia Docker/nginx na máquina para executar o contêiner. O template nginx foi preservado integralmente da main; somente a etapa de build React foi incorporada ao Dockerfile.

Compatibilidade v1.4: catálogo aceita `opcional:true`; Usar padrão remove a propriedade do JSON e permite ativar a concentração por cedente sem inventar agrupamento. Obrigatórios e valores inválidos continuam sendo validados, com testes específicos. Go/Postgres e carga real permanecem fora desta rodada.

## Histórico de implementação e verificações

As seções abaixo registram rodadas anteriores. Contagens de testes, condições do backend e observações de ausência de push/deploy dizem respeito àquela etapa, não ao estado atual da publicação. A validação atual está registrada acima.

Na integração inicial, a referência era o contrato v1.1, revisão `23cdd440e73355e73085817755296ff0de0e8f86`.

## Compatibilidade

| Parte | Implementação no frontend |
|---|---|
| Configuração | `/api/config`; data-base, limite de arquivo e descrição do preço vêm da API |
| Upload | XHR, progresso, `data_base`, 202 e replay200 |
| Andamento | GET do lote a cada500ms; cinco etapas; fila, percentual, linhas e tempo |
| Dinheiro | Centavos inteiros; formatação exata; sem motor financeiro de produção na interface |
| Notas | Páginas de50, busca/filtro/ordenação na API e nulos em linhas inválidas |
| Exportação | Endpoint CSV com os filtros atuais; sem percorrer páginas no navegador |
| Cessão | UUID por abertura do diálogo, mesma chave nas tentativas, valores finais e notas excluídas |
| Gestor | Verificação204, senha em sessionStorage e header `X-Gestor-Senha` |
| Regras | Catálogo dinâmico; método e parâmetros sobrescritos YAML somente leitura; prévia/publicação/histórico |
| Histórico de lotes | Somente IDs de envios deste navegador, pois o contrato não define listagem de lotes |

## Verificações executadas

- 42 testes: 22 de contrato/transporte MSW, 16 de cliente/formatação e 4 de download CSV.
- TypeScript estrito, lint e builds normal e de demonstração.
- Instalação `npm ci`, testes e build em cópia limpa e autônoma do frontend.
- Navegador: upload → fila/etapas → resultado; filtro por motivo; detalhe de FORMATO_INVALIDO; Escape restaurando foco; cessão com uma nota excluída e atualização do valor final.
- Gestor: senha de demonstração, método YAML somente leitura, prazo máximo180→120, prévia, publicação com autor e histórico separado.
- Responsividade em360px, tabela substituída por cartões, ordenação e ações disponíveis; corrigido botão de novo lote para não transbordar.
- Data-base escolhida26/10/2026 verificada no resultado e nos vencimentos relativos do exemplo.
- Corrigido o fallback dos mocks para não interceptar módulos do Vite em `/src/api/…`; teste com servidor HTTP real garante a passagem desses arquivos.

## Limites desta validação

Na rodada inicial, o backend Go ainda estava em desenvolvimento. Não foi executada uma cessão contra o Go/Postgres real, nem o Compose do grupo, nem testes de carga de100mil/500mil linhas. O Dockerfile e o nginx foram preparados, mas o Docker não estava disponível nesta máquina para executar a imagem. O processamento das fixtures é simulado e não constitui medição de desempenho do backend.

Naquela rodada, nenhuma mudança no contrato foi necessária, nenhum serviço remoto foi provisionado e nenhum deploy ou push no repositório do grupo foi realizado. A pasta frontend e o patch estavam preparados para incorporação.


## Interface do Figma

Referência: arquivo EFdeJ9T6mnBayCJcbh60aT, seção 1:913, frames Login 1:364 e Dashboard 1:427. Implementados menu lateral, paleta verde, tipografia Parkinsans, assets locais originais, hero e cartões de lotes. A geometria desktop do dashboard foi conferida em 1440 × 1024: hero em x170/y179, largura1206 e altura188. Login e dashboard conferidos em360px sem rolagem horizontal.

O login de demonstração não é autenticação de produção: aceita dados fictícios ou permite entrada sem formulário; só grava um booleano em sessionStorage. Não envia nem persiste e-mail/senha. Acesso do gestor continua no endpoint oficial. O fluxo login → dashboard → novo envio → exemplo → resultado foi verificado. O exemplo com data-base27/10/2026 retornou15aprovadas/15recusadas eR$82.989,08 sob a versão2 das regras da demonstração. Saída pelo rodapé retorna ao login e remove o marcador de acesso.

Os cartões do dashboard mostram até três lotes consultados por ID, sem endpoint novo nem valores fictícios inseridos no layout. A ajuda do mascote é um guia local de uso. Login de usuário, recuperação de senha, cadastro e IA não foram adicionados ao contrato Go.

A demonstração passou a usar intercepção de Fetch e XHR na própria página, com os handlers MSW e sem depender de Service Worker. Isso remove a dependência de um worker que vinha deixando chamadas passarem para a API indisponível após períodos sem interação na prévia. Os testes cobrem inicialização única/HMR, passagem de assets sem transformação, rotas desconhecidas501, upload/replay, tempo ocioso e migração seletiva da inscrição antiga. A causa interna exata dos episódios do transporte anterior não foi atribuída ao login ou à captura de tela.


## Modal Novo lote — seção Figma1:1365

- Frames1:1133 (recolhido),1:1198 (instruções abertas) e1:1584 (dashboard sem lotes). Assets originais novos em `public/figma/`; o mascote vazio usa export isolado110×124 para conservar o recorte.
- Geometria desktop1440×1024 conferida: grupo recolhido480×749,6 emx480/y137,2; aberto998×749,6 emx221/y137,2; formulário480×653,6 e onboarding502×416,4. Diferenças fracionárias vêm do line-height do navegador.
- Campos de nome e data, CSV selecionado/troca/remoção, envio XHR, progresso e erros422 preservados; nome salvo só como metadado local, sem novo campo no multipart.
- Navegador: envio sem arquivo informa o ajuste; CSV com cabeçalho inválido mostra os campos ausentes; CSV válido com uma nota chega ao resultado como “Novo lote — Figma”,R$9.803,92 e uma aprovação; Escape fecha e restaura foco no botão do dashboard.
- Mobile360×800: modal328px, sem rolagem horizontal; instruções se empilham e o conteúdo do overlay é rolável.
- O download “Baixar recusadas” com filtro FORMATO_INVALIDO foi executado no navegador: CSV não vazio com uma linha recusada. Bytes, filtros, filename e revogação do Blob também são cobertos por testes.
- Backend Go/Docker continuam fora da validação disponível nesta máquina; nenhum deploy ou push foi realizado.


## Loading — Figma1:1631

Modal480px com vídeo416×234, barra8px e contagem real de linhas; renderização em360×800 com painel328px/vídeo280×157,5 e sem overflow. O blur solicitado permanece no fundo. Vídeo H.264 sem áudio, com poster e respeito a movimento reduzido.

Upload e exemplo ficam na tela de análise até o estado terminal; consultas por500ms sem acumular notas. Uma falhaHTTP interrompe o polling e oferece nova consulta sem repetir o POST. Cancelar fecha o acompanhamento; a API não possui endpoint de cancelamento. Um lote pendente reaberto pelo histórico usa o mesmo painel.

O exemplo mock passa por fila500ms+5etapas de1,8s. Nova abertura/ação de exemplo usa uma chave explícita opcional do contrato; tentativas da mesma ação/data reutilizam a chave. Isso não muda os bytes do arquivo, o dedupe do upload normal ou o modo conectado ao Go. Os testes verificam as fronteiras do relógio e uma falha de envio seguida de tentativa com chave estável.

Browser: exemplo novo e repetido até o resultado; CSV real fictício com uma nota atéR$9.803,92; Cancelar retornando ao dashboard e histórico atualizado; reabertura pendente mostrando22/30linhas e75,2%. Sem erros de execução no fluxo. Na inspeção surgiram IDs antigos ausentes entre duas abas: limitação anterior de snapshots dos mocks por aba, mantida fora do escopo visual e registrada para o time; não indica um teste contra o Go.


## Grade única do sistema

As regras divergentes de1180px/1370px e o padding horizontal próprio da Dashboard foram substituídos por `src/app-layout.css`: máximo1280px, gutter64/40/20px e `.container` nas páginas. Sidebar continua fora do cálculo interno; rodapé, perfil e confirmação usam os mesmos limites. A faixa do perfil foi reservada também no Resultado para evitar sobreposição. As medidas desta seção substituem a geometria horizontal antiga registrada na revisão inicial do Figma.

Navegador: Dashboard/Meus lotes/Resultado com x365,5 e largura1280 em1920×1080; x170/largura1191 em1440×1024; x146/largura823 em1024×900; x92/largura573 em700×900; x20/largura305 em360×800. A reserva de scrollbar é incluída nesses valores do navegador. Conferidos também768px, rodapé e barra de confirmação.

A inspeção corrigiu overflow do cabeçalho acessível da tabela, que agora fica contido no wrapper de rolagem, e do botão de novo envio entre620–700px. A busca ocupa uma linha própria nessa faixa. O resultado permanece sem rolagem lateral da página; a tabela conserva rolagem local quando necessária. Diálogo aberto/fechado em1920px manteve x365,5/largura1280, sem compensação dupla da scrollbar.


## Cabeçalho compartilhado do portal

O padrão da Dashboard agora é o componente `PageHeader`: título/descrição à esquerda e seletor Usuário/Gestor à direita. Aplicado a Dashboard, Meus lotes, Resultado (inclusive loading/erro), Regras, Gestor (acesso/editor) e404. Ações, caminho, versão e sessão ficam abaixo; não há um segundo seletor absoluto. O envio mantém a Dashboard ao fundo do modal. O login demo e os títulos dos modais continuam próprios.

O padding superior é64/48/36px, controlado por `src/app-layout.css`; essas medidas substituem a faixa reservada antiga do perfil mencionada na seção anterior. Em1440px, os cinco headers foram medidos emx170/y64 com largura1191, título32px, descrição16px e seletor alinhado no topo. Em1024px, y48; em700px e360px, y36. No celular o seletor pode quebrar para a linha seguinte.

Conferidos os cinco em360px, entrada/editor do gestor, navegação e abrir/fechar Novo lote. Corrigido overflow no histórico público de versões, permitindo quebra dos cartões. Lint e build com TypeScript passaram. A revisão independente não encontrou regressões na cascata nem importações cíclicas. O teste continua usando mocks; não houve publicação de regras, push ou deploy.


## Meus lotes: cards e filtros

A antiga lista foi substituída pelo mesmo LotCard da Dashboard, em3/2/1colunas. NewLotLink compartilha o botão verde e o asset original de24px; ícones dos cards mantêm24/18px. Inputs do login Figma1:364:60px, raio24px, Parkinsans16px no desktop. Grade máxima1280px e PageHeader preservados.

Busca, situação, data-base e ordem atuam em resumos dos IDs locais. Não foi criado GET de listagem, nem há varredura de notas. Data-base desconhecida não usa a data de envio como substituta. Consulta pendente não equivale a processamento; concluído sem aprovadas é Analisado. Valores de cessão prevalecem na ordenação; desconhecidos ficam por último e zero é preservado.

Conferidos busca/filtros/reset, teclado, vazio, modal de envio e retorno do exemplo; desktop1440, tablet1024 e celular360. Os cards mantêm163px nos exemplos e permitem crescimento para badges maiores. Avisos agregados evitam esconder falhas de consulta em filtros; limpar devolve foco à busca.53testes e builds normal/demo aprovados. A limitação anterior de IDs mock antigos entre abas continua fora deste ajuste; nenhum dado foi apagado para ocultá-la. API Go e Docker continuam sem validação real disponível.


## Perfil no lugar das regras públicas

A navegação do usuário passou a `/perfil`; `/regras` redireciona, e os links da ajuda/landing foram ajustados. A gestão das regras e a senha do gestor permanecem existentes. Nenhuma autorização do backend foi modificada. Perfil demo usa dados fictícios em sessionStorage, uma lista explícita de campos permitidos e nenhuma senha persistida/enviada. Logout remove o acesso demo e a senha do gestor. No modo Go, perfil tem estado não configurado.

Conferidos no navegador: salvar/recuperar nome após reload, pausa do mascote por preferência, validação e conclusão da troca demonstrativa, confirmação de logout até o login e retorno ao perfil. Desktop com grade1280px e celular360px sem overflow; rota antiga redirecionando corretamente, sem erros de execução.69testes aprovados, incluindo armazenamento inválido/bloqueado e descarte de campos de credenciais; TypeScript/lint e builds aprovados.
