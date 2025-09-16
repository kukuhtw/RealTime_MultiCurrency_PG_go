

```
# Real-Time Multi-Currency Payment Gateway (PoC)

Monorepo **Proof of Concept** untuk _real-time multi-currency payment gateway_ berbasis **microservices** (API Gateway, Payments, FX, Wallet, Risk) dengan **HTTP/JSON**, _observability_ (Prometheus + Grafana), serta _tooling_ untuk dummy data dan testing.

> ⚠️ PoC ini untuk edukasi/demonstrasi. **Bukan** siap produksi (belum ada persistence DB, auth lengkap, HA, dsb.).

---

## Arsitektur Singkat

- **api-gateway**: _entrypoint_ HTTP, melayani _static frontend_ (embed Grafana) + health/metrics.
- **payments**: _mock_ pembuatan transaksi pembayaran.
- **fx**: _mock_ FX rate & conversion.
- **wallet**: _mock_ informasi saldo.
- **risk**: _mock_ score risiko transaksi.
- **prometheus**: scrape metrics dari tiap service.
- **grafana**: dashboard metrik (auto-provision via file JSON).

```

Browser ──> API Gateway (:8080) ──> Services (:8081..8084)
│
└── embeds Grafana (:3000)
Prometheus (:9090) <───── scrape ───── Services
Grafana (:3000)  <────── datasource ─── Prometheus

```

---

## Struktur Repo

```

payment-gateway-poc/
├─ README.md
├─ Makefile
├─ go.mod / go.sum
│
├─ pkg/                  # shared libs/proto (skeleton)
│  ├─ proto/             # \*.proto (tambahkan sesuai kebutuhan)
│  ├─ auth/              # helper auth/JWT (placeholder)
│  ├─ tracing/           # OpenTelemetry init (placeholder)
│  └─ errors/            # error wrapper (placeholder)
│
├─ services/
│  ├─ api-gateway/
│  │  ├─ main.go
│  │  └─ static/index.html   # frontend minimal + embed Grafana
│  ├─ payments/main.go
│  ├─ fx/main.go
│  ├─ wallet/main.go
│  └─ risk/main.go
│
├─ tools/
│  └─ cmd/dummygen/main.go   # generator CSV dummy transactions (-n)
│
├─ tests/
│  ├─ data/dummy\_transactions.csv
│  └─ integration/load\_from\_csv\_test.go
│
├─ deployments/
│  ├─ docker/Dockerfile      # single multi-service builder
│  └─ compose/docker-compose.dev.yaml
│
├─ grafana/
│  ├─ grafana\_payment\_gateway\_dashboard.json
│  └─ provisioning/
│     ├─ dashboards/dashboard.yml
│     └─ datasources/datasource.yml
│
└─ prometheus/prometheus.yml

````

---

## Prasyarat

- **Docker** & **Docker Compose v2**.
- (Opsional) **Go** ≥ 1.22 untuk menjalankan `go` secara lokal.
  - Jika **Go tidak terpasang**, semua build/test dapat dijalankan **via Docker** (disediakan target Makefile).

---

## Quick Start (Dev – Docker Compose)

Jalankan seluruh stack:

```bash
make dev
# atau manual:
# docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build
````

Akses:

* **Frontend (embed Grafana)**: [http://localhost:8080](http://localhost:8080)
* **Grafana**: [http://localhost:3000](http://localhost:3000)  (anonymous viewer aktif; embed diizinkan)
* **Prometheus**: [http://localhost:9090](http://localhost:9090)
* **Services**:

  * payments: [http://localhost:8081](http://localhost:8081)
  * fx: [http://localhost:8082](http://localhost:8082)
  * wallet: [http://localhost:8083](http://localhost:8083)
  * risk: [http://localhost:8084](http://localhost:8084)

Hentikan stack:

```bash
make down
```

Lihat log:

```bash
make logs
```

Status container:

```bash
make ps
```

---

## Smoke Test Cepat

Health & metrics:

```bash
curl -s http://localhost:8080/healthz | jq
curl -s http://localhost:8080/metrics | head

curl -s http://localhost:8081/healthz | jq
curl -s http://localhost:8081/metrics | head
```

Fungsi *mock*:

```bash
# Payments
curl -s -X POST http://localhost:8081/payments \
  -H "Content-Type: application/json" \
  -d '{"id":"PAY-001","currency":"USD","amount":123.45,"source_account":"ACC_SRC_ABC","destination_account":"ACC_DST_DEF"}' | jq

# FX
curl -s "http://localhost:8082/rate?base=USD&quote=IDR" | jq
curl -s "http://localhost:8082/convert?from=USD&to=IDR&amount=1" | jq

# Wallet
curl -s http://localhost:8083/balance/ACC_001 | jq

# Risk
curl -s -X POST http://localhost:8084/score \
  -H "Content-Type: application/json" \
  -d '{"account":"ACC_001","amount":999.99,"currency":"USD"}' | jq
```

---

## Observability

* **Prometheus**: buka **Status → Targets** dan pastikan semua job `UP`.
* **Grafana**:

  * Dashboard disediakan melalui `grafana/grafana_payment_gateway_dashboard.json`.
  * Embed URL yang dipakai `services/api-gateway/static/index.html`:

    ```
    http://localhost:3000/d/paygw-poc?orgId=1&kiosk
    ```
  * Panel contoh: *Go goroutines by service*, *RSS Memory*, *CPU seconds rate*, *Targets UP*, *Scrape duration*.

> Jika iframe di 8080 sempat “Page not found”, pastikan file `index.html` terbaru ter-*serve* (rebuild api-gateway atau gunakan bind-mount untuk `/app/static` di compose).

---

## Dummy Data (CSV)

Generator: `tools/cmd/dummygen`
Output default: `tests/data/dummy_transactions.csv`

### Generate via Docker (tanpa Go lokal)

```bash
# 100 baris (default)
make dummy-docker

