COMPOSE_FILE := infra-local/docker-compose.yml
ENV_FILE := infra-local/.env
ENV_FILE_OPT := $(if $(wildcard $(ENV_FILE)),--env-file $(ENV_FILE),)

TAGS ?=

.PHONY: help up down restart logs ps build test swagger run fmt tidy

help:
	@echo "Available commands:"
	@echo "  make up       - Build and start local infra + API"
	@echo "  make down     - Stop and remove containers"
	@echo "  make restart  - Restart the local stack"
	@echo "  make logs     - Follow container logs"
	@echo "  make ps       - List running containers"
	@echo "  make build    - Build API docker image"
	@echo "  make test     - Run Go tests"
	@echo "  make swagger  - Regenerate Swagger docs"
	@echo "  make run      - Run API locally without Docker"
	@echo "  make fmt      - Format Go files"
	@echo "  make tidy     - Tidy Go modules"

up:
	docker compose -f $(COMPOSE_FILE) $(ENV_FILE_OPT) up -d --build

down:
	docker compose -f $(COMPOSE_FILE) down

restart: down up

logs:
	docker compose -f $(COMPOSE_FILE) logs -f

ps:
	docker compose -f $(COMPOSE_FILE) ps

build:
	docker build -t petfinder-api:local .

swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs

run:
	@test -f $(ENV_FILE) && export $$(grep -v '^#' $(ENV_FILE) | xargs) 2>/dev/null; \
	MONGO_URI=$${MONGO_URI:-mongodb://admin:admin@localhost:27017/?authSource=admin} \
	go run $(if $(TAGS),-tags $(TAGS)) ./cmd/api

test:
	go test $(if $(TAGS),-tags $(TAGS)) ./...

fmt:
	gofmt -w cmd internal

tidy:
	go mod tidy
