package app

import (
	"context"
	"errors"
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
	pageConfig  *page.PageConfig
}

func New(port string) (*App, error) {
	err := loadEnv()
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	pool, err := db.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to create database pool: %w", err)
	}
	app := &App{}
	app.pool = pool

	appRouter := router.NewRouter()

	server := &http.Server{
		Addr:    port,
		Handler: appRouter,
	}
	app.server = server
	tokenConfig, err := authtoken.NewTokenConfig()
	if err != nil {
		if shutdownErr := app.Shutdown(context.Background()); shutdownErr != nil {
			return nil, fmt.Errorf("multiple errors: %w", errors.Join(err, shutdownErr))
		}
		return nil, fmt.Errorf("error creating token config: %w", err)
	}
	app.tokenConfig = tokenConfig
	app.pageConfig = page.NewPageConfig(1000)
	err = app.registerHandlers(appRouter)
	if err != nil {
		if shutdownErr := app.Shutdown(context.Background()); shutdownErr != nil {
			return nil, fmt.Errorf("multiple errors: %w", errors.Join(err, shutdownErr))
		}

		return nil, fmt.Errorf("error registering routes: %w", err)
	}

	return app, nil
}

func (app *App) registerHandlers(appRouter *gin.Engine) error {
	signin, err := handlers.NewSignInHandler(app.pool, app.tokenConfig)
	if err != nil {
		return fmt.Errorf("error creating signin handler: %w", err)
	}
	signup, err := handlers.NewSignUpHandler(app.pool, app.tokenConfig)
	if err != nil {
		return fmt.Errorf("error creating signup handler: %w", err)
	}

	// public routes
	apiv1 := appRouter.Group("/api/v1")
	for _, r := range []Handler{signup, signin} {
		r.RegisterRoutes(apiv1)
	}

	newPage, err := handlers.NewCreatePageHandler(app.pool, app.pageConfig)
	if err != nil {
		return fmt.Errorf("error creating page handler: %w", err)
	}

	updatePage, err := handlers.NewUpdatePageHandler(app.pool)
	if err != nil {
		return fmt.Errorf("error creating update page handler: %w", err)
	}

	deletePage, err := handlers.NewDeletePageHandler(app.pool)
	if err != nil {
		return fmt.Errorf("error creating delete page handler: %w", err)
	}

	// protected routes
	protectedRoutes := []Handler{newPage, updatePage, deletePage}
	protectedApiGroup := apiv1.Group("", app.tokenConfig.AuthMiddleware())
	for _, r := range protectedRoutes {
		r.RegisterRoutes(protectedApiGroup)
	}

	return nil
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
