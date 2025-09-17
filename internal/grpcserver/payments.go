// internal/grpcserver/payments.go
// internal/grpcserver/payments.go

package grpcserver

import (
  "context"
  "fmt"
  "time"

  paymentsv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
  riskv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
  walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
  fxv1 "github.com/example/payment-gateway-poc/proto/gen/fx/v1"

)

type PaymentsServer struct {
  paymentsv1.UnimplementedPaymentsServiceServer
  Risk   riskv1.RiskServiceClient
  Wallet walletv1.WalletServiceClient
  Fx     fxv1.FxServiceClient // <-- tambahkan ini
}

func (s *PaymentsServer) CreatePayment(ctx context.Context, in *paymentsv1.CreatePaymentRequest) (*paymentsv1.CreatePaymentResponse, error) {
  if in == nil {
    return nil, fmt.Errorf("nil request")
  }

  
  // 1) Risk scoring
if s.Risk != nil {
  _, err := s.Risk.Score(ctx, &riskv1.ScoreRequest{
    AmountMinor: in.GetAmountMinor(),
    UserId:      in.GetUserId(),
  })
  if err != nil {
    return nil, fmt.Errorf("risk score error: %w", err)
  }
}

  // 2) Generate payment_id
  paymentID := fmt.Sprintf("pay_%d", time.Now().UnixNano())

  // 3) Reserve di wallet
  if s.Wallet != nil {
    resv, err := s.Wallet.Reserve(ctx, &walletv1.ReserveRequest{
      PaymentId:   paymentID,
      AmountMinor: in.GetAmountMinor(),
      Currency:    in.GetCurrency(),
    })
    if err != nil {
      return nil, fmt.Errorf("wallet reserve error: %w", err)
    }
    if !resv.GetOk() {
      return nil, fmt.Errorf("wallet reserve not ok")
    }

    // 4) Capture
    capRes, err := s.Wallet.Capture(ctx, &walletv1.CaptureRequest{
      ReservationId: resv.GetReservationId(),
    })
    if err != nil {
      return nil, fmt.Errorf("wallet capture error: %w", err)
    }
    if !capRes.GetOk() {
      return nil, fmt.Errorf("wallet capture not ok")
    }
  }

  return &paymentsv1.CreatePaymentResponse{PaymentId: paymentID}, nil
}
