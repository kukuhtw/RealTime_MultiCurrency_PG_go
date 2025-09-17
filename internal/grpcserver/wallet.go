// internal/grpcserver/wallet.go
package grpcserver

import (
  "context"

  commonv1 "github.com/example/payment-gateway-poc/proto/gen/common/v1"
  walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
)

type WalletServer struct{ walletv1.UnimplementedWalletServiceServer }

func (s *WalletServer) GetBalance(ctx context.Context, in *walletv1.GetBalanceRequest) (*walletv1.GetBalanceResponse, error) {
  return &walletv1.GetBalanceResponse{
    BalanceMinor: 5000,
    Currency:     commonv1.Currency_USD,
  }, nil
}

func (s *WalletServer) Reserve(ctx context.Context, in *walletv1.ReserveRequest) (*walletv1.ReserveResponse, error) {
  ok := in.GetAmountMinor() <= 5000
  rid := ""
  if ok {
    rid = "RSV-" + in.GetPaymentId()
  }
  return &walletv1.ReserveResponse{Ok: ok, ReservationId: rid}, nil
}

func (s *WalletServer) Capture(ctx context.Context, in *walletv1.CaptureRequest) (*walletv1.CaptureResponse, error) {
  return &walletv1.CaptureResponse{Ok: true}, nil
}
