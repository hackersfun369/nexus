.PHONY: build test lint clean install run

build:
	go build -o nexus ./cmd/nexus/

test:
	go test ./... -v

test-short:
	go test ./... -short -v

bench:
	go test ./tests/bench/... -bench=. -benchtime=3s

lint:
	go vet ./...

clean:
	rm -f nexus
	go clean

install:
	go install ./cmd/nexus/

run:
	go run ./cmd/nexus/
