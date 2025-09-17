// services/api-gateway/handlers/random_accounts.go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"google.golang.org/grpc"

	walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
)

type RandomAccountsResponse struct {
	SenderID   string `json:"sender_id"`
	ReceiverID string `json:"receiver_id"`
}

type walletClientCompat interface {
	GetRandomAccounts(ctx context.Context, in *walletv1.GetRandomAccountsRequest, opts ...grpc.CallOption) (*walletv1.GetRandomAccountsResponse, error)
}

// services/api-gateway/handlers/random_accounts.go
func RandomAccountsHandler(wallet walletClientCompat) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		resp, err := wallet.GetRandomAccounts(ctx, &walletv1.GetRandomAccountsRequest{Count: 2})
		if err != nil {
			http.Error(w, "wallet.GetRandomAccounts: "+err.Error(), http.StatusBadGateway)
			return
		}

		accountIds := resp.GetAccountIds()
		if len(accountIds) < 2 {
			http.Error(w, "not enough accounts", http.StatusBadGateway)
			return
		}

		out := RandomAccountsResponse{
			SenderID:   accountIds[0],
			ReceiverID: accountIds[1],
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}
