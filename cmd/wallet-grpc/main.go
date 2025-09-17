// cmd/wallet-grpc/main.go
package main

import (
  "context"
  "log"
  "net"
  "net/http"
  "sync"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
  "github.com/example/payment-gateway-poc/internal/grpcserver"
)

type accountStore struct {
  mu   sync.Mutex
  data map[string]*walletv1.AccountSeed
}

var store = &accountStore{data: map[string]*walletv1.AccountSeed{}}

type adminWalletServer struct {
  walletv1.UnimplementedAdminServer
}

func (s *adminWalletServer) SeedAccounts(ctx context.Context, req *walletv1.SeedAccountsRequest) (*walletv1.SeedAccountsResponse, error) {
  store.mu.Lock()
  defer store.mu.Unlock()
  var upserted uint32
  for _, a := range req.GetAccounts() {
    store.data[a.GetAccountId()] = a // idempotent upsert
    upserted++
  }
  return &walletv1.SeedAccountsResponse{Upserted: upserted}, nil
}

func registerAdmin(grpcServer *grpc.Server) {
  walletv1.RegisterAdminServer(grpcServer, &adminWalletServer{})
}

func main() {
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )

  // ==== REGISTRASI SERVICE UTAMA (pilih sesuai generate) ====
  // walletv1.RegisterWalletServer(grpcSrv, &grpcserver.WalletServer{})
  walletv1.RegisterWalletServiceServer(grpcSrv, &grpcserver.WalletServer{})

  // Admin/Seed
  registerAdmin(grpcSrv)

  // Metrics
  gp.Register(grpcSrv)

  // gRPC
  go func() {
    lis, err := net.Listen("tcp", ":9093")
    if err != nil {
      log.Fatalf("[wallet-grpc] listen :9093: %v", err)
    }
    log.Println("[wallet-grpc] serving gRPC on :9093")
    if err := grpcSrv.Serve(lis); err != nil {
      log.Fatalf("[wallet-grpc] serve: %v", err)
    }
  }()

  // Metrics HTTP
  mux := http.NewServeMux()
  mux.Handle("/metrics", promhttp.Handler())
  log.Println("[wallet-grpc] serving metrics on :9103 /metrics")
  if err := http.ListenAndServe(":9103", mux); err != nil && err != http.ErrServerClosed {
    log.Fatalf("[wallet-grpc] metrics: %v", err)
  }
}
