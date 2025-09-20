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
	ClickHouse     ClickHouseConfig
	Metrics        MetricsConfig
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Debug    bool
}

type MetricsConfig struct {
	Enabled       bool
	BatchSize     int
	FlushInterval time.Duration
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

	clickhousePort, err := strconv.Atoi(getEnv("CLICKHOUSE_PORT", "9000"))
	if err != nil {
		clickhousePort = 9000
	}

	metricsEnabled, err := strconv.ParseBool(getEnv("METRICS_ENABLED", "true"))
	if err != nil {
		metricsEnabled = true
	}

	metricsBatchSize, err := strconv.Atoi(getEnv("METRICS_BATCH_SIZE", "100"))
	if err != nil {
		metricsBatchSize = 100
	}

	metricsFlushInterval, err := strconv.Atoi(getEnv("METRICS_FLUSH_INTERVAL", "10"))
	if err != nil {
		metricsFlushInterval = 10
	}

	clickhouseDebug, err := strconv.ParseBool(getEnv("CLICKHOUSE_DEBUG", "false"))
	if err != nil {
		clickhouseDebug = false
	}

	return &Config{
		DatabaseURL:    getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/site_monitor?sslmode=disable"),
		ServerAddress:  ":" + port,
		CheckInterval:  time.Duration(checkIntervalInt) * time.Second,
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     clickhousePort,
			Database: getEnv("CLICKHOUSE_DATABASE", "site_monitor"),
			Username: getEnv("CLICKHOUSE_USERNAME", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
			Debug:    clickhouseDebug,
		},
		Metrics: MetricsConfig{
			Enabled:       metricsEnabled,
			BatchSize:     metricsBatchSize,
			FlushInterval: time.Duration(metricsFlushInterval) * time.Second,
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}