# 1000 baris
make dummy-docker N=1000
```

Atau langsung:

```bash
docker run --rm -v "$(pwd)":/app -w /app \
  --entrypoint /usr/local/go/bin/go golang:1.22 run ./tools/cmd/dummygen -n 1000
```

Verifikasi:

```bash
wc -l tests/data/dummy_transactions.csv
# 1001 (1 header + 1000 data)
```

---

## Testing

### Integration test (CSV loader)

```bash
# via Docker (works even if Go not installed)
make test-integration-docker
```

### Semua paket

```bash
# via Docker
make test-docker
```

Jika memiliki Go lokal:

```bash
make test
```

> **Catatan:** file `deployments/docker/Dockerfile` **bukan** kode Go. Pastikan namanya **Dockerfile** (jangan `Dockerfile.go`) agar `go test ./...` tidak menganggapnya file Go (kalau `.go` akan error `illegal character U+0023 '#')`.

---

## Makefile – Target Penting

* `make dev` – *Up* stack dev (Compose, rebuild bila perlu).
* `make down` – stop & remove containers.
* `make logs` – tail logs semua service.
* `make ps` – status container.
* `make dummy-docker N=1000` – generate CSV dummy (via Docker; ubah jumlah dengan `N`).
* `make test-integration-docker` – jalanin integration test (Dockerized).
* `make test-docker` – jalanin semua test (Dockerized).
* `make test` – jalanin semua test (butuh Go lokal).
* `make build` – build semua paket (Go lokal atau fallback Docker – tergantung konfigurasi Makefile kamu).

---

## Endpoints

| Service     | Port | Healthz    | Metrics    | Catatan                           |
| ----------- | ---- | ---------- | ---------- | --------------------------------- |
| api-gateway | 8080 | `/healthz` | `/metrics` | *serve* `static/` + embed Grafana |
| payments    | 8081 | `/healthz` | `/metrics` | `POST /payments`                  |
| fx          | 8082 | `/healthz` | `/metrics` | `/rate`, `/convert`               |
| wallet      | 8083 | `/healthz` | `/metrics` | `/balance/{id}`                   |
| risk        | 8084 | `/healthz` | `/metrics` | `POST /score`                     |
| prometheus  | 9090 | –          | –          | UI & query di 9090                |
| grafana     | 3000 | –          | –          | Anonymous + embedding             |

---

## Development Notes

### Hot reload static frontend

Untuk menghindari rebuild image saat mengubah `services/api-gateway/static/index.html`, aktifkan **bind-mount**:

```yaml
# deployments/compose/docker-compose.dev.yaml (service api-gateway)
volumes:
  - ../../services/api-gateway/static:/app/static:ro
```

### Jika Go lokal belum terpasang

Semua perintah `go` dalam README bisa diganti dengan **Dockerized Go**:

```bash
docker run --rm -v "$(pwd)":/app -w /app \
  --entrypoint /usr/local/go/bin/go golang:1.22 <COMMAND>
# contoh:
# ... go test ./... -v
# ... go run ./tools/cmd/dummygen -n 1000
```

---

## Troubleshooting

* **`Makefile: missing separator`**
  Pastikan baris perintah setelah target memakai **TAB** (bukan spasi).

* **`go: Permission denied` / `go: not found` saat `make test`**
  Jalankan `make test-docker` atau pasang Go lokal. README ini sudah menyiapkan target Dockerized.

* **`illegal character U+0023 '#'` saat `go test ./...`**
  Ubah nama `deployments/docker/Dockerfile.go` → `deployments/docker/Dockerfile`.

* **Grafana iframe “Page not found” di 8080**

  * Pastikan URL iframe berbasis UID: `http://localhost:3000/d/paygw-poc?orgId=1&kiosk`.
  * Pastikan `index.html` terbaru tersaji (rebuild api-gateway atau pakai bind-mount).
  * Env grafana: `GF_AUTH_ANONYMOUS_ENABLED=true`, `GF_SECURITY_ALLOW_EMBEDDING=true`.

* **Compose error `services.<name> additional properties 'args' not allowed`**
  Pastikan `args:` berada **di bawah `build:`** (indentasi tepat).

* **Prometheus/Grafana blank**
  Cek **Prometheus → Status → Targets** harus `UP`.
  Re-provision Grafana: `docker compose ... restart grafana`.

---

## Protobuf / gRPC (Placeholder)

Folder `pkg/proto/*.proto` disiapkan untuk menambah kontrak gRPC. Contoh *workflow*:

```bash
# install tool chain (buf/protoc) sesuai preferensi Anda
# contoh:
# buf generate
# atau
# protoc --go_out=. --go-grpc_out=. pkg/proto/*.proto
```

Tambahkan target `proto` di Makefile sesuai tool yang Anda gunakan.

---

## Security & Production Gaps

* Tidak ada DB/persistence.
* Auth/authorization dummy.
* Tidak ada rate limiting, circuit breaker, retry, tracing lengkap.
* TLS, secret management, dan hardening kontainer belum di-set.

> Untuk produksi, siapkan: penyimpanan transaksi, idempotency key, retry-safe queue, audit log, observability end-to-end (trace), *blue/green* deployment, dsb.

---

## Lisensi

Tentukan lisensi sesuai kebutuhan (MIT/Apache-2.0/dll).

---

```

