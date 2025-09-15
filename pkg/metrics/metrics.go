// payment-gateway-poc/pkg/metrics/metrics.go
package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
    // Tambah label "service" supaya 1 query bisa bandingkan antar service
    PaymentRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "payment",
            Name:      "requests_total",
            Help:      "Total request pembayaran per service",
        },
        []string{"service", "status", "method"},
    )

    PaymentRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Namespace: "payment",
            Name:      "request_duration_seconds",
            Help:      "Durasi proses request pembayaran per service",
            // bucket cukup rapat di sub-second
            Buckets: []float64{
                0.01, 0.02, 0.03, 0.05, 0.08, 0.12,
                0.2, 0.3, 0.5, 0.8, 1.2, 2, 3, 5,
            },
        },
        []string{"service", "status"},
    )
)

func init() {
    prometheus.MustRegister(PaymentRequestsTotal, PaymentRequestDuration)
}

// Helper biar rapi dipanggil dari handler
func IncRequest(service, status, method string) {
    PaymentRequestsTotal.WithLabelValues(service, status, method).Inc()
}
func ObserveDuration(service, status string, seconds float64) {
    PaymentRequestDuration.WithLabelValues(service, status).Observe(seconds)
}
