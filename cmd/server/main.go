// cmd/server/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/seatedro/drl"
	drlv1 "github.com/seatedro/drl/api/v1"
	"github.com/seatedro/drl/internal/service"
	"github.com/seatedro/drl/store/redis"
)

func getEnvInt(key string, defaultVal int) int {
	if val, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}

func main() {
	// Parse flags
	httpAddr := flag.String("http-addr", ":8080", "HTTP server address")
	grpcAddr := flag.String("grpc-addr", ":9090", "gRPC server address")
	redisAddr := flag.String("redis-addr", "localhost:6379", "Redis address")
	flag.Parse()

	// Get configuration from environment
	rate := getEnvInt("RATE_LIMIT", 100)
	window := getEnvDuration("RATE_WINDOW", time.Minute)
	burstSize := getEnvInt("BURST_SIZE", 150)

	log.Printf("Starting rate limiter with config: rate=%d/%.0fs burst=%d",
		rate, window.Seconds(), burstSize)

	// Create Redis store
	store, err := redis.NewStore(redis.Options{
		Addresses: []string{*redisAddr},
	})
	if err != nil {
		log.Fatalf("Failed to create Redis store: %v", err)
	}

	// Create rate limiter
	rateLimiter, err := drl.New(drl.Config{
		Rate:      rate,
		Window:    window,
		BurstSize: burstSize,
	}, store)
	if err != nil {
		log.Fatalf("Failed to create rate limiter: %v", err)
	}

	// Create services
	grpcService := service.NewGRPCService(rateLimiter)
	httpService := service.NewHTTPService(rateLimiter)

	// Create gRPC server
	grpcServer := grpc.NewServer()
	drlv1.RegisterRateLimiterServer(grpcServer, grpcService)
	reflection.Register(grpcServer)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    *httpAddr,
		Handler: httpService,
	}

	// Run servers
	errChan := make(chan error, 1)

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			errChan <- fmt.Errorf("failed to listen on gRPC addr: %w", err)
			return
		}

		log.Printf("gRPC server listening on %s", *grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start HTTP server
	go func() {
		log.Printf("HTTP server listening on %s", *httpAddr)
		if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Printf("Server error: %v", err)
	case sig := <-sigChan:
		log.Printf("Received signal: %v", sig)
	}

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	grpcServer.GracefulStop()

	log.Println("Servers shut down")
}
