// services/risk-gprc/main.go

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
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
		grpc.StreamInterceptor(gp.StreamServerInterceptor),
	)
	gp.Register(grpcServer)

	lis, err := net.Listen("tcp", ":9094")
	if err != nil {
		log.Fatalf("[risk-grpc] listen error: %v", err)
	}
	go func() {
		log.Println("[risk-grpc] serving gRPC on :9094")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[risk-grpc] serve error: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Println("[risk-grpc] serving metrics on :9104 /metrics")
	log.Fatal(http.ListenAndServe(":9104", nil))
}
