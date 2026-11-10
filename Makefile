.PHONY: all test build up down clean test-backend build-frontend

all: test build

test: test-backend

test-backend:
	cd backend && go test -v ./...

build: build-backend build-frontend

build-backend:
	cd backend && go build -o ../reconpilot-api ./cmd/api
	cd backend && go build -o ../reconpilot-cli ./cmd/cli

build-frontend:
	cd frontend && npm install && npm run build

up:
	docker-compose up -d --build

down:
	docker-compose down

clean:
	rm -f reconpilot-api reconpilot-cli reconpilot-api.exe reconpilot-cli.exe
	rm -rf frontend/dist

