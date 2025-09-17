// cmd/payments-worker/main.go
package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/segmentio/kafka-go"

	commonv1   "github.com/example/payment-gateway-poc/proto/gen/common/v1"
	
)

type inMsg struct {
	IdempotencyKey string `json:"idempotency_key"`
	SenderID       string `json:"sender_id"`
	ReceiverID     string `json:"receiver_id"`
	AmountIDR      int64  `json:"amount_idr"`
	CurrencyInput  string `json:"currency_input"`
	TxDate         string `json:"tx_date"`
}

func main() {
	brokers := env("KAFKA_BROKERS", "kafka:9092")
	reqTopic := env("KAFKA_REQ_TOPIC", "payments.request")
	resTopic := env("KAFKA_RES_TOPIC", "payments.result")

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokers},
		Topic:    reqTopic,
		GroupID:  "payments-worker",
		MinBytes: 1,
		MaxBytes: 10e6,
	})
	defer r.Close()

	w := &kafka.Writer{
		Addr:     kafka.TCP(brokers),
		Topic:    resTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer w.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[payments-worker] started")
	for {
		m, err := r.ReadMessage(ctx)
		if err != nil {
			log.Printf("read err: %v", err)
			return
		}
		var in inMsg
		if err := json.Unmarshal(m.Value, &in); err != nil {
			log.Printf("bad msg: %v", err)
			continue
		}

		// Contoh: panggil PaymentsService.LogAndSettle (disederhanakan; di sini kita mock)
		_ = commonv1.Currency_IDR // touch import

		// Simulasi proses OK
		out := map[string]any{
			"status": "SUCCESS",
			"ref":    "RSV-" + in.IdempotencyKey,
		}
		b, _ := json.Marshal(out)

		if err := w.WriteMessages(ctx, kafka.Message{Key: m.Key, Value: b, Time: time.Now()}); err != nil {
			log.Printf("write err: %v", err)
		}
	}
}

func env(k, d string) string { if v := os.Getenv(k); v != "" { return v }; return d }
