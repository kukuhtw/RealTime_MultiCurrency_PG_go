

# 🧩 gRPC Developer Guide — Payment Gateway PoC

Dokumen ini khusus untuk **developer** yang ingin mengembangkan dan menjalankan microservices berbasis **gRPC** pada repo `payment-gateway-poc`.

---

## 🔧 Build Protobuf & Stub

Semua kontrak API ada di folder `proto/`.

### Generate Go stubs

```bash
make proto-gen
```

Hasil generate akan muncul di folder sesuai package:

```
proto/payments/v1/payments.pb.go
proto/payments/v1/payments_grpc.pb.go
...
```

---

## 🚀 Jalankan gRPC Services

### Semua gRPC services via Docker Compose

```bash
make dev-grpc
```

### Hentikan semua gRPC services

```bash
make down-grpc
```

### Jalankan satu per satu (opsional)

```bash
# Jalankan payments-grpc
go run ./cmd/payments-grpc

# Jalankan wallet-grpc
go run ./cmd/wallet-grpc
```

---

## 📡 Ports & Metrics

| Service       | gRPC Port | Metrics Port |
| ------------- | --------- | ------------ |
| payments-grpc | 9091      | 9101         |
| fx-grpc       | 9092      | 9102         |
| wallet-grpc   | 9093      | 9103         |
| risk-grpc     | 9094      | 9104         |

---

## 🧪 Testing gRPC Services

### Manual via [grpcurl](https://github.com/fullstorydev/grpcurl)

#### Create Payment

```bash
grpcurl -plaintext -d '{
  "id": "PAY-123",
  "currency": "USD",
  "amount": 99.95,
  "source_account": "ACC_SRC_A",
  "destination_account": "ACC_DST_B"
}' localhost:9091 payments.v1.PaymentsService/CreatePayment
```

#### Score Risk

```bash
grpcurl -plaintext -d '{
  "tx_id": "PAY-123",
  "customer_id": "CUST-99",
  "amount": 250
}' localhost:9094 risk.v1.RiskService/Score
```

---

### Load Test dengan k6 (xk6-grpc)

#### Basic scenario

```bash
make e2e-grpc
```

#### Custom env vars

```bash
TX_ID=PAY-001 CUSTOMER_ID=CUST-1 AMOUNT=150 \
  make e2e-grpc
```

Script ada di:

```
tests/e2e/payment_grpc_test.js
tests/e2e/payment_grpc_param_test.js
```

---

## 🏗️ Struktur Terkait gRPC

```
payment-gateway-poc/
├─ cmd/
│   ├─ payments-grpc/       # Entrypoint Payments gRPC server
│   ├─ wallet-grpc/         # Entrypoint Wallet gRPC server
│   ├─ fx-grpc/             # Entrypoint FX gRPC server
│   └─ risk-grpc/           # Entrypoint Risk gRPC server
├─ proto/
│   ├─ common/v1/           # Shared messages
│   ├─ payments/v1/         # Payments proto
│   ├─ wallet/v1/           # Wallet proto
│   ├─ fx/v1/               # FX proto
│   └─ risk/v1/             # Risk proto
├─ internal/grpcserver/     # Server implementation
└─ tests/e2e/               # k6/xk6-grpc tests
```

---

## ⚠️ Catatan

* gRPC dipakai untuk **komunikasi antar service internal**
* REST endpoint (`/healthz`, `/metrics`) tetap ada untuk observability dan orchestration
* Protobuf contract adalah **source of truth** → setiap perubahan harus regenerate stub (`make proto-gen`)

