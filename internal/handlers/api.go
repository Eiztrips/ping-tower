package handlers

import (
	"encoding/json"
	"net/http"
	"site-monitor/internal/database"
	"site-monitor/internal/models"
	"site-monitor/internal/monitor"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/mux"
)

type SiteStatusResponse struct {
	URL    string `json:"url"`
	Status string `json:"status"`
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
	r.HandleFunc("/", WebInterfaceHandler()).Methods("GET")
	r.HandleFunc("/api/sites", AddSiteHandler(db)).Methods("POST")
	r.HandleFunc("/api/sites", GetAllSitesHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}/status", GetSiteStatusHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/delete", DeleteSiteByURLHandler(db)).Methods("DELETE")
	r.HandleFunc("/api/sites/{id}/history", GetSiteHistoryHandler(db)).Methods("GET")
	r.HandleFunc("/api/dashboard/stats", GetDashboardStatsHandler(db)).Methods("GET")
	r.HandleFunc("/api/check", TriggerCheckHandler(db)).Methods("POST")
	r.HandleFunc("/api/sse", SSEHandler()).Methods("GET")
	r.HandleFunc("/api/sites/{id}/config", GetSiteConfigHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{id}/config", UpdateSiteConfigHandler(db)).Methods("PUT")

	r.HandleFunc("/api/metrics/sites/{id}/hourly", HandleGetHourlyMetrics).Methods("GET")
	r.HandleFunc("/api/metrics/sites/{id}/performance", HandleGetPerformanceSummary).Methods("GET")
	r.HandleFunc("/api/metrics/ssl/alerts", HandleGetSSLAlerts).Methods("GET")
	r.HandleFunc("/api/metrics/health", HandleGetSystemHealth).Methods("GET")
	r.HandleFunc("/api/metrics/stats", HandleGetMetricsStats).Methods("GET")
}

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

		BroadcastSSE("site_added", map[string]string{"url": req.URL})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Сайт успешно добавлен для мониторинга"})
	}
}

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
			URL:    site.URL,
			Status: site.Status,
			LastChecked: site.LastChecked.Format("2006-01-02 15:04:05"),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func CheckSiteStatus(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Проверка сайтов инициирована"})
	}
}

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

func GetSiteConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		siteID := vars["id"]

		var id int
		if _, err := fmt.Sscanf(siteID, "%d", &id); err != nil {
			w.Header().Set("Content-Type", "application/json")
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

func UpdateSiteConfigHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		siteID := vars["id"]

		var id int
		if _, err := fmt.Sscanf(siteID, "%d", &id); err != nil {
			w.Header().Set("Content-Type", "application/json")
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
			"config": config,
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}
