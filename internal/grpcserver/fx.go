// internal/grpcserver/fx.go
// Example complete gRPC server file structure
// File: internal/grpcserver/fx.go
package grpcserver

import (
    "context"
    "log"
    
    // Essential gRPC imports
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/reflection"
    
    // Your proto imports (adjust path as needed)
    fxv1 "github.com/example/payment-gateway-poc/proto/gen/fx/v1"
)

// FxServer implements the FX service
type FxServer struct {
    fxv1.UnimplementedFxServiceServer // Embed for forward compatibility
}

// NewFxServer creates a new FX service server
func NewFxServer() *FxServer {
    return &FxServer{}
}

// Example method implementation
func (s *FxServer) GetExchangeRate(ctx context.Context, req *fxv1.GetExchangeRateRequest) (*fxv1.GetExchangeRateResponse, error) {
    // Validate request
    if req == nil {
        return nil, status.Errorf(codes.InvalidArgument, "request cannot be nil")
    }
    
    if req.FromCurrency == "" || req.ToCurrency == "" {
        return nil, status.Errorf(codes.InvalidArgument, "from_currency and to_currency are required")
    }
    
    // Your business logic here...
    rate := 1.0 // Placeholder
    
    return &fxv1.GetExchangeRateResponse{
        FromCurrency: req.FromCurrency,
        ToCurrency:   req.ToCurrency,
        Rate:         rate,
    }, nil
}

// RegisterServer registers the FX server with a gRPC server
func (s *FxServer) RegisterServer(grpcServer *grpc.Server) {
    fxv1.RegisterFxServiceServer(grpcServer, s)
    
    // Enable reflection for development (remove in production)
    reflection.Register(grpcServer)
    
    log.Println("FX gRPC server registered")
}