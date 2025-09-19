package main

import (
    "log"
    "net/http"
    "site-monitor/internal/config"
    "site-monitor/internal/database"
    "site-monitor/internal/handlers"
    "site-monitor/internal/monitor"

    "github.com/gorilla/mux"
)

func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Ошибка загрузки конфигурации: %v", err)
    }

    db, err := database.NewDB(cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Ошибка подключения к базе данных: %v", err)
    }
    defer db.Close()

    go monitor.StartMonitoring(db.DB, cfg.CheckInterval)
    
    log.Printf("Сервер запущен на %s", cfg.ServerAddress)
    if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
        log.Fatalf("Ошибка запуска сервера: %v", err)
    }
}