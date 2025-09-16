package grpcserver

import (
  "context"
  walletv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/wallet/v1"
  commonv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/common/v1"
)

type WalletServer struct{ walletv1.UnimplementedWalletServiceServer }

func (s *WalletServer) GetBalance(ctx context.Context, in *walletv1.GetBalanceRequest) (*walletv1.GetBalanceResponse, error) {
  return &walletv1.GetBalanceResponse{Balance: &commonv1.Money{Currency: commonv1.Currency_USD, Amount: 5000}}, nil
}
func (s *WalletServer) Reserve(ctx context.Context, in *walletv1.ReserveRequest) (*walletv1.ReserveResponse, error) {
  ok := in.Amount.Amount <= 5000
  rid := ""
  if ok { rid = "RSV-" + in.Tx.TxId }
  return &walletv1.ReserveResponse{Ok: ok, ReservationId: rid}, nil
}
func (s *WalletServer) Capture(ctx context.Context, in *walletv1.CaptureRequest) (*walletv1.CaptureResponse, error) {
  return &walletv1.CaptureResponse{Ok: true}, nil
}
