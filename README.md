

# Real-Time Multi-Currency Payment Gateway (PoC)

Monorepo **Proof of Concept** untuk *real-time multi-currency payment gateway* berbasis **microservices + gRPC + Kafka** dengan orkestrasi di **API Gateway**, *observability* (Prometheus + Grafana), dan tooling untuk dummy data & testing.

> ⚠️ PoC untuk edukasi/demonstrasi. Belum siap produksi (auth/HA/hardening/db migration penuh, dsb).

---

## Arsitektur Ringkas

* **api-gateway** (HTTP): Validasi request + orchestrator:

  1. panggil **fx-grpc** (konversi USD/SGD → IDR),
  2. cek saldo **wallet-grpc** (fail-fast jika kurang),
  3. cek **risk-grpc** (fail-fast jika deny),
  4. publish ke **Kafka** `payments.request` dan tunggu hasil di `payments.result`.
* **payments-worker** (consumer): ambil pesan Kafka → panggil **payments-rs** (`LogAndSettle`).
* **payments-rs**: bisnis proses pembayaran → panggil **db-rs** untuk **Reserve → Commit / Rollback**.
* **db-rs**: koneksi Postgres; transaksi atomik (lock, update saldo, `reservations`, `payments`).
* **Observability**: **Prometheus** (scrape) + **Grafana** (dashboard).
* **Kafka UI**: memantau topik/partisi traffic.

### Sequence (Mermaid)

```mermaid
sequenceDiagram
    title Payment Orchestration (Gateway + Kafka + Worker + DB)

    actor Client
    participant GW as API Gateway (HTTP)
    participant FX as fx-grpc
    participant WAL as wallet-grpc
    participant RISK as risk-grpc
    participant KREQ as Kafka payments.request
    participant KRES as Kafka payments.result
    participant WORK as payments-worker
    participant PAY as payments-rs (LogAndSettle)
    participant DB as db-rs (Reserve/Commit/Rollback)
    database PG as Postgres

    Client->>GW: POST /api/payments {sender, receiver, currency, amount, idempo}

    rect rgba(200,200,255,0.15)
      alt currency != IDR
        GW->>FX: Convert(USD/SGD → IDR)
        FX-->>GW: amount_idr
      else currency == IDR
        Note right of GW: amount_idr = amount
      end
      alt FX error
        GW-->>Client: {status:FAILED, reason:"fx_unavailable"}
        return
      end
    end

    rect rgba(200,255,200,0.15)
      GW->>WAL: GetAccount(sender_id)
      WAL-->>GW: balance_idr
      alt balance < amount_idr
        GW-->>Client: {status:FAILED, reason:"insufficient_funds"}
        return
      end
    end

    rect rgba(255,220,200,0.15)
      GW->>RISK: Evaluate(sender, receiver, amount_idr, currency, tx_date)
      RISK-->>GW: {allow:true|false, reason}
      alt allow == false
        GW-->>Client: {status:FAILED, reason:"risk_<reason>"}
        return
      end
    end

    rect rgba(235,235,235,0.5)
      GW->>KREQ: key=idempo\nvalue={sender,receiver,amount_idr,...}
      Note over GW: Wait result (≤5s)
      KREQ-->>WORK: consume
      WORK->>PAY: LogAndSettle(request)
      activate PAY

      %% Reserve
      PAY->>DB: ReserveFunds(idempo, sender, receiver, amount_idr, currency)
      DB->>PG: BEGIN; lock sender; check; deduct; insert reservation(PENDING); COMMIT
      DB-->>PAY: {status:OK|INSUFFICIENT|DUPLICATE, reservation_id?}

      alt INSUFFICIENT
        PAY-->>WORK: FAILED(insufficient_funds)
      else DUPLICATE
        PAY-->>WORK: SUCCESS_REPLAY(duplicate_idempo)
      else OK
        %% Commit
        PAY->>DB: CommitReservation(reservation_id, idempo)
        DB->>PG: lock reservation; credit receiver;\nreservation→COMMITTED; payments(SUCCESS); COMMIT
        DB-->>PAY: {status:OK|NOT_FOUND|BAD_STATUS}

        alt OK
          PAY-->>WORK: SUCCESS(committed, ref=reservation_id)
        else NOT_FOUND or BAD_STATUS
          %% Rollback safety
          PAY->>DB: RollbackReservation(reservation_id, "commit_failed")
          DB->>PG: credit back sender;\nreservation→ROLLEDBACK; payments(FAILED); COMMIT
          PAY-->>WORK: FAILED(commit_failed)
        end
      end
      deactivate PAY

      WORK->>KRES: key=idempo\nvalue={status, reason, ref?}
      KRES-->>GW: result match by idempo

      alt timeout / no result
        GW-->>Client: {status:FAILED, reason:"queue_timeout"}
      else got result
        GW-->>Client: {status, reason, ref?}
      end
    end

    Note over DB,PG: Idempotency:\n- reservations.idempotency_key (reserve)\n- payments.idempotency_key (commit replay)
```

