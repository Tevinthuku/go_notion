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

	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}
	pool, err := db.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %v", err)
	}

	router := router.NewRouter()

	server := &http.Server{
		Addr:    port,
		Handler: router,
	}
	tokenConfig, err := authtoken.NewTokenConfig()
	if err != nil {
		return nil, fmt.Errorf("error creating token config: %v", err)
	}

	signin := usecase.NewSignIn(pool, tokenConfig)
	signup := usecase.NewSignUp(pool, tokenConfig)

	usecases := []UseCase{signup, signin}

	apiGroup := router.Group("/api")
	for _, usecase := range usecases {
		usecase.RegisterRoutes(apiGroup)
	}

	return &App{pool, server, tokenConfig}, nil
}

func (app *App) Run() error {
	return app.server.ListenAndServe()
}

func (app *App) Shutdown(ctx context.Context) error {
	defer app.pool.Close()
	log.Println("shutting down app")
	return app.server.Shutdown(ctx)
}

func (app *App) Server() *http.Server {
	return app.server
}
