.PHONY: run test migrate-up migrate-down seed

run:
	go run ./cmd/api/main.go

test:
	go test ./... -v -count=1

test-integration:
	go test ./... -v -count=1 -tags integration

migrate-up:
	migrate -path ./migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" up

migrate-down:
	migrate -path ./migrations -database "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}" down

seed:
	go run ./seeds/finance_seed.go
