# =========================================
# Payment Gateway PoC - Makefile (gRPC stack)
# =========================================

# ---- Go (dockerized) ----
MODULE        ?= github.com/example/payment-gateway-poc
DOCKER_GO     ?= docker run --rm -v "$(CURDIR)":/app -w /app --entrypoint /usr/local/go/bin/go golang:1.22

# ---- Dummy data (python) ----
.PHONY: gen-dummy
gen-dummy:
	python3 tools/generate_dummy_data.py

# ---- Compose (gRPC-only stack) ----
COMPOSE_FILE    ?= deployments/compose/docker-compose.dev.v3.yaml
COMPOSE_PROJECT ?= compose
COMPOSE_NETWORK ?= payment-network

.PHONY: dev-grpc down-grpc restart-grpc rebuild-grpc up-nb-grpc logs-grpc ps-grpc
dev-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d --build

down-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) down --remove-orphans

restart-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) down --remove-orphans
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d --build

rebuild-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) down --remove-orphans
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) build --no-cache
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d

up-nb-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) up -d --no-build

logs-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) logs -f

ps-grpc:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) ps

# ---- grpcurl (seeding Admin services) ----
GRPCURL_IMG ?= fullstorydev/grpcurl:latest
GRPCURL     ?= docker run --rm --network $(COMPOSE_NETWORK) -v "$(CURDIR)/seeds":/seeds $(GRPCURL_IMG)

.PHONY: seed-grpc
seed-grpc: dev-grpc
	@echo "⏳ waiting services..."
	sleep 10
	$(GRPCURL) -plaintext -d @/seeds/wallet_accounts.json wallet-grpc:9093 wallet.v1.Admin/SeedAccounts
	$(GRPCURL) -plaintext -d @/seeds/fx_rates.json      fx-grpc:9102     fx.v1.Admin/SeedRates
	$(GRPCURL) -plaintext -d @/seeds/risk_rules.json    risk-grpc:9094   risk.v1.Admin/SeedRules
	@echo "✅ gRPC seeding done"

.PHONY: seed-and-run
seed-and-run: gen-dummy seed-grpc
	@echo "Seeding done. Next: make e2e-grpc-csv"

# ---- k6 runners (E2E gRPC) ----
K6_IMG        ?= grafana/k6:latest
K6_RUN        ?= docker run --rm -it --network $(COMPOSE_NETWORK) -v "$(CURDIR)":/work -w /work $(K6_IMG)
PAYMENTS_TARGET ?= payments-rs:9096
CSV_PATH      ?= ./data/dummy_transactions.csv

.PHONY: e2e-grpc e2e-grpc-csv
e2e-grpc: dev-grpc
	$(K6_RUN) -e TARGET=$(PAYMENTS_TARGET) run tests/e2e/payment_grpc_test.js

e2e-grpc-csv: dev-grpc
	$(K6_RUN) -e TARGET=$(PAYMENTS_TARGET) -e CSV_PATH=$(CSV_PATH) run tests/e2e/payment_grpc_from_csv.js

# ---- Go build & test (dockerized) ----
.PHONY: all build test test-docker test-integration-docker tidy-docker clean deps
all: build

build:
	$(DOCKER_GO) build ./...

test:
	$(DOCKER_GO) test ./... -v

test-docker: test

test-integration-docker:
	$(DOCKER_GO) test ./tests/integration -v

tidy-docker:
	$(DOCKER_GO) mod tidy

deps:
	$(DOCKER_GO) get github.com/segmentio/kafka-go@latest
	$(DOCKER_GO) get github.com/jackc/pgx/v5@latest
	$(DOCKER_GO) get github.com/google/uuid@latest
	$(DOCKER_GO) mod tidy

clean:
	rm -rf bin/

# ---- Protobuf (dockerized protoc toolchain) ----
PROTOC_VER ?= 27.1
.PHONY: proto-gen-docker proto-gen
proto-gen-docker:
	docker run --rm -v "$(CURDIR)":/work -w /work golang:1.23 bash -c "\
		set -euo pipefail && \
		apt-get update && apt-get install -y curl unzip >/dev/null && \
		curl -sSL -o /tmp/protoc-$(PROTOC_VER)-linux-x86_64.zip https://github.com/protocolbuffers/protobuf/releases/download/v$(PROTOC_VER)/protoc-$(PROTOC_VER)-linux-x86_64.zip && \
		unzip -qo /tmp/protoc-$(PROTOC_VER)-linux-x86_64.zip -d /usr/local 'bin/*' 'include/*' && \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest && \
		protoc -I. \
		  --go_out=. --go_opt=paths=source_relative \
		  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
		  proto/gen/common/v1/common.proto \
		  proto/gen/risk/v1/risk.proto \
		  proto/gen/risk/v1/risk_admin.proto \
		  proto/gen/wallet/v1/wallet.proto \
		  proto/gen/fx/v1/fx.proto \
		  proto/gen/fx/v1/fx_admin.proto \
		  proto/gen/payments/v1/payments.proto \
	"

proto-gen:
	protoc -I. \
	  --go_out=. --go_opt=paths=source_relative \
	  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	  proto/gen/common/v1/common.proto \
	  proto/gen/risk/v1/risk.proto \
	  proto/gen/risk/v1/risk_admin.proto \
	  proto/gen/wallet/v1/wallet.proto \
	  proto/gen/fx/v1/fx.proto \
	  proto/gen/fx/v1/fx_admin.proto \
	  proto/gen/payments/v1/payments.proto

# ---- Housekeeping Docker/Compose ----
.PHONY: down-orphans prune-containers prune-images prune-volumes clean-all nuke
down-orphans:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) down --remove-orphans

prune-containers:
	docker container prune -f

prune-images:
	docker image prune -f

prune-volumes:
	docker volume prune -f

clean-all: down-orphans prune-containers prune-images

nuke:
	docker compose -f $(COMPOSE_FILE) -p $(COMPOSE_PROJECT) down -v --remove-orphans
	docker network prune -f
	docker image prune -f
	docker container prune -f
	docker volume prune -f
