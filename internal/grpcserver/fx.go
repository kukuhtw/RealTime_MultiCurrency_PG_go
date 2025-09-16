package grpcserver

import (
  "context"
  fxv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/fx/v1"
  commonv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/common/v1"
)

type FxServer struct{ fxv1.UnimplementedFxServiceServer }

func (s *FxServer) GetRate(ctx context.Context, in *fxv1.GetRateRequest) (*fxv1.GetRateResponse, error) {
  rate := 1.0
  if in.Base == commonv1.Currency_USD && in.Quote == commonv1.Currency_IDR { rate = 15500 }
  return &fxv1.GetRateResponse{Rate: rate}, nil
}
func (s *FxServer) Convert(ctx context.Context, in *fxv1.ConvertRequest) (*fxv1.ConvertResponse, error) {
  rate := 1.0
  if in.From.Currency == commonv1.Currency_USD && in.To == commonv1.Currency_IDR { rate = 15500 }
  return &fxv1.ConvertResponse{Result: &commonv1.Money{Currency: in.To, Amount: in.From.Amount * rate}}, nil
}
