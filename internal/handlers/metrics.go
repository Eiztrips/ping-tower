package handlers

import (
	"encoding/json"
	"fmt"
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
		vars := mux.Vars(r)
		siteIDStr := vars["id"]
		hoursStr := r.URL.Query().Get("hours")
		hours := 24
		if hoursStr != "" {
			if h, err := strconv.Atoi(hoursStr); err == nil {
				hours = h
			}
		}

		mockResponse := map[string]interface{}{
			"site_id": siteIDStr,
			"hours":   hours,
			"metrics": []map[string]interface{}{
				{
					"hour":              "2024-01-01T12:00:00Z",
					"total_checks":      12,
					"successful_checks": 11,
					"avg_response_time": 450,
					"avg_dns_time":      25,
					"avg_connect_time":  35,
					"avg_tls_time":      45,
					"avg_ttfb":          120,
				},
			},
		}
		json.NewEncoder(w).Encode(mockResponse)
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
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch hourly metrics: %v"}`, err), http.StatusInternalServerError)
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
		mockSummary := map[string]interface{}{
			"site_id":            1,
			"site_url":           "https://example.com",
			"period":             "Last 24 hours",
			"total_checks":       144,
			"successful_checks":  142,
			"uptime_percent":     98.6,
			"avg_response_time":  450,
			"avg_dns_time":      25,
			"avg_connect_time":  35,
			"avg_tls_time":      45,
			"avg_ttfb":          120,
		}
		json.NewEncoder(w).Encode(mockSummary)
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
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch performance summary: %v"}`, err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(summary)
}

func HandleGetSSLAlerts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	daysStr := r.URL.Query().Get("days")
	days := 30
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil {
			days = d
		}
	}

	if metricsService == nil {
		mockAlerts := map[string]interface{}{
			"expiring_within_days": days,
			"certificates": []map[string]interface{}{
				{
					"site_id":          1,
					"site_url":         "https://example.com",
					"ssl_issuer":      "Let's Encrypt",
					"days_until_expiry": 15,
				},
			},
		}
		json.NewEncoder(w).Encode(mockAlerts)
		return
	}

	expiring, err := metricsService.GetExpiringSSLCertificates(days)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "Failed to fetch SSL alerts: %v"}`, err), http.StatusInternalServerError)
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
			"current_size":  stats["buffer_size"],
			"batch_size":    stats["batch_size"],
			"flush_interval": stats["flush_interval"],
		},
		"database_status": map[string]interface{}{
			"clickhouse": stats["clickhouse_connected"],
			"postgres":   stats["postgres_connected"],
		},
	}

	json.NewEncoder(w).Encode(response)
}