// services/wallet/main.go

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	commonv1 "github.com/example/payment-gateway-poc/proto/gen/common/v1"
	walletv1 "github.com/example/payment-gateway-poc/proto/gen/wallet/v1"
	"github.com/google/uuid"
	gp "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

type server struct {
	walletv1.UnimplementedWalletServiceServer
	pool *pgxpool.Pool
}

func main() {
	// === DB pool ===
	dsn := getenv("DATABASE_URL", "postgresql://postgres:secret@host.docker.internal:5432/poc?sslmode=disable")
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("[wallet-grpc] parse dsn: %v", err)
	}
	cfg.MinConns = 1
	cfg.MaxConns = 8
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalf("[wallet-grpc] connect db: %v", err)
	}
	defer pool.Close()

	// Buat tabel reservations (PoC) jika belum ada
	if err := ensureSchema(context.Background(), pool); err != nil {
		log.Fatalf("[wallet-grpc] ensure schema: %v", err)
	}

	// === gRPC server + metrics interceptors ===
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(gp.UnaryServerInterceptor),
		grpc.StreamInterceptor(gp.StreamServerInterceptor),
	)
	gp.Register(grpcServer)

	walletv1.RegisterWalletServiceServer(grpcServer, &server{pool: pool})

	// === gRPC listener ===
	grpcAddr := getenv("GRPC_ADDR", ":9093")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("[wallet-grpc] listen error on %s: %v", grpcAddr, err)
	}
	go func() {
		log.Printf("[wallet-grpc] serving gRPC on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("[wallet-grpc] serve error: %v", err)
		}
	}()

	// === Prometheus /metrics ===
	metricsAddr := getenv("METRICS_ADDR", ":9103")
	http.Handle("/metrics", promhttp.Handler())
	log.Printf("[wallet-grpc] serving metrics on %s /metrics", metricsAddr)
	log.Fatal(http.ListenAndServe(metricsAddr, nil))
}

// ======== RPC implementations ========

func (s *server) GetRandomAccounts(ctx context.Context, req *walletv1.GetRandomAccountsRequest) (*walletv1.GetRandomAccountsResponse, error) {
	n := req.GetCount()
	if n == 0 {
		n = 2
	}

	q := fmt.Sprintf(`
		SELECT account_id
		FROM wallet_accounts
		ORDER BY random()
		LIMIT %d;
	`, n)

	rows, err := s.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query random accounts: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	if uint32(len(ids)) < n {
		return nil, fmt.Errorf("not enough accounts (need %d, got %d)", n, len(ids))
	}
	return &walletv1.GetRandomAccountsResponse{AccountIds: ids}, nil
}


func (s *server) GetBalance(ctx context.Context, req *walletv1.GetBalanceRequest) (*walletv1.GetBalanceResponse, error) {
	if req.GetAccountId() == "" {
		return nil, errors.New("account_id required")
	}
	const q = `SELECT balance_idr FROM wallet_accounts WHERE account_id=$1`
	var bal int64
	err := s.pool.QueryRow(ctx, q, req.GetAccountId()).Scan(&bal)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("account not found")
		}
		return nil, fmt.Errorf("query balance: %w", err)
	}
	return &walletv1.GetBalanceResponse{
		BalanceMinor: bal,
		Currency:     commonv1.Currency_IDR,
	}, nil
}

func (s *server) Reserve(ctx context.Context, req *walletv1.ReserveRequest) (*walletv1.ReserveResponse, error) {
	if req.GetAccountId() == "" || req.GetPaymentId() == "" {
		return &walletv1.ReserveResponse{Ok: false, Reason: "account_id and payment_id required"}, nil
	}
	if req.GetAmountMinor() <= 0 {
		return &walletv1.ReserveResponse{Ok: false, Reason: "amount_minor must be > 0"}, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// PoC sederhana: langsung potong saldo saat reserve (tanpa kolom reserved terpisah)
	cmd, err := tx.Exec(ctx, `
		UPDATE wallet_accounts
		SET balance_idr = balance_idr - $2, updated_at = now()
		WHERE account_id = $1 AND balance_idr >= $2
	`, req.GetAccountId(), req.GetAmountMinor())
	if err != nil {
		return nil, fmt.Errorf("update balance: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return &walletv1.ReserveResponse{Ok: false, Reason: "insufficient balance"}, nil
	}

	resID := uuid.New().String()
	_, err = tx.Exec(ctx, `
		INSERT INTO wallet_reservations
			(reservation_id, payment_id, account_id, amount_minor, currency, status)
		VALUES ($1, $2, $3, $4, $5, 'RESERVED')
	`, resID, req.GetPaymentId(), req.GetAccountId(), req.GetAmountMinor(), req.GetCurrency().String())
	if err != nil {
		return nil, fmt.Errorf("insert reservation: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return &walletv1.ReserveResponse{Ok: true, ReservationId: resID}, nil
}

func (s *server) Capture(ctx context.Context, req *walletv1.CaptureRequest) (*walletv1.CaptureResponse, error) {
	if req.GetReservationId() == "" {
		return &walletv1.CaptureResponse{Ok: false, Reason: "reservation_id required"}, nil
	}
	cmd, err := s.pool.Exec(ctx, `
		UPDATE wallet_reservations
		SET status = 'CAPTURED'
		WHERE reservation_id = $1 AND status = 'RESERVED'
	`, req.GetReservationId())
	if err != nil {
		return nil, fmt.Errorf("update reservation: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return &walletv1.CaptureResponse{Ok: false, Reason: "invalid reservation or already captured"}, nil
	}
	return &walletv1.CaptureResponse{Ok: true}, nil
}

// ======== helpers ========

func ensureSchema(ctx context.Context, pool *pgxpool.Pool) error {
	// wallet_accounts diasumsikan sudah ada (sesuai skema kamu).
	// Buat tabel reservations minimal untuk PoC.
	sql := `
CREATE TABLE IF NOT EXISTS wallet_reservations (
  reservation_id UUID PRIMARY KEY,
  payment_id     TEXT UNIQUE NOT NULL,
  account_id     VARCHAR NOT NULL,
  amount_minor   BIGINT  NOT NULL,
  currency       TEXT    NOT NULL,
  status         TEXT    NOT NULL DEFAULT 'RESERVED', -- RESERVED | CAPTURED | CANCELED
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
`
	_, err := pool.Exec(ctx, sql)
	return err
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
