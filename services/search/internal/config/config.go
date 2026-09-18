package config

import (
	"os"
)

type Config struct {
	Port      string
	ESHost    string
	BrokerURL string
}

func Load() *Config {
	return &Config{
		Port:      getEnv("SEARCH_PORT", "8080"),
		ESHost:    getEnv("SEARCH_ES_HOST", "http://search-engine:9200"),
		BrokerURL: getEnv("SEARCH_BROKER_URL", "amqp://guest:guest@broker:5672/"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
