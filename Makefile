.PHONY: build run test clean install release web version

VERSION    ?= dev
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -s -w \
	-X github.com/hackersfun369/nexus/internal/version.Version=$(VERSION) \
	-X github.com/hackersfun369/nexus/internal/version.Commit=$(COMMIT) \
	-X github.com/hackersfun369/nexus/internal/version.BuildDate=$(DATE)

## Build web UI then Go binary
build: web
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o bin/nexus ./cmd/nexus

## Build without rebuilding web
build-go:
	CGO_ENABLED=1 go build -ldflags "$(LDFLAGS)" -o bin/nexus ./cmd/nexus

## Build React frontend
web:
	cd web && pnpm install --frozen-lockfile && pnpm build

## Run the server
run: build
	./bin/nexus serve

## Run all tests
test:
	go test ./... -timeout 120s

## Run tests with verbose output
test-v:
	go test ./... -v -timeout 120s

## Install to /usr/local/bin
install: build
	cp bin/nexus /usr/local/bin/nexus
	@echo "✓ nexus installed to /usr/local/bin/nexus"

## Cross-compile for all platforms (CGO_ENABLED=0 for cross)
release: web
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/nexus-linux-amd64   ./cmd/nexus
	GOOS=linux   GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/nexus-linux-arm64   ./cmd/nexus
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/nexus-darwin-amd64  ./cmd/nexus
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/nexus-darwin-arm64  ./cmd/nexus
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o dist/nexus-windows-amd64.exe ./cmd/nexus
	cd dist && sha256sum nexus-* > checksums.txt
	@echo "✓ Binaries in dist/"
	@ls -lh dist/

## Tag and push a release
tag:
	@read -p "Version (e.g. v0.1.0): " v && \
	git tag -a "$$v" -m "Release $$v" && \
	git push origin "$$v" && \
	echo "✓ Tagged and pushed $$v — GitHub Actions will build the release"

## Show version
version:
	@go run -ldflags "$(LDFLAGS)" ./cmd/nexus version

## Clean build artifacts
clean:
	rm -rf bin/ dist/

## Show help
help:
	@echo "NEXUS build targets:"
	@echo "  make build    — build web + binary"
	@echo "  make test     — run all tests"
	@echo "  make release  — cross-compile all platforms"
	@echo "  make install  — install to /usr/local/bin"
	@echo "  make tag      — tag and push a release"
	@echo "  make clean    — remove build artifacts"
