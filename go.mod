module github.com/example/payment-gateway-poc

go 1.22

require (
	// Web / HTTP
	github.com/gorilla/mux v1.8.1
	github.com/rs/cors v1.11.0

	// Kafka client
	github.com/segmentio/kafka-go v0.4.47

	// Postgres client
	github.com/jackc/pgx/v5 v5.5.5

	// UUID generator
	github.com/google/uuid v1.6.0

	// Prometheus
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/prometheus/client_golang v1.19.0

	// gRPC core
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.1
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.17.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.48.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
)
