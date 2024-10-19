package main

import (
	"context"
	"os"
	"os/signal"

	"data-pipelines-worker/api"
	"data-pipelines-worker/types/config"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	worker := api.NewServer(config.GetConfig())
	worker.SetAPIMiddlewares()
	worker.SetAPIHandlers()

	worker.Start(ctx)
}
