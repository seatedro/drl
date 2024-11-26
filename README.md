# Distributed Rate Limiter (DRL)

A high-performance, distributed rate limiter built in Go, supporting both HTTP and gRPC protocols. Uses Redis for distributed state management and implements the Token Bucket algorithm for precise rate limiting.

## Features

- **Distributed Rate Limiting**: Consistent rate limiting across multiple service instances
- **Dual Protocol Support**: Both HTTP and gRPC interfaces
- **Token Bucket Algorithm**: Efficient and flexible rate limiting with burst support
- **Redis Backend**: Scalable and reliable state management
- **Configurable Limits**: Adjust rates, windows, and burst sizes via environment variables
- **Docker Ready**: Easy deployment with Docker and Docker Compose

## Architecture

```
[Client Requests] 
       ↓
[Load Balancer]
       ↓
[Rate Limiter Service Instances] ←→ [Redis Cluster]
```

## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Redis (for development)

### Installation

1. Clone the repository:
```bash
git clone https://github.com/seatedro/drl.git
cd drl
```

2. Install dependencies:
```bash
go mod download
```

3. Generate protocol buffers:
```bash
make generate
```

### Running with Docker Compose

```bash
docker-compose up --build
```

This will start:
- Rate limiter service (HTTP :8080, gRPC :9090)
- Redis instance
- Load testing service

## Usage

### HTTP API

```bash
# Check if request is allowed
curl -X POST "http://localhost:8080/v1/allow/my-key?namespace=myapp"

# Reset rate limit for a key
curl -X POST "http://localhost:8080/v1/reset/my-key?namespace=myapp"
```

### gRPC API

```bash
# Using grpcurl
grpcurl -d '{"key": "my-key", "namespace": "myapp"}' \
    -plaintext localhost:9090 drl.v1.RateLimiter/Allow
```

### Response Headers (HTTP)

```
X-RateLimit-Remaining: <remaining requests>
X-RateLimit-Reset: <reset timestamp>
Retry-After: <seconds until next available request>
```

## Configuration

Environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| RATE_LIMIT | Requests allowed per window | 100 |
| RATE_WINDOW | Time window in seconds | 60 |
| BURST_SIZE | Maximum burst size | 150 |
| DEBUG | Enable debug logging | false |

## Load Testing

The project includes comprehensive load testing for both HTTP and gRPC endpoints:

```bash
# Run load tests
docker-compose up loadtest
```

Test scenarios include:
- Burst capacity testing
- Sustained load testing
- High concurrency testing
- Rate limit recovery testing
