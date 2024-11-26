// cmd/test/main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	drlv1 "github.com/seatedro/drl/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type HTTPResponse struct {
	Allowed    bool    `json:"allowed"`
	Remaining  int     `json:"remaining"`
	ResetAfter float64 `json:"reset_after_sec"`
	RetryAfter float64 `json:"retry_after_sec,omitempty"`
}

func burstRequests(count int, delay time.Duration) {
	log.Printf("Starting burst test with %d requests, %v delay between requests", count, delay)
}

func testHTTP(wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	log.Println("=== Starting HTTP Test ===")

	// Burst test: Send 150 requests quickly (should hit rate limit as burst is 150)
	for i := 0; i < 150; i++ {
		resp, err := client.Post(
			"http://ratelimiter:8080/v1/allow/test-key?namespace=testing",
			"application/json",
			nil,
		)
		if err != nil {
			results <- fmt.Sprintf("❌ HTTP error: %v", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		var response HTTPResponse
		json.Unmarshal(body, &response)

		if resp.StatusCode == http.StatusTooManyRequests {
			results <- fmt.Sprintf("🚫 HTTP request %d: Rate limited (Retry after: %.2f seconds)",
				i+1, response.RetryAfter)
		} else if resp.StatusCode != http.StatusOK {
			results <- fmt.Sprintf("❌ HTTP request %d: Unexpected status: %d", i+1, resp.StatusCode)
		} else {
			results <- fmt.Sprintf("✅ HTTP request %d: Allowed (Remaining: %d)",
				i+1, response.Remaining)
		}

		resp.Body.Close()
		time.Sleep(10 * time.Millisecond) // Small delay to make output readable
	}

	// Wait a bit and then try a few more requests to see the recovery
	time.Sleep(5 * time.Second)
	results <- "\n=== After 5 second wait ===\n"

	for i := 0; i < 5; i++ {
		resp, err := client.Post(
			"http://ratelimiter:8080/v1/allow/test-key?namespace=testing",
			"application/json",
			nil,
		)
		if err != nil {
			results <- fmt.Sprintf("❌ HTTP error: %v", err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		var response HTTPResponse
		json.Unmarshal(body, &response)

		if resp.StatusCode == http.StatusTooManyRequests {
			results <- fmt.Sprintf("🚫 Recovery request %d: Still rate limited (Retry after: %.2f seconds)",
				i+1, response.RetryAfter)
		} else {
			results <- fmt.Sprintf("✅ Recovery request %d: Allowed (Remaining: %d)",
				i+1, response.Remaining)
		}

		resp.Body.Close()
		time.Sleep(100 * time.Millisecond)
	}
}

func testGRPC(wg *sync.WaitGroup, results chan<- string) {
	defer wg.Done()

	conn, err := grpc.NewClient(
		"ratelimiter:9090",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		results <- fmt.Sprintf("❌ gRPC connection error: %v", err)
		return
	}
	defer conn.Close()

	client := drlv1.NewRateLimiterClient(conn)

	log.Println("=== Starting gRPC Test ===")

	// Burst test
	for i := 0; i < 150; i++ {
		resp, err := client.Allow(context.Background(), &drlv1.AllowRequest{
			Key:       "test-key",
			Namespace: "testing-grpc",
		})
		if err != nil {
			results <- fmt.Sprintf("❌ gRPC error: %v", err)
			continue
		}

		if !resp.Allowed {
			results <- fmt.Sprintf("🚫 gRPC request %d: Rate limited (Retry after: %d seconds)",
				i+1, resp.RetryAfterSec)
		} else {
			results <- fmt.Sprintf("✅ gRPC request %d: Allowed (Remaining: %d)",
				i+1, resp.Remaining)
		}

		time.Sleep(10 * time.Millisecond) // Small delay to make output readable
	}

	// Wait and test recovery
	time.Sleep(5 * time.Second)
	results <- "\n=== After 5 second wait ===\n"

	for i := 0; i < 5; i++ {
		resp, err := client.Allow(context.Background(), &drlv1.AllowRequest{
			Key:       "test-key",
			Namespace: "testing-grpc",
		})
		if err != nil {
			results <- fmt.Sprintf("❌ gRPC error: %v", err)
			continue
		}

		if !resp.Allowed {
			results <- fmt.Sprintf("🚫 Recovery request %d: Still rate limited (Retry after: %d seconds)",
				i+1, resp.RetryAfterSec)
		} else {
			results <- fmt.Sprintf("✅ Recovery request %d: Allowed (Remaining: %d)",
				i+1, resp.Remaining)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	log.Println("Waiting for rate limiter service to be ready...")
	time.Sleep(5 * time.Second)

	results := make(chan string, 1000)
	var wg sync.WaitGroup

	// Run HTTP and gRPC tests
	wg.Add(2)
	go testHTTP(&wg, results)
	go testGRPC(&wg, results)

	// Wait for tests to complete in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Print results as they come in
	for result := range results {
		log.Println(result)
	}
}
