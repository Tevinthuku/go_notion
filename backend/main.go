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

func runInner(ctx context.Context, stop context.CancelFunc) error {

	app, err := app.New(":8080")
	if err != nil {
		return fmt.Errorf("error creating app: %v", err)
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := app.Run(); err != nil && err != http.ErrServerClosed {
			log.Printf("error running app: %v", err)
			stop()
		}
	}()

	// listen for interrupt signal from OS
	<-ctx.Done()

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down app: %v", err)
	}

	return nil
}
