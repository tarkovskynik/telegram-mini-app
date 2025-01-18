migrate-create:
	goose -dir=$(MIGRATIONS_DIR) create $(name) sql

migrate-up:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" up

migrate-down:
	goose -dir=$(MIGRATIONS_DIR) postgres "$(DB_URL)" down

go-generate:
	go generate ./...
