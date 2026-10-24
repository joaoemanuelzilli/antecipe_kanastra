FROM node:24-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
# Tenta o registry oficial; se a rede interceptar o HTTPS dele (ex.: firewall de evento),
# usa o espelho registry.yarnpkg.com (mesmos pacotes; o npm troca o host do lock sozinho).
RUN npm ci --fetch-retries=1 --fetch-retry-mintimeout=2000 --fetch-retry-maxtimeout=5000 \
 || npm ci --registry=https://registry.yarnpkg.com/
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY templates/ /etc/nginx/templates/
ENV API_UPSTREAM=api:8080 \
    AI_UPSTREAM=ai:7777 \
    NGINX_ENTRYPOINT_LOCAL_RESOLVERS=1
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s CMD wget -q -O /dev/null http://127.0.0.1/ || exit 1
