# Provisório: serve uma página de espera e repassa /api para a API.
# Vai ser trocado pelo build do app React (node → nginx) na integração.
FROM nginx:1.27-alpine
# o nginx oficial aplica envsubst nos templates na subida (só nas variáveis definidas, como AI_SECURITY_KEY)
COPY templates/ /etc/nginx/templates/
# padrões do compose; na Railway, troque por <serviço>.railway.internal:<porta>
ENV API_UPSTREAM=api:8080 \
    AI_UPSTREAM=ai:7777 \
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1
COPY public/ /usr/share/nginx/html/
