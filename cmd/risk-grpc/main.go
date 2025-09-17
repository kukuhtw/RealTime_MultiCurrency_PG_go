// cmd/risk-grpc/main.go
package main

import (
  "context"
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  riskv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
  "github.com/example/payment-gateway-poc/internal/grpcserver"
)

type rulesConfig struct {
  MaxAmount        uint32
  VelocityPerMin   uint32
  BlockedAccounts  map[string]struct{}
  BlockedCountries map[string]struct{}
}

var rules = &rulesConfig{
  BlockedAccounts:  map[string]struct{}{},
  BlockedCountries: map[string]struct{}{},
}

type adminRiskServer struct {
  riskv1.UnimplementedAdminServer
}

func (s *adminRiskServer) SeedRules(ctx context.Context, req *riskv1.SeedRulesRequest) (*riskv1.SeedRulesResponse, error) {
  rules.MaxAmount = req.GetMaxAmount()
  rules.VelocityPerMin = req.GetVelocityPerMin()

  rules.BlockedAccounts = map[string]struct{}{}
  for _, a := range req.GetBlockedAccounts() {
    rules.BlockedAccounts[a] = struct{}{}
  }
  rules.BlockedCountries = map[string]struct{}{}
  for _, c := range req.GetBlockedCountries() {
    rules.BlockedCountries[c] = struct{}{}
  }
  return &riskv1.SeedRulesResponse{Ok: true}, nil
}

func registerAdmin(grpcServer *grpc.Server) {
  riskv1.RegisterAdminServer(grpcServer, &adminRiskServer{})
}

func main() {
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )

  // ==== REGISTRASI SERVICE UTAMA (pilih sesuai generate) ====
  // riskv1.RegisterRiskServer(grpcSrv, &grpcserver.RiskServer{})
  riskv1.RegisterRiskServiceServer(grpcSrv, &grpcserver.RiskServer{})

  // Admin/Seed
  registerAdmin(grpcSrv)

  // Metrics
  gp.Register(grpcSrv)

  // gRPC
  go func() {
    lis, err := net.Listen("tcp", ":9094")
    if err != nil {
      log.Fatalf("[risk-grpc] listen :9094: %v", err)
    }
    log.Println("[risk-grpc] serving gRPC on :9094")
    if err := grpcSrv.Serve(lis); err != nil {
      log.Fatalf("[risk-grpc] serve: %v", err)
    }
  }()

  // Metrics HTTP
  mux := http.NewServeMux()
  mux.Handle("/metrics", promhttp.Handler())
  log.Println("[risk-grpc] serving metrics on :9104 /metrics")
  if err := http.ListenAndServe(":9104", mux); err != nil && err != http.ErrServerClosed {
    log.Fatalf("[risk-grpc] metrics: %v", err)
  }
}
