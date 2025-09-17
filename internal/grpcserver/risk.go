// internal/grpcserver/risk.go
package grpcserver

import (
  "context"

  riskv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
)

type RiskServer struct{ riskv1.UnimplementedRiskServiceServer }

func (s *RiskServer) Score(ctx context.Context, in *riskv1.ScoreRequest) (*riskv1.ScoreResponse, error) {
  // Implementasi minimal kompatibel dengan proto yang ada.
  // (Kalau nanti kamu mau pakai field tertentu dari ScoreResponse,
  // cocokin dulu dengan definisi proto risk/v1/risk.proto kamu.)
  return &riskv1.ScoreResponse{}, nil
}
