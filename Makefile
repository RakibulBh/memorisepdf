APP := api
# Scoped rather than ./..., which walks into web/node_modules and finds a
# stray Go package shipped inside one of the npm dependencies.
PKGS := ./cmd/... ./internal/...
BIN_DIR := bin
MAIN := ./cmd/api
BINARY := $(BIN_DIR)/$(APP)

COMPOSE := docker compose -f deploy/docker-compose.dev.yml

.PHONY: build run test fmt vet clean tidy web-dev web-build dev-up dev-down dev-logs psql check

## Go

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BINARY) $(MAIN)

run: build
	./$(BINARY)

test:
	go test $(PKGS)

vet:
	go vet $(PKGS)

fmt:
	go fmt $(PKGS)

tidy:
	go mod tidy

## Frontend

web-dev:
	cd web && npm run dev

web-build:
	cd web && npm run build

## Local dependencies (postgres and tika)

dev-up:
	$(COMPOSE) up -d

dev-down:
	$(COMPOSE) down

dev-logs:
	$(COMPOSE) logs -f

psql:
	$(COMPOSE) exec db psql -U memorisepdf -d memorisepdf

## Everything CI runs

check: fmt vet test
	cd web && npx tsc --noEmit && npm run lint

clean:
	rm -rf $(BIN_DIR)
