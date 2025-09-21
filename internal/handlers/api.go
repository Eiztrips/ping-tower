// @title Site Monitor API
// @version 1.0.0
// @description Полнофункциональный API для мониторинга сайтов с детальной аналитикой, планировщиком заданий и SSL мониторингом
// @termsOfService https://sitemonitor.com/terms/

// @contact.name Site Monitor Support
// @contact.email support@sitemonitor.com
// @contact.url https://sitemonitor.com/support

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api

// @schemes http https

// @tag.name sites
// @tag.description Управление мониторингом сайтов

// @tag.name dashboard  
// @tag.description Статистика и дашборд

// @tag.name metrics
// @tag.description Детальные метрики и аналитика

// @tag.name ssl
// @tag.description SSL сертификаты и безопасность

// @tag.name health
// @tag.description Состояние системы

package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"site-monitor/internal/config"
	"site-monitor/internal/database"
	"site-monitor/internal/models"
	"site-monitor/internal/monitor"
	"site-monitor/internal/notifications"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

var apiDatabase *database.DB

func SetAPIDatabase(db *database.DB) {
	apiDatabase = db
}

type SiteStatusResponse struct {
	URL         string `json:"url"`
	Status      string `json:"status"`
	LastChecked string `json:"last_checked"`
}

type AddSiteRequest struct {
	URL string `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type DashboardStats struct {
	TotalSites      int     `json:"total_sites"`
	SitesUp         int     `json:"sites_up"`
	SitesDown       int     `json:"sites_down"`
	AvgUptime       float64 `json:"avg_uptime"`
	AvgResponseTime float64 `json:"avg_response_time"`
}

type SSEMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

var sseClients = make(map[chan SSEMessage]bool)
var sseClientsMutex = make(chan bool, 1)

func init() {
	sseClientsMutex <- true
}

func RegisterRoutes(r *mux.Router, db *database.DB) {
	// Set database for handlers
	SetDemoDatabase(db)
	SetAPIDatabase(db)

	// Main interface routes
	r.HandleFunc("/", WebInterfaceHandler()).Methods("GET")
	r.HandleFunc("/demo", DemoHandler()).Methods("GET")
	r.HandleFunc("/metrics", MetricsWebHandler()).Methods("GET")
	r.HandleFunc("/alerts", AlertsWebHandler()).Methods("GET")

	// Swagger documentation
	r.HandleFunc("/swagger", SwaggerUIHandler()).Methods("GET")
	r.HandleFunc("/api/swagger/swagger.yaml", SwaggerYAMLHandler()).Methods("GET")
	r.HandleFunc("/api/swagger/swagger.json", SwaggerJSONHandler()).Methods("GET")
	r.PathPrefix("/swagger-ui/").Handler(http.StripPrefix("/swagger-ui/", http.FileServer(http.Dir("./swagger-ui/"))))

	// API routes for sites management
	r.HandleFunc("/api/sites", AddSiteHandler(db)).Methods("POST")
	r.HandleFunc("/api/sites", GetAllSitesHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}/status", GetSiteStatusHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/delete", DeleteSiteByURLHandler(db)).Methods("DELETE")
	r.HandleFunc("/api/sites/{id}/history", GetSiteHistoryHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{id}/config", GetSiteConfigHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{id}/config", UpdateSiteConfigHandler(db)).Methods("PUT")

	// Dashboard and monitoring
	r.HandleFunc("/api/dashboard/stats", GetDashboardStatsHandler(db)).Methods("GET")
	r.HandleFunc("/api/check", TriggerCheckHandler(db)).Methods("POST")
	r.HandleFunc("/api/sse", SSEHandler()).Methods("GET")
	r.HandleFunc("/api/health", HandleHealthCheck).Methods("GET")

	// Real SSL alerts endpoint using database
	r.HandleFunc("/api/ssl/alerts", HandleGetSSLAlertsFromDB(db)).Methods("GET")

	// Alert configuration endpoints
	r.HandleFunc("/api/alerts/configs", GetAlertConfigsHandler(db)).Methods("GET")
	r.HandleFunc("/api/alerts/configs/{name}", GetAlertConfigHandler(db)).Methods("GET")
	r.HandleFunc("/api/alerts/configs/{name}", UpdateAlertConfigHandler(db)).Methods("PUT")
	r.HandleFunc("/api/alerts/configs", CreateAlertConfigHandler(db)).Methods("POST")
	r.HandleFunc("/api/alerts/configs/{name}", DeleteAlertConfigHandler(db)).Methods("DELETE")
	r.HandleFunc("/api/alerts/test", TestAlertHandler(db)).Methods("POST")

	// Metrics API endpoints - real data from database
	r.HandleFunc("/api/metrics/sites/{id}/hourly", HandleGetHourlyMetricsFromDB(db)).Methods("GET")
	r.HandleFunc("/api/metrics/sites/{id}/performance", HandleGetPerformanceSummaryFromDB(db)).Methods("GET")
	r.HandleFunc("/api/metrics/aggregated", HandleGetAggregatedMetricsFromDB(db)).Methods("GET")
	r.HandleFunc("/api/metrics/health", HandleGetSystemHealthFromDB(db)).Methods("GET")
	r.HandleFunc("/api/metrics/stats", HandleGetMetricsStatsFromDB(db)).Methods("GET")

	// ClickHouse metrics endpoints (if available)
	if metricsService != nil {
		r.HandleFunc("/api/clickhouse/metrics/sites/{id}/hourly", HandleGetHourlyMetrics).Methods("GET")
		r.HandleFunc("/api/clickhouse/metrics/sites/{id}/performance", HandleGetPerformanceSummary).Methods("GET")
		r.HandleFunc("/api/clickhouse/ssl/alerts", HandleGetSSLAlerts).Methods("GET")
		r.HandleFunc("/api/clickhouse/metrics/health", HandleGetSystemHealth).Methods("GET")
		r.HandleFunc("/api/clickhouse/metrics/stats", HandleGetMetricsStats).Methods("GET")
	}
}

// HandleGetSSLAlertsFromDB - получить SSL алерты из PostgreSQL
func HandleGetSSLAlertsFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		daysStr := r.URL.Query().Get("days")
		days := 30
		if daysStr != "" {
			if d, err := strconv.Atoi(daysStr); err == nil {
				days = d
			}
		}

		// Получаем сайты с истекающими SSL сертификатами из PostgreSQL
		query := `SELECT id, url, ssl_issuer, ssl_expiry, ssl_valid
				  FROM sites 
				  WHERE url LIKE 'https://%' 
				  AND ssl_expiry IS NOT NULL 
				  AND ssl_expiry > NOW() 
				  AND ssl_expiry <= NOW() + INTERVAL '%d days'
				  ORDER BY ssl_expiry ASC`

		rows, err := db.Query(fmt.Sprintf(query, days))
		if err != nil {
			log.Printf("Ошибка запроса SSL алертов: %v", err)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"expiring_within_days": days,
				"certificates":         []interface{}{},
			})
			return
		}
		defer rows.Close()

		var certificates []map[string]interface{}
		for rows.Next() {
			var id int
			var url, issuer string
			var expiry time.Time
			var valid bool

			if err := rows.Scan(&id, &url, &issuer, &expiry, &valid); err != nil {
				continue
			}

			daysUntilExpiry := int(time.Until(expiry).Hours() / 24)
			certificates = append(certificates, map[string]interface{}{
				"site_id":           id,
				"site_url":          url,
				"ssl_issuer":        issuer,
				"ssl_expiry":        expiry,
				"days_until_expiry": daysUntilExpiry,
				"is_valid":          valid,
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"expiring_within_days": days,
			"certificates":         certificates,
		})
	}
}

// HandleGetHourlyMetricsFromDB - получить почасовые метрики из PostgreSQL
func HandleGetHourlyMetricsFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		siteIDStr := vars["id"]
		siteID, err := strconv.Atoi(siteIDStr)
		if err != nil {
			http.Error(w, `{"error": "Invalid site ID"}`, http.StatusBadRequest)
			return
		}

		hoursStr := r.URL.Query().Get("hours")
		hours := 24
		if hoursStr != "" {
			if h, err := strconv.Atoi(hoursStr); err == nil {
				hours = h
			}
		}

		// Получаем историю проверок из PostgreSQL
		history, err := db.GetSiteHistory(siteID, hours*4) // Приблизительно по 4 проверки в час
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch history: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Группируем по часам
		hourlyData := make(map[string]*HourlyMetrics)
		for _, record := range history {
			hour := record.CheckedAt.Truncate(time.Hour).Format("2006-01-02T15:04:05Z")
			
			if hourlyData[hour] == nil {
				hourlyData[hour] = &HourlyMetrics{
					Hour:    record.CheckedAt.Truncate(time.Hour),
					SiteID:  uint32(siteID),
					SiteURL: "",
				}
			}
			
			hourlyData[hour].TotalChecks++
			if record.Status == "up" {
				hourlyData[hour].SuccessfulChecks++
			}
			
			if record.ResponseTime > 0 {
				if hourlyData[hour].AvgResponseTime == 0 {
					hourlyData[hour].AvgResponseTime = float64(record.ResponseTime)
				} else {
					hourlyData[hour].AvgResponseTime = (hourlyData[hour].AvgResponseTime + float64(record.ResponseTime)) / 2
				}
			}
		}

		// Преобразуем в слайс
		var metrics []HourlyMetrics
		for _, data := range hourlyData {
			metrics = append(metrics, *data)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"site_id": siteID,
			"hours":   hours,
			"metrics": metrics,
		})
	}
}

// HandleGetPerformanceSummaryFromDB - получить сводку производительности из PostgreSQL
func HandleGetPerformanceSummaryFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		siteIDStr := vars["id"]
		siteID, err := strconv.Atoi(siteIDStr)
		if err != nil {
			http.Error(w, `{"error": "Invalid site ID"}`, http.StatusBadRequest)
			return
		}

		// Получаем сайт из базы данных
		site, err := db.GetSiteByURL("") // Получим через ID
		if err != nil {
			// Получаем через запрос по ID
			query := `SELECT 
						url, status, response_time, content_length, dns_time, connect_time, 
						tls_time, ttfb, total_checks, successful_checks, last_checked
					  FROM sites WHERE id = $1`
			
			var url, status string
			var responseTime, contentLength, dnsTime, connectTime, tlsTime, ttfb int64
			var totalChecks, successfulChecks int
			var lastChecked time.Time
			
			err = db.QueryRow(query, siteID).Scan(
				&url, &status, &responseTime, &contentLength, &dnsTime,
				&connectTime, &tlsTime, &ttfb, &totalChecks, &successfulChecks, &lastChecked,
			)
			
			if err != nil {
				http.Error(w, `{"error": "Site not found"}`, http.StatusNotFound)
				return
			}
			
			uptimePercent := 0.0
			if totalChecks > 0 {
				uptimePercent = float64(successfulChecks) / float64(totalChecks) * 100
			}
			
			summary := map[string]interface{}{
				"site_id":            siteID,
				"site_url":           url,
				"period":             "Реальные данные из БД",
				"total_checks":       totalChecks,
				"successful_checks":  successfulChecks,
				"uptime_percent":     uptimePercent,
				"avg_response_time":  responseTime,
				"avg_dns_time":       dnsTime,
				"avg_connect_time":   connectTime,
				"avg_tls_time":       tlsTime,
				"avg_ttfb":           ttfb,
				"last_checked":       lastChecked,
			}
			
			json.NewEncoder(w).Encode(summary)
			return
		}

		// Если сайт найден через URL, используем его данные
		uptimePercent := 0.0
		if site.TotalChecks > 0 {
			uptimePercent = float64(site.SuccessfulChecks) / float64(site.TotalChecks) * 100
		}
		
		summary := map[string]interface{}{
			"site_id":            site.ID,
			"site_url":           site.URL,
			"period":             "Реальные данные из БД",
			"total_checks":       site.TotalChecks,
			"successful_checks":  site.SuccessfulChecks,
			"uptime_percent":     uptimePercent,
			"avg_response_time":  site.ResponseTime,
			"avg_dns_time":       site.DNSTime,
			"avg_connect_time":   site.ConnectTime,
			"avg_tls_time":       site.TLSTime,
			"avg_ttfb":           site.TTFB,
			"last_checked":       site.LastChecked,
		}
		
		json.NewEncoder(w).Encode(summary)
	}
}

// HandleGetAggregatedMetricsFromDB - получить агрегированные метрики из PostgreSQL
func HandleGetAggregatedMetricsFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		hoursStr := r.URL.Query().Get("hours")
		// Удаляем неиспользуемую переменную hours
		// hours := 24
		// if hoursStr != "" {
		//	if h, err := strconv.Atoi(hoursStr); err == nil {
		//		hours = h
		//	}
		// }

		// Получаем агрегированные данные из PostgreSQL
		sites, err := db.GetAllSites()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch sites: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Агрегируем данные
		totalChecks := 0
		totalResponseTime := int64(0)
		totalDnsTime := int64(0)
		totalConnectTime := int64(0)
		totalTlsTime := int64(0)
		totalTtfb := int64(0)
		responseCount := 0
		sslIssues := 0
		uptimeSum := 0.0

		for _, site := range sites {
			totalChecks += site.TotalChecks
			
			if site.ResponseTime > 0 {
				totalResponseTime += site.ResponseTime
				responseCount++
			}
			if site.DNSTime > 0 {
				totalDnsTime += site.DNSTime
			}
			if site.ConnectTime > 0 {
				totalConnectTime += site.ConnectTime
			}
			if site.TLSTime > 0 {
				totalTlsTime += site.TLSTime
			}
			if site.TTFB > 0 {
				totalTtfb += site.TTFB
			}
			
			if len(site.URL) >= 8 && site.URL[:8] == "https://" && !site.SSLValid {
				sslIssues++
			}
			
			uptimePercent := 0.0
			if site.TotalChecks > 0 {
				uptimePercent = float64(site.SuccessfulChecks) / float64(site.TotalChecks) * 100
			}
			uptimeSum += uptimePercent
		}

		siteCount := len(sites)
		avgResponseTime := 0.0
		avgDnsTime := 0.0
		avgConnectTime := 0.0
		avgTlsTime := 0.0
		avgTtfb := 0.0
		avgUptime := 0.0

		if responseCount > 0 {
			avgResponseTime = float64(totalResponseTime) / float64(responseCount)
		}
		if siteCount > 0 {
			avgDnsTime = float64(totalDnsTime) / float64(siteCount)
			avgConnectTime = float64(totalConnectTime) / float64(siteCount)
			avgTlsTime = float64(totalTlsTime) / float64(siteCount)
			avgTtfb = float64(totalTtfb) / float64(siteCount)
			avgUptime = uptimeSum / float64(siteCount)
		}

		// Определяем период из параметра запроса
		period := "Реальные данные из БД"
		if hoursStr != "" {
			period = fmt.Sprintf("Реальные данные из БД за последние %s часов", hoursStr)
		}

		response := map[string]interface{}{
			"period":             fmt.Sprintf("%s (анализ %d сайтов)", period, siteCount),
			"total_checks":       totalChecks,
			"avg_response_time":  avgResponseTime,
			"avg_dns_time":       avgDnsTime,
			"avg_connect_time":   avgConnectTime,
			"avg_tls_time":       avgTlsTime,
			"avg_ttfb":           avgTtfb,
			"uptime_percent":     avgUptime,
			"ssl_issues":         sslIssues,
			"sites_analyzed":     siteCount,
		}

		json.NewEncoder(w).Encode(response)
	}
}

// HandleGetSystemHealthFromDB - получить состояние системы на основе PostgreSQL
func HandleGetSystemHealthFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		health := map[string]interface{}{
			"postgres_connected":   false,
			"clickhouse_connected": false,
			"sites_count":          0,
			"active_sites":         0,
			"last_check":           nil,
		}

		// Проверяем PostgreSQL
		if err := db.DB.Ping(); err == nil {
			health["postgres_connected"] = true

			// Получаем статистику сайтов
			var sitesCount, activeSites int
			var lastCheck time.Time

			db.QueryRow("SELECT COUNT(*) FROM sites").Scan(&sitesCount)
			db.QueryRow("SELECT COUNT(*) FROM sites WHERE status = 'up'").Scan(&activeSites)
			db.QueryRow("SELECT MAX(last_checked) FROM sites").Scan(&lastCheck)

			health["sites_count"] = sitesCount
			health["active_sites"] = activeSites
			health["last_check"] = lastCheck
		}

		// Проверяем ClickHouse если доступен
		if metricsService != nil {
			systemHealth := metricsService.GetSystemHealth()
			if connected, ok := systemHealth["clickhouse_connected"].(bool); ok {
				health["clickhouse_connected"] = connected
			}
		}

		json.NewEncoder(w).Encode(health)
	}
}

// HandleGetMetricsStatsFromDB - получить статистику метрик из PostgreSQL  
func HandleGetMetricsStatsFromDB(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Получаем статистику из PostgreSQL
		var totalSites, activeSites, totalChecks int
		var avgResponseTime float64

		db.QueryRow("SELECT COUNT(*) FROM sites").Scan(&totalSites)
		db.QueryRow("SELECT COUNT(*) FROM sites WHERE status = 'up'").Scan(&activeSites)
		db.QueryRow("SELECT COALESCE(SUM(total_checks), 0) FROM sites").Scan(&totalChecks)
		db.QueryRow("SELECT COALESCE(AVG(response_time), 0) FROM sites WHERE response_time > 0").Scan(&avgResponseTime)

		response := map[string]interface{}{
			"service_status": "active",
			"database_status": map[string]interface{}{
				"postgres": true,
				"clickhouse": metricsService != nil,
			},
			"sites_metrics": map[string]interface{}{
				"total_sites": totalSites,
				"active_sites": activeSites,
				"total_checks": totalChecks,
				"avg_response_time": avgResponseTime,
			},
		}

		// Добавляем ClickHouse метрики если доступны
		if metricsService != nil {
			stats := metricsService.GetSystemHealth()
			response["clickhouse_metrics"] = map[string]interface{}{
				"buffer_size": stats["buffer_size"],
				"batch_size": stats["batch_size"],
				"flush_interval": stats["flush_interval"],
			}
		}

		json.NewEncoder(w).Encode(response)
	}
}

// HandleGetAllSites - получить все сайты
// @Summary Получить список всех сайтов
// @Description Возвращает список всех сайтов с их статусами, конфигурациями и детальными метриками
// @Tags sites
// @Accept json
// @Produce json
// @Success 200 {array} models.Site "Список сайтов получен успешно"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /sites [get]
func HandleGetAllSites(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	sites, err := apiDatabase.GetAllSites()
	if err != nil {
		log.Printf("Ошибка получения сайтов: %v", err)
		http.Error(w, `{"error": "Failed to fetch sites"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(sites)
}

// HandleAddSite - добавить новый сайт
// @Summary Добавить новый сайт для мониторинга
// @Description Добавляет новый сайт в систему мониторинга с базовой конфигурацией и запускает автоматическую проверку
// @Tags sites
// @Accept json
// @Produce json
// @Param site body AddSiteRequest true "Данные сайта"
// @Success 201 {object} SuccessResponse "Сайт успешно добавлен"
// @Failure 400 {object} ErrorResponse "Неверный формат запроса"
// @Failure 409 {object} ErrorResponse "Сайт уже существует"
// @Router /sites [post]
func HandleAddSite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if request.URL == "" {
		http.Error(w, `{"error": "URL is required"}`, http.StatusBadRequest)
		return
	}

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	err := apiDatabase.AddSite(request.URL)
	if err != nil {
		log.Printf("Ошибка добавления сайта: %v", err)
		http.Error(w, `{"error": "Failed to add site"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Site added successfully",
		"url":     request.URL,
	})
}

// HandleDeleteSite - удалить сайт
// @Summary Удалить сайт из мониторинга
// @Description Удаляет сайт и всю связанную с ним информацию из системы мониторинга
// @Tags sites
// @Accept json
// @Produce json
// @Param site body DeleteSiteRequest true "URL сайта для удаления"
// @Success 200 {object} SuccessResponse "Сайт успешно удален"
// @Failure 400 {object} ErrorResponse "Неверный формат запроса"
// @Failure 404 {object} ErrorResponse "Сайт не найден"
// @Router /sites/delete [delete]
func HandleDeleteSite(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var request struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	err := apiDatabase.DeleteSite(request.URL)
	if err != nil {
		log.Printf("Ошибка удаления сайта: %v", err)
		http.Error(w, `{"error": "Failed to delete site"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Site deleted successfully",
		"url":     request.URL,
	})
}

// HandleGetSiteConfig - получить конфигурацию сайта
// @Summary Получить конфигурацию сайта
// @Description Возвращает детальную конфигурацию мониторинга для конкретного сайта
// @Tags sites
// @Accept json
// @Produce json
// @Param id path int true "ID сайта"
// @Success 200 {object} models.SiteConfig "Конфигурация получена успешно"
// @Failure 400 {object} ErrorResponse "Неверный ID сайта"
// @Failure 404 {object} ErrorResponse "Конфигурация не найдена"
// @Router /sites/{id}/config [get]
func HandleGetSiteConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	siteIDStr := vars["id"]
	siteID, err := strconv.Atoi(siteIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid site ID"}`, http.StatusBadRequest)
		return
	}

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	config, err := apiDatabase.GetSiteConfig(siteID)
	if err != nil {
		log.Printf("Ошибка получения конфигурации: %v", err)
		// Возвращаем базовую конфигурацию если не найдена
		config = &models.SiteConfig{
			SiteID:               siteID,
			CheckInterval:        300,
			Timeout:              30,
			ExpectedStatus:       200,
			FollowRedirects:      true,
			MaxRedirects:         10,
			CheckSSL:             true,
			SSLAlertDays:         30,
			UserAgent:            "Site-Monitor/1.0",
			Enabled:              true,
			NotifyOnDown:         true,
			NotifyOnUp:           true,
			CollectSSLDetails:    true,
			ShowResponseTime:     true,
			ShowContentLength:    true,
			ShowUptime:           true,
			ShowSSLInfo:          true,
		}
	}

	json.NewEncoder(w).Encode(config)
}

