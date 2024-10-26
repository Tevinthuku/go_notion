package main

import (
	"go_notion/backend/app"

	_ "github.com/jackc/pgx/v5/stdlib"

	_ "github.com/golang-migrate/migrate/source/file"
)

func main() {
	app := app.New()
	app.Run()
}
