# Chat de IA

O robô abre o chat em todas as telas. O lote aberto entra no contexto; os detalhes da nota preenchem uma pergunta para revisão e envio. Há sugestões, conversa em memória por lote, nova sessão, carregamento e recuperação de erros.

A conexão é real, inclusive em npm run demo, sem respostas simuladas. Usa GET /ai/assistente/status e POST /ai/agents/antecipe-assistente/runs conforme ai/README.md do grupo. Envia FormData com stream=true (SSE), UUID por conversa e dependencies.lote_id; nenhuma chave no navegador. Renderiza Markdown sem HTML.

Em desenvolvimento, AI_PROXY_TARGET aponta para o nginx do grupo (padrão http://localhost:3000). Se a porta estiver ocupada, publique o nginx em uma porta livre e ajuste a variável em .env.local, reiniciando o Vite. Não exponha a porta interna do AgentOS ao navegador.

Suba a infraestrutura conforme o README do grupo e configure os segredos apenas no backend. O status deve retornar disponivel=true. Sem serviço, o painel mostra indisponibilidade e permite tentar conectar novamente. Lotes mockados locais não existem no backend: para consultar lotes reais, use o modo Go e envie um lote à API do grupo.

O chat recebe RunContent progressivamente e exige RunCompleted. RunError e interrupções descartam a resposta parcial, preservando a pergunta para nova tentativa. Requisições têm timeout e cancelamento ao sair do contexto.

Validação: desktop e 360px, indisponibilidade local, testes de status, FormData, erro HTTP200 e conteúdo vazio. Resposta real de modelo ainda não validada: o serviço local não estava ativo.

## Endereço por ambiente

Em `.env.local` (desenvolvimento) ou nas variáveis de build da hospedagem:

```env
VITE_AI_BASE_URL=https://seu-backend.example/ai
```

A base inclui `/ai` quando usa o nginx do grupo. Vazia usa `/ai` na mesma origem. A variável é pública: não recebe chave ou senha. Status e conversas usam a mesma base. Uma origem diferente exige CORS no serviço. Reinicie o Vite após alterar; em produção, gere um novo build.

Para usar o proxy de desenvolvimento sem CORS, deixe VITE_AI_BASE_URL vazia e configure AI_PROXY_TARGET com a origem do nginx, por exemplo http://localhost:3000. O proxy conserva o prefixo /ai.

No Docker, VITE_AI_BASE_URL é argumento de build. Com o nginx do grupo, prefira deixá-la vazia e configure AI_UPSTREAM no servidor; a chave interna continua apenas no nginx/backend.
