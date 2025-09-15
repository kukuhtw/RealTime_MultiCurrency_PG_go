.PHONY: all dev down logs ps build test test-docker test-integration-docker dummy-docker tidy-docker clean

MODULE=github.com/example/payment-gateway-poc
DOCKER_GO = docker run --rm -v "$(CURDIR)":/app -w /app --entrypoint /usr/local/go/bin/go golang:1.22

N ?= 100
OUT ?= tests/data/dummy_transactions.csv

all: build

# ---- Build & Test via Dockerized Go ----
build:
	$(DOCKER_GO) build ./...

test:
	$(DOCKER_GO) test ./... -v

test-docker: test
test-integration-docker:
	$(DOCKER_GO) test ./tests/integration -v

tidy-docker:
	$(DOCKER_GO) mod tidy

dummy-docker: tidy-docker
	$(DOCKER_GO) run ./tools/cmd/dummygen -n $(N) -out $(OUT)

# ---- Runtime (Compose) ----
dev:
	docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build

down:
	docker compose -f deployments/compose/docker-compose.dev.yaml down

logs:
	docker compose -f deployments/compose/docker-compose.dev.yaml logs -f

ps:
	docker compose -f deployments/compose/docker-compose.dev.yaml ps

clean:
	rm -rf bin/
