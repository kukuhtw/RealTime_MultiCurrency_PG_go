// services/api-gateway/handlers/random_accounts.go

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
)

type RandomAccountsResponse struct {
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
}

type WalletClient interface {
	GetRandomAccounts(ctx context.Context, in *walletv1.GetRandomAccountsRequest, opts ...interface{}) (*walletv1.GetRandomAccountsResponse, error)
}

// Gunakan tipe ini agar kompatibel dengan grpc generated client method signature.
type walletClientCompat interface {
	GetRandomAccounts(ctx context.Context, in *walletv1.GetRandomAccountsRequest, opts ...grpcCallOption) (*walletv1.GetRandomAccountsResponse, error)
}
type grpcCallOption = interface{}

func RandomAccountsHandler(wallet walletClientCompat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp, err := wallet.GetRandomAccounts(ctx, &walletv1.GetRandomAccountsRequest{})
		if err != nil {
			http.Error(w, "wallet.GetRandomAccounts: "+err.Error(), http.StatusBadGateway)
			return
		}

		out := RandomAccountsResponse{
			SenderID:   resp.GetSenderId(),
			ReceiverID: resp.GetReceiverId(),
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
