// cmd/fx-grpc/main.go

package main

import (
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"

  fxv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/fx/v1"
  "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/internal/grpcserver"
)

func main() {
  // gRPC server + Prometheus interceptors
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )
  gp.Register(grpcSrv)

  // Register service
  fxv1.RegisterFxServiceServer(grpcSrv, &grpcserver.FxServer{})

  // gRPC listener
  go func() {
    lis, err := net.Listen("tcp", ":9092")
    if err != nil {
      log.Fatal(err)
    }
    log.Println("fx gRPC :9092")
    log.Fatal(grpcSrv.Serve(lis))
  }()

  // HTTP listener untuk /metrics
  http.Handle("/metrics", promhttp.Handler())
  log.Println("fx metrics :9102/metrics")
  log.Fatal(http.ListenAndServe(":9102", nil))
}
