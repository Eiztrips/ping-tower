package handlers

import (
	"encoding/json"
	"net/http"
	"site-monitor/internal/metrics"
	"strconv"

	"github.com/gorilla/mux"
)

var metricsService *metrics.Service

func SetMetricsService(service *metrics.Service) {
	metricsService = service
}

func HandleGetHourlyMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if metricsService == nil {
		http.Error(w, `{"error": "Metrics service not available"}`, http.StatusServiceUnavailable)
		return
	}

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

	hourlyMetrics, err := metricsService.GetHourlyMetrics(siteID, hours)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch hourly metrics"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"site_id": siteID,
		"hours":   hours,
		"metrics": hourlyMetrics,
	})
}

func HandleGetPerformanceSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if metricsService == nil {
		http.Error(w, `{"error": "Metrics service not available"}`, http.StatusServiceUnavailable)
		return
	}

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

	summary, err := metricsService.GetSitePerformanceSummary(siteID, hours)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch performance summary"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

func HandleGetSSLAlerts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if metricsService == nil {
		http.Error(w, `{"error": "Metrics service not available"}`, http.StatusServiceUnavailable)
		return
	}

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	expiring, err := metricsService.GetExpiringSSLCertificates(days)
	if err != nil {
		http.Error(w, `{"error": "Failed to fetch SSL alerts"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"expiring_within_days": days,
		"certificates":         expiring,
	})
}

func HandleGetSystemHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if metricsService == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"metrics_service": "disabled",
			"clickhouse":      false,
		})
		return
	}

	health := metricsService.GetSystemHealth()
	json.NewEncoder(w).Encode(health)
}

func HandleGetMetricsStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if metricsService == nil {
		http.Error(w, `{"error": "Metrics service not available"}`, http.StatusServiceUnavailable)
		return
	}

	stats := metricsService.GetSystemHealth()

	response := map[string]interface{}{
		"service_status": "active",
		"buffer_status": map[string]interface{}{
			"current_size": stats["buffer_size"],
			"batch_size":   stats["batch_size"],
			"flush_interval": stats["flush_interval"],
		},
		"database_status": map[string]interface{}{
			"clickhouse": stats["clickhouse_connected"],
			"postgres":   stats["postgres_connected"],
		},
	}

	json.NewEncoder(w).Encode(response)
}