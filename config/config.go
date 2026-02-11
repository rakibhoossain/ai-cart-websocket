package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	RabbitMQURL   string
	RabbitMQQueue string
	JWTPublicKey  []byte
	Port          string
	APISecret     string
}

func Load() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	pubKeyPath := getEnv("JWT_PUBLIC_KEY_PATH", "public.pem")
	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		// Try to read from env var directly if file not found
		pubKeyContent := getEnv("JWT_PUBLIC_KEY", "")
		if pubKeyContent != "" {
			pubKeyBytes = []byte(pubKeyContent)
		} else {
			log.Fatalf("Fatal: Could not load public key from %s or JWT_PUBLIC_KEY env", pubKeyPath)
		}
	}

	return &Config{
		RabbitMQURL:   getEnv("RABBITMQ_URL", ""), // Made optional by removing default
		RabbitMQQueue: getEnv("RABBITMQ_QUEUE", "notifications"),
		JWTPublicKey:  pubKeyBytes,
		Port:          getEnv("PORT", "8080"),
		APISecret:     getEnv("API_SECRET", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
