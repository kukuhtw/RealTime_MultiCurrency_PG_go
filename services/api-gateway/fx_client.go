// // services/api-gateway/fx_client.go

package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	pb "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
	"google.golang.org/grpc"
)

func updateRates() {
	conn, err := grpc.Dial("fx-grpc:9102", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFxServiceClient(conn)

	rand.Seed(time.Now().UnixNano())

	// random kurs
	usdIdr := float64(rand.Intn(15700-15200) + 15200) // 15200–15700
	sgdIdr := float64(rand.Intn(11600-11200) + 11200) // 11200–11600

	req := &pb.UpdateRatesRequest{
		Rates: []*pb.Rate{
			{BaseCurrency: "USD", QuoteCurrency: "IDR", Rate: usdIdr},
			{BaseCurrency: "SGD", QuoteCurrency: "IDR", Rate: sgdIdr},
			{BaseCurrency: "IDR", QuoteCurrency: "USD", Rate: 1 / usdIdr},
			{BaseCurrency: "IDR", QuoteCurrency: "SGD", Rate: 1 / sgdIdr},
			{BaseCurrency: "USD", QuoteCurrency: "SGD", Rate: usdIdr / sgdIdr},
			{BaseCurrency: "SGD", QuoteCurrency: "USD", Rate: sgdIdr / usdIdr},
		},
	}

	resp, err := client.UpdateRates(context.Background(), req)
	if err != nil {
		log.Fatalf("could not update rates: %v", err)
	}
	log.Printf("FX update response: %v", resp.Message)
}
