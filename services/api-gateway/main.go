// services/api-gateway/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"

	m "github.com/example/payment-gateway-poc/pkg/metrics" // <-- sesuaikan dgn module path di go.mod
)

const serviceName = "api-gateway"

func main() {
	r := mux.NewRouter()

	// Instrument semua request (kecuali /metrics)
	r.Use(metricsMiddleware)

	// 1) metrics & healthz DULU (agar tidak ketimpa static)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"service": serviceName,
			"ts":      time.Now().UTC(),
		})
	}).Methods(http.MethodGet)

	// 2) static terakhir (catch-all)
	staticDir := "./static"
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	addr := getEnv("HTTP_ADDR", ":8080")
	handler := cors.AllowAll().Handler(r)
	log.Printf("%s listening at %s", serviceName, addr)
	log.Fatal(http.ListenAndServe(addr, handler))
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
		// Jangan instrument endpoint /metrics itu sendiri
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
