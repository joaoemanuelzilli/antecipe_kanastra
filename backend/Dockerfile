# Build com contexto na raiz do repositório (docker-compose.yml: context: ., dockerfile: backend/Dockerfile)
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/antecipe ./cmd/antecipe

FROM alpine:3.22
RUN adduser -D -u 10001 antecipe
WORKDIR /app
COPY --from=build /out/ /app/bin/
COPY data/ /app/data/
COPY config/ /app/config/
ENV PATH=/app/bin:$PATH
USER antecipe
EXPOSE 8080
CMD ["api"]
