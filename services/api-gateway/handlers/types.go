// services/api-gateway/handlers/types.go
package handlers

type PaymentIn struct {
    SenderID       string  `json:"sender_id"`
    ReceiverID     string  `json:"receiver_id"`
    Currency       string  `json:"currency"` // IDR / USD / SGD
    Amount         float64 `json:"amount"`
    TxDateISO      string  `json:"tx_date"`
    IdempotencyKey string  `json:"idempotency_key"`
}

type PaymentOut struct {
    Status string `json:"status"`
    Reason string `json:"reason,omitempty"`
    Ref    string `json:"ref,omitempty"` // reservation_id / reference
}
