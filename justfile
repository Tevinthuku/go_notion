export DATABASE_URL := "postgres://postgres:postgres@localhost:5432?sslmode=disable"

db-up:
	docker compose up -d

db-migrate:
	migrate -path backend/db/migrations/sql -database $DATABASE_URL up

db-down:
	docker compose down


backend_test: db-up db-migrate
	go test -v ./backend/...

clean-db:
	docker compose down -v

reset-db: clean-db
	docker compose up -d
	migrate -path backend/db/migrations/sql -database ${DATABASE_URL} up
