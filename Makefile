.PHONY: all server tools lsh duh clean test test-integration

# Output directory
BIN := bin

# Default target
all: server tools

# Server binary
server: $(BIN)/goshell

$(BIN)/goshell: cmd/goshell/*.go
	@mkdir -p $(BIN)
	go build -o $(BIN)/goshell ./cmd/goshell

# All tool binaries
tools: lsh duh

# Individual tools
lsh: $(BIN)/lsh

$(BIN)/lsh: cmd/lsh/*.go
	@mkdir -p $(BIN)
	go build -o $(BIN)/lsh ./cmd/lsh

duh: $(BIN)/duh

$(BIN)/duh: cmd/duh/*.go
	@mkdir -p $(BIN)
	go build -o $(BIN)/duh ./cmd/duh

# Run unit tests (default, fast)
test:
	go test ./...

# Run all tests including integration tests
test-integration:
	go test -tags=integration ./...

# Clean build artifacts
clean:
	rm -rf $(BIN)
