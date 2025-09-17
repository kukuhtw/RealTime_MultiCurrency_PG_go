#!/usr/bin/env bash
set -euo pipefail

PROJECT=compose
COMPOSE_FILE="deployments/compose/docker-compose.dev.v3.yaml"

echo "==> Cleaning up existing resources (project: $PROJECT)..."
# Hentikan & hapus containers, network, tapi simpan volumes (data Postgres dsb)
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" down --remove-orphans || true

echo "==> Pruning dangling networks (safe)..."
docker network prune -f >/dev/null || true

echo "==> Checking for port conflicts..."
# Semua port yang dipublish di compose:
# - postgres:        15432
# - kafka:           9092
# - kafka-ui:        9081
# - kafka-exporter:  9308
# - api-gateway:     18080
# - wallet-grpc:     19093 (gRPC), 19103 (/metrics)
# - fx-grpc:         19102 (gRPC)
# - risk-grpc:       19094 (gRPC), 19104 (/metrics)
# - db-rs:           19095 (gRPC), 19105 (/metrics)
# - payments-rs:     19096 (gRPC), 19106 (/metrics)
# - prometheus:      19097
# - grafana:         3000
PORTS=(15432 9092 9081 9308 18080 19093 19103 19102 19094 19104 19095 19105 19096 19106 19097 3000)

for port in "${PORTS[@]}"; do
  if command -v lsof >/dev/null 2>&1; then
    if lsof -i :"$port" >/dev/null 2>&1; then
      echo "  - Port $port in use → trying to kill holder process..."
      # PID bisa lebih dari satu; kita hentikan semuanya dengan hati-hati
      for pid in $(lsof -ti :"$port"); do
        kill -9 "$pid" 2>/dev/null || true
      done
    fi
  else
    # fallback dengan ss (Linux)
    if ss -lnt "( sport = :$port )" | grep -q ":$port"; then
      echo "  - Port $port in use (detected via ss). Silakan hentikan prosesnya manual."
      exit 1
    fi
  fi
done

echo "==> Building and starting services..."
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" up -d --build

echo "==> Waiting for services to warm up..."
sleep 15

echo "==> Services status:"
docker compose -f "$COMPOSE_FILE" -p "$PROJECT" ps

echo
echo "==> Endpoints:"
cat <<'EOF'
Infra:
  - Postgres     : localhost:15432  (DB: poc, user: postgres, pass: secret)
  - Kafka (PLAINTEXT) : localhost:9092
  - Kafka UI     : http://localhost:9081
  - Kafka Exporter: http://localhost:9308/metrics

Gateway & Services:
  - API Gateway  : http://localhost:18080        (UI & /metrics)
  - wallet-grpc  : localhost:19093 (gRPC), http://localhost:19103/metrics
  - fx-grpc      : localhost:19102 (gRPC)
  - risk-grpc    : localhost:19094 (gRPC), http://localhost:19104/metrics
  - db-rs        : localhost:19095 (gRPC), http://localhost:19105/metrics
  - payments-rs  : localhost:19096 (gRPC), http://localhost:19106/metrics

Observability:
  - Prometheus   : http://localhost:19097
  - Grafana      : http://localhost:3000  (admin / admin)
EOF
