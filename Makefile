.PHONY: generate clean

# Generate protobuf files
generate:
	buf generate

# Clean generated files
clean:
	rm -f api/v1/*.pb.go

# Install required tools
tools:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

# Run buf lint
lint:
	buf lint
