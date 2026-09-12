.PHONY: migrate-up migrate-down sqlc-gen test test-integration lint run build

# Load .env for local dev targets
include .env
export

MIGRATE = migrate -database "$(DATABASE_URL)" -path internal/db/migrations

migrate-up:
	$(MIGRATE) up

migrate-down:
	$(MIGRATE) down 1

migrate-force:
	$(MIGRATE) force $(V)

sqlc-gen:
	sqlc generate

test:
	go test -short -count=1 ./...

test-integration:
	go test -race -count=1 ./...

lint:
	golangci-lint run ./...

build:
	go build -o server ./cmd/server

run:
	go run ./cmd/server

vet:
	go vet ./...
