## Setup

1. Useful tools to install

- Migrate CLI - https://github.com/golang-migrate/migrate/tree/master/cmd/migrate

2. Run docker compose file

```bash
docker compose up -d
```

## Contributing:

1. Creating a new migration

In the `backend` directory, run:

```bash
migrate create -ext sql -dir db/migrations/sql -seq <migration_name>
```
