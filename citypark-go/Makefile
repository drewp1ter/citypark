.PHONY: build run seed db-up db-test-up test lint

build:
	go build -o bin/server ./cmd/server

run:
	set -a; [ ! -f .env ] || . ./.env; set +a; go run ./cmd/server

seed:
	set -a; [ ! -f .env ] || . ./.env; set +a; go run ./cmd/seed

db-up:
	docker compose up -d postgres

db-test-up:
	docker compose up -d --wait postgres_test

test: db-test-up
	set -a; source .env.development; set +a; go test -p 1 ./...

lint:
	gofmt -l . && go vet ./...
