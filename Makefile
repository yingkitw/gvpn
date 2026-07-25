.PHONY: all build clean run-server run-client test deps

all: build

deps:
	go mod download
	go mod tidy

build: deps
	@echo "Building VPN server and client..."
	@mkdir -p bin
	go build -o bin/server ./cmd/server
	go build -o bin/client ./cmd/client
	@echo "Build complete!"

run-server: build
	@echo "Starting VPN server..."
	sudo ./bin/server

run-client: build
	@echo "Starting VPN client..."
	./bin/client

clean:
	@echo "Cleaning up..."
	rm -rf bin/
	@echo "Clean complete!"

test:
	go test -v ./...

lint:
	golangci-lint run

fmt:
	go fmt ./...

vendor:
	go mod vendor

install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest