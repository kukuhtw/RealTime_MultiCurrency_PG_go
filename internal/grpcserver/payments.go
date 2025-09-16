package grpcserver

import (
  "context"
  paymentsv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/payments/v1"
  fxv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/fx/v1"
  riskv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/risk/v1"
  walletv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/wallet/v1"
)

type PaymentsServer struct{
  paymentsv1.UnimplementedPaymentsServiceServer
  Risk  riskv1.RiskServiceClient
  Fx    fxv1.FxServiceClient
  Wallet walletv1.WalletServiceClient
}

func (s *PaymentsServer) CreatePayment(ctx context.Context, in *paymentsv1.CreatePaymentRequest) (*paymentsv1.CreatePaymentResponse, error) {
  r, err := s.Risk.Score(ctx, &riskv1.ScoreRequest{
    Tx: in.Tx, Customer: in.Customer, Amount: in.Amount,
    MerchantId: in.MerchantId, Ip: in.Ip, DeviceId: in.DeviceId,
    BillingCountry: in.BillingCountry, Mcc: in.Mcc,
  })
  if err != nil || r.Decision == riskv1.Decision_DENY {
    return &paymentsv1.CreatePaymentResponse{
      TxId: in.Tx.TxId, Status: paymentsv1.PaymentStatus_FAILED,
      RiskDecision: r.GetDecision(), RiskScore: r.GetRiskScore(),
      Reasons: append(r.GetReasons(), "risk_block"),
    }, nil
  }

  final := in.Amount
  if in.Amount.Currency != in.SettlementCurrency {
    cvt, err := s.Fx.Convert(ctx, &fxv1.ConvertRequest{ From: in.Amount, To: in.SettlementCurrency })
    if err != nil {
      return &paymentsv1.CreatePaymentResponse{TxId: in.Tx.TxId, Status: paymentsv1.PaymentStatus_FAILED, Reasons: []string{"fx_error"}}, nil
    }
    final = cvt.Result
  }

  resv, err := s.Wallet.Reserve(ctx, &walletv1.ReserveRequest{ Tx: in.Tx, Account: in.Account, Amount: final })
  if err != nil || !resv.Ok {
    return &paymentsv1.CreatePaymentResponse{TxId: in.Tx.TxId, Status: paymentsv1.PaymentStatus_FAILED, Reasons: []string{"insufficient_funds"}}, nil
  }
  if _, err := s.Wallet.Capture(ctx, &walletv1.CaptureRequest{ReservationId: resv.ReservationId}); err != nil {
    return &paymentsv1.CreatePaymentResponse{TxId: in.Tx.TxId, Status: paymentsv1.PaymentStatus_FAILED, Reasons: []string{"capture_error"}}, nil
  }

  return &paymentsv1.CreatePaymentResponse{
    TxId: in.Tx.TxId, Status: paymentsv1.PaymentStatus_CAPTURED,
    RiskDecision: r.Decision, RiskScore: r.RiskScore, FinalAmount: final, ReservationId: resv.ReservationId,
  }, nil
}
