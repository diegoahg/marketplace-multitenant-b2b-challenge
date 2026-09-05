package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port, MongoURI, Database, Emulator, Project, Topic, Subscription, ERP, Push, LogLevel string
	QuoteTTL                                                                              time.Duration
}

func Load() (Config, error) {
	c := Config{Port: get("HTTP_PORT", "8080"), MongoURI: get("MONGO_URI", "mongodb://localhost:27017/?replicaSet=rs0&directConnection=true"), Database: get("MONGO_DATABASE", "marketplace"), Emulator: get("PUBSUB_EMULATOR_HOST", "localhost:8085"), Project: get("PUBSUB_PROJECT_ID", "local-marketplace"), Topic: get("PUBSUB_TOPIC", "orders-confirmed"), Subscription: get("PUBSUB_SUBSCRIPTION", "orders-confirmed-worker"), ERP: os.Getenv("ERP_BASE_URL"), Push: os.Getenv("PUSH_BASE_URL"), LogLevel: get("LOG_LEVEL", "INFO")}
	var e error
	c.QuoteTTL, e = time.ParseDuration(get("QUOTE_TTL", "15m"))
	if e != nil || c.QuoteTTL <= 0 {
		return c, fmt.Errorf("invalid QUOTE_TTL")
	}
	port, e := strconv.Atoi(c.Port)
	if e != nil || port < 1 || port > 65535 {
		return c, fmt.Errorf("invalid HTTP_PORT")
	}
	return c, nil
}
func get(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
