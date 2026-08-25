.PHONY: test race vet benchmark run build

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

benchmark:
	go run ./cmd/benchmark -suite configs/benchmark.json

run:
	go run ./cmd/gateway -config configs/local.json

build:
	go build ./cmd/gateway ./cmd/benchmark
