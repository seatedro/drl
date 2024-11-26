#!/bin/bash
# test.sh

echo "Waiting for rate limiter to be ready..."
sleep 5

echo "=== Rate Limiter Load Tests ==="

# Function to run hey and parse results
run_http_test() {
    local name=$1
    local url=$2
    local args=$3
    
    echo -e "\n=== HTTP: $name ==="
    echo "Running: hey $args -m POST $url"
    
    output=$(hey $args -m POST "$url")
    
    echo -e "\nResults:"
    echo "$output" | grep -A 1 "Status code distribution:"
    echo "$output" | grep "Requests/sec:"
}

# Function to run ghz and parse results
run_grpc_test() {
    local name=$1
    local args=$2
    local key=$3
    
    echo -e "\n=== gRPC: $name ==="
    echo "Running: ghz $args"
    
    ghz \
        --insecure \
        --proto /app/api/v1/drl.proto \
        --call drl.v1.RateLimiter.Allow \
        --data '{"key": "'"${key}"'", "namespace": "testing-grpc"}' \
        --format pretty \
        $args \
        ratelimiter:9090
}

echo "🌐 Testing HTTP Endpoints"
echo "========================"

# Test 1: HTTP Burst Test
run_http_test "Burst Test" \
    "http://ratelimiter:8080/v1/allow/burst-test?namespace=testing" \
    "-n 50 -c 10"

# Test 2: HTTP Sustained Load
run_http_test "Sustained Load" \
    "http://ratelimiter:8080/v1/allow/sustained-test?namespace=testing" \
    "-n 30 -c 2 -q 2"

# Test 3: HTTP High Concurrency
run_http_test "High Concurrency" \
    "http://ratelimiter:8080/v1/allow/concurrent-test?namespace=testing" \
    "-n 100 -c 20"

echo -e "\n\n📡 Testing gRPC Endpoints"
echo "========================="

# Test 1: gRPC Burst Test
run_grpc_test "Burst Test" \
    "--connections 10 --concurrency 10 --total 50 --duration=5s" \
    "burst-test"

echo -e "\nWaiting 5 seconds for rate limit window..."
sleep 5

# Test 2: gRPC Sustained Load
run_grpc_test "Sustained Load" \
    "--connections 2 --concurrency 2 --total 30 --rps 2" \
    "sustained-test"

echo -e "\nWaiting 5 seconds for rate limit window..."
sleep 5

# Test 3: gRPC High Concurrency
run_grpc_test "High Concurrency" \
    "--connections 20 --concurrency 20 --total 100" \
    "concurrent-test"

# Test 4: Recovery Test for both protocols
echo -e "\n♻️  Testing Rate Limit Recovery (Both Protocols)"

echo "HTTP Recovery Test:"
run_http_test "Initial HTTP Burst" \
    "http://ratelimiter:8080/v1/allow/recovery-test?namespace=testing" \
    "-n 10 -c 5"

echo "gRPC Recovery Test:"
run_grpc_test "Initial gRPC Burst" \
    "--connections 5 --concurrency 5 --total 10" \
    "recovery-test"

echo -e "\nWaiting for rate limit window to expire (5 seconds)..."
sleep 5

echo "HTTP Recovery Burst:"
run_http_test "HTTP Recovery" \
    "http://ratelimiter:8080/v1/allow/recovery-test?namespace=testing" \
    "-n 10 -c 5"

echo "gRPC Recovery Burst:"
run_grpc_test "gRPC Recovery" \
    "--connections 5 --concurrency 5 --total 10" \
    "recovery-test"

echo -e "\n=== Test Summary ==="
echo "✓ HTTP Endpoints tested"
echo "  ✓ Burst capacity"
echo "  ✓ Sustained load"
echo "  ✓ High concurrency"
echo "  ✓ Rate limit recovery"
echo "✓ gRPC Endpoints tested"
echo "  ✓ Burst capacity"
echo "  ✓ Sustained load"
echo "  ✓ High concurrency"
echo "  ✓ Rate limit recovery"
