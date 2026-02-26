.PHONY: build test lint fmt ci clean

build:
	go build -o bin/agent ./cmd/agent
	go build -o bin/ctl ./cmd/ctl

test:
	go test ./... -count=1 -race

lint:
	golangci-lint run ./...

fmt:
	gofmt -s -w .

# CI: format check + test (+ lint if golangci-lint available)
ci: fmt
	@test -z "$$(gofmt -l .)" || (echo "run 'make fmt'"; exit 1)
	go test ./... -count=1 -race
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run ./... || true

clean:
	rm -rf bin/
