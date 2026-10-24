SHELL := /bin/bash
.DEFAULT_GOAL := ajuda

N ?= 500000
SEMENTE ?= 42

.PHONY: ajuda up down limpar publicar logs logs-ai test test-ai test-integracao gerar carga check demo-concorrencia

ajuda: ## Mostra os comandos
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-20s %s\n", $$1, $$2}'

.env:
	@cp .env.example .env
	@sed -i.bak "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')/" .env
	@sed -i.bak "s/^GESTOR_SENHA=.*/GESTOR_SENHA=$$(head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n')/" .env
	@sed -i.bak "s/^AI_SECURITY_KEY=.*/AI_SECURITY_KEY=$$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')/" .env
	@rm -f .env.bak
	@echo "Criei o .env com senhas aleatórias (ele não vai para o git)."

up: .env ## Sobe Postgres, API e front (primeira vez: alguns minutos)
	@grep -qE '^AI_SECURITY_KEY=.+' .env || echo "AI_SECURITY_KEY=$$(head -c 24 /dev/urandom | od -An -tx1 | tr -d ' \n')" >> .env
	docker compose up --build -d
	@echo "Esperando a API ficar pronta..."
	@for i in $$(seq 1 90); do \
		if curl -fsS http://localhost:8080/api/saude >/dev/null 2>&1; then break; fi; \
		sleep 2; \
		if [ $$i -eq 90 ]; then echo "A API não respondeu. Veja: make logs"; exit 1; fi; \
	done
	@echo ""
	@echo "  Front:            http://localhost:3000"
	@echo "  API:              http://localhost:8080/api/saude"
	@echo "  Traces (Jaeger):  http://localhost:16686"
	@echo "  Métricas:         http://localhost:8080/metrics"
	@echo "  Assistente (IA):  http://localhost:3000/ai/assistente/status  (só via nginx)"
	@echo "  Senha do gestor:  $$(grep '^GESTOR_SENHA=' .env | cut -d= -f2)"
	@echo ""
	@grep -qE '^OPENROUTER_API_KEY=.+' .env || { echo "  Assistente de IA desligado: coloque OPENROUTER_API_KEY no .env e rode make up de novo."; echo ""; }

down: ## Para tudo (os dados do banco ficam no volume)
	docker compose down

limpar: ## Para tudo e APAGA o banco local (volta ao estoque original na próxima subida)
	docker compose down -v

logs: ## Acompanha os logs da API
	docker compose logs -f api

logs-ai: ## Acompanha os logs do assistente de IA
	docker compose logs -f ai

test: ## Testes de unidade do backend (precisa de Go 1.27)
	cd backend && go test ./...

test-ai: ## Testes do assistente de IA (precisa do uv; não usa rede nem chave)
	cd ai && uv venv -q --allow-existing --python 3.13 .venv && uv pip sync -q --python .venv/bin/python requirements-dev.txt && .venv/bin/python -m pytest -q

test-integracao: .env ## Todos os testes, inclusive banco e ponta a ponta (Postgres do compose; precisa de Go 1.27)
	docker compose up -d --wait postgres
	-docker compose exec -T postgres psql -U antecipe -d antecipe -qc "CREATE DATABASE antecipe_test" 2>/dev/null
	set -a && . ./.env && set +a && cd backend && \
		TEST_DATABASE_URL="postgres://antecipe:$${POSTGRES_PASSWORD}@localhost:5433/antecipe_test?sslmode=disable" go test -count=1 -p 1 ./...

gerar: .env ## Gera um lote de teste em amostras/ (ex.: make gerar N=500000)
	@mkdir -p amostras
	docker compose run --rm --no-deps -T api antecipe gerar --linhas $(N) --semente $(SEMENTE) --estoque data/estoque_antecipe.zip > amostras/lote_$(N).csv
	@echo "Gerado amostras/lote_$(N).csv"

carga: .env ## Mede lotes de 100k, 250k e 500k linhas (precisa do make up)
	./scripts/carga.sh

check: ## Procura segredos no repositório (gitleaks)
	docker run --rm -v "$$PWD:/repo" zricethezav/gitleaks:v8.30.1 detect --source /repo --no-banner

demo-concorrencia: .env ## Mostra duas cessões simultâneas sem venda duplicada (precisa do make up)
	./scripts/demo-concorrencia.sh

REGISTRO ?=
TAG ?= latest

publicar: ## Monta e envia as imagens para um registry (ex.: make publicar REGISTRO=docker.io/seu-usuario TAG=v1)
	@test -n "$(REGISTRO)" || { echo "Informe o registry: make publicar REGISTRO=docker.io/seu-usuario (ou ghcr.io/seu-usuario)"; exit 1; }
	docker build --platform linux/amd64 -f backend/Dockerfile -t $(REGISTRO)/antecipe-api:$(TAG) .
	docker build --platform linux/amd64 -t $(REGISTRO)/antecipe-web:$(TAG) ./frontend
	docker build --platform linux/amd64 -t $(REGISTRO)/antecipe-ai:$(TAG) ./ai
	docker push $(REGISTRO)/antecipe-api:$(TAG)
	docker push $(REGISTRO)/antecipe-web:$(TAG)
	docker push $(REGISTRO)/antecipe-ai:$(TAG)
	@echo ""
	@echo "  Imagens publicadas:"
	@echo "    $(REGISTRO)/antecipe-api:$(TAG)"
	@echo "    $(REGISTRO)/antecipe-web:$(TAG)"
	@echo "    $(REGISTRO)/antecipe-ai:$(TAG)"
	@echo "  Passo a passo da Railway: docs/deploy-railway.md"
