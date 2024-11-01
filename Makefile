

DB_URL=postgresql://postgres:12345@127.0.0.1:5555/ud_telegram_miniapp_test?sslmode=disable
MIGRATIONS_DIR=./migrations

migrate-create:
	goose -dir=$(MIGRATIONS_DIR) create $(name) sql

migrate-up:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" up

migrate-down:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" down

go-generate:
	go generate ./...
