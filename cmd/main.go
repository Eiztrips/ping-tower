package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"ping-tower/internal/config"
	"ping-tower/internal/database"
	"ping-tower/internal/handlers"
	"ping-tower/internal/metrics"
	"ping-tower/internal/models"
	"ping-tower/internal/monitor"
	"ping-tower/internal/notifications"
	"ping-tower/internal/scheduler"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("🚀 Запуск Site Monitor...")

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
	}

	db, err := database.NewDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("❌ Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	var metricsService *metrics.Service
	if cfg.Metrics.Enabled {
		clickhouseConfig := database.ClickHouseConfig{
			Host:     cfg.ClickHouse.Host,
			Port:     cfg.ClickHouse.Port,
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.Username,
			Password: cfg.ClickHouse.Password,
			Debug:    cfg.ClickHouse.Debug,
		}

		metricsConfig := metrics.Config{
			ClickHouse:      clickhouseConfig,
			BatchSize:       cfg.Metrics.BatchSize,
			FlushInterval:   cfg.Metrics.FlushInterval,
			MaxDailyRows:    50000,
			MinMetricGap:    5 * time.Minute,
		}

		metricsService, err = metrics.NewService(metricsConfig, db)
		if err != nil {
			log.Printf("⚠️ Ошибка инициализации сервиса метрик: %v", err)
		} else {
			defer metricsService.Close()
			handlers.SetMetricsService(metricsService)
			
			go func() {
				ticker := time.NewTicker(6 * time.Hour)
				defer ticker.Stop()
				for range ticker.C {
					log.Println("🧹 Running ClickHouse optimization...")
				}
			}()
		}
	}

	checker := monitor.NewChecker(db, 15*time.Minute)

	// Initialize alert manager with database configurations
	var globalAlertManager *notifications.AlertManager
	if cfg.Alerts.Enabled {
		globalAlertManager = notifications.NewAlertManager(&cfg.Alerts)
		log.Println("🔔 Система оповещений инициализирована (из переменных окружения)")
	}

	// Try to load global configuration from database
	globalAlertConfig, err := db.GetAlertConfig("global")
	if err == nil && globalAlertConfig.Enabled {
		// Convert database config to notifications config
		dbAlertsConfig := convertDBToNotificationsConfig(globalAlertConfig)
		globalAlertManager = notifications.NewAlertManager(dbAlertsConfig)
		log.Println("🔔 Система оповещений инициализирована (из базы данных)")

		if globalAlertConfig.EmailEnabled {
			log.Println("📧 Email оповещения включены")
		}
		if globalAlertConfig.WebhookEnabled {
			log.Println("🔗 Webhook оповещения включены")
		}
		if globalAlertConfig.TelegramEnabled {
			log.Println("📱 Telegram оповещения включены")
		}
	} else if globalAlertManager == nil {
		log.Println("🔕 Система оповещений отключена")
	}

	if metricsService != nil {
		monitor.MetricsRecorder = func(siteID int, siteURL string, result monitor.CheckResult, checkType string) {
			err := metricsService.RecordCheckResult(siteID, siteURL, result, checkType)
			if err != nil {
				log.Printf("⚠️ Ошибка записи метрик: %v", err)
			}
		}
	}

	// Set up alert notifications for site checks
	if globalAlertManager != nil {
		monitor.NotifySiteChecked = func(siteURL string, result monitor.CheckResult) {
			// Try to get site-specific alert config first
			site, err := db.GetSiteByURL(siteURL)
			var alertManager *notifications.AlertManager = globalAlertManager

			// Check if site has specific alert configuration
			if err == nil {
				// TODO: Implement site-specific alert configs if needed
				// For now, use global configuration
			}

			// Determine if we should send alert based on conditions
			shouldAlert := false
			alertType := "unknown"

			// Get the global alert config to check conditions
			if globalAlertConfig, err := db.GetAlertConfig("global"); err == nil {
				if result.Status == "down" && globalAlertConfig.AlertOnDown {
					shouldAlert = true
					alertType = "site_down"
				} else if result.Status == "up" && globalAlertConfig.AlertOnUp {
					shouldAlert = true
					alertType = "site_up"
				} else if result.StatusCode >= 500 && globalAlertConfig.AlertOnDown {
					shouldAlert = true
					alertType = "server_error"
				} else if globalAlertConfig.AlertOnResponseTimeThreshold &&
					result.ResponseTime > int64(globalAlertConfig.ResponseTimeThreshold) {
					shouldAlert = true
					alertType = "slow_response"
				}
			} else {
				// Fallback to basic conditions if no config
				log.Printf("⚠️ Нет конфигурации алертов, используем fallback логику для %s", siteURL)
				if result.Status == "down" {
					shouldAlert = true
					alertType = "site_down"
					log.Printf("📢 Fallback: сайт %s недоступен, отправляем алерт", siteURL)
				} else if result.StatusCode >= 500 {
					shouldAlert = true
					alertType = "server_error"
					log.Printf("📢 Fallback: сайт %s вернул ошибку сервера (%d), отправляем алерт", siteURL, result.StatusCode)
				}
			}

			if shouldAlert {
				siteID := 0
				if site != nil {
					siteID = site.ID
				}

				err := alertManager.SendAlert(siteID, siteURL, result, alertType)
				if err != nil {
					log.Printf("⚠️ Ошибка отправки оповещения для %s: %v", siteURL, err)

					// Log to alert history if possible
					if site != nil && globalAlertConfig != nil {
						db.LogAlert(site.ID, globalAlertConfig.ID, alertType, "all", "failed", "", err.Error())
					}
				} else {
					log.Printf("✅ Оповещение отправлено для %s (тип: %s)", siteURL, alertType)

					// Log to alert history
					if site != nil && globalAlertConfig != nil {
						db.LogAlert(site.ID, globalAlertConfig.ID, alertType, "all", "sent", "Alert sent successfully", "")
					}
				}
			}
		}
	}

	cronScheduler := scheduler.NewCronScheduler()
	err = cronScheduler.Start()
	if err != nil {
		log.Fatalf("❌ Ошибка запуска cron планировщика: %v", err)
	}
	defer cronScheduler.Stop()

	err = cronScheduler.AddJob(
		"global-check",
		"Общая проверка всех сайтов",
		"*/30 * * * *",
		monitor.CreateGlobalMonitoringJob(checker),
	)
	if err != nil {
		log.Printf("⚠️ Ошибка добавления задания общей проверки: %v", err)
	}

	sites, err := db.GetAllSites()
	if err != nil {
		log.Printf("⚠️ Ошибка загрузки сайтов: %v", err)
	} else {
		for _, site := range sites {
			if site.Config == nil {
				continue
			}
			
			schedule := site.Config.GetEffectiveSchedule()
			if schedule == "*/5 * * * *" {
				schedule = "*/15 * * * *"
			}
			
			jobID := fmt.Sprintf("site-%d", site.ID)
			err = cronScheduler.AddJob(
				jobID,
				fmt.Sprintf("Проверка сайта %s", site.URL),
				schedule,
				monitor.CreateSiteMonitoringJob(site.ID, site.URL, checker),
			)
			if err != nil {
				log.Printf("⚠️ Ошибка добавления задания для сайта %s: %v", site.URL, err)
			}
		}
	}

	r := mux.NewRouter()
	handlers.RegisterRoutes(r, db)

	log.Printf("🌐 Сервер запущен на порту %s", cfg.ServerAddress)
	log.Println("📊 Метрики оптимизированы для экономного использования места")
	log.Println("⏰ Интервалы проверки увеличены для снижения нагрузки")
	
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("🛑 Получен сигнал остановки, завершаем работу...")
		if metricsService != nil {
			metricsService.Close()
		}
		cronScheduler.Stop()
		os.Exit(0)
	}()

	log.Fatal(http.ListenAndServe(cfg.ServerAddress, r))
}

