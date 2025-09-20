package main

import (
    "log"
    "net/http"
    "site-monitor/internal/config"
    "site-monitor/internal/database"
    "site-monitor/internal/handlers"
    "site-monitor/internal/monitor"
    "site-monitor/internal/metrics"
    "time"

    "github.com/gorilla/mux"
)

func main() {
    log.Println("🚀 Запуск Site Monitor...")
    
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Ошибка загрузки конфигурации: %v", err)
    }

    log.Printf("⚙️ Конфигурация: DB=%s, Port=%s", 
        cfg.DatabaseURL, cfg.ServerAddress)

    db, err := database.NewDB(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Ошибка подключения к базе данных: %v", err)
    }
    defer db.Close()

    var metricsService *metrics.Service
    if cfg.Metrics.Enabled {
        log.Println("📊 Инициализация ClickHouse для метрик...")

        metricsConfig := metrics.Config{
            ClickHouse: database.ClickHouseConfig{
                Host:     cfg.ClickHouse.Host,
                Port:     cfg.ClickHouse.Port,
                Database: cfg.ClickHouse.Database,
                Username: cfg.ClickHouse.Username,
                Password: cfg.ClickHouse.Password,
                Debug:    cfg.ClickHouse.Debug,
            },
            BatchSize:     cfg.Metrics.BatchSize,
            FlushInterval: cfg.Metrics.FlushInterval,
        }

        metricsService, err = metrics.NewService(metricsConfig, db)
        if err != nil {
            log.Printf("⚠️ Ошибка инициализации ClickHouse (продолжаем без метрик): %v", err)
            metricsService = nil
        } else {
            defer metricsService.Close()
            log.Println("✅ ClickHouse метрики активированы")
        }
    } else {
        log.Println("📊 Метрики отключены в конфигурации")
    }

    monitor.NotifySiteChecked = func(url string, result monitor.CheckResult) {
        handlers.BroadcastSSE("site_checked", map[string]interface{}{
            "url": url,
            "status": result.Status,
            "status_code": result.StatusCode,
            "response_time": result.ResponseTime,
			"ssl_valid": result.SSLValid,
            "timestamp": time.Now().Format(time.RFC3339),
        })
    }

    if metricsService != nil {
        monitor.MetricsRecorder = func(siteID int, siteURL string, result monitor.CheckResult, checkType string) {
            if err := metricsService.RecordCheckResult(siteID, siteURL, result, checkType); err != nil {
                log.Printf("⚠️ Ошибка записи метрик: %v", err)
            }
        }
    }

    monitor.StartPeriodicMonitoring(db)

    r := mux.NewRouter()
    handlers.RegisterRoutes(r, db)

    if metricsService != nil {
        handlers.SetMetricsService(metricsService)
    }
    
    log.Printf("🌐 Сервер запущен на http://localhost%s", cfg.ServerAddress)
    log.Println("� Фоновый мониторинг запущен с индивидуальными интервалами")
    
    if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
        log.Fatalf("Ошибка запуска сервера: %v", err)
    }
}