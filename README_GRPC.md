# gRPC Enablement (Drop-in)

**What you got here**
- `.proto` contracts for Payments, Risk, Wallet, FX (+ common types)
- Minimal Go gRPC servers (stubs) for each service
- Ready to generate code with `protoc`

## How to integrate into your repo
1. Copy all folders into the repo root.
2. Install codegen:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```
3. Generate stubs:
   ```bash
   protoc -I.      --go_out=. --go_opt=paths=source_relative      --go-grpc_out=. --go-grpc_opt=paths=source_relative      proto/common/v1/common.proto      proto/risk/v1/risk.proto      proto/wallet/v1/wallet.proto      proto/fx/v1/fx.proto      proto/payments/v1/payments.proto
   ```
4. Run servers (separate terminals):
   ```bash
   go run ./cmd/risk-grpc
   go run ./cmd/wallet-grpc
   go run ./cmd/fx-grpc
   go run ./cmd/payments-grpc
   ```
5. Test with grpcurl:
   ```bash
   grpcurl -plaintext localhost:9094 list
   grpcurl -plaintext -d '{"tx":{"tx_id":"TX-1"},"customer":{"customer_id":"C1"},"amount":{"currency":1,"amount":1200}}' localhost:9094 risk.v1.RiskService/Score
   grpcurl -plaintext -d '{"tx":{"tx_id":"TX-1"},"customer":{"customer_id":"C1"},"account":{"account_id":"ACC-1"},"amount":{"currency":1,"amount":10},"settlement_currency":2}' localhost:9091 payments.v1.PaymentsService/CreatePayment
   ```

> Optional: wire these into docker-compose as separate services or add grpc-gateway later.
