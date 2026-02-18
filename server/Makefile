.PHONY: build run test check-ddd check-architecture check-layer-dependencies check-domain-purity check-no-legacy-layer check-dto-selftest check-repo-selftest check-layer-selftest check-wiring-selftest check-guards-selftest check-common-httpdto check-naming-conventions check-mapper-naming check-wiring-conventions sync-common-httpdto lint clean deps fmt generate check-router-boundaries check-router-selftest check-handler-boundaries check-handler-selftest

# Build the server binary
build:
	go build -o bin/server ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Architecture guards (DDD-Strict + DTO + Repository + HTTPDTO)
check-architecture:
	bash ./scripts/check-no-legacy-layer.sh
	bash ./scripts/check-layer-dependencies.sh
	bash ./scripts/check-domain-purity.sh
	bash ./scripts/check-ddd-boundaries.sh
	bash ./scripts/check-dto-boundaries.sh
	bash ./scripts/check-repository-boundaries.sh
	bash ./scripts/check-naming-conventions.sh
	bash ./scripts/check-mapper-naming.sh
	bash ./scripts/check-router-boundaries.sh
	bash ./scripts/check-handler-boundaries.sh
	bash ./scripts/check-wiring-conventions.sh
	bash ./scripts/check-common-httpdto.sh

check-naming-conventions:
	bash ./scripts/check-naming-conventions.sh

check-mapper-naming:
	bash ./scripts/check-mapper-naming.sh

check-wiring-conventions:
	bash ./scripts/check-wiring-conventions.sh

check-layer-dependencies:
	bash ./scripts/check-layer-dependencies.sh

check-domain-purity:
	bash ./scripts/check-domain-purity.sh

check-no-legacy-layer:
	bash ./scripts/check-no-legacy-layer.sh

# Keep backward-compatible entrypoint
check-ddd:
	$(MAKE) check-architecture

# Local optional: DTO guard self-test
check-dto-selftest:
	bash ./scripts/check-dto-boundaries-selftest.sh

# Local optional: repository guard self-test
check-repo-selftest:
	bash ./scripts/check-repository-boundaries-selftest.sh

# Local optional: layer dependency guard self-test
check-layer-selftest:
	bash ./scripts/check-layer-dependencies-selftest.sh

# Local optional: wiring conventions guard self-test
check-wiring-selftest:
	bash ./scripts/check-wiring-conventions-selftest.sh

# Local optional: all guards self-test
check-guards-selftest:
	bash ./scripts/check-guards-selftest.sh

# Router boundary guard
check-router-boundaries:
	bash ./scripts/check-router-boundaries.sh

# Local optional: router boundary guard self-test
check-router-selftest:
	bash ./scripts/check-router-boundaries-selftest.sh

# Handler boundary guard
check-handler-boundaries:
	bash ./scripts/check-handler-boundaries.sh

# Local optional: handler boundary guard self-test
check-handler-selftest:
	bash ./scripts/check-handler-boundaries-selftest.sh

# Check common_http template consistency
check-common-httpdto:
	bash ./scripts/check-common-httpdto.sh

# Sync common_http files from template
sync-common-httpdto:
	bash ./scripts/sync-common-httpdto.sh

# Run tests
test:
	go test -v ./...

# Run tests with coverage report
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Generate code (if configured)
generate:
	go generate ./...
