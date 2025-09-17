// cmd/payments-worker/main.go (WORKER: Kafka → payments-rs)

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
	pv1 "github.com/example/payment-gateway-poc/proto/gen/payments/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Req struct {
	IdempotencyKey string `json:"idempotency_key"`
	SenderID       string `json:"sender_id"`
	ReceiverID     string `json:"receiver_id"`
	AmountIDR      int64  `json:"amount_idr"`
	CurrencyInput  string `json:"currency_input"`
	TxDate         string `json:"tx_date"`
}

type Res struct {
	Status string `json:"status"`
	Reason string `json:"reason,omitempty"`
	Ref    string `json:"ref,omitempty"`
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

func main() {
	brokers := []string{getenv("KAFKA_BROKERS", "kafka:9092")}
	reqTopic := getenv("KAFKA_REQ_TOPIC", "payments.request")
	resTopic := getenv("KAFKA_RES_TOPIC", "payments.result")
	paymentsAddr := getenv("PAYMENTS_ADDR", "payments-grpc:9096")

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: brokers,
		Topic:   reqTopic,
		GroupID: "payments-worker",
		MinBytes: 1, MaxBytes: 10e6,
	})
	w := &kafka.Writer{Addr: kafka.TCP(brokers...), Topic: resTopic}
	defer r.Close()
	defer w.Close()

	conn, err := grpc.Dial(paymentsAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { log.Fatalf("dial payments: %v", err) }
	defer conn.Close()
	cli := pv1.NewPaymentsClient(conn)

	log.Println("payments-worker running...")
	for {
		msg, err := r.ReadMessage(context.Background())
		if err != nil { log.Fatalf("kafka read: %v", err) }

		var req Req
		if err := json.Unmarshal(msg.Value, &req); err != nil {
			log.Printf("bad payload: %v", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		resp, err := cli.LogAndSettle(ctx, &pv1.LogAndSettleRequest{
			SenderId:       req.SenderID,
			ReceiverId:     req.ReceiverID,
			AmountIdr:      req.AmountIDR,
			Currency:       req.CurrencyInput,
			TxDate:         req.TxDate,
			IdempotencyKey: &req.IdempotencyKey,
		})
		cancel()

		out := Res{}
		if err != nil {
			out = Res{Status: "FAILED", Reason: "settlement_error"}
		} else {
			out = Res{
				Status: resp.GetStatus(),
				Reason: resp.GetMessage(),
				Ref:    resp.GetReservationId(),
			}
		}
		b, _ := json.Marshal(out)
		if err := w.WriteMessages(context.Background(), kafka.Message{Key: msg.Key, Value: b}); err != nil {
			log.Printf("publish result error: %v", err)
		}
	}
}
