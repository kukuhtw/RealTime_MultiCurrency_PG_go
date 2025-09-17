// cmd/fx-grpc/main.go
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

  fxv1 "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
  "github.com/example/payment-gateway-poc/internal/grpcserver"
)

// ===== In-memory FX rates store (untuk PoC) =====
type rateStore struct {
  mu   sync.Mutex
  data map[string]*fxv1.RateSeed
}

var fxStore = &rateStore{data: map[string]*fxv1.RateSeed{}}

// ===== Admin service (seeding) =====
type adminFxServer struct {
  fxv1.UnimplementedAdminServer
}

func (s *adminFxServer) SeedRates(ctx context.Context, req *fxv1.SeedRatesRequest) (*fxv1.SeedRatesResponse, error) {
  fxStore.mu.Lock()
  defer fxStore.mu.Unlock()
  var upserted uint32
  for _, r := range req.GetRates() {
    fxStore.data[r.GetPair()] = r
    upserted++
  }
  return &fxv1.SeedRatesResponse{Upserted: upserted}, nil
}

func registerAdmin(grpcServer *grpc.Server) {
    fxv1.RegisterAdminServer(grpcServer, &adminFxServer{})
}



func main() {
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )

  // ==== REGISTRASI SERVICE UTAMA (pilih salah satu sesuai kode generate) ====
  // 1) Jika service di proto bernama `service FX { ... }`
  // fxv1.RegisterFXServer(grpcSrv, &grpcserver.FxServer{})
  // 2) Jika service di proto bernama `service FxService { ... }`
  fxv1.RegisterFxServiceServer(grpcSrv, &grpcserver.FxServer{})

  // Admin/Seed
  registerAdmin(grpcSrv)

  // Daftarkan metrics
  gp.Register(grpcSrv)

  // gRPC listener
  go func() {
    lis, err := net.Listen("tcp", ":9092")
    if err != nil {
      log.Fatalf("[fx-grpc] listen :9092: %v", err)
    }
    log.Println("[fx-grpc] serving gRPC on :9092")
    if err := grpcSrv.Serve(lis); err != nil {
      log.Fatalf("[fx-grpc] serve error: %v", err)
    }
  }()

  // HTTP /metrics
  mux := http.NewServeMux()
  mux.Handle("/metrics", promhttp.Handler())
  log.Println("[fx-grpc] serving metrics on :9102 /metrics")
  if err := http.ListenAndServe(":9102", mux); err != nil && err != http.ErrServerClosed {
    log.Fatalf("[fx-grpc] metrics server error: %v", err)
  }
}
