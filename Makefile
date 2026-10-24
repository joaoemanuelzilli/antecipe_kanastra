SHELL := /bin/bash
.DEFAULT_GOAL := ajuda

N ?= 500000
SEMENTE ?= 42

.PHONY: ajuda up down logs test

ajuda: ## Mostra os comandos
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  make %-20s %s\n", $$1, $$2}'

.env:
	@cp .env.example .env
	@sed -i.bak "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$$(head -c 16 /dev/urandom | od -An -tx1 | tr -d ' \n')/" .env
	@sed -i.bak "s/^GESTOR_SENHA=.*/GESTOR_SENHA=$$(head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n')/" .env
	@rm -f .env.bak
	@echo "Criei o .env com senhas aleatórias (ele não vai para o git)."

up: .env ## Sobe Postgres, API e front (primeira vez: alguns minutos)
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
	@echo "  Senha do gestor:  $$(grep '^GESTOR_SENHA=' .env | cut -d= -f2)"
	@echo ""

down: ## Para tudo (os dados do banco ficam no volume)
	docker compose down

logs: ## Acompanha os logs da API
	docker compose logs -f api

test: ## Testes de unidade do backend (precisa de Go 1.27)
	cd backend && go test ./...
