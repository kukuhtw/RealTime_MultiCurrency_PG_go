// services/api-gateway/main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"

	"github.com/example/payment-gateway-poc/services/api-gateway/clients"
	"github.com/example/payment-gateway-poc/services/api-gateway/handlers"
	"github.com/example/payment-gateway-poc/services/api-gateway/queue"
	m "github.com/example/payment-gateway-poc/pkg/metrics"
)

const serviceName = "api-gateway"

func main() {
	grpcClients, err := clients.NewGRPC()
	if err != nil {
		log.Fatalf("init grpc clients: %v", err)
	}
	defer grpcClients.Close()

	r := mux.NewRouter()
	r.Use(metricsMiddleware)

	// metrics & health
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":      true,
			"service": serviceName,
			"ts":      time.Now().UTC(),
		})
	}).Methods(http.MethodGet)

	// API
	bus := queue.New(
		strings.Split(getenv("KAFKA_BROKERS", "kafka:9092"), ","),
		getenv("KAFKA_REQ_TOPIC", "payments.request"),
		getenv("KAFKA_RES_TOPIC", "payments.result"),
	)
	r.HandleFunc("/api/random-accounts", handlers.RandomAccountsHandler(grpcClients.Wallet)).Methods(http.MethodGet)
	r.HandleFunc("/api/payments", handlers.PaymentsHandler(handlers.Deps{
		Fx:     grpcClients.Fx,
		Wallet: grpcClients.Wallet,
		Risk:   grpcClients.Risk,
		Bus:    bus,
	})).Methods(http.MethodPost)

	// static
	staticDir := "./static"
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticDir)))

	addr := getenv("HTTP_ADDR", ":8080")
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

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
