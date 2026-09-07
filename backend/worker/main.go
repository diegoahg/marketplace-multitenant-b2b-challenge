package main

import (
	"context"
	"fmt"
	"log/slog"
	"marketplace/internal/config"
	"marketplace/internal/integration"
	"marketplace/internal/messaging"
	mongo "marketplace/internal/repository/mongo"
	"os"
	"os/signal"
	"sync"
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
	erpSub, e := messaging.NewEmulator(cfg.Emulator, cfg.Project, cfg.Topic, cfg.ERPSubscription)
	if e != nil {
		return e
	}
	pushSub, e := messaging.NewEmulator(cfg.Emulator, cfg.Project, cfg.Topic, cfg.PushSubscription)
	if e != nil {
		return e
	}
	if cfg.ERPSubscription == cfg.PushSubscription {
		return fmt.Errorf("ERP and PUSH subscriptions must differ")
	}
	for {
		e = erpSub.Setup(startup)
		if e == nil {
			e = pushSub.Setup(startup)
		}
		if e == nil {
			break
		}
		select {
		case <-startup.Done():
			return e
		case <-time.After(time.Second):
		}
	}
	erp := &messaging.Destination{Kind: "erp", Store: store, Send: integration.New("erp", cfg.ERP, store.DB).SendOrder, Now: time.Now}
	push := &messaging.Destination{Kind: "push", Store: store, Send: integration.New("push", cfg.Push, store.DB).SendOrderConfirmed, Now: time.Now}
	mail := integration.Mail{Address: cfg.SMTPAddress, From: cfg.MailFrom, To: cfg.MailTo}
	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		messaging.RunPublisher(ctx, store, messaging.FanoutPublisher{ERP: erpSub, Push: pushSub}, log)
	}()
	go func() { defer wg.Done(); messaging.RunDestination(ctx, erpSub, erp, log) }()
	go func() { defer wg.Done(); messaging.RunDestination(ctx, pushSub, push, log) }()
	go func() { defer wg.Done(); messaging.RunAlerts(ctx, store, mail, log) }()
	log.Info("worker started", "topic", cfg.Topic, "erpSubscription", cfg.ERPSubscription, "pushSubscription", cfg.PushSubscription)
	<-ctx.Done()
	wg.Wait()
	log.Info("worker stopped")
	return nil
}
