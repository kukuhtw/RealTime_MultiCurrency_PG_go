// payment-gateway-poc/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	paymentsv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
)

type APIServer struct {
	paymentsClient paymentsv1.PaymentServiceClient
	router         *mux.Router
}

func NewAPIServer() (*APIServer, error) {
	// Connect to payments gRPC service
	paymentsConn, err := grpc.Dial(
		os.Getenv("PAYMENTS_ADDR"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	paymentsClient := paymentsv1.NewPaymentServiceClient(paymentsConn)

	router := mux.NewRouter()
	server := &APIServer{
		paymentsClient: paymentsClient,
		router:         router,
	}

	server.setupRoutes()
	return server, nil
}

func (s *APIServer) setupRoutes() {
	// Business routes
	s.router.HandleFunc("/payments", s.createPaymentHandler).Methods("POST")
	s.router.HandleFunc("/payments/{id}", s.getPaymentHandler).Methods("GET")
	
	// Health check
	s.router.HandleFunc("/healthz", s.healthHandler).Methods("GET")
	
	// Metrics
	s.router.Handle("/metrics", promhttp.Handler())
}

func (s *APIServer) createPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation for creating payment
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "processing", "message": "Payment creation endpoint"}`))
}

func (s *APIServer) getPaymentHandler(w http.ResponseWriter, r *http.Request) {
	// Implementation for getting payment status
	vars := mux.Vars(r)
	paymentID := vars["id"]
	
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"id": "` + paymentID + `", "status": "completed"}`))
}

func (s *APIServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
}

func (s *APIServer) Start(addr string) error {
	log.Printf("API Gateway server starting on %s", addr)
	
	server := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP server Shutdown: %v", err)
		}
	}()

	return server.ListenAndServe()
}

func main() {
	server, err := NewAPIServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(":8080"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}