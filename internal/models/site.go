package models

import (
	"time"
)

type Site struct {
	ID              int       `json:"id"`
	URL             string    `json:"url"`
	Status          string    `json:"status"`
	StatusCode      int       `json:"status_code"`
	ResponseTime    int64     `json:"response_time_ms"`
	ContentLength   int64     `json:"content_length"`
	SSLValid        bool      `json:"ssl_valid"`
	SSLExpiry       *time.Time `json:"ssl_expiry"`
	LastError       string    `json:"last_error"`
	UptimePercent   float64   `json:"uptime_percent"`
	TotalChecks     int       `json:"total_checks"`
	SuccessfulChecks int      `json:"successful_checks"`
	LastChecked     time.Time `json:"last_checked"`
	CreatedAt       time.Time `json:"created_at"`
}

type SiteHistory struct {
	ID           int       `json:"id"`
	SiteID       int       `json:"site_id"`
	Status       string    `json:"status"`
	StatusCode   int       `json:"status_code"`
	ResponseTime int64     `json:"response_time_ms"`
	Error        string    `json:"error"`
	CheckedAt    time.Time `json:"checked_at"`
}