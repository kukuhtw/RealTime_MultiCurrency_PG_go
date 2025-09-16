.PHONY: all dev down logs ps build test test-docker test-integration-docker dummy-docker tidy-docker clean

MODULE=github.com/example/payment-gateway-poc
DOCKER_GO = docker run --rm -v "$(CURDIR)":/app -w /app --entrypoint /usr/local/go/bin/go golang:1.22

N ?= 1000
OUT ?= tests/data/dummy_transactions.csv

.PHONY: e2e-host e2e-compose e2e-csv-host e2e-csv-compose

# ==== k6 runner (pakai container, jadi tidak perlu install k6 di host) ====
K6_IMG ?= grafana/k6:latest
K6_RUN  = docker run --rm -it -v "$(CURDIR)":/work -w /work $(K6_IMG)

# Nama network default docker compose (lihat: `docker network ls | grep _default`)
COMPOSE_PROJECT ?= compose
COMPOSE_NETWORK ?= $(COMPOSE_PROJECT)_default

# ---- E2E: skenario campuran (FX, wallet, risk, payments) ----
# ==== k6 opts (override via: make e2e-compose K6_OPTS="--vus 20 --duration 1m") ====
K6_OPTS ?=


# ---- E2E: skenario campuran (FX, wallet, risk, payments) ----
e2e-host: dev
	$(K6_RUN) $(K6_OPTS) \
		-e API_GATEWAY=http://host.docker.internal:8080 \
		-e PAYMENTS=http://host.docker.internal:8081 \
		-e FX=http://host.docker.internal:8082 \
		-e WALLET=http://host.docker.internal:8083 \
		-e RISK=http://host.docker.internal:8084 \
		

e2e-compose: dev
	$(K6_RUN) --network $(COMPOSE_NETWORK) $(K6_OPTS) \
		-e API_GATEWAY=http://api-gateway:8080 \
		-e PAYMENTS=http://payments:8081 \
		-e FX=http://fx:8082 \
		-e WALLET=http://wallet:8083 \
		-e RISK=http://risk:8084 \
		

# ---- E2E: dari CSV ke /payments ----
CSV_COMPOSE_DEFAULT ?= ../data/dummy_transactions.csv

e2e-csv-host: dummy-docker dev
	$(K6_RUN) $(K6_OPTS) \
		-e PAYMENTS=http://host.docker.internal:8081 \
		-e CSV_PATH=./tests/data/dummy_transactions.csv \
		tests/e2e/payment_load_from_csv.js

e2e-csv-compose: dummy-docker dev
	$(K6_RUN) --network $(COMPOSE_NETWORK) $(K6_OPTS) \
		-e PAYMENTS=http://payments:8081 \
		-e CSV_PATH=$(CSV_COMPOSE_DEFAULT) \
		tests/e2e/payment_load_from_csv.js

# Aggregator & bantuan
.PHONY: e2e help
e2e: e2e-compose
help:
	@echo "Targets: dev | down | logs | ps | build | test | test-integration-docker |"
	@echo "         dummy-docker (N=$(N)) | e2e-host | e2e-compose | e2e-csv-host | e2e-csv-compose"
	@echo "Vars:    K6_OPTS='--vus 10 --duration 1m' | COMPOSE_NETWORK=$(COMPOSE_NETWORK)"

dev-grpc:
	docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build risk-grpc wallet-grpc fx-grpc payments-grpc

down-grpc:
	docker compose -f deployments/compose/docker-compose.dev.yaml down risk-grpc wallet-grpc fx-grpc payments-grpc

proto-gen:
	protoc -I. \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  proto/common/v1/common.proto \
	  proto/risk/v1/risk.proto \
	  proto/wallet/v1/wallet.proto \
	  proto/fx/v1/fx.proto \
	  proto/payments/v1/payments.proto

e2e-grpc:
	docker run --rm -it \
	  -v "$(CURDIR)":/work -w /work \
	  grafana/xk6-grpc:latest run tests/e2e/payment_grpc_test.js


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
