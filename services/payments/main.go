// services/payments/main.go
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	m "github.com/example/payment-gateway-poc/pkg/metrics" // sesuaikan dgn go.mod
)

const serviceName = "payments"

type PaymentRequest struct {
	ID                string  `json:"id"`
	Currency          string  `json:"currency,omitempty"`
	Amount            float64 `json:"amount,omitempty"`
	SourceAccount     string  `json:"source_account,omitempty"`
	DestinationAccount string `json:"destination_account,omitempty"`
}

func main() {
	rand.Seed(time.Now().UnixNano())

	r := mux.NewRouter()
	r.Use(metricsMiddleware) // catat counter & histogram utk semua endpoint (kecuali /metrics)

	// health
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": serviceName})
	}).Methods(http.MethodGet)

	// bisnis
	r.HandleFunc("/payments", paymentHandler).Methods(http.MethodPost)

	// expose metrics
	r.Handle("/metrics", promhttp.Handler())

	addr := getEnv("HTTP_ADDR", ":8081")
	log.Printf("%s listening at %s", serviceName, addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func paymentHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req PaymentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	// simulasi kerja 50–350 ms
	time.Sleep(time.Duration(50+rand.Intn(300)) * time.Millisecond)

	// simulasi gagal sesuai env FAIL_RATE (0.0–1.0). default 0.0 (tanpa gagal).
	if rand.Float64() < failRate() {
		http.Error(w, `{"status":"error","message":"upstream failed"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "id": req.ID})
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

func failRate() float64 {
	if v := os.Getenv("FAIL_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f >= 0 && f <= 1 {
			return f
		}
	}
	return 0.0 // default: tidak gagal
}
