// payment-gateway-poc/main.go
package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type PaymentRequest struct {
	ID                 string  `json:"id"`
	Currency           string  `json:"currency"`
	Amount             float64 `json:"amount"`
	SourceAccount      string  `json:"source_account"`
	DestinationAccount string  `json:"destination_account"`
}

type PaymentResponse struct {
	ID      string `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	r := mux.NewRouter()

	// bisnis
	r.HandleFunc("/payments", paymentsHandler).Methods(http.MethodPost)

	// healthcheck
	r.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// expose prometheus
	r.Handle("/metrics", promhttp.Handler())

	addr := ":8081"
	log.Printf("Payment Gateway POC listening on %s ...", addr)
	log.Fatal(http.ListenAndServe(addr,
