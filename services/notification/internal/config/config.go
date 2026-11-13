package config

import "os"

type Config struct {
	RMQHost          string
	RMQPort          string
	RMQUser          string
	RMQPassword      string
	MongoUser        string
	MongoPassword    string
	MongoHost        string
	MongoPort        string
	MongoDatabase    string
	AuthPublicKeyURL string
	SMTPHost         string
	SMTPPort         string
	SMTPFrom         string
}

func Load() *Config {
	return &Config{
		RMQHost:          os.Getenv("RMQ_HOST"),
		RMQPort:          os.Getenv("RMQ_PORT"),
		RMQUser:          os.Getenv("RMQ_USER"),
		RMQPassword:      os.Getenv("RMQ_PASSWORD"),
		MongoUser:        os.Getenv("MONGO_USER"),
		MongoPassword:    os.Getenv("MONGO_PASSWORD"),
		MongoHost:        os.Getenv("MONGO_HOST"),
		MongoPort:        os.Getenv("MONGO_PORT"),
		MongoDatabase:    os.Getenv("MONGO_DATABASE"),
		AuthPublicKeyURL: os.Getenv("AUTH_PUBLIC_KEY_URL"),
		SMTPHost:         os.Getenv("SMTP_HOST"),
		SMTPPort:         os.Getenv("SMTP_PORT"),
		SMTPFrom:         os.Getenv("SMTP_FROM"),
	}
}
