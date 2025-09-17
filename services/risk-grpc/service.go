// services/risk-grpc/service.go
package main

import (
	"context"
	"crypto/rand"
	"log"
	"math/big"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	riskv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
)

var (
	riskRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "poc",
			Subsystem: "risk",
			Name:      "evaluate_total",
			Help:      "Total Evaluate() calls",
		},
		[]string{"decision", "reason"},
	)
	riskLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "poc",
			Subsystem: "risk",
			Name:      "evaluate_seconds",
			Help:      "Latency of Evaluate()",
			Buckets:   prometheus.DefBuckets,
		},
	)
)

// konstanta aturan sederhana
const (
	minAmountIDR int64 = 1_000       // < 1000 → reject
	maxAmountIDR int64 = 10_000_000  // > 10_000_000 → reject
)

// 2% probabilistic reject
func hitTwoPercent() bool {
	n, err := rand.Int(rand.Reader, big.NewInt(100)) // 0..99
	if err != nil {
		// kalau gagal random cryptographic, fallback aman: tidak reject
		log.Printf("[risk] crypto/rand error: %v", err)
		return false
	}
	return n.Int64() < 2 // 0 atau 1 => 2%
}

type RiskService struct {
	riskv1.UnimplementedRiskServiceServer
}

func NewRiskServiceFromEnv() *RiskService {
	return &RiskService{}
}

func (s *RiskService) Evaluate(ctx context.Context, req *riskv1.ScoreRequest) (*riskv1.EvaluateResponse, error) {
	start := time.Now()
	defer func() { riskLatency.Observe(time.Since(start).Seconds()) }()

	// sanity
	if req.SenderId == "" || req.ReceiverId == "" {
		riskRequests.WithLabelValues("deny", "bad_input").Inc()
		return &riskv1.EvaluateResponse{Allow: false, Reason: "bad_input"}, nil
	}
	if req.SenderId == req.ReceiverId {
		riskRequests.WithLabelValues("deny", "same_party").Inc()
		return &riskv1.EvaluateResponse{Allow: false, Reason: "same_party"}, nil
	}

	// aturan nominal
	if req.AmountIdr < minAmountIDR {
		riskRequests.WithLabelValues("deny", "below_min").Inc()
		return &riskv1.EvaluateResponse{Allow: false, Reason: "below_min"}, nil
	}
	if req.AmountIdr > maxAmountIDR {
		riskRequests.WithLabelValues("deny", "above_max").Inc()
		return &riskv1.EvaluateResponse{Allow: false, Reason: "above_max"}, nil
	}

	// 2% random reject
	if hitTwoPercent() {
		riskRequests.WithLabelValues("deny", "random_reject").Inc()
		return &riskv1.EvaluateResponse{Allow: false, Reason: "random_reject"}, nil
	}

	// allow (98%)
	riskRequests.WithLabelValues("allow", "").Inc()
	return &riskv1.EvaluateResponse{Allow: true, Reason: ""}, nil
}
