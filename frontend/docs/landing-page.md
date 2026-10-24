# Landing page

## Direção

Apresentação pública baseada no Figma `EFdeJ9T6mnBayCJcbh60aT`, seção LP `6:1335`, frames desktop `1:1780` e mobile `6:1071`. Mantém a identidade atual do portal: Parkinsans local, verde vivo `#02d866`, texto `#1d1d1d`, branco e mint `#ccf8e1`.

O visitante deve entender o fluxo e acessar a simulação. Conteúdo em sequência: promessa de clareza → envio, resultado e cessão → acompanhamento da análise → detalhes para decidir → dúvidas com Fred → acesso. A estrutura visual e imagens vêm da referência; conteúdo provisório ou duplicado foi substituído por capacidades presentes no frontend e contrato Go.

## Composição

- Cabeçalho de 68px, marca à esquerda, repositório e acesso à direita.
- Hero com imagem local, título de até64px, proteção de contraste e ações.
- Conteúdo de até1200px, margens de40px no desktop e20px no celular.
- Fotos em superfícies de32px de raio no desktop e24px no celular; composição empilhada abaixo de760px.
- Seções claras alternadas com preto; Fred mantém as três ilustrações da base. Dúvidas usam `details` nativo e só exibem explicação quando abertas.
- Prévia de lote explicitamente ilustrativa. Nenhum valor da landing é usado no portal.

## Integração e verificação

`/` é pública. `/dashboard` recebe o dashboard anterior. Envio e consultas mantêm as rotas atuais. No modo demo, a navegação preserva um destino permitido após a entrada. Esse marcador não fornece autenticação de produção.

Build, lint e53 testes aprovados. Conferência visual em1440px e360px, com todas as imagens carregadas e sem rolagem horizontal. Exercitados: dúvida expansível, abertura/fechamento por Escape de Sobre o uso, entrada demo até Novo lote e retorno ao dashboard com foco em Novo envio.

Capturas locais: `output/playwright/landing-desktop.png` e `output/playwright/landing-mobile.png` na raiz do projeto. Origem dos assets em `public/figma/landing/README.md`.

## Animação

Movimento progressivo sem dependências: abertura com aproximação fotográfica e revelação do título; fotos e resumo em camadas; sequência dos três passos; entrada dos cards Fred; wordmark revelado no rodapé. As entradas executam uma vez por montagem via IntersectionObserver e Web Animations API, sem esconder conteúdo no CSS de repouso.

A foto da padaria acompanha discretamente a rolagem somente no desktop com ponteiro preciso, com atualização limitada por requestAnimationFrame e observação de visibilidade. Sem loops contínuos. Botões, setas, Fred e abertura de dúvidas respondem à interação. Ao mudar para movimento reduzido, as animações ativas são canceladas e a foto volta ao repouso. Eventos, observadores e animações são limpos na saída da landing; abas ocultas pausam as animações.

Verificado em1440px e360px, com animações de entrada ativas, teclado e troca dinâmica para movimento reduzido (zero animações e sem overflow). Build e lint aprovados.

## Vídeo de apresentação

Vídeo fornecido pelo usuário (`antecipe video.mp4`),48,8s,1920×1080. Inserido entre Como funciona e a seção da padaria, com atalho Assistir vídeo no hero. Reprodução por clique, som original e controles nativos; sem autoplay, com pausa ao ocultar a aba. Capa extraída aos24s e arquivo otimizado em H.264/AAC com faststart (~4,5MB contra30MB original). Assets locais em `public/media/antecipe-apresentacao.mp4` e `.jpg`. Dados exibidos no vídeo identificados como ilustrativos. Verificados reprodução real e layout em1440px e360px.
