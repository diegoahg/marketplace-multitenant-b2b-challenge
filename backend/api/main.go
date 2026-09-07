package main

import (
	"context"
	"errors"
	"log/slog"
	"marketplace/internal/application"
	"marketplace/internal/config"
	api "marketplace/internal/http"
	mongo "marketplace/internal/repository/mongo"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("application stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	cfg, e := config.Load()
	if e != nil {
		return e
	}
	var level slog.Level
	if e = level.UnmarshalText([]byte(cfg.LogLevel)); e != nil {
		return e
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(log.With("service", "api"))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	store, e := mongo.Connect(startup, cfg.MongoURI, cfg.Database)
	if e != nil {
		return e
	}
	defer func() {
		closeCtx, done := context.WithTimeout(context.Background(), 5*time.Second)
		defer done()
		_ = store.Close(closeCtx)
	}()
	if e = store.Init(startup); e != nil {
		return e
	}
	service := &application.Service{Catalog: store, Quotes: store, Orders: store, Tracking: store, Now: time.Now, QuoteTTL: cfg.QuoteTTL}
	ready := store.Ping
	server := &http.Server{Addr: ":" + cfg.Port, Handler: api.Router(service, ready, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 35 * time.Second, IdleTimeout: 60 * time.Second}
	result := make(chan error, 1)
	go func() { log.Info("API listening", "port", cfg.Port); result <- server.ListenAndServe() }()
	var serveErr error
	select {
	case <-ctx.Done():
	case serveErr = <-result:
	}
	shutdown, done := context.WithTimeout(context.Background(), 35*time.Second)
	defer done()
	shutdownErr := server.Shutdown(shutdown)
	if shutdownErr != nil {
		_ = server.Close()
	}
	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		return serveErr
	}
	return shutdownErr
}
