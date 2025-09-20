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
	
	// New detailed metrics
	DNSTime       int64     `json:"dns_time"`
	ConnectTime   int64     `json:"connect_time"`
	TLSTime       int64     `json:"tls_time"`
	TTFB          int64     `json:"ttfb"`
	ContentHash   string    `json:"content_hash"`
	RedirectCount int       `json:"redirect_count"`
	FinalURL      string    `json:"final_url"`
	
	// SSL Details
	SSLKeyLength  int       `json:"ssl_key_length"`
	SSLAlgorithm  string    `json:"ssl_algorithm"`
	SSLIssuer     string    `json:"ssl_issuer"`
	
	// Server Info
	ServerType    string    `json:"server_type"`
	PoweredBy     string    `json:"powered_by"`
	ContentType   string    `json:"content_type"`
	CacheControl  string    `json:"cache_control"`
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