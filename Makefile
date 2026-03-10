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

# ── RELEASE TARGETS ───────────────────────────────────
DIST_DIR := dist
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build-all checksums clean-dist

build-all: clean-dist
	@echo "Building all platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-linux-amd64   ./cmd/nexus
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-darwin-amd64  ./cmd/nexus
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-darwin-arm64  ./cmd/nexus
	@echo "Built binaries in $(DIST_DIR)/"

checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && for f in nexus-*; do \
		sha256sum "$$f" > "$$f.sha256"; \
		echo "  $$f.sha256"; \
	done
	@echo "Done"

clean-dist:
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)

# ── RELEASE TARGETS ───────────────────────────────────
DIST_DIR := dist
VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: build-all checksums clean-dist

build-all: clean-dist
	@echo "Building all platforms..."
	@mkdir -p $(DIST_DIR)
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-linux-amd64   ./cmd/nexus
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-darwin-amd64  ./cmd/nexus
	GOOS=darwin  GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(DIST_DIR)/nexus-darwin-arm64  ./cmd/nexus
	@echo "Built binaries in $(DIST_DIR)/"

checksums:
	@echo "Generating checksums..."
	@cd $(DIST_DIR) && for f in nexus-*; do \
		sha256sum "$$f" > "$$f.sha256"; \
		echo "  $$f.sha256"; \
	done
	@echo "Done"

clean-dist:
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_DIR)
