## Setup

1. Useful tools to install

- Migrate CLI - https://github.com/golang-migrate/migrate/tree/master/cmd/migrate

2. Run docker compose file

```bash
docker compose up -d
```

3. Running the migrations

In the `backend` directory, run:

```bash
migrate -path db/migrations/sql -database "postgres://postgres:postgres@localhost:5432/go_notion?sslmode=disable" up
```

## Contributing:

1. Creating a new migration

In the `backend` directory, run:

```bash
migrate create -ext sql -dir db/migrations/sql -seq <migration_name>
```
