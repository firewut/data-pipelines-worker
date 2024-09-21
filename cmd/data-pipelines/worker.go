package main

import (
	"context"
	"os"
	"os/signal"

	"data-pipelines-worker/api"
)

func main() {
	worker := api.NewServer()
	worker.SetAPIMiddlewares()
	worker.SetAPIHandlers()

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	worker.Start(ctx)
}
