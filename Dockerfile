# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install required tools
RUN apk add --no-cache git make

# Install protoc and dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 go build -o /bin/server ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /bin/server /bin/server

EXPOSE 8080 9090
ENTRYPOINT ["/bin/server"]