# Contas reais e gráficos do assistente

## Integração com as mudanças do João

Base: `origin/main` em `ef95af8` (contas Go v1.5). A cópia integral é `api-grupo-v1.5.md`.

- `/login` conectado usa `POST /api/auth/login`; `/cadastro` cria empresa e inicia sessão em `POST /api/auth/cadastro`.
- Etapa1: nome, e-mail, senha e repetição; etapa2: nome da empresa e CNPJ, obrigatórios no Go. A confirmação não é enviada. Erros de campos do backend são apresentados sem reescrever.
- Cookies HttpOnly, sem token ou senha no armazenamento do navegador. Requisições fetch e upload XHR incluem credenciais.
- Abertura do portal consulta `GET /api/auth/eu`. Sessão ausente leva ao login; indisponibilidade mostra erro e permite tentar novamente, sem mock automático.
- Perfil real usa nome, e-mail, empresa, CNPJ e papel devolvidos pelo Go. Edição de conta e troca de senha não existem no contrato: são explicitamente indisponíveis. Preferências são locais.
- Histórico conectado usa `GET /api/lotes` (até200 resumos); não consulta todas as notas. Filtros locais são sobre esses resumos. A consulta de notas de cada lote permanece paginada no servidor.
- Conta empresa não recebe seletor de gestão. A gestão reconhece cookie de gestor; autorização continua no Go. Logout encerra sessão no servidor.
- `VITE_MOCK_API=true` mantém login/perfil demonstrativos. Cadastro nessa modalidade é explicitamente simulado e não persiste senhas nem cria conta real.

## Gráficos no Markdown

O serviço da IA pode emitir um bloco cercado de linguagem `antecipe-chart`:

```antecipe-chart
{"tipo":"rosca","titulo":"Resultado das notas","metrica":"status"}
```

`tipo`: `barras` ou `rosca`. `metrica`: `status`, `motivos` ou `valores` (esta somente barras). `lote_id`: UUID opcional, usa o lote aberto se omitido. O objeto não aceita dados numéricos fornecidos pelo modelo. Limite de2KB e título de100 caracteres. JSON parcial/inválido continua como código, sem executar nada.

O renderer consulta `GET /api/lotes/{id}` com a sessão atual e usa `totais`/`motivos` oficiais. Não altera textos, não baixa todas as notas e não calcula decisões financeiras. Gráficos incluem alternativa tabular acessível, valores pt-BR e estados de carregamento, vazio, processamento e erro. Barras exibem no máximo12 motivos e a tabela contém os dados completos recebidos no resumo.

Instruções do agente atualizadas em `ai/agents/assistente.py` no clone integrado. É necessário atualizar o serviço de IA para que o modelo conheça o formato. Gráficos não concedem acesso a lotes: o Go verifica a sessão da consulta.

## Validação

Testes unitários verificam ausência de sessão, falha da API, envio das credenciais por cookie, contrato de cadastro/logout e recusa de configurações de gráfico com valores arbitrários ou tipos inválidos. Fluxos visuais testados com respostas HTTP controladas no navegador, em1440px e360px: confirmação de senha, cadastro em duas etapas, perfil conectado sem dados demo e gráfico com tabela alternativa.

Não havia Go/Postgres nem o modelo de IA executando localmente para validar cadastro persistido ou geração espontânea de gráficos. Os testes de navegador exercitam o contrato do frontend, não substituem os testes e2e do backend.

## Layout dos formulários

Login real, demonstração e cadastro compartilham espaçamento e campos em `auth-page.css`: card de até552px, padding40px (24px no celular), rótulos visíveis, campos56px e rodapé de navegação separado por linha e24px de respiro. Cadastro mostra as duas etapas com estado atual acessível. O banner do Fred permanece exclusivo da demonstração. Conferidos login demo/real e as duas etapas em1440px e360px; transição entre etapas preserva os dados.
