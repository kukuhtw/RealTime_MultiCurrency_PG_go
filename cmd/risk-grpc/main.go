// cmd/risk-grpc/main.go

// cmd/risk-grpc/main.go
package main

import (
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  riskv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/risk/v1"
  "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/internal/grpcserver"
)

func main() {
  // gRPC server + Prometheus interceptors (RPC metrics)
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )
  gp.Register(grpcSrv) // register default gRPC metrics

  // Register Risk service implementation
  riskv1.RegisterRiskServiceServer(grpcSrv, &grpcserver.RiskServer{})

  // Start gRPC listener (business traffic)
  go func() {
    lis, err := net.Listen("tcp", ":9094")
    if err != nil {
      log.Fatal(err)
    }
    log.Println("risk gRPC :9094")
    log.Fatal(grpcSrv.Serve(lis))
  }()

  // Start HTTP endpoint for Prometheus scrape
  http.Handle("/metrics", promhttp.Handler())
  log.Println("risk metrics :9104/metrics")
  log.Fatal(http.ListenAndServe(":9104", nil))
}
