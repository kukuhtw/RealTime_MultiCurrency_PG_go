

# Real-Time Multi-Currency Payment Gateway (PoC)

```
==============================================================================
Project : Real-Time Multi-Currency Payment Gateway (PoC)
Version : 0.1.0
Author  : Kukuh Tripamungkas Wicaksono (Kukuh TW)
Email   : kukuhtw@gmail.com
WhatsApp: https://wa.me/628129893706
LinkedIn: https://id.linkedin.com/in/kukuhtw
License : MIT (see LICENSE)

Summary : Monorepo Proof of Concept untuk real-time multi-currency payment
          gateway berbasis microservices (API Gateway, Payments, FX, Wallet,
          Risk) dengan gRPC, observability (Prometheus + Grafana), serta
          tooling untuk dummy data dan testing.
==============================================================================
```

---

## 📖 Ringkasan

Proyek ini adalah **Proof of Concept (PoC)** untuk sistem **pembayaran lintas mata uang real-time** berbasis **microservices**.
Menggunakan kombinasi:

* **Golang** → layanan domain (Wallet, FX, Risk, Payments, API Gateway)
* **Rust** → layanan berperforma tinggi (Database handler, Payment Worker)
* **gRPC** → komunikasi antar service
* **Postgres** → database utama
* **Kafka** → message broker untuk event-driven payment worker
* **Prometheus + Grafana** → observability metrics & dashboard

Tujuan: memberikan **arsitektur modular, scalable, resilient** yang dapat dijadikan blueprint untuk sistem pembayaran modern.

---

## ⚙️ Fitur Utama

* **gRPC Microservices** untuk domain Wallet, FX, Risk, Payments.
* **Multi-currency FX Service** dengan dummy kurs USD, IDR, SGD.
* **Idempotency**: menghindari double spend/reservasi ganda.
* **Risk Service**: rule engine sederhana untuk fraud detection.
* **Async Worker (Rust)**: settlement via Kafka.
* **Observability**: Prometheus + Grafana dashboard siap pakai.
* **Testing Tools**: e2e tests, load tests, dummy data generator.

---

## 🏗️ Arsitektur

flowchart LR;

%% Clients
C1[Web / Mobile Client]:::client
G[API Gateway (Go)\nHTTP + gRPC]:::gw

%% Go Services
subgraph GO[Go Services]
  W[Wallet Svc]:::svc
  FX[FX Svc]:::svc
  R[Risk Svc]()


## 🔄 Sequence Diagram: MakePayment Flow

```mermaid
sequenceDiagram
  autonumber
  participant Client
  participant GW as API Gateway
  participant Pay as Payments Orchestrator
  participant Risk as Risk Svc
  participant FX as FX Svc
  participant Wal as Wallet Svc
  participant DB as DB Svc (Rust)
  participant K as Kafka
  participant Wrk as Payment Worker (Rust)
  participant PG as Postgres

  Client->>GW: MakePayment(req)
  GW->>Pay: gRPC MakePayment(req)
  Pay->>Risk: Check(txnCtx)
  Risk-->>Pay: ok
  Pay->>FX: Convert(USD->IDR)
  FX-->>Pay: rate + amount
  Pay->>DB: reserve_funds(idempotency_key)
  DB->>PG: INSERT reservation
  DB-->>Pay: Ok{reservation_id}
  Pay->>K: Produce "PAYMENT_RESERVED"
  Pay-->>GW: Accepted + reservation_id
  GW-->>Client: 202 Accepted

  Wrk->>K: Consume "PAYMENT_RESERVED"
  Wrk->>DB: commit_reservation()
  DB->>PG: update reservation + ledger
  par Balances
    Wrk->>Wal: Debit(sender)
    Wrk->>Wal: Credit(receiver)
  end
  Wrk->>K: Produce "PAYMENT_SETTLED"
  Client->>GW: GetStatus(reservation_id)
  GW->>Pay: GetStatus(reservation_id)
  Pay-->>GW: success
  GW-->>Client: 200 OK
```

---

## 📂 Struktur Direktori

Beberapa direktori penting:

* `cmd/` → entrypoint tiap service (wallet-grpc, payments-grpc, dll)
* `services/` → implementasi service (`api-gateway`, `db-rs`, `payments-rs`, dll)
* `proto/` → definisi protobuf
* `deployments/` → docker-compose, k8s manifest
* `grafana/` & `prometheus/` → observability setup
* `tests/` → e2e & load testing
* `tools/` → generator dummy data

---

## ⚙️ Setup Lingkungan

### Prasyarat

* Docker & Docker Compose
* Go 1.23+
* Rust (nightly, cargo, sqlx-cli)
* Protoc compiler
* Node.js (untuk e2e test)

### Jalankan Stack

```bash
# Clone repo
git clone https://github.com/your-org/realtime-payment-gateway.git
cd realtime-payment-gateway

# Generate dummy data
make gen-dummy

# Jalankan stack dengan Docker Compose
make dev-grpc

# Stop
make down-grpc
```

### Akses

* API Gateway → `http://localhost:8080`
* Prometheus → `http://localhost:9090`
* Grafana → `http://localhost:3000`

---

## 🔌 Endpoint gRPC

* **WalletService**: `GetBalance`, `Debit`, `Credit`
* **FXService**: `Convert(From, To, Amount)`
* **PaymentsService**: `MakePayment`, `GetStatus`
* **RiskService**: `Check(Transaction)`

---

## 📊 Monitoring

* Prometheus config → `prometheus/prometheus.yml`
* Grafana dashboard → `grafana/grafana_payment_gateway_dashboard.json`

---

## 🧪 Testing

### Go Integration Test

```bash
go test ./tests/integration/...
```

### Node.js Load Test

```bash
node tests/e2e/payment_load.js
```

### gRPC Test dari CSV

```bash
node tests/e2e/payment_grpc_from_csv.js
```

---

## 📌 Catatan

* Rust services dipakai untuk path kritikal performa tinggi.
* Go services dipakai untuk orchestrator & domain logic.
* PoC ini bisa jadi dasar implementasi production.

---

## 👨‍💻 Kontributor

* **Kukuh Tripamungkas Wicaksono (Kukuh TW)**

  * ✉️ Email: [kukuhtw@gmail.com](mailto:kukuhtw@gmail.com)
  * 💬 WhatsApp: [https://wa.me/628129893706](https://wa.me/628129893706)
  * 🔗 LinkedIn: [id.linkedin.com/in/kukuhtw](https://id.linkedin.com/in/kukuhtw)

---
