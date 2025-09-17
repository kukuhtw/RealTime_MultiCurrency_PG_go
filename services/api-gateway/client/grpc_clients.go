// services/api-gateway/clients/grpc_clients.go
package clients

import (
	"fmt"
	"os"
	"time"

	paymentsv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
	walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPC struct {
	Wallet   walletv1.WalletServiceClient
	Payments paymentsv1.PaymentsServiceClient

	rawConnWallet   *grpc.ClientConn
	rawConnPayments *grpc.ClientConn
}

func NewGRPC() (*GRPC, error) {
	walletAddr := getenv("WALLET_ADDR", "wallet-grpc:9103")
	payAddr := getenv("PAYMENTS_ADDR", "payments-grpc:9101")

	dial := func(addr string) (*grpc.ClientConn, error) {
		return grpc.Dial(
			addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithTimeout(5*time.Second),
		)
	}

	wc, err := dial(walletAddr)
	if err != nil {
		return nil, fmt.Errorf("dial wallet %s: %w", walletAddr, err)
	}
	pc, err := dial(payAddr)
	if err != nil {
		_ = wc.Close()
		return nil, fmt.Errorf("dial payments %s: %w", payAddr, err)
	}

	return &GRPC{
		Wallet:          walletv1.NewWalletServiceClient(wc),
		Payments:        paymentsv1.NewPaymentsServiceClient(pc),
		rawConnWallet:   wc,
		rawConnPayments: pc,
	}, nil
}

func (g *GRPC) Close() {
	if g.rawConnWallet != nil {
		_ = g.rawConnWallet.Close()
	}
	if g.rawConnPayments != nil {
		_ = g.rawConnPayments.Close()
	}
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
