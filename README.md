## Setup

1. Useful tools to install

- Migrate CLI - https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
- Just - https://github.com/casey/just

2. Running the backend tests:

```bash
just backend_test
```

## Contributing:

1. Creating a new migration

In the `backend` directory, run:

```bash
migrate create -ext sql -dir db/migrations/sql -seq <migration_name>
```

2. Fixing a migration

In the backend directory: when you encounter the following error when a migration is faulty

```
error: Dirty database version 2. Fix and force version.
```

run the following command:

```bash
migrate -database "your_database_url" -path db/migrations/sql force 1
```

then:

```bash
migrate -path backend/db/migrations/sql -database "your_database_url" up
```
