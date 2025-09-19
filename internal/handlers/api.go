package handlers

import (
	"encoding/json"
	"net/http"
	"site-monitor/internal/database"
	"fmt"
	"log"

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

func RegisterRoutes(r *mux.Router, db *database.DB) {
	r.HandleFunc("/", WebInterfaceHandler()).Methods("GET")
	r.HandleFunc("/api/sites", AddSiteHandler(db)).Methods("POST")
	r.HandleFunc("/api/sites", GetAllSitesHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}/status", GetSiteStatusHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}", DeleteSiteHandler(db)).Methods("DELETE")
	r.HandleFunc("/api/sites/{id}/history", GetSiteHistoryHandler(db)).Methods("GET")
	r.HandleFunc("/api/dashboard/stats", GetDashboardStatsHandler(db)).Methods("GET")
	r.HandleFunc("/api/check", TriggerCheckHandler(db)).Methods("POST")
}

func TriggerCheckHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := db.TriggerCheck()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка запуска проверки"})
			return
		}

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
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения списка сайтов"})
			return
		}

		log.Printf("✅ Успешно получен список из %d сайтов", len(sites))
		
		// Логируем первый сайт для отладки
		if len(sites) > 0 {
			log.Printf("🔍 Первый сайт: ID=%d, URL=%s, Status=%s", 
				sites[0].ID, sites[0].URL, sites[0].Status)
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sites)
	}
}

func DeleteSiteHandler(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		url := vars["url"]

		err := db.DeleteSite(url)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Сайт не найден"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SuccessResponse{Message: "Сайт удален из мониторинга"})
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

		// Convert string to int
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
		stats := DashboardStats{}
		
		// First, get basic count
		countQuery := `SELECT COUNT(*) FROM sites`
		err := db.QueryRow(countQuery).Scan(&stats.TotalSites)
		if err != nil {
			log.Printf("Ошибка получения количества сайтов: %v", err)
		}
		
		// Get detailed stats only if we have sites
		if stats.TotalSites > 0 {
			statsQuery := `SELECT 
							COUNT(CASE WHEN status = 'up' THEN 1 END) as up,
							COUNT(CASE WHEN status = 'down' THEN 1 END) as down,
							COALESCE(AVG(CASE WHEN COALESCE(total_checks, 0) > 0 THEN (COALESCE(successful_checks, 0)::float / COALESCE(total_checks, 1)::float * 100) ELSE 0 END), 0) as avg_uptime,
							COALESCE(AVG(COALESCE(response_time, 0)::float), 0) as avg_response_time
						  FROM sites`
			
			err = db.QueryRow(statsQuery).Scan(&stats.SitesUp, &stats.SitesDown, &stats.AvgUptime, &stats.AvgResponseTime)
			if err != nil {
				log.Printf("Ошибка получения детальной статистики: %v", err)
				// Set default values on error
				stats.SitesUp = 0
				stats.SitesDown = 0
				stats.AvgUptime = 0.0
				stats.AvgResponseTime = 0.0
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	}
}
