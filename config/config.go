package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RabbitMQURL   string
	RabbitMQQueue string
	Port          string
	APISecret     string
	JWTSecret     string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	apiSecret := getEnv("API_SECRET", "")
	jwtSecret := getEnv("JWT_SECRET", apiSecret)
	if jwtSecret == "" {
		log.Fatal("Fatal: Either JWT_SECRET or API_SECRET must be configured")
	}

	return &Config{
		RabbitMQURL:   getEnv("RABBITMQ_URL", ""), // Made optional by removing default
		RabbitMQQueue: getEnv("RABBITMQ_QUEUE", "notifications"),
		Port:          getEnv("PORT", "8080"),
		APISecret:     apiSecret,
		JWTSecret:     jwtSecret,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
