# Provisório: serve uma página de espera e repassa /api para a API.
# Vai ser trocado pelo build do app React (node → nginx) na integração.
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY public/ /usr/share/nginx/html/
