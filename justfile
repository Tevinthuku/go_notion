db-up:
	docker compose up -d

db-migrate:
	migrate -path backend/db/migrations/sql -database "postgres://postgres:postgres@localhost:5432/notion_test?sslmode=disable" up

db-down:
	docker compose down


backend_test: db-up db-migrate
	go test -v ./backend/...
