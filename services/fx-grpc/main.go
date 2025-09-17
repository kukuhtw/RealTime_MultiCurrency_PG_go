// services/fx-rpc/main.go 
// services/fx-rpc/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	pb "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedFxServiceServer
}

// bentuk JSON yang akan disimpan: { "rates": [ ... ] }
type rateJSON struct {
	BaseCurrency  string  `json:"base_currency"`
	QuoteCurrency string  `json:"quote_currency"`
	Rate          float64 `json:"rate"`
}
type filePayload struct {
	Rates []rateJSON `json:"rates"`
}

func (s *server) UpdateRates(ctx context.Context, req *pb.UpdateRatesRequest) (*pb.UpdateRatesResponse, error) {
	// log ke console
	for _, r := range req.Rates {
		log.Printf("[FX] Update rate: %s/%s = %.6f", r.BaseCurrency, r.QuoteCurrency, r.Rate)
	}

	// map dari protobuf ke bentuk JSON
	out := make([]rateJSON, 0, len(req.Rates))
	for _, r := range req.Rates {
		out = append(out, rateJSON{
			BaseCurrency:  r.BaseCurrency,
			QuoteCurrency: r.QuoteCurrency,
			Rate:          r.Rate,
		})
	}

	// tulis ke seeds/fx_rates.json (atomic write)
	if err := writeRatesJSON("seeds", "fx_rates.json", filePayload{Rates: out}); err != nil {
		log.Printf("[FX] write JSON error: %v", err)
		return &pb.UpdateRatesResponse{Success: false, Message: fmt.Sprintf("write error: %v", err)}, nil
	}

	return &pb.UpdateRatesResponse{Success: true, Message: "Rates updated & saved to seeds/fx_rates.json"}, nil
}

func writeRatesJSON(dir, filename string, payload filePayload) error {
	// pastikan folder ada
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	path := filepath.Join(dir, filename)
	tmp := path + ".tmp"

	// tulis ke file sementara dulu, lalu rename (atomic)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

/*
 // Jika kamu BENAR-BENAR ingin format "rates = [ ... ]" (JavaScript, bukan JSON),
 // ganti fungsi di atas dengan versi ini:

func writeRatesJS(dir, filename string, payload filePayload) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// hanya arraynya
	arr, err := json.MarshalIndent(payload.Rates, "", "  ")
	if err != nil {
		return err
	}
	content := append([]byte("rates = "), arr...)
	content = append(content, byte('\n'))
	path := filepath.Join(dir, filename)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
*/

func main() {
	lis, err := net.Listen("tcp", ":9102")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterFxServiceServer(s, &server{})
	log.Println("FX gRPC server listening on :9102")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