// HandleUpdateSiteConfig - обновить конфигурацию сайта
// @Summary Обновить конфигурацию сайта
// @Description Обновляет настройки мониторинга: интервалы проверки, cron расписания, параметры сбора метрик
// @Tags sites
// @Accept json
// @Produce json
// @Param id path int true "ID сайта"
// @Param config body models.SiteConfig true "Новая конфигурация сайта"
// @Success 200 {object} SuccessResponse "Конфигурация обновлена успешно"
// @Failure 400 {object} ErrorResponse "Неверные данные конфигурации"
// @Failure 500 {object} ErrorResponse "Ошибка обновления"
// @Router /sites/{id}/config [put]
func HandleUpdateSiteConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	siteIDStr := vars["id"]
	siteID, err := strconv.Atoi(siteIDStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid site ID"}`, http.StatusBadRequest)
		return
	}

	var config models.SiteConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		http.Error(w, `{"error": "Invalid request format"}`, http.StatusBadRequest)
		return
	}

	config.SiteID = siteID
	err = apiDatabase.UpdateSiteConfig(&config)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	BroadcastSSE("site_config_updated", map[string]interface{}{
		"site_id": siteID,
		"config":  config,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// HandleTriggerCheck - запустить проверку всех сайтов
func HandleTriggerCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	err := apiDatabase.TriggerCheck()
	if err != nil {
		log.Printf("Ошибка запуска проверки: %v", err)
		http.Error(w, `{"error": "Failed to trigger check"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"message": "Check triggered successfully",
	})
}

// HandleDashboardStats - статистика для дашборда
func HandleDashboardStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if apiDatabase == nil {
		http.Error(w, `{"error": "Database not available"}`, http.StatusServiceUnavailable)
		return
	}

	var stats DashboardStats

	// Получаем базовую статистику
	countQuery := `SELECT COUNT(*) FROM sites`
	apiDatabase.QueryRow(countQuery).Scan(&stats.TotalSites)

	if stats.TotalSites > 0 {
		statsQuery := `SELECT 
						COUNT(CASE WHEN status = 'up' THEN 1 END) as up,
						COUNT(CASE WHEN status = 'down' THEN 1 END) as down,
						COALESCE(AVG(CASE WHEN COALESCE(total_checks, 0) > 0 THEN (COALESCE(successful_checks, 0)::float / COALESCE(total_checks, 1)::float * 100) ELSE 0 END), 0) as avg_uptime,
						COALESCE(AVG(COALESCE(response_time, 0)::float), 0) as avg_response_time
					  FROM sites`

		apiDatabase.QueryRow(statsQuery).Scan(&stats.SitesUp, &stats.SitesDown, &stats.AvgUptime, &stats.AvgResponseTime)
	}

	json.NewEncoder(w).Encode(stats)
}

// HandleHealthCheck - проверка здоровья системы
// @Summary Состояние системы
// @Description Возвращает информацию о состоянии всех компонентов системы: PostgreSQL, ClickHouse, количество сайтов
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse "Состояние системы получено успешно"
// @Router /health [get]
func HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"postgres_connected":   false,
		"clickhouse_connected": false,
		"timestamp":            time.Now().Format(time.RFC3339),
	}

	if apiDatabase != nil {
		if err := apiDatabase.DB.Ping(); err == nil {
			health["postgres_connected"] = true
		}
	}

	// Если есть метрики сервис, проверяем ClickHouse
	if metricsService != nil {
		systemHealth := metricsService.GetSystemHealth()
		if connected, ok := systemHealth["clickhouse_connected"].(bool); ok {
			health["clickhouse_connected"] = connected
		}
	}

	json.NewEncoder(w).Encode(health)
}

// SSEHandler - обработчик Server-Sent Events
// @Summary Подключение к реальному времени
// @Description Подключается к потоку Server-Sent Events для получения обновлений в реальном времени
// @Tags health
// @Accept text/event-stream
// @Produce text/event-stream
// @Success 200 {string} string "text/event-stream"
// @Router /sse [get]
func SSEHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		clientChan := make(chan SSEMessage, 10)

		<-sseClientsMutex
		sseClients[clientChan] = true
		sseClientsMutex <- true

		defer func() {
			<-sseClientsMutex
			delete(sseClients, clientChan)
			sseClientsMutex <- true
			close(clientChan)
		}()

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "data: %s\n\n", `{"type":"connected","data":{"message":"Connected to SSE"}}`)
		flusher.Flush()

		for {
			select {
			case msg := <-clientChan:
				data, _ := json.Marshal(msg)
				fmt.Fprintf(w, "data: %s\n\n", string(data))
				flusher.Flush()
			case <-r.Context().Done():
				return
			case <-time.After(30 * time.Second):
				fmt.Fprintf(w, "data: %s\n\n", `{"type":"ping","data":{"timestamp":"`+time.Now().Format(time.RFC3339)+`"}}`)
				flusher.Flush()
			}
		}
	}
}

// BroadcastSSE - отправка SSE сообщений всем клиентам
func BroadcastSSE(msgType string, data interface{}) {
	message := SSEMessage{
		Type: msgType,
		Data: data,
	}

	<-sseClientsMutex
	for client := range sseClients {
		select {
		case client <- message:
		default:
		}
	}
	sseClientsMutex <- true
}

// TriggerCheckHandler - запуск проверки по требованию
// @Summary Запустить проверку всех сайтов
// @Description Инициирует немедленную проверку всех сайтов в системе мониторинга
// @Tags sites
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse "Проверка запущена успешно"
// @Failure 500 {object} ErrorResponse "Ошибка запуска проверки"
// @Router /check [post]
func TriggerCheckHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("🔄 Принудительный запуск проверки всех сайтов")

		BroadcastSSE("check_started", map[string]string{"message": "Проверка запущена"})

		go func() {
			monitor.CheckOnDemand(db)
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Проверка инициирована"})
	}
}

// AddSiteHandler - добавление нового сайта
func AddSiteHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AddSiteRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Неверный формат запроса"})
			return
		}

		if req.URL == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "URL обязателен для заполнения"})
			return
		}

		err := db.AddSite(req.URL)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Сайт уже добавлен для мониторинга"})
			return
		}

		// Получаем ID добавленного сайта для проверки
		site, err := db.GetSiteByURL(req.URL)
		if err != nil {
			log.Printf("❌ Не удалось получить данные добавленного сайта: %v", err)
		} else {
			// Запускаем проверку нового сайта в фоне
			go func() {
				log.Printf("🔍 Запуск автоматической проверки нового сайта: %s", req.URL)

				// Создаем временный checker для проверки
				checker := monitor.NewChecker(db, 0)

				// Получаем конфигурацию или используем базовую
				config, err := db.GetSiteConfig(site.ID)
				if err != nil {
					// Создаем базовую конфигурацию для нового сайта
					defaultConfig := monitor.DefaultSiteConfig
					defaultConfig.SiteID = site.ID
					config = &defaultConfig
				}

				// Выполняем проверку
				result := checker.CheckSiteWithConfig(req.URL, config)

				// Обновляем статус в базе данных
				checker.UpdateSiteStatus(&monitor.Site{ID: site.ID, URL: req.URL}, result)
				checker.SaveCheckHistory(site.ID, result)

				// Отправляем SSE уведомление о проверке
				if monitor.NotifySiteChecked != nil {
					monitor.NotifySiteChecked(req.URL, result)
				}

				log.Printf("✅ Автоматическая проверка нового сайта завершена: %s - %s", req.URL, result.Status)
			}()
		}

		BroadcastSSE("site_added", map[string]string{"url": req.URL})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Сайт успешно добавлен для мониторинга"})
	}
}

// GetAllSitesHandler - получение всех сайтов
func GetAllSitesHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("🔍 Получение списка всех сайтов...")

		sites, err := db.GetAllSites()
		if err != nil {
			log.Printf("❌ Ошибка получения списка сайтов: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения списка сайтов: " + err.Error()})
			return
		}

		for i, site := range sites {
			config, err := db.GetSiteConfig(site.ID)
			if err == nil {
				sites[i].Config = config
			}
		}

		log.Printf("✅ Успешно получен список из %d сайтов с конфигурациями", len(sites))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sites)
	}
}

// DeleteSiteByURLHandler - удаление сайта по URL
func DeleteSiteByURLHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}

		w.Header().Set("Content-Type", "application/json")

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Неверный формат запроса"})
			return
		}

		if req.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "URL обязателен"})
			return
		}

		err := db.DeleteSite(req.URL)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Сайт не найден"})
			return
		}

		BroadcastSSE("site_deleted", map[string]string{"url": req.URL})

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Сайт успешно удален из мониторинга"})
	}
}

// GetSiteStatusHandler - получение статуса сайта
func GetSiteStatusHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		url := vars["url"]

		site, err := db.GetSiteByURL(url)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Сайт не найден"})
			return
		}

		response := SiteStatusResponse{
			URL:         site.URL,
			Status:      site.Status,
			LastChecked: site.LastChecked.Format("2006-01-02 15:04:05"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// CheckSiteStatus - проверка статуса сайта
func CheckSiteStatus(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Проверка сайтов инициирована"})
	}
}

// GetMonitoringResults - получение результатов мониторинга
func GetMonitoringResults(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sites, err := db.GetAllSites()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения результатов мониторинга"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sites)
	}
}

// GetSiteHistoryHandler - получение истории проверок сайта
// @Summary История проверок сайта
// @Description Возвращает историю проверок конкретного сайта с детальной информацией о каждой проверке
// @Tags sites
// @Accept json
// @Produce json
// @Param id path int true "ID сайта"
// @Param limit query int false "Количество записей" default(100)
// @Success 200 {array} models.SiteHistory "История получена успешно"
// @Failure 400 {object} ErrorResponse "Неверный ID сайта"
// @Failure 500 {object} ErrorResponse "Ошибка получения истории"
// @Router /sites/{id}/history [get]
func GetSiteHistoryHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		siteID := vars["id"]

		var id int
		if _, err := fmt.Sscanf(siteID, "%d", &id); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Неверный ID сайта"})
			return
		}

		history, err := db.GetSiteHistory(id, 100)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения истории сайта"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(history)
	}
}

// GetDashboardStatsHandler - получение статистики дашборда
func GetDashboardStatsHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("📊 Получение статистики дашборда...")

		stats := DashboardStats{}

		countQuery := `SELECT COUNT(*) FROM sites`
		err := db.QueryRow(countQuery).Scan(&stats.TotalSites)
		if err != nil {
			log.Printf("❌ Ошибка получения количества сайтов: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения статистики: " + err.Error()})
			return
		}

		if stats.TotalSites > 0 {
			statsQuery := `SELECT 
							COUNT(CASE WHEN status = 'up' THEN 1 END) as up,
							COUNT(CASE WHEN status = 'down' THEN 1 END) as down,
							COALESCE(AVG(CASE WHEN COALESCE(total_checks, 0) > 0 THEN (COALESCE(successful_checks, 0)::float / COALESCE(total_checks, 1)::float * 100) ELSE 0 END), 0) as avg_uptime,
							COALESCE(AVG(COALESCE(response_time, 0)::float), 0) as avg_response_time
						  FROM sites`

			err = db.QueryRow(statsQuery).Scan(&stats.SitesUp, &stats.SitesDown, &stats.AvgUptime, &stats.AvgResponseTime)
			if err != nil {
				log.Printf("❌ Ошибка получения детальной статистики: %v", err)
				stats.SitesUp = 0
				stats.SitesDown = 0
				stats.AvgUptime = 0.0
				stats.AvgResponseTime = 0.0
			}
		}

		log.Printf("📊 Статистика: сайтов всего=%d, онлайн=%d, оффлайн=%d, аптайм=%.1f%%, среднее время=%.0fмс",
			stats.TotalSites, stats.SitesUp, stats.SitesDown, stats.AvgUptime, stats.AvgResponseTime)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}

// GetSiteConfigHandler - получение конфигурации сайта
func GetSiteConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		siteID := vars["id"]

		var id int
		if _, err := fmt.Sscanf(siteID, "%d", &id); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid site ID"})
			return
		}

		config, err := db.GetSiteConfig(id)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	}
}

// UpdateSiteConfigHandler - обновление конфигурации сайта
func UpdateSiteConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		siteID := vars["id"]

		var id int
		if _, err := fmt.Sscanf(siteID, "%d", &id); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid site ID"})
			return
		}

		var config models.SiteConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid request format"})
			return
		}

		config.SiteID = id
		err := db.UpdateSiteConfig(&config)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		BroadcastSSE("site_config_updated", map[string]interface{}{
			"site_id": id,
			"config":  config,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// HourlyMetrics definition
type HourlyMetrics struct {
	Hour              time.Time `json:"hour"`
	SiteID            uint32    `json:"site_id"`
	SiteURL           string    `json:"site_url"`
	TotalChecks       uint64    `json:"total_checks"`
	SuccessfulChecks  uint64    `json:"successful_checks"`
	AvgResponseTime   float64   `json:"avg_response_time"`
	AvgDNSTime        float64   `json:"avg_dns_time"`
	AvgConnectTime    float64   `json:"avg_connect_time"`
	AvgTLSTime        float64   `json:"avg_tls_time"`
	AvgTTFB           float64   `json:"avg_ttfb"`
}

// Alert configuration handlers

// GetAlertConfigsHandler - получить все конфигурации алертов
// @Summary Получить все конфигурации алертов
// @Description Возвращает список всех конфигураций алертов в системе
// @Tags alerts
// @Accept json
// @Produce json
// @Success 200 {array} models.AlertConfig "Список конфигураций алертов"
// @Failure 500 {object} ErrorResponse "Ошибка сервера"
// @Router /alerts/configs [get]
func GetAlertConfigsHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		configs, err := db.GetAllAlertConfigs()
		if err != nil {
			log.Printf("❌ Ошибка получения конфигураций алертов: %v", err)
			http.Error(w, `{"error": "Failed to fetch alert configs"}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(configs)
	}
}

// GetAlertConfigHandler - получить конфигурацию алертов по имени
// @Summary Получить конфигурацию алертов
// @Description Возвращает конфигурацию алертов по имени
// @Tags alerts
// @Accept json
// @Produce json
// @Param name path string true "Имя конфигурации"
// @Success 200 {object} models.AlertConfig "Конфигурация алертов"
// @Failure 404 {object} ErrorResponse "Конфигурация не найдена"
// @Router /alerts/configs/{name} [get]
func GetAlertConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		name := vars["name"]

		config, err := db.GetAlertConfig(name)
		if err != nil {
			log.Printf("❌ Ошибка получения конфигурации алертов %s: %v", name, err)
			http.Error(w, `{"error": "Alert config not found"}`, http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(config)
	}
}

// CreateAlertConfigHandler - создать новую конфигурацию алертов
// @Summary Создать конфигурацию алертов
// @Description Создает новую конфигурацию алертов
// @Tags alerts
// @Accept json
// @Produce json
// @Param config body models.AlertConfig true "Данные конфигурации"
// @Success 201 {object} models.AlertConfig "Конфигурация создана"
// @Failure 400 {object} ErrorResponse "Неверные данные"
// @Failure 409 {object} ErrorResponse "Конфигурация уже существует"
// @Router /alerts/configs [post]
func CreateAlertConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var config models.AlertConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if config.Name == "" {
			http.Error(w, `{"error": "Name is required"}`, http.StatusBadRequest)
			return
		}

		// Initialize webhook headers if nil
		if config.WebhookHeaders == nil {
			config.WebhookHeaders = make(map[string]string)
		}

		err := db.CreateAlertConfig(&config)
		if err != nil {
			log.Printf("❌ Ошибка создания конфигурации алертов: %v", err)
			http.Error(w, `{"error": "Failed to create alert config"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Создана конфигурация алертов: %s", config.Name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(config)
	}
}

// UpdateAlertConfigHandler - обновить конфигурацию алертов
// @Summary Обновить конфигурацию алертов
// @Description Обновляет существующую конфигурацию алертов
// @Tags alerts
// @Accept json
// @Produce json
// @Param name path string true "Имя конфигурации"
// @Param config body models.AlertConfig true "Обновленные данные"
// @Success 200 {object} models.AlertConfig "Конфигурация обновлена"
// @Failure 400 {object} ErrorResponse "Неверные данные"
// @Failure 404 {object} ErrorResponse "Конфигурация не найдена"
// @Router /alerts/configs/{name} [put]
func UpdateAlertConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		name := vars["name"]

		var config models.AlertConfig
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		config.Name = name

		// Initialize webhook headers if nil
		if config.WebhookHeaders == nil {
			config.WebhookHeaders = make(map[string]string)
		}

		err := db.UpdateAlertConfig(&config)
		if err != nil {
			log.Printf("❌ Ошибка обновления конфигурации алертов %s: %v", name, err)
			http.Error(w, `{"error": "Failed to update alert config"}`, http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Обновлена конфигурация алертов: %s", name)
		json.NewEncoder(w).Encode(config)
	}
}

// DeleteAlertConfigHandler - удалить конфигурацию алертов
// @Summary Удалить конфигурацию алертов
// @Description Удаляет конфигурацию алертов по имени
// @Tags alerts
// @Accept json
// @Produce json
// @Param name path string true "Имя конфигурации"
// @Success 200 {object} SuccessResponse "Конфигурация удалена"
// @Failure 404 {object} ErrorResponse "Конфигурация не найдена"
// @Router /alerts/configs/{name} [delete]
func DeleteAlertConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		vars := mux.Vars(r)
		name := vars["name"]

		// Prevent deletion of global config
		if name == "global" {
			http.Error(w, `{"error": "Cannot delete global configuration"}`, http.StatusBadRequest)
			return
		}

		err := db.DeleteAlertConfig(name)
		if err != nil {
			log.Printf("❌ Ошибка удаления конфигурации алертов %s: %v", name, err)
			http.Error(w, `{"error": "Failed to delete alert config"}`, http.StatusNotFound)
			return
		}

		log.Printf("✅ Удалена конфигурация алертов: %s", name)
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Alert configuration deleted successfully"})
	}
}

// TestAlertHandler - тестовая отправка алерта
// @Summary Тестовая отправка алерта
// @Description Отправляет тестовый алерт для проверки конфигурации
// @Tags alerts
// @Accept json
// @Produce json
// @Param test body TestAlertRequest true "Параметры тестового алерта"
// @Success 200 {object} SuccessResponse "Тестовый алерт отправлен"
// @Failure 400 {object} ErrorResponse "Неверные данные"
// @Router /alerts/test [post]
func TestAlertHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var request struct {
			ConfigName string `json:"config_name"`
			TestURL    string `json:"test_url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
			return
		}

		if request.ConfigName == "" {
			http.Error(w, `{"error": "Config name is required"}`, http.StatusBadRequest)
			return
		}

		// Get alert config
		alertConfig, err := db.GetAlertConfig(request.ConfigName)
		if err != nil {
			http.Error(w, `{"error": "Alert config not found"}`, http.StatusNotFound)
			return
		}

		// Convert AlertConfig to notifications config format
		alertsConfig := convertToNotificationsConfig(alertConfig)

		// Create test alert manager
		alertManager := notifications.NewAlertManager(alertsConfig)

		testURL := request.TestURL
		if testURL == "" {
			testURL = "https://example.com"
		}

		// Create test result
		testResult := monitor.CheckResult{
			Status:       "down",
			StatusCode:   200,
			ResponseTime: 150,
			Error:        "Test alert message",
		}

		// Send test alert
		err = alertManager.SendAlert(0, testURL, testResult, "test")
		if err != nil {
			log.Printf("❌ Ошибка отправки тестового алерта: %v", err)
			http.Error(w, fmt.Sprintf(`{"error": "Failed to send test alert: %v"}`, err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Тестовый алерт отправлен для конфигурации: %s", request.ConfigName)
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Test alert sent successfully"})
	}
}

// convertToNotificationsConfig converts database AlertConfig to notifications AlertsConfig
func convertToNotificationsConfig(dbConfig *models.AlertConfig) *config.AlertsConfig {
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
