// payment-gateway-poc/services/risk/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	m "github.com/example/payment-gateway-poc/pkg/metrics" // <-- sesuaikan dgn module path di go.mod
)

type RiskInput struct {
	Account  string  `json:"account"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type RiskResult struct {
	Score  int    `json:"score"`  // 0-100
	Action string `json:"action"` // ALLOW, REVIEW, BLOCK
}

const serviceName = "risk"

func main() {
	r := mux.NewRouter()

	// Middleware metrics
	r.Use(metricsMiddleware)

	// Healthcheck
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": serviceName})
	}).Methods(http.MethodGet)

	// Endpoint utama: risk scoring
	r.HandleFunc("/score", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var in RiskInput
		_ = json.NewDecoder(r.Body).Decode(&in)

		// dummy logic
		res := RiskResult{Score: 42, Action: "REVIEW"}
		_ = json.NewEncoder(w).Encode(res)
	}).Methods(http.MethodPost)

	// Expose prometheus
	r.Handle("/metrics", promhttp.Handler())

	addr := getEnv("HTTP_ADDR", ":8084")
	log.Printf("%s listening at %s", serviceName, addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

/*************** Metrics middleware ***************/
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		statusLabel := "FAILED"
		if rec.status >= 200 && rec.status < 400 {
			statusLabel = "SUCCESS"
		}

		m.IncRequest(serviceName, statusLabel, r.Method)
		m.ObserveDuration(serviceName, statusLabel, time.Since(start).Seconds())
	})
}

/******************** Utils ********************/
func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
