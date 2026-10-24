# Chat de IA

O robô abre o chat em todas as telas. O lote aberto entra no contexto; os detalhes da nota preenchem uma pergunta para revisão e envio. Há sugestões, conversa em memória por lote, nova sessão, carregamento e recuperação de erros.

A conexão é real, inclusive em npm run demo, sem respostas simuladas. Usa GET /ai/assistente/status e POST /ai/agents/antecipe-assistente/runs conforme ai/README.md do grupo. Envia FormData com stream=false, UUID por conversa e dependencies.lote_id; nenhuma chave no navegador. Renderiza Markdown sem HTML.

Em desenvolvimento, AI_PROXY_TARGET aponta para o nginx do grupo (padrão http://localhost:3000). Se a porta estiver ocupada, publique o nginx em uma porta livre e ajuste a variável em .env.local, reiniciando o Vite. Não exponha a porta interna do AgentOS ao navegador.

Suba a infraestrutura conforme o README do grupo e configure os segredos apenas no backend. O status deve retornar disponivel=true. Sem serviço, o painel mostra indisponibilidade e permite tentar conectar novamente. Lotes mockados locais não existem no backend: para consultar lotes reais, use o modo Go e envie um lote à API do grupo.

Validação: desktop e 360px, indisponibilidade local, testes de status, FormData, erro HTTP200 e conteúdo vazio. Resposta real de modelo ainda não validada: o serviço local não estava ativo.
