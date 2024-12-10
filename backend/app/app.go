package app

import (
	"context"
	"fmt"
	"go_notion/backend/authtoken"
	"go_notion/backend/db"
	"go_notion/backend/handlers"
	"go_notion/backend/page"
	"go_notion/backend/router"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Handler interface {
	RegisterRoutes(router *gin.RouterGroup)
}

type App struct {
	pool        *pgxpool.Pool
	server      *http.Server
	tokenConfig *authtoken.TokenConfig
}

func New(port string) (*App, error) {
	app := &App{}
	err := loadEnv()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	pool, err := db.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %w", err)
	}

	app.pool = pool

	appRouter := router.NewRouter()

	server := &http.Server{
		Addr:    port,
		Handler: appRouter,
	}
	app.server = server
	tokenConfig, err := authtoken.NewTokenConfig()
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating token config: %w", err)
	}
	app.tokenConfig = tokenConfig

	signin, err := handlers.NewSignInHandler(pool, tokenConfig)
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating signin handler: %w", err)
	}
	signup, err := handlers.NewSignUpHandler(pool, tokenConfig)
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating signup handler: %w", err)
	}

	// public routes
	apiv1 := appRouter.Group("/api/v1")
	for _, r := range []Handler{signup, signin} {
		r.RegisterRoutes(apiv1)
	}

	pageConfig := page.NewPageConfig(1000)
	newPage, err := handlers.NewCreatePageHandler(pool, pageConfig)
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating page handler: %w", err)
	}

	updatePage, err := handlers.NewUpdatePageHandler(pool)
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating update page handler: %w", err)
	}

	deletePage, err := handlers.NewDeletePageHandler(pool)
	if err != nil {
		app.Shutdown(context.Background())
		return nil, fmt.Errorf("error creating delete page handler: %w", err)
	}

	// protected routes
	protectedRoutes := []Handler{newPage, updatePage, deletePage}
	protectedApiGroup := apiv1.Group("", tokenConfig.AuthMiddleware())
	for _, r := range protectedRoutes {
		r.RegisterRoutes(protectedApiGroup)
	}

	return app, nil
}

func (app *App) Run() error {
	if err := app.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
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
