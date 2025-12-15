.PHONY: build test lint

build:
	go build -o fdup .

test:
	go test ./...

lint:
	@echo "Running Go lint..."
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.0.2 run ./...
