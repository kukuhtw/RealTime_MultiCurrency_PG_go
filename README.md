# Real-Time Multi-Currency Payment Gateway (PoC) - Problem & Solution Analysis

## 🎯 **Problem yang Ingin Dipecahkan**

### 1. **Kompleksitas Sistem Payment Multi-Currency**
- Transaksi lintas mata uang membutuhkan **konversi real-time** dengan rate yang akurat
- Perlu handling **fluktuasi nilai tukar** yang dapat berubah cepat
- Validasi **ketersediaan saldo** dalam berbagai currency

### 2. **Risk Management yang Real-Time**
- Deteksi **transaksi mencurigakan** secara instan
- Penilaian **risk score** berdasarkan berbagai faktor
- Pencegahan **fraud** sebelum transaksi diproses

### 3. **Observability dan Monitoring**
- Kesulitan **melacak performance** sistem secara real-time
- Kurangnya **visibility** pada metrics penting (latency, error rate, throughput)
- **Troubleshooting** yang lambat ketika terjadi issues

### 4. **Testing dan Data Realistik**
- Kesulitan mendapatkan **data testing** yang menyerupai production
- **Load testing** dengan scenario yang realistic
- Validasi **end-to-end flow** tanpa environment production

### 5. **Arsitektur Microservices yang Terkelola**
- Koordinasi antara **multiple services** yang specialized
- **Communication patterns** yang efisien antara services
- **Service discovery** dan health checking

## ⚡ **Solusi yang Diterapkan dalam Code Ini**

### 1. **Microservices Architecture**
```go
// Setiap service memiliki responsibility khusus
services/
├─ api-gateway/     // Entry point & routing
├─ payments/        // Processing transaksi
├─ fx/             // Currency conversion
├─ wallet/         // Balance management
├─ risk/           // Fraud detection
```

### 2. **Real-Time Currency Exchange**
```go
// FX service menangani konversi real-time
GET /rate?base=USD&quote=IDR
GET /convert?from=USD&to=IDR&amount=100
```

### 3. **Comprehensive Observability Stack**
```
Prometheus ───┐
              │ Scrape metrics
Services ─────┘
              │ Query data
Grafana ──────┘
```

### 4. **Data-Driven Testing Infrastructure**
```go
// Generator data dummy yang realistic
tools/cmd/dummygen -n 1000
// Output: PAY-000001,IDR,188.52,ACC_SRC_5081,ACC_DST_1228
```

### 5. **Load Testing dengan k6**
```javascript
// Simulasi traffic realistik dari CSV data
const randomPayment = csvData[Math.floor(Math.random() * csvData.length)];
http.post(`${paymentsUrl}/payments`, payload, params);
```

### 6. **Health Checking & Metrics**
```bash
# Standard health check endpoints
curl http://localhost:8081/healthz
curl http://localhost:8081/metrics
```

### 7. **API Gateway Pattern**
```go
// Single entry point dengan embedded dashboard
api-gateway/
├─ main.go              // Routing logic
└─ static/index.html    // Embedded Grafana UI
```

## 🏗️ **Architecture Solution**

```
Browser/Client
    │
    ▼
API Gateway (8080) ────┐
    │                  │
    ├─ Static Content  │
    │  (Grafana Embed) │
    │                  │
    └─ API Routing ────┤
                       │
                       ▼
               [Microservices]
    ┌─────────────┼─────────────┐
    │             │             │
    ▼             ▼             ▼
Payments       FX Service    Wallet Service
(8081)         (8082)        (8083)
    │             │             │
    └─────────────┼─────────────┘
                  │
                  ▼
              Risk Service
                 (8084)
```

## 📊 **Value yang Diberikan**

### 1. **Real-Time Visibility**
- Dashboard Grafana menunjukkan **metrics live**
- **Monitoring** performance setiap service
- **Alerting** potential issues

### 2. **Scalability**
- Setiap service dapat **di-scale independently**
- **Load balancing** yang mudah diimplementasikan
- **Resource allocation** yang optimal

### 3. **Development Velocity**
- **Testing** yang comprehensive dengan data realistic
- **Debugging** yang lebih mudah dengan observability
- **Deployment** yang terisolasi per service

### 4. **Risk Mitigation**
- **Fraud detection** sebelum transaksi diproses
- **Validation** multi-layer
- **Fallback mechanisms** yang dapat diimplementasikan

## 🚀 **PoC sebagai Foundation**

Meskipun ini masih **Proof of Concept**, architecture ini memberikan:

1. **Blueprint** untuk system production-ready
2. **Patterns** yang dapat di-extend (database, auth, etc.)
3. **Testing framework** yang comprehensive
4. **Observability foundation** yang solid
5. **Documentation** dan examples untuk development selanjutnya

**Kesimpulan:** PoC ini memecahkan masalah kompleksitas payment system multi-currency dengan approach microservices yang observable, testable, dan scalable! 🎉  
# Payment Gateway Architecture Overview

## 1. System Architecture Diagram

```mermaid
flowchart TD
    A[Browser/Client] -->|HTTP Requests| G(API Gateway<br>:8080)
    
    G --> P(Payments Service<br>:8081)
    G --> F(FX Service<br>:8082)
    G --> W(Wallet Service<br>:8083)
    G --> R(Risk Service<br>:8084)
    
    subgraph Observability [Observability Stack]
        PR(Prometheus<br>:9090)
        GR(Grafana<br>:3000)
    end

    P -.->|Scrape /metrics| PR
    F -.->|Scrape /metrics| PR
    W -.->|Scrape /metrics| PR
    R -.->|Scrape /metrics| PR
    G -.->|Scrape /metrics| PR

    GR -->|Query Data| PR
    A -->|Access Dashboard| GR
    
    style Observability fill:#f9f,stroke:#333,stroke-width:2px
```

## 2. Payment Processing Sequence Diagram

```mermaid
sequenceDiagram
    participant C as Client
    participant G as API Gateway
    participant P as Payments Service
    participant R as Risk Service
    participant F as FX Service
    participant W as Wallet Service

    C->>G: POST /payments {payment_data}
    G->>P: Forward payment request
    P->>R: Check risk score
    R-->>P: Return risk assessment
    P->>F: Convert currency (if needed)
    F-->>P: Return conversion rate
    P->>W: Verify balance
    W-->>P: Return balance status
    
    alt Risk is acceptable & balance sufficient
        P-->>G: Payment processed successfully
        G-->>C: 200 OK {status: "CAPTURED"}
    else Risk too high or insufficient balance
        P-->>G: Payment rejected
        G-->>C: 400 Bad Request {status: "FAILED"}
    end
    
    Note right of P: Metrics sent to<br>Prometheus throughout
```

## 3. Service Interaction Details

### API Gateway (:8080)
- Single entry point for all incoming requests
- Routes requests to appropriate microservices
- Serves static content (including embedded Grafana)
- Health check endpoint: `/healthz`
- Metrics endpoint: `/metrics`

### Payments Service (:8081)
- Core payment processing logic
- Coordinates with other services for risk, FX, and wallet checks
- Endpoint: `POST /payments`
- Returns payment status: PENDING, CAPTURED, or FAILED

### FX Service (:8082)
- Handles currency exchange rates and conversions
- Endpoints:
  - `GET /rate?base=USD&quote=IDR`
  - `GET /convert?from=USD&to=IDR&amount=100`

### Wallet Service (:8083)
- Manages account balances and transactions
- Endpoint: `GET /balance/{account_id}`
- Validates sufficient funds for transactions

### Risk Service (:8084)
- Assesses transaction risk and fraud potential
- Endpoint: `POST /score`
- Returns risk score based on transaction details

## 4. Observability Stack

### Prometheus (:9090)
- Collects metrics from all services every 15 seconds
- Stores time-series data for monitoring
- Provides query language for data analysis
- Accessible at: http://localhost:9090

### Grafana (:3000)
- Visualizes metrics from Prometheus
- Pre-configured with payment gateway dashboard
- Embedded in API Gateway frontend
- Anonymous access enabled for demo purposes
- Accessible at: http://localhost:3000

## 5. Data Flow

1. **Client** makes HTTP request to **API Gateway**
2. **API Gateway** routes to appropriate microservice
3. **Microservices** process request and communicate if needed
4. **Services** expose metrics at `/metrics` endpoints
5. **Prometheus** scrapes metrics from all services
6. **Grafana** queries Prometheus for visualization
7. **Client** can view real-time metrics via Grafana UI

This architecture provides a scalable, observable foundation for payment processing with clear separation of concerns and comprehensive monitoring capabilities.
---

