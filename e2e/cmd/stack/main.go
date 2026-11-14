//go:build e2e

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/BoxingOctopus/squirrelbot/e2e/harness"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	stack, err := harness.Start(ctx, harness.Options{
		Addr:      env("E2E_HTTP_ADDR", "127.0.0.1:18080"),
		PublicURL: env("E2E_PUBLIC_URL", "http://127.0.0.1:4173"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer stack.Close()
	fmt.Printf("e2e stack ready %s\n", stack.BaseURL)
	stop, stopCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopCancel()
	<-stop.Done()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