// convertDBToNotificationsConfig converts database AlertConfig to notifications AlertsConfig
func convertDBToNotificationsConfig(dbConfig *models.AlertConfig) *config.AlertsConfig {
	// Parse email recipients
	var emailTo []string
	if dbConfig.EmailTo != "" {
		emailTo = strings.Split(dbConfig.EmailTo, ",")
		for i := range emailTo {
			emailTo[i] = strings.TrimSpace(emailTo[i])
		}
	}

	return &config.AlertsConfig{
		Enabled: dbConfig.Enabled,
		Email: config.EmailAlertConfig{
			Enabled:    dbConfig.EmailEnabled,
			SMTPServer: dbConfig.SMTPServer,
			Port:       dbConfig.SMTPPort,
			Username:   dbConfig.SMTPUsername,
			Password:   dbConfig.SMTPPassword,
			From:       dbConfig.EmailFrom,
			To:         emailTo,
		},
		Webhook: config.WebhookAlertConfig{
			Enabled: dbConfig.WebhookEnabled,
			URL:     dbConfig.WebhookURL,
			Headers: dbConfig.WebhookHeaders,
			Timeout: dbConfig.WebhookTimeout,
		},
		Telegram: config.TelegramAlertConfig{
			Enabled:  dbConfig.TelegramEnabled,
			BotToken: dbConfig.TelegramBotToken,
			ChatID:   dbConfig.TelegramChatID,
		},
	}
}