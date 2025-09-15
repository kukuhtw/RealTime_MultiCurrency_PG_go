// payment-gateway-poc/services/fx/main.go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	m "github.com/example/payment-gateway-poc/pkg/metrics" // <-- paket metrics bersama
)

const serviceName = "fx"

func main() {
	r := mux.NewRouter()

	// middleware metrics untuk semua handler di bawahnya
	r.Use(metricsMiddleware)

	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "service": serviceName})
	}).Methods(http.MethodGet)

	r.HandleFunc("/rate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		base := q.Get("base")
		quote := q.Get("quote")
		if base == "" {
			base = "USD"
		}
		if quote == "" {
			quote = "IDR"
		}
		rate := mockRate(base, quote)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pair": fmt.Sprintf("%s/%s", base, quote),
			"rate": rate,
			"ts":   time.Now().UTC().Unix(),
		})
	}).Methods(http.MethodGet)

	r.HandleFunc("/convert", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		from, to := q.Get("from"), q.Get("to")
		if from == "" {
			from = "USD"
		}
		if to == "" {
			to = "IDR"
		}
		amount := 1.0
		rate := mockRate(from, to)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"amount_converted": amount * rate,
			"rate_used":        rate,
		})
	}).Methods(http.MethodGet)

	// expose Prometheus metrics
	r.Handle("/metrics", promhttp.Handler())

	addr := getEnv("HTTP_ADDR", ":8082")
	log.Printf("%s listening at %s", serviceName, addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

// ------------ helpers ------------

// middleware: ukur durasi & catat status per request
func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		statusLabel := httpStatusToBiz(rec.status) // "SUCCESS"/"FAILED"
		m.IncRequest(serviceName, statusLabel, r.Method)
		m.ObserveDuration(serviceName, statusLabel, time.Since(start).Seconds())
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func httpStatusToBiz(code int) string {
	if code >= 200 && code < 400 {
		return "SUCCESS"
	}
	return "FAILED"
}

func mockRate(base, quote string) float64 {
	if base == "" || quote == "" {
		return 1
	}
	// dummy oscillating rate
	return math.Abs(float64(len(base))*1000.0-float64(len(quote))*100.0) + 15000.0
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
