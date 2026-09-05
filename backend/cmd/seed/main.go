package main

import (
	"context"
	"log/slog"
	"marketplace/internal/config"
	mongo "marketplace/internal/repository/mongo"
	"os"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("seed failed", "error", e)
		os.Exit(1)
	}
}
func run() error {
	cfg, e := config.Load()
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	store, e := mongo.Connect(ctx, cfg.MongoURI, cfg.Database)
	if e != nil {
		return e
	}
	defer store.Close(ctx)
	if e = store.Init(ctx); e != nil {
		return e
	}
	file, e := os.Open("testdata/seed.json")
	if e != nil {
		return e
	}
	defer file.Close()
	if e = store.Seed(ctx, file); e != nil {
		return e
	}
	slog.Info("seed complete; existing data preserved")
	return nil
}
