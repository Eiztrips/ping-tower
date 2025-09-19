package main

import (
    "log"
    "net/http"
    "site-monitor/internal/config"
    "site-monitor/internal/database"
    "site-monitor/internal/handlers"
    "site-monitor/internal/monitor"
    "time"

    "github.com/gorilla/mux"
)

func main() {
    log.Println("🚀 Запуск Site Monitor...")
    
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Ошибка загрузки конфигурации: %v", err)
    }

    log.Printf("⚙️ Конфигурация: DB=%s, Port=%s, Interval=%v", 
        cfg.DatabaseURL, cfg.ServerAddress, cfg.CheckInterval)

    db, err := database.NewDB(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Ошибка подключения к базе данных: %v", err)
    }
    defer db.Close()

    // Настраиваем связь между модулями для SSE уведомлений
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

    r := mux.NewRouter()
    handlers.RegisterRoutes(r, db)

    time.Sleep(2 * time.Second)
    
    log.Printf("🔍 Запуск мониторинга с интервалом %v", cfg.CheckInterval)
    go monitor.StartMonitoring(db.DB, cfg.CheckInterval)
    
    log.Printf("🌐 Сервер запущен на http://localhost%s", cfg.ServerAddress)
    if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
        log.Fatalf("Ошибка запуска сервера: %v", err)
    }
}