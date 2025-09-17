// cmd/payments-grpc/main.go
package main

import (
  "context"
  "log"
  "net"
  "net/http"
  "os"
  "os/signal"
  "syscall"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  payv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
  "github.com/example/payment-gateway-poc/internal/grpcserver"
)

func main() {
  grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )

  // ==== REGISTRASI SERVICE UTAMA (pilih yang sesuai generate) ====
  // payv1.RegisterPaymentsServer(grpcServer, &grpcserver.PaymentsServer{})
  payv1.RegisterPaymentsServiceServer(grpcServer, &grpcserver.PaymentsServer{})

  // Default gRPC metrics
  gp.Register(grpcServer)

  lis, err := net.Listen("tcp", ":9091")
  if err != nil {
    log.Fatalf("[payments-grpc] listen :9091: %v", err)
  }
  go func() {
    log.Println("[payments-grpc] serving gRPC on :9091")
    if err := grpcServer.Serve(lis); err != nil {
      log.Fatalf("[payments-grpc] grpc serve: %v", err)
    }
  }()

  mux := http.NewServeMux()
  mux.Handle("/metrics", promhttp.Handler())
  metricsSrv := &http.Server{Addr: ":9101", Handler: mux}
  go func() {
    log.Println("[payments-grpc] serving metrics on :9101 /metrics")
    if err := metricsSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
      log.Fatalf("[payments-grpc] metrics serve: %v", err)
    }
  }()

  // Graceful shutdown
  sig := make(chan os.Signal, 1)
  signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
  <-sig
  log.Println("[payments-grpc] shutting down...")
  grpcServer.GracefulStop()
  _ = metricsSrv.Shutdown(context.Background())
  log.Println("[payments-grpc] bye")
}
