.PHONY: build run test test-integration migrate-up migrate-down migrate-create seed proto-gen lint clean docker-up docker-down

# Variables
APP_NAME = skintrader-api
DB_URL ?= postgres://skintrader:skintrader_secret@localhost:5432/skintrader?sslmode=disable
MIGRATE = migrate -path migrations -database "$(DB_URL)"

# Build
build:
	go build -ldflags="-w -s" -o bin/$(APP_NAME) ./cmd/api

run:
	go run ./cmd/api

# Database
migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-down-all:
	$(MIGRATE) down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

migrate-force:
	@read -p "Version: " version; \
	$(MIGRATE) force $$version

# Seeding
seed:
	go run ./seeds/...

# Testing
test:
	go test ./tests/unit/... -v -count=1

test-integration:
	go test ./tests/integration/... -v -count=1 -tags=integration

test-all:
	go test ./... -v -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out -count=1
	go tool cover -html=coverage.out -o coverage.html

# Proto
proto-gen:
	protoc --go_out=. --go-grpc_out=. proto/face_match/face_match.proto

# Linting
lint:
	golangci-lint run ./...

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker-compose build

docker-logs:
	docker-compose logs -f api

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html

# Dev dependencies
deps:
	go mod tidy
	go mod download

# Face service
face-service-build:
	cd face-service && docker build -t skintrader-face-service .

face-service-run:
	cd face-service && docker run -p 50051:50051 skintrader-face-service
