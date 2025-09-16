// cmd/payments-grpc/main.go

package main

import (
  "log"
  "net"
  "net/http"

  gp "github.com/grpc-ecosystem/go-grpc-prometheus"
  "github.com/prometheus/client_golang/prometheus/promhttp"
  "google.golang.org/grpc"
  "google.golang.org/grpc/credentials/insecure"

  paymentsv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/payments/v1"
  riskv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/risk/v1"
  walletv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/wallet/v1"
  fxv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/fx/v1"
  "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/internal/grpcserver"
)

func main() {
  // koneksi ke service lain
  riskConn, _ := grpc.Dial("localhost:9094", grpc.WithTransportCredentials(insecure.NewCredentials()))
  walConn, _ := grpc.Dial("localhost:9093", grpc.WithTransportCredentials(insecure.NewCredentials()))
  fxConn, _ := grpc.Dial("localhost:9092", grpc.WithTransportCredentials(insecure.NewCredentials()))

  // gRPC server dengan prometheus interceptors
  grpcSrv := grpc.NewServer(
    grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
    grpc.StreamInterceptor(gp.StreamServerInterceptor),
  )
  gp.Register(grpcSrv)

  // register service Payments
  paymentsv1.RegisterPaymentsServiceServer(grpcSrv, &grpcserver.PaymentsServer{
    Risk:   riskv1.NewRiskServiceClient(riskConn),
    Wallet: walletv1.NewWalletServiceClient(walConn),
    Fx:     fxv1.NewFxServiceClient(fxConn),
  })

  // listener gRPC
  go func() {
    lis, err := net.Listen("tcp", ":9091")
    if err != nil {
      log.Fatal(err)
    }
    log.Println("payments gRPC :9091")
    log.Fatal(grpcSrv.Serve(lis))
  }()

  // listener HTTP untuk /metrics
  http.Handle("/metrics", promhttp.Handler())
  log.Println("payments metrics :9101/metrics")
  log.Fatal(http.ListenAndServe(":9101", nil))
}