---

## Struktur Repo (bagian relevan)

```
services/
├─ api-gateway/
│  ├─ handlers/
│  │  ├─ payments.go        # FX → Wallet → Risk → Kafka publish/wait
│  │  └─ types.go           # JSON request/response
│  ├─ queue/kafka.go        # Publish()/WaitResult()
│  ├─ client/grpc_clients.go# init Fx/Wallet/Risk/Payments clients
│  └─ static/index.html     # UI PoC (form + iframe Grafana)
├─ payments-worker/
│  └─ main.go               # Consume Kafka → call payments-rs → publish result
├─ payments-rs/
│  └─ src/main.rs           # LogAndSettle → call db-rs (Reserve/Commit/Rollback)
├─ db-rs/
│  └─ src/
│     ├─ handlers.rs        # gRPC handlers (Reserve/Commit/Rollback)
│     ├─ store.rs           # SQL logic (FOR UPDATE, insert/update)
│     └─ schema.sql         # DDL: reservations, payments, (wallet_accounts)
├─ fx-grpc/ (cmd/fx-grpc)   # FX convert gRPC
├─ wallet (cmd/wallet-grpc) # Wallet gRPC (GetAccount, dst)
└─ risk-grpc/
   ├─ main.go               # gRPC server
   └─ service.go            # Rules: <1000 or >10_000_000 reject; 2% random reject
```

Kontrak proto tersedia di `proto/gen/{fx,wallet,risk,payments,db}/v1`.

---

## Quick Start (Dev – Docker Compose)

Jalankan seluruh stack (v2):

```bash
docker compose -f deployments/compose/docker-compose.dev.v2.yaml up -d --build
```

**Akses:**

