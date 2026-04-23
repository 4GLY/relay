package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"relay/internal/app"
	"relay/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.RunCuratorWorker(ctx, config.Load()); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
