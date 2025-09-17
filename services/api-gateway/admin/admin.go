// services/api-gateway/admin/admin.go
package admin

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	fxv1 "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AdminServer struct {
	fxv1.UnimplementedFxAdminServiceServer
	fxAddr string
}

func NewAdminServer() *AdminServer {
	addr := os.Getenv("FX_ADDR")
	if addr == "" {
		// fallback: service name + port di docker compose
		addr = "fx-grpc:9102"
	}
	return &AdminServer{fxAddr: addr}
}

func (s *AdminServer) RefreshRandomRates(ctx context.Context, req *fxv1.RefreshRandomRatesRequest) (*fxv1.RefreshRandomRatesResponse, error) {
	// Seed RNG
	rand.Seed(time.Now().UnixNano())

	// 1) Generate random base rates
	usdIdr := float64(rand.Intn(15700-15200+1) + 15200) // inklusif
	sgdIdr := float64(rand.Intn(11600-11200+1) + 11200) // inklusif

	// 2) Turunan pasangan lain
	idrUsd := 1.0 / usdIdr
	idrSgd := 1.0 / sgdIdr
	usdSgd := usdIdr / sgdIdr
	sgdUsd := sgdIdr / usdIdr

	// 3) Susun payload ke FxService.UpdateRates
	outRates := []*fxv1.Rate{
		{BaseCurrency: "USD", QuoteCurrency: "IDR", Rate: usdIdr},
		{BaseCurrency: "SGD", QuoteCurrency: "IDR", Rate: sgdIdr},
		{BaseCurrency: "IDR", QuoteCurrency: "USD", Rate: idrUsd},
		{BaseCurrency: "IDR", QuoteCurrency: "SGD", Rate: idrSgd},
		{BaseCurrency: "USD", QuoteCurrency: "SGD", Rate: usdSgd},
		{BaseCurrency: "SGD", QuoteCurrency: "USD", Rate: sgdUsd},
	}

	conn, err := grpc.Dial(s.fxAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect FX %s: %w", s.fxAddr, err)
	}
	defer conn.Close()

	fxClient := fxv1.NewFxServiceClient(conn)

	_, err = fxClient.UpdateRates(ctx, &fxv1.UpdateRatesRequest{Rates: outRates})
	if err != nil {
		return nil, fmt.Errorf("call UpdateRates: %w", err)
	}

	log.Printf("[FxAdmin] Pushed %d rates to FX (%s): USD/IDR=%.2f, SGD/IDR=%.2f, USD/SGD=%.4f",
		len(outRates), s.fxAddr, usdIdr, sgdIdr, usdSgd,
	)

	// 4) Balas ke caller
	return &fxv1.RefreshRandomRatesResponse{
		Success: true,
		Message: "Random FX rates generated and pushed",
		Pushed:  outRates,
	}, nil
}
