package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go_notion/backend/api_error"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL, ok := os.LookupEnv("DATABASE_URL")
	if !ok {
		fmt.Fprintf(os.Stderr, "DATABASE_URL is not set\n")
		os.Exit(1)
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create database: %v\n", err)
		os.Exit(1)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create driver: %v\n", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://backend/migrations",
		"go_notion",
		driver,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create migration: %v\n", err)
		os.Exit(1)
	}
	if err := m.Up(); errors.Is(err, migrate.ErrNoChange) {
		log.Println(err)
	} else if err != nil {
		log.Fatalln(err)
	}

	dbpool, err := pgxpool.New(context.Background(), dbURL)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	r := gin.Default()
	r.Use(api_error.Errorhandler())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.Run()
}
