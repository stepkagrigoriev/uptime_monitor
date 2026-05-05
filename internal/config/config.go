package config

import (
	"os"
	"github.com/joho/godotenv"
	"uptime-monitor/internal/logger"
)

type Config struct {
	DBURL     string
	Port      string
	JWTSecret string
}

func LoadConfig() *Config {
	if err := godotenv.Load("../../.env"); err != nil {
		logger.Log.Info("Файл .env не найден, используются системные переменные")
	}

	return &Config{
		DBURL:     getEnv("DATABASE_URL", "postgres://postgres:pass@localhost:5432/uptime?sslmode=disable"),
		Port:      getEnv("PORT", "8080"),
		JWTSecret: getEnv("JWT_SECRET", "default_secret_key"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}