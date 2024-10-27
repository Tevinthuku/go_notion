package main

import (
	"context"
	"fmt"
	"go_notion/backend/app"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := runInner(ctx, stop); err != nil {
		log.Printf("app shutdown failed: %v", err)
		return 1
	}
	log.Println("app shutdown gracefully")
	return 0
}

const defaultShutdownTimeout = 5 * time.Second

func runInner(ctx context.Context, stop context.CancelFunc) error {

	app, err := app.New(":8080")
	if err != nil {
		return fmt.Errorf("error creating app: %v", err)
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	errChan := make(chan error, 1)
	go func() {
		if err := app.Run(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("error running app: %v", err)
			stop()
		}
	}()

	// wait for either context cancellation or server error
	select {
	case <-ctx.Done():
	case err := <-errChan:
		return err
	}

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	timeout := defaultShutdownTimeout
	// in production we want to give the app a longer shutdown time to finish processing requests
	// hence APP_SHUTDOWN_TIMEOUT is used and its longer than the default 5s
	if envTimeout := os.Getenv("APP_SHUTDOWN_TIMEOUT"); envTimeout != "" {
		if d, err := time.ParseDuration(envTimeout); err == nil {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down app: %v", err)
	}

	return nil
}
