package handlers

import (
	"encoding/json"
	"net/http"
	"site-monitor/internal/database"

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

func RegisterRoutes(r *mux.Router, db *database.DB) {
	r.HandleFunc("/", handlers.WebInterfaceHandler()).Methods("GET")
	r.HandleFunc("/api/sites", AddSiteHandler(db)).Methods("POST")
	r.HandleFunc("/api/sites", GetAllSitesHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}/status", GetSiteStatusHandler(db)).Methods("GET")
	r.HandleFunc("/api/sites/{url}", DeleteSiteHandler(db)).Methods("DELETE")
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
		sites, err := db.GetAllSites()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "Ошибка получения списка сайтов"})
			return
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
