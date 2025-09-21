package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	ServerAddress  string
	CheckInterval  time.Duration
	ClickHouse     ClickHouseConfig
	Metrics        MetricsConfig
	Alerts         AlertsConfig
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

type AlertsConfig struct {
	Enabled   bool
	Email     EmailAlertConfig
	Webhook   WebhookAlertConfig
	Telegram  TelegramAlertConfig
}

type EmailAlertConfig struct {
	Enabled    bool
	SMTPServer string
	Port       string
	Username   string
	Password   string
	From       string
	To         []string
}

type WebhookAlertConfig struct {
	Enabled bool
	URL     string
	Headers map[string]string
	Timeout int
}

type TelegramAlertConfig struct {
	Enabled bool
	BotToken string
	ChatID   string
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

	// Alert configuration
	alertsEnabled, err := strconv.ParseBool(getEnv("ALERTS_ENABLED", "false"))
	if err != nil {
		alertsEnabled = false
	}

	emailEnabled, err := strconv.ParseBool(getEnv("EMAIL_ALERTS_ENABLED", "false"))
	if err != nil {
		emailEnabled = false
	}

	webhookEnabled, err := strconv.ParseBool(getEnv("WEBHOOK_ALERTS_ENABLED", "false"))
	if err != nil {
		webhookEnabled = false
	}

	telegramEnabled, err := strconv.ParseBool(getEnv("TELEGRAM_ALERTS_ENABLED", "false"))
	if err != nil {
		telegramEnabled = false
	}

	webhookTimeout, err := strconv.Atoi(getEnv("WEBHOOK_TIMEOUT", "10"))
	if err != nil {
		webhookTimeout = 10
	}

	// Parse email recipients
	emailTo := []string{}
	if emailToStr := getEnv("EMAIL_TO", ""); emailToStr != "" {
		emailTo = strings.Split(emailToStr, ",")
		for i := range emailTo {
			emailTo[i] = strings.TrimSpace(emailTo[i])
		}
	}

	// Parse webhook headers
	webhookHeaders := make(map[string]string)
	if headersStr := getEnv("WEBHOOK_HEADERS", ""); headersStr != "" {
		headerPairs := strings.Split(headersStr, ",")
		for _, pair := range headerPairs {
			if kv := strings.SplitN(pair, ":", 2); len(kv) == 2 {
				webhookHeaders[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
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
		Alerts: AlertsConfig{
			Enabled: alertsEnabled,
			Email: EmailAlertConfig{
				Enabled:    emailEnabled,
				SMTPServer: getEnv("SMTP_SERVER", ""),
				Port:       getEnv("SMTP_PORT", "587"),
				Username:   getEnv("SMTP_USERNAME", ""),
				Password:   getEnv("SMTP_PASSWORD", ""),
				From:       getEnv("EMAIL_FROM", ""),
				To:         emailTo,
			},
			Webhook: WebhookAlertConfig{
				Enabled: webhookEnabled,
				URL:     getEnv("WEBHOOK_URL", ""),
				Headers: webhookHeaders,
				Timeout: webhookTimeout,
			},
			Telegram: TelegramAlertConfig{
				Enabled:  telegramEnabled,
				BotToken: getEnv("TELEGRAM_BOT_TOKEN", ""),
				ChatID:   getEnv("TELEGRAM_CHAT_ID", ""),
			},
		},
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}