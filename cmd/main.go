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
	"ping-tower/internal/monitor"
	"ping-tower/internal/scheduler"
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

	if metricsService != nil {
		monitor.MetricsRecorder = func(siteID int, siteURL string, result monitor.CheckResult, checkType string) {
			err := metricsService.RecordCheckResult(siteID, siteURL, result, checkType)
			if err != nil {
				log.Printf("⚠️ Ошибка записи метрик: %v", err)
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