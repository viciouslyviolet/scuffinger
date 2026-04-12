APP_NAME := scuffinger
DOCKER_COMPOSE := docker compose

.PHONY: build test run stop clean

## build: Compile the Go binary
build:
	@echo "==> Building $(APP_NAME)..."
	go build -ldflags="-s -w" -o bin/$(APP_NAME) .
	@echo "==> Done. Binary at bin/$(APP_NAME)"

## test: Run all tests
test:
	@echo "==> Running tests..."
	go test ./... -v
	@echo "==> Tests complete."

## run: Build and start all services with Docker Compose
run:
	@echo "==> Starting services..."
	$(DOCKER_COMPOSE) up --build -d
	@echo "==> Services running. App available at http://localhost:8080"

## stop: Stop and remove all Docker Compose services
stop:
	@echo "==> Stopping services..."
	$(DOCKER_COMPOSE) down
	@echo "==> Services stopped."

## clean: Remove build artifacts and data volumes
clean:
	@echo "==> Cleaning..."
	rm -rf bin/
	$(DOCKER_COMPOSE) down -v
	rm -rf data/
	@echo "==> Clean complete."

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'