## Struktur Repo

```
payment-gateway-poc/
├─ README.md
├─ Makefile
├─ go.mod / go.sum
│
├─ pkg/
│  ├─ metrics/                 # Prometheus metrics (Counter/Histogram)
│  ├─ proto/                   # *.proto (opsional, placeholder)
│  ├─ auth/ tracing/ errors/   # placeholder libs
│
├─ services/
│  ├─ api-gateway/
│  │  ├─ main.go
│  │  └─ static/index.html     # frontend minimal + embed Grafana
│  ├─ payments/main.go
│  ├─ fx/main.go
│  ├─ wallet/main.go
│  └─ risk/main.go
│
├─ tools/
│  └─ cmd/dummygen/main.go     # generator CSV dummy (-n)
│
├─ tests/
│  ├─ data/dummy_transactions.csv
│  ├─ e2e/payment_load.js
│  └─ e2e/payment_load_from_csv.js
│
├─ deployments/
│  ├─ docker/Dockerfile
│  └─ compose/docker-compose.dev.yaml
│
├─ grafana/
│  ├─ dashboards/payment.json  # grafana_payment_gateway_dashboard*.json
│  └─ provisioning/
│     ├─ dashboards/provider.yaml
│     └─ datasources/prometheus.yaml
│
└─ prometheus/prometheus.yml
```

---

## Prasyarat

* **Docker** & **Docker Compose v2**
* (Opsional) **Go** ≥ 1.22 bila ingin menjalankan `go` secara lokal.
  Tanpa Go lokal pun, semua build/test bisa dijalankan via Docker.

---

## Quick Start (Dev)

Jalankan seluruh stack:

```bash
make dev
# atau manual:
docker compose -f deployments/compose/docker-compose.dev.yaml up -d --build
```

Akses cepat:

