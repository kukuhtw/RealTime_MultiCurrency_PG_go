package grpcserver

import (
  "context"
  riskv1 "github.com/kukuhtw/RealTime_MultiCurrency_PG_go/gen/risk/v1"
)

type RiskServer struct{ riskv1.UnimplementedRiskServiceServer }

func (s *RiskServer) Score(ctx context.Context, in *riskv1.ScoreRequest) (*riskv1.ScoreResponse, error) {
  var score uint32
  reasons := []string{}
  if in.Amount.Amount > 1000 { score += 20; reasons = append(reasons, "amount_high") }
  if in.Decline_24H > 3 { score += 30; reasons = append(reasons, "recent_declines") }
  decision := riskv1.Decision_ACCEPT
  switch {
  case score > 60: decision = riskv1.Decision_DENY
  case score > 30: decision = riskv1.Decision_REVIEW
  }
  return &riskv1.ScoreResponse{TxId: in.Tx.TxId, RiskScore: score, Decision: decision, Reasons: reasons}, nil
}
