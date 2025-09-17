// services/api-gateway/handlers/payments.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	fxv1   "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
	riskv1 "github.com/example/payment-gateway-poc/proto/gen/risk/v1"
	wv1    "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"

	"github.com/example/payment-gateway-poc/services/api-gateway/queue"
	m "github.com/example/payment-gateway-poc/pkg/metrics"
)

type Deps struct {
	Fx     fxv1.FxServiceClient
	Wallet wv1.WalletServiceClient
	Risk   riskv1.RiskServiceClient
	Bus    *queue.Bus
}

func PaymentsHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer func() { m.ObserveDuration("api-gateway", "REQUEST", time.Since(start).Seconds()) }()

		var in PaymentIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, PaymentOut{Status: "FAILED", Reason: "bad_json"})
			return
		}
		if in.SenderID == "" || in.ReceiverID == "" || in.Amount <= 0 || in.IdempotencyKey == "" || in.TxDateISO == "" {
			writeJSON(w, http.StatusBadRequest, PaymentOut{Status: "FAILED", Reason: "invalid_input"})
			return
		}
		cur := strings.ToUpper(strings.TrimSpace(in.Currency))
		if cur != "IDR" && cur != "USD" && cur != "SGD" {
			writeJSON(w, http.StatusBadRequest, PaymentOut{Status: "FAILED", Reason: "unsupported_currency"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		// 1) FX convert → amount_idr
		amountIDR := in.Amount
		if cur != "IDR" {
			fxRes, err := d.Fx.Convert(ctx, &fxv1.ConvertRequest{
				FromCurrency: cur,
				ToCurrency:   "IDR",
				Amount:       in.Amount,
			})
			if err != nil {
				m.IncRequest("api-gateway", "FAILED", "FX")
				writeJSON(w, http.StatusBadGateway, PaymentOut{Status: "FAILED", Reason: "fx_unavailable"})
				return
			}
			amountIDR = fxRes.GetAmount()
			m.IncRequest("api-gateway", "SUCCESS", "FX")
		}

		// 2) Wallet check saldo pengirim
		acc, err := d.Wallet.GetAccount(ctx, &wv1.GetAccountRequest{AccountId: in.SenderID})
		if err != nil {
			m.IncRequest("api-gateway", "FAILED", "WALLET_GET")
			writeJSON(w, http.StatusBadGateway, PaymentOut{Status: "FAILED", Reason: "wallet_unavailable"})
			return
		}
		if acc.GetBalanceIdr() < int64(amountIDR) {
			m.IncRequest("api-gateway", "FAILED", "WALLET_INSUFFICIENT")
			writeJSON(w, http.StatusOK, PaymentOut{Status: "FAILED", Reason: "insufficient_funds"})
			return
		}
		m.IncRequest("api-gateway", "SUCCESS", "WALLET_CHECK")

		// 3) Risk check
		riskRes, err := d.Risk.Evaluate(ctx, &riskv1.ScoreRequest{
			SenderId:      in.SenderID,
			ReceiverId:    in.ReceiverID,
			AmountIdr:     int64(amountIDR),
			CurrencyInput: cur,
			TxDateIso:     in.TxDateISO,
		})
		if err != nil {
			m.IncRequest("api-gateway", "FAILED", "RISK_CALL")
			writeJSON(w, http.StatusBadGateway, PaymentOut{Status: "FAILED", Reason: "risk_unavailable"})
			return
		}
		if !riskRes.GetAllow() {
			m.IncRequest("api-gateway", "FAILED", "RISK_DENY")
			writeJSON(w, http.StatusOK, PaymentOut{Status: "FAILED", Reason: riskRes.GetReason()})
			return
		}
		m.IncRequest("api-gateway", "SUCCESS", "RISK_ALLOW")

		// 4) Publish ke Kafka & tunggu result
		payload, _ := json.Marshal(map[string]any{
			"idempotency_key": in.IdempotencyKey,
			"sender_id":       in.SenderID,
			"receiver_id":     in.ReceiverID,
			"amount_idr":      int64(amountIDR),
			"currency_input":  cur,
			"tx_date":         in.TxDateISO,
		})

		if err := d.Bus.Publish(ctx, []byte(in.IdempotencyKey), payload); err != nil {
			m.IncRequest("api-gateway", "FAILED", "KAFKA_PUBLISH")
			writeJSON(w, http.StatusBadGateway, PaymentOut{Status: "FAILED", Reason: "queue_publish_error"})
			return
		}
		m.IncRequest("api-gateway", "SUCCESS", "KAFKA_PUBLISH")

		result, ok, err := d.Bus.WaitResult(r.Context(), []byte(in.IdempotencyKey), 5*time.Second)
		if err != nil || !ok {
			m.IncRequest("api-gateway", "FAILED", "KAFKA_WAIT")
			writeJSON(w, http.StatusGatewayTimeout, PaymentOut{Status: "FAILED", Reason: "queue_timeout"})
			return
		}
		m.IncRequest("api-gateway", "SUCCESS", "KAFKA_WAIT")

		var out PaymentOut
		if err := json.Unmarshal(result, &out); err != nil || out.Status == "" {
			writeJSON(w, http.StatusOK, PaymentOut{Status: "FAILED", Reason: "bad_worker_result"})
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}