| Komponen    | URL (host)                                       | Catatan                                      |
| ----------- | ------------------------------------------------ | -------------------------------------------- |
| API Gateway | [http://localhost:18080](http://localhost:18080) | UI PoC + `/metrics`                          |
| Grafana     | [http://localhost:3000](http://localhost:3000)   | Dashboard & Explore                          |
| Prometheus  | [http://localhost:19097](http://localhost:19097) | Scrape status/queries                        |
| Kafka UI    | [http://localhost:9081](http://localhost:9081)   | Lihat `payments.request` & `payments.result` |

> gRPC port lain ikut dipublish (mis. `payments-rs` di `19096`), tapi tidak perlu diakses langsung untuk alur normal.

Hentikan:

```bash
docker compose -f deployments/compose/docker-compose.dev.v2.yaml down -v
```

---

## Smoke Test

Kirim 1 transaksi (USD 10 → otomatis dikonversi ke IDR oleh FX):

```bash
curl -s -XPOST http://localhost:18080/api/payments \
  -H 'Content-Type: application/json' \
  -d '{
    "sender_id":"ACC001",
    "receiver_id":"ACC002",
    "currency":"USD",
    "amount":10,
    "tx_date":"2025-09-17T03:00:00Z",
    "idempotency_key":"test-001"
  }' | jq
```

Hasil tipikal:

```json
{ "status":"SUCCESS", "reason":"committed", "ref":"<reservation_uuid>" }
```

Atau:

* `FAILED / insufficient_funds` (saldo kurang),
* `FAILED / risk_<reason>` (ditolak risk),
* `SUCCESS_REPLAY` (idempotency key sama),
* `FAILED / queue_timeout` (worker lambat/tidak jalan).

Buka **Kafka UI** ([http://localhost:9081](http://localhost:9081)) untuk melihat pesan di topic `payments.request` dan `payments.result`.

---

## Detail Proses (5 Tahap)

1. **FX Convert**
   Gateway panggil **fx-grpc** jika `currency ∈ {USD, SGD}` → dapat `amount_idr`.

2. **Wallet Check**
   Gateway panggil **wallet-grpc\:GetAccount(sender\_id)** → jika `balance_idr < amount_idr` → **FAILED/insufficient\_funds** (fail-fast).

3. **Risk Check**
   Gateway panggil **risk-grpc\:Evaluate** → jika `Allow=false` → **FAILED/risk\_<reason>** (fail-fast).
   *Rule default PoC*: `< 1,000` atau `> 10,000,000` ditolak; selain itu 2% acak ditolak.

4. **Publish & Wait (Kafka)**
   Gateway publish ke **`payments.request`** (key = `idempotency_key`) dan **menunggu ≤5s** di **`payments.result`**.
   Worker konsumsi request → panggil **payments-rs.LogAndSettle**.

5. **Process Payment di Backend**
   **payments-rs** → **db-rs**:

   * **Reserve**: lock sender, cek saldo lagi (defense-in-depth), **deduct**, insert `reservations(PENDING)`.
     Jika **duplicate** idempo → **SUCCESS\_REPLAY**. Jika kurang → **FAILED/insufficient\_funds**.
   * **Commit**: lock reservation, **credit receiver**, `reservations→COMMITTED`, insert `payments(SUCCESS)`.
     Jika commit gagal → **Rollback**: credit balik sender, `reservations→ROLLEDBACK`, `payments(FAILED)`.

**Tabel**:

* `wallet_accounts(account_id, balance_idr, updated_at)`
* `reservations(reservation_id, idempotency_key, sender_id, receiver_id, amount_idr, currency_input, status)`
* `payments(payment_id, idempotency_key, sender_id, receiver_id, currency_input, amount_idr, status)`

---

## Endpoints (Dev)

| Service     | Host Port | Path                   | Catatan                              |
| ----------- | --------- | ---------------------- | ------------------------------------ |
| api-gateway | 18080     | `/api/payments` (POST) | JSON in/out                          |
|             |           | `/api/random-accounts` | Dummy helper                         |
|             |           | `/metrics`, `/healthz` | Prometheus/health                    |
| fx-grpc     | 19102     | gRPC                   | Dipanggil oleh gateway               |
| wallet-grpc | 19093     | gRPC                   | Dipanggil oleh gateway (GetAccount)  |
| risk-grpc   | 19094     | gRPC                   | Dipanggil oleh gateway (Evaluate)    |
| payments-rs | 19096     | gRPC                   | Dipanggil oleh worker (LogAndSettle) |
| db-rs       | 19095     | gRPC                   | Dipanggil oleh payments-rs           |
| Prometheus  | 19097     | Web UI                 | Scrape metrics dari semua service    |
| Grafana     | 3000      | Web UI                 | Dashboard (provisioned)              |
| Kafka UI    | 9081      | Web UI                 | Observasi topik Kafka                |

> Port dapat berbeda jika kamu mengubah `docker-compose.dev.v2.yaml`.

---

## Konfigurasi & ENV Penting

* **api-gateway**:
  `FX_ADDR`, `WALLET_ADDR`, `RISK_ADDR`, `PAYMENTS_ADDR`, `KAFKA_BROKERS`, `KAFKA_REQ_TOPIC`, `KAFKA_RES_TOPIC`
* **payments-worker**:
  `KAFKA_BROKERS`, `KAFKA_REQ_TOPIC`, `KAFKA_RES_TOPIC`, `PAYMENTS_ADDR`
* **payments-rs**:
  `DB_ADDR` (alamat gRPC db-rs)
* **db-rs**:
  `DATABASE_URL` (Postgres)
* **risk-grpc** (opsional):
  rules via ENV jika ditambahkan (PoC default hard-coded)

Semua sudah di-wire di `deployments/compose/docker-compose.dev.v2.yaml`.

---

## Observability

* **Prometheus**: `Status → Targets` harus `UP`.
* **Grafana**: dashboards auto-provisioned (lihat folder `grafana/`).
* **Metrics contoh**:

  * Gateway: `pgw_fx_calls_total`, `pgw_wallet_check_total`, `pgw_risk_total`, `pgw_kafka_publish_total`, `pgw_kafka_wait_result_total`, latensi per tahap.
  * Worker/Payments/DB: jumlah reserve/commit/rollback, latensi RPC, hasil sukses/gagal.

---

## Testing & Data Dummy

* Dummy wallets / rates / rules di `seeds/`.
* Integration test contoh di `tests/`.
* Tools generator dummy di `tools/`.

---

## Troubleshooting

* **`queue_timeout`**: worker tidak consume / lambat. Cek `payments-worker` + `payments-rs` logs; pastikan `payments.result` punya message untuk key yang sama.
* **FX atau Risk unavailable**: pastikan servisnya `UP`.
* **Saldo kurang**: wajar—coba ganti akun (`/api/random-accounts` di UI) atau turunkan nominal.
* **Idempotency**: gunakan `idempotency_key` sama → harus dapat `SUCCESS_REPLAY`.

---

## Lisensi

MIT

---

