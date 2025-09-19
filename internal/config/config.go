package config

import (
	"log"
	"os"
	"time"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	ServerAddress  string
	CheckInterval  time.Duration
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println("Файл .env не найден, используются переменные окружения системы")
	}

	port := getEnv("PORT", "8080")
	checkIntervalStr := getEnv("CHECK_INTERVAL", "10")
	checkIntervalInt, err := strconv.Atoi(checkIntervalStr)
	if err != nil {
		checkIntervalInt = 10
	}

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/site_monitor?sslmode=disable"),
		ServerAddress:  ":" + port,
		CheckInterval:  time.Duration(checkIntervalInt) * time.Second,
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}