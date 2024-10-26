package app

import (
	"go_notion/backend/api_error"
	"go_notion/backend/authentication"
	"go_notion/backend/db"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type App struct {
	pool        *pgxpool.Pool
	router      *gin.Engine
	tokenConfig *authentication.TokenConfig
	auth        *authentication.AuthenticationRoutesContext
}

func New() *App {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	pool := db.Run()

	router := gin.Default()
	router.Use(api_error.Errorhandler())
	tokenConfig, err := authentication.NewTokenConfig()
	if err != nil {
		log.Fatalf("Error creating token config: %v", err)
	}

	auth := authentication.NewAuthentication(pool, tokenConfig)

	return &App{pool, router, tokenConfig, auth}
}

func (app *App) Run() {
	app.SetupRoutes()

	defer app.pool.Close()
	app.router.Run()
}

func (app *App) SetupRoutes() {
	router := app.router.Group("/api")
	app.auth.RegisterRoutes(router)
}
