.PHONY: build run test test-okx clean docker-up docker-down generate help

TEST_PACKAGES = $(shell go list ./... | grep -v '/tests$$')
GQLGEN_VERSION = $(shell go list -m -f '{{.Version}}' github.com/99designs/gqlgen)

help:
	@echo "Available commands:"
	@echo "  build       - Build the application"
	@echo "  serve       - Build and serve the application"
	@echo "  run         - Run the application"
	@echo "  test        - Run default regression tests"
	@echo "  test-okx    - Run OKX integration tests"
	@echo "  clean       - Clean build artifacts"
	@echo "  docker-up   - Start Docker services"
	@echo "  docker-down - Stop Docker services"
	@echo "  generate    - Generate GraphQL code"

build:
	go build -o bin/degov-server cmd/main.go

run:
	go run cmd/main.go

serve: build
	./bin/degov-server

# serve with logging to file
serve-with-log: build
	@mkdir -p _tmp/logs
	./bin/degov-server 2>&1 | tee _tmp/logs/serve-$$(date +%Y%m%d-%H%M%S).log.txt

test: generate
	go test $(TEST_PACKAGES)

test-okx: generate
	go test ./tests/...

clean:
	rm -rf bin/
	rm -rf .data/

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

generate:
	go get github.com/99designs/gqlgen@$(GQLGEN_VERSION)
	go generate ./...
	go mod tidy

deps:
	go mod tidy
	go mod download

# complete setup for development environment
setup: generate deps docker-up
	@echo "Setting up development environment..."
	@sleep 2
	@echo "Development environment ready!"
	@echo "Run 'make run' to start the application"

reset: docker-down clean setup serve
	@echo "Resetting development environment..."
	@sleep 2
	@echo "Development environment reset!"
