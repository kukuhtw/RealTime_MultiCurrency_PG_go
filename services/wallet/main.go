// payment-gateway-poc/services/wallet/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	m "github.com/example/payment-gateway-poc/pkg/metrics" // <-- sesuaikan dengan module path di go.mod
)

type Balance struct {
	Account  string  `json:"account"`
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

const serviceName = "wallet"

func main() {
	r := mux.NewRouter()

	// Instrument semua request (kecuali /metrics)
	r.Use(metricsMiddleware)

	// Healthcheck
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": serviceName})
	}).Methods(http.MethodGet)

	// Endpoint bisnis: saldo
	r.HandleFunc("/balance/{account}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		acct := mux.Vars(r)["account"]
		resp := Balance{Account: acct, Currency: "IDR", Amount: 1_000_000}
		_ = json.NewEncoder(w).Encode(resp)
	}).Methods(http.MethodGet)

	// Expose Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	addr := getEnv("HTTP_ADDR", ":8083")
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
		// Jangan instrument endpoint /metrics
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