* **Frontend (embed Grafana)** → [http://localhost:8080](http://localhost:8080)
* **Grafana** → [http://localhost:3000](http://localhost:3000) (anonymous viewer aktif; embedding diizinkan)
* **Prometheus** → [http://localhost:9090](http://localhost:9090)
* **Services**:

  * Payments → [http://localhost:8081](http://localhost:8081)
  * FX       → [http://localhost:8082](http://localhost:8082)
  * Wallet   → [http://localhost:8083](http://localhost:8083)
  * Risk     → [http://localhost:8084](http://localhost:8084)

Hentikan stack:

```bash
make down
```

Lihat log / status:

```bash
make logs
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

* **Prometheus** → *Status → Targets* harus **UP**.
* **Grafana**:

  * Dashboard diprovision dari `grafana/dashboards/payment.json` (UID contoh: `paygw-poc`).
  * Contoh URL embed (di `services/api-gateway/static/index.html`):

    ```
    http://localhost:3000/d/paygw-poc?orgId=1&kiosk
    ```
  * Panel utama:

    * **Request rate** per service
    * **Error rate** (5m) per service
    * **p95/p99 latency** per service
    * **Requests in last 5m** per service
  * Contoh PromQL:

    ```promql
    sum by (service) (rate(payment_requests_total[1m]))
    100 * (sum by (service) (increase(payment_requests_total{status="FAILED"}[5m])) / sum by (service) (increase(payment_requests_total[5m])))
    histogram_quantile(0.95, sum by (service, le) (rate(payment_request_duration_seconds_bucket[5m])))
    histogram_quantile(0.99, sum by (service, le) (rate(payment_request_duration_seconds_bucket[5m])))
    ```

---

## Dummy Data (CSV)

Generator: `tools/cmd/dummygen` → output default `tests/data/dummy_transactions.csv`.

**Via Docker (tanpa Go lokal):**

```bash
make dummy-docker            # 100 baris (default)
make dummy-docker N=1000     # 1000 baris
```

Atau langsung:

```bash
docker run --rm -v "$(pwd)":/app -w /app \
  --entrypoint /usr/local/go/bin/go golang:1.22 \
  run ./tools/cmd/dummygen -n 1000
```

Verifikasi:

```bash
wc -l tests/data/dummy_transactions.csv
# 1001 (1 header + 1000 data)
```

---

## Testing

### End-to-End (k6)

Contoh **CSV load** (parametrik, aman untuk Docker):

```bash
# lewat network host (Docker Desktop/WSL)
docker run --rm -it \
  -v "$PWD:/work" -w /work grafana/k6 run \
  -e PAYMENTS=http://host.docker.internal:8081 \
  -e CSV_PATH=./tests/data/dummy_transactions.csv \
  tests/e2e/payment_load_from_csv.js
```

Atau **join network compose** (akses service via nama container):

```bash
docker run --rm -it \
  --network compose_default \
  -v "$PWD:/work" -w /work grafana/k6 run \
  -e PAYMENTS=http://payments:8081 \
  -e CSV_PATH=../data/dummy_transactions.csv \
  tests/e2e/payment_load_from_csv.js
```

### Integration test (Go)

```bash
make test-integration-docker     # Dockerized
# atau
make test                         # butuh Go lokal
```

---

## Makefile – Target Penting

| Target                         | Deskripsi                                |
| ------------------------------ | ---------------------------------------- |
| `make dev`                     | Up stack dev (Compose, build bila perlu) |
| `make down`                    | Stop & remove containers                 |
| `make logs`                    | Tail logs semua service                  |
| `make ps`                      | Status container                         |
| `make dummy-docker N=1000`     | Generate CSV dummy via Docker            |
| `make test-integration-docker` | Integration test (Dockerized)            |
| `make test-docker`             | Semua test (Dockerized)                  |
| `make test`                    | Semua test (Go lokal)                    |
| `make build`                   | Build semua paket                        |

---

## Endpoints

| Service     | Port | Healthz    | Metrics    | Catatan                         |
| ----------- | ---- | ---------- | ---------- | ------------------------------- |
| api-gateway | 8080 | `/healthz` | `/metrics` | serve `static/` + embed Grafana |
| payments    | 8081 | `/healthz` | `/metrics` | `POST /payments`                |
| fx          | 8082 | `/healthz` | `/metrics` | `/rate`, `/convert`             |
| wallet      | 8083 | `/healthz` | `/metrics` | `/balance/{id}`                 |
| risk        | 8084 | `/healthz` | `/metrics` | `POST /score`                   |
| prometheus  | 9090 | –          | –          | UI & query                      |
| grafana     | 3000 | –          | –          | Anonymous + embedding           |

---

## Development Notes

### Hot-reload static frontend

Aktifkan **bind-mount** agar ubah `index.html` langsung tersaji:

```yaml
# deployments/compose/docker-compose.dev.yaml (service api-gateway)
volumes:
  - ../../services/api-gateway/static:/app/static:ro
```

### Bila Go lokal belum terpasang

Jalankan perintah `go` via Docker:

```bash
docker run --rm -v "$(pwd)":/app -w /app \
  --entrypoint /usr/local/go/bin/go golang:1.22 <COMMAND>
# contoh:
# ... go test ./... -v
# ... go run ./tools/cmd/dummygen -n 1000
```

---

## Troubleshooting

* **Makefile: `missing separator`** → baris resep wajib **TAB**, bukan spasi.
* **`go: not found`** → pakai target Dockerized (`make test-docker`) atau install Go.
* **`illegal character U+0023 '#'` saat `go test ./...`** → pastikan `deployments/docker/Dockerfile` **bukan** `Dockerfile.go`.
* **Grafana iframe “Page not found”** → gunakan URL berbasis **UID** (mis. `paygw-poc`), pastikan provision & mount folder-to-folder.
* **Datasource duplicate default** → hanya **satu** `isDefault: true` di `grafana/provisioning/datasources/*.yaml`.
* **Grafana gagal start karena mount** → hindari file-to-file; pakai **folder-to-folder**:

  ```yaml
  - ../../grafana/dashboards:/var/lib/grafana/dashboards:ro
  - ../../grafana/provisioning:/etc/grafana/provisioning:ro
  ```
* **Error rate tinggi** → cek handler `payments` & variabel `FAIL_RATE`; error dihitung dari **HTTP status** di middleware. Set `FAIL_RATE=0.0` bila ingin nol.

---

## Security & Production Gaps

* Tidak ada DB/persistence.
* Auth/authorization dummy.
* Tidak ada rate limiting, circuit breaker, retry, tracing lengkap.
* TLS, secret management, dan hardening container belum disiapkan.

> Untuk produksi: siapkan penyimpanan transaksi, idempotency key, retry-safe queue, audit log, tracing E2E, *blue/green* deployment, dsb.

---


