package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "site-monitor/internal/config"
    "site-monitor/internal/database"
    "site-monitor/internal/handlers"
    "site-monitor/internal/monitor"
    "site-monitor/internal/metrics"
    "site-monitor/internal/models"
    "site-monitor/internal/scheduler"
    "syscall"
    "time"

    "github.com/gorilla/mux"
)

func main() {
    log.Println("🚀 Запуск Site Monitor с cron-подобной архитектурой...")
    
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

    // Инициализация ClickHouse метрик
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

    // Создаем планировщик заданий
    cronScheduler := scheduler.NewCronScheduler()

    // Создаем checker для мониторинга
    checker := monitor.NewChecker(db, 0)

    // Настройка SSE уведомлений
    monitor.NotifySiteChecked = func(url string, result monitor.CheckResult) {
        handlers.BroadcastSSE("site_checked", map[string]interface{}{
            "url": url,
            "status": result.Status,
            "status_code": result.StatusCode,
            "response_time": result.ResponseTime,
			"ssl_valid": result.SSLValid,
            "timestamp": time.Now().Format(time.RFC3339),
            "check_type": "scheduled",
        })
    }

    // Настройка записи метрик
    if metricsService != nil {
        monitor.MetricsRecorder = func(siteID int, siteURL string, result monitor.CheckResult, checkType string) {
            if err := metricsService.RecordCheckResult(siteID, siteURL, result, checkType); err != nil {
                log.Printf("⚠️ Ошибка записи метрик: %v", err)
            }
        }
    }

    // Добавляем глобальное задание для проверки всех сайтов (каждые 5 минут)
    globalSchedule := "*/5 * * * *" // каждые 5 минут
    err = cronScheduler.AddJob(
        "global_check", 
        "Проверка всех сайтов", 
        globalSchedule,
        createGlobalMonitoringJob(checker),
    )
    if err != nil {
        log.Fatalf("Ошибка добавления глобального задания: %v", err)
    }

    // Загружаем сайты и создаем для них индивидуальные задания
    if err := setupSiteJobs(db, cronScheduler, checker); err != nil {
        log.Printf("⚠️ Ошибка настройки заданий для сайтов: %v", err)
    }

    // Запускаем планировщик
    if err := cronScheduler.Start(); err != nil {
        log.Fatalf("Ошибка запуска планировщика: %v", err)
    }
    defer cronScheduler.Stop()

    // Настройка веб-сервера
    r := mux.NewRouter()
    handlers.RegisterRoutes(r, db)

    // Добавляем роуты для управления планировщиком
    r.HandleFunc("/api/scheduler/jobs", getSchedulerJobs(cronScheduler)).Methods("GET")
    r.HandleFunc("/api/scheduler/jobs/{id}/enable", enableJob(cronScheduler)).Methods("POST")
    r.HandleFunc("/api/scheduler/jobs/{id}/disable", disableJob(cronScheduler)).Methods("POST")
    r.HandleFunc("/api/scheduler/jobs/{id}/status", getJobStatus(cronScheduler)).Methods("GET")

    if metricsService != nil {
        handlers.SetMetricsService(metricsService)
    }

    // Обработка сигналов для graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        log.Printf("🌐 Сервер запущен на http://localhost%s", cfg.ServerAddress)
        log.Println("🕐 Cron-подобный планировщик активен")
        
        if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
            log.Printf("Ошибка сервера: %v", err)
        }
    }()

    // Ожидание сигнала завершения
    <-sigChan
    log.Println("🛑 Получен сигнал завершения, останавливаем сервисы...")
    
    cronScheduler.Stop()
    if metricsService != nil {
        metricsService.Close()
    }
    db.Close()
    
    log.Println("✅ Сервисы остановлены")
}

// createGlobalMonitoringJob создает задание для проверки всех сайтов
func createGlobalMonitoringJob(checker *monitor.Checker) func() error {
    return func() error {
        return checker.CheckAllSitesScheduled()
    }
}

// createSiteMonitoringJob создает задание для мониторинга конкретного сайта
func createSiteMonitoringJob(siteID int, siteURL string, checker *monitor.Checker) func() error {
    return func() error {
        return checker.CheckSiteScheduled(siteID, siteURL)
    }
}

// setupSiteJobs настраивает задания для мониторинга отдельных сайтов
func setupSiteJobs(db *database.DB, cronScheduler *scheduler.CronScheduler, checker *monitor.Checker) error {
    log.Println("⚙️ Настройка индивидуальных заданий для сайтов...")
    
    sites, err := db.GetAllSites()
    if err != nil {
        return fmt.Errorf("ошибка получения списка сайтов: %w", err)
    }

    for _, site := range sites {
        config, err := db.GetSiteConfig(site.ID)
        if err != nil {
            log.Printf("⚠️ Не удалось получить конфигурацию для сайта %s, используем базовые настройки", site.URL)
            config = &models.SiteConfig{
                SiteID: site.ID,
                CheckInterval: 300, // 5 минут
                ScheduleEnabled: false,
                CronSchedule: "",
                Enabled: true,
            }
        }

        if !config.Enabled {
            log.Printf("⏭️ Сайт %s отключен, пропускаем", site.URL)
            continue
        }

        // Получаем эффективное расписание
        schedule := config.GetEffectiveSchedule()
        jobID := fmt.Sprintf("site_%d", site.ID)
        jobName := fmt.Sprintf("Мониторинг %s", site.URL)

        err = cronScheduler.AddJob(
            jobID,
            jobName,
            schedule,
            createSiteMonitoringJob(site.ID, site.URL, checker),
        )
        
        if err != nil {
            log.Printf("❌ Ошибка добавления задания для сайта %s: %v", site.URL, err)
            continue
        }

        log.Printf("✅ Добавлено задание для сайта %s: %s (%s)", 
            site.URL, schedule, config.GetScheduleDescription())
    }

    return nil
}

// HTTP handlers для управления планировщиком

func getSchedulerJobs(cronScheduler *scheduler.CronScheduler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        jobs := cronScheduler.GetJobs()
        
        result := make([]map[string]interface{}, 0)
        for _, job := range jobs {
            status := cronScheduler.GetJobStatus(job.ID)
            result = append(result, status)
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "jobs": result,
            "total": len(result),
        })
    }
}

func enableJob(cronScheduler *scheduler.CronScheduler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        jobID := vars["id"]

        err := cronScheduler.EnableJob(jobID)
        if err != nil {
            http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "enabled"})
    }
}

func disableJob(cronScheduler *scheduler.CronScheduler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        jobID := vars["id"]

        err := cronScheduler.DisableJob(jobID)
        if err != nil {
            http.Error(w, fmt.Sprintf(`{"error": "%s"}`, err.Error()), http.StatusNotFound)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "disabled"})
    }
}

func getJobStatus(cronScheduler *scheduler.CronScheduler) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        vars := mux.Vars(r)
        jobID := vars["id"]

        status := cronScheduler.GetJobStatus(jobID)
        
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(status)
    }
}