// internal/service/grpc.go
package service

import (
	"context"
	"fmt"

	"github.com/seatedro/drl"
	drlv1 "github.com/seatedro/drl/api/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GRPCService struct {
	drlv1.UnimplementedRateLimiterServer
	limiter drl.RateLimiter
}

func NewGRPCService(limiter drl.RateLimiter) *GRPCService {
	return &GRPCService{
		limiter: limiter,
	}
}

func (s *GRPCService) formatKey(namespace, key string) string {
	if namespace == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", namespace, key)
}

func (s *GRPCService) Allow(ctx context.Context, req *drlv1.AllowRequest) (*drlv1.AllowResponse, error) {
	key := s.formatKey(req.Namespace, req.Key)

	allowed, remaining, resetAfter, err := s.limiter.Allow(ctx, key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "rate limit check failed: %v", err)
	}

	resp := &drlv1.AllowResponse{
		Allowed:       allowed,
		Remaining:     int32(remaining),
		ResetAfterSec: int64(resetAfter.Seconds()),
	}

	if !allowed {
		resp.RetryAfterSec = resp.ResetAfterSec
	}

	return resp, nil
}

func (s *GRPCService) Reset(ctx context.Context, req *drlv1.ResetRequest) (*drlv1.ResetResponse, error) {
	key := s.formatKey(req.Namespace, req.Key)

	err := s.limiter.Reset(ctx, key)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "reset failed: %v", err)
	}

	return &drlv1.ResetResponse{
		Success: true,
	}, nil
}
