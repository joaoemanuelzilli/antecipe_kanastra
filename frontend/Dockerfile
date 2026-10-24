# Provisório: serve uma página de espera e repassa /api para a API.
# Vai ser trocado pelo build do app React (node → nginx) na integração.
FROM nginx:1.27-alpine
# o nginx oficial aplica envsubst nos templates na subida (só nas variáveis definidas, como AI_SECURITY_KEY)
COPY templates/ /etc/nginx/templates/
COPY public/ /usr/share/nginx/html/
