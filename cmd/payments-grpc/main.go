// cmd/payments-grpc/main.go


// cmd/payments-grpc/main.go
package main

import (
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"
)

func main() {
  // gRPC server + Prometheus interceptors
  grpcServer := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )
  gp.Register(grpcServer)

  // Listen gRPC
  lis, err := net.Listen("tcp", ":9091")
  if err != nil {
    log.Fatalf("[payments-grpc] listen error: %v", err)
  }

  // Jalankan gRPC di goroutine
  go func() {
    log.Println("[payments-grpc] serving gRPC on :9091")
    if err := grpcServer.Serve(lis); err != nil {
      log.Fatalf("[payments-grpc] serve error: %v", err)
    }
  }()

  // Endpoint Prometheus metrics (HTTP terpisah)
  http.Handle("/metrics", promhttp.Handler())
  log.Println("[payments-grpc] serving metrics on :9101 /metrics")
  log.Fatal(http.ListenAndServe(":9101", nil))
}
