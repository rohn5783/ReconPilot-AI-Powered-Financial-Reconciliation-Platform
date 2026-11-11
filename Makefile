.PHONY: all build build-backend build-frontend test run-api run-cli seed docker-up docker-down clean

all: build

build: build-backend build-frontend

build-backend:
	@echo "Building ReconPilot Golang API and CLI binaries..."
	@mkdir -p bin
	cd backend && go build -ldflags="-w -s" -o ../bin/reconpilot-api ./cmd/api
	cd backend && go build -ldflags="-w -s" -o ../bin/reconpilot ./cmd/cli

build-frontend:
	@echo "Building React 19 Frontend workstation..."
	cd frontend && npm run build

test: test-backend test-frontend

test-backend:
	@echo "Running Golang test suites..."
	cd backend && go test -v -race ./...

test-frontend:
	@echo "Typechecking React frontend..."
	cd frontend && npm run build

run-api:
	cd backend && go run ./cmd/api

run-cli:
	cd backend && go run ./cmd/cli

seed:
	@echo "Generating sample CSV files in ./samples..."
	@mkdir -p samples
	./bin/reconpilot seed --output-dir ./samples

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down -v

clean:
	@rm -rf bin samples frontend/dist
	@echo "Cleaned build artifacts."
