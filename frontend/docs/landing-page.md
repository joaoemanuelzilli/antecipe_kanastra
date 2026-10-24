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
