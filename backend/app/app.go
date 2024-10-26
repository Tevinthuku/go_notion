package app

import (
	"go_notion/backend/api_error"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"go_notion/backend/usecase"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type UseCase interface {
	RegisterRoutes(router *gin.RouterGroup)
}

type App struct {
	pool        *pgxpool.Pool
	router      *gin.Engine
	tokenConfig *authtoken.TokenConfig
	usecases    []UseCase
}

func New() *App {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	pool := db.Run()

	router := gin.Default()
	router.Use(api_error.Errorhandler())
	tokenConfig, err := authtoken.NewTokenConfig()
	if err != nil {
		log.Fatalf("Error creating token config: %v", err)
	}

	signin := usecase.NewSignIn(pool, tokenConfig)
	signup := usecase.NewSignUp(pool, tokenConfig)

	usecases := []UseCase{signup, signin}

	return &App{pool, router, tokenConfig, usecases}
}

func (app *App) Run() {
	app.SetupRoutes()

	defer app.pool.Close()
	app.router.Run()
}

func (app *App) SetupRoutes() {
	router := app.router.Group("/api")
	for _, usecase := range app.usecases {
		usecase.RegisterRoutes(router)
	}
}
