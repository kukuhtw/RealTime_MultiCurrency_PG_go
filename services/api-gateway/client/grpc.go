// services/api-gateway/client/grpc.go
// services/api-gateway/clients/grpc.go
package clients

import (
    "fmt"
    "os"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    fxv1 "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
    wv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
    rv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
    pv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
)

type GRPC struct {
    Fx       fxv1.FxServiceClient
    Wallet   wv1.WalletServiceClient
    Risk     rv1.RiskServiceClient
    Payments pv1.PaymentsServiceClient
    conns    []*grpc.ClientConn
}

func dial(addr string) (*grpc.ClientConn, error) {
    return grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func NewGRPC() (*GRPC, error) {
    fxAddr := getenv("FX_ADDR", "fx-grpc:9102")
    wAddr  := getenv("WALLET_ADDR", "wallet-grpc:9103")
    rAddr  := getenv("RISK_ADDR", "risk-grpc:9104")
    pAddr  := getenv("PAYMENTS_ADDR", "payments-grpc:9096")

    conns := make([]*grpc.ClientConn, 0, 4)

    cfx, err := dial(fmt.Sprintf("%s", fxAddr)); if err != nil { return nil, err }; conns = append(conns, cfx)
    cw , err := dial(fmt.Sprintf("%s", wAddr));  if err != nil { return nil, err }; conns = append(conns, cw)
    cr , err := dial(fmt.Sprintf("%s", rAddr));  if err != nil { return nil, err }; conns = append(conns, cr)
    cp , err := dial(fmt.Sprintf("%s", pAddr));  if err != nil { return nil, err }; conns = append(conns, cp)

    return &GRPC{
        Fx:       fxv1.NewFxServiceClient(cfx),
        Wallet:   wv1.NewWalletServiceClient(cw),
        Risk:     rv1.NewRiskServiceClient(cr),
        Payments: pv1.NewPaymentsServiceClient(cp),
        conns:    conns,
    }, nil
}

func (g *GRPC) Close() {
    for _, c := range g.conns { _ = c.Close() }
}

func getenv(k, d string) string {
    if v := os.Getenv(k); v != "" { return v }
    return d
}
