package app

import (
	"context"
	"fmt"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"go_notion/backend/router"
	"go_notion/backend/usecase"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type UseCase interface {
	RegisterRoutes(router *gin.RouterGroup)
}

type App struct {
	pool        *pgxpool.Pool
	server      *http.Server
	tokenConfig *authtoken.TokenConfig
}

func New(port string) (*App, error) {

	err := loadEnv()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}
	pool, err := db.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %v", err)
	}

	appRouter := router.NewRouter()

	server := &http.Server{
		Addr:    port,
		Handler: appRouter,
	}
	tokenConfig, err := authtoken.NewTokenConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating token config: %v", err)
	}

	signin, err := usecase.NewSignIn(pool, tokenConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating signin usecase: %v", err)
	}
	signup, err := usecase.NewSignUp(pool, tokenConfig)
	if err != nil {
		return nil, fmt.Errorf("error creating signup usecase: %v", err)
	}

	usecases := []UseCase{signup, signin}

	apiGroup := appRouter.Group("/api/v1")
	for _, usecase := range usecases {
		usecase.RegisterRoutes(apiGroup)
	}

	return &App{pool, server, tokenConfig}, nil
}

func (app *App) Run() error {
	return app.server.ListenAndServe()
}

func (app *App) Shutdown(ctx context.Context) error {
	log.Println("shutting down app")
	app.pool.Close()
	return app.server.Shutdown(ctx)
}

func (app *App) Server() *http.Server {
	return app.server
}

func loadEnv() error {
	env := os.Getenv("GO_ENV")
	if env == "" {
		env = "development"
	}

	// Try environment-specific file first
	err := godotenv.Load(fmt.Sprintf(".env.%s", env))
	if err != nil {
		// Fallback to default .env
		return godotenv.Load()
	}
	return nil
}
