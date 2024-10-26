

DB_URL=postgresql://postgres:12345@127.0.0.1:5555/ud_telegram_miniapp_test?sslmode=disable
MIGRATIONS_DIR=./migrations

migrate-create:
	goose -dir=$(MIGRATIONS_DIR) create $(name) sql

# Apply all available migrations
migrate-up:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" up

# Roll back the last applied migration
migrate-down:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" down