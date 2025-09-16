// cmd/wallet-grpc/main.go
package main

import (
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  walletv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/wallet/v1"
  "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/internal/grpcserver"
)

func main() {
  // gRPC server + Prometheus interceptors
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )
  gp.Register(grpcSrv) // register default gRPC metrics

  // Register service
  walletv1.RegisterWalletServiceServer(grpcSrv, &grpcserver.WalletServer{})

  // gRPC listener (business)
  go func() {
    lis, err := net.Listen("tcp", ":9093")
    if err != nil {
      log.Fatal(err)
    }
    log.Println("wallet gRPC :9093")
    log.Fatal(grpcSrv.Serve(lis))
  }()

  // HTTP listener untuk /metrics
  http.Handle("/metrics", promhttp.Handler())
  log.Println("wallet metrics :9103/metrics")
  log.Fatal(http.ListenAndServe(":9103", nil))
}
