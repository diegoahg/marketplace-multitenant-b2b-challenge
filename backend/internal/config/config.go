package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ERPSubscription, PushSubscription, SMTPAddress, MailFrom, MailTo        string
	Port, MongoURI, Database, Emulator, Project, Topic, ERP, Push, LogLevel string
	QuoteTTL                                                                time.Duration
}

func Load() (Config, error) {
	c := Config{Port: get("HTTP_PORT", "8080"), MongoURI: get("MONGO_URI", "mongodb://localhost:27017/?replicaSet=rs0&directConnection=true"), Database: get("MONGO_DATABASE", "marketplace"), Emulator: get("PUBSUB_EMULATOR_HOST", "localhost:8085"), Project: get("PUBSUB_PROJECT_ID", "local-marketplace"), Topic: get("PUBSUB_TOPIC", "orders-confirmed"), ERP: os.Getenv("ERP_BASE_URL"), Push: os.Getenv("PUSH_BASE_URL"), LogLevel: get("LOG_LEVEL", "INFO")}
	var e error
	c.ERPSubscription = get("PUBSUB_ERP_SUBSCRIPTION", "orders-confirmed-erp")
	c.PushSubscription = get("PUBSUB_PUSH_SUBSCRIPTION", "orders-confirmed-push")
	c.SMTPAddress = get("SMTP_ADDRESS", "localhost:1025")
	c.MailFrom = get("DLQ_MAIL_FROM", "worker@mariposamarket.test")
	c.MailTo = get("DLQ_MAIL_TO", "operations@mariposamarket.test")
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
