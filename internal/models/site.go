package models

import (
	"fmt"
	"time"
)

type Site struct {
	ID                int         `json:"id"`
	URL               string      `json:"url"`
	Status            string      `json:"status"`
	StatusCode        int         `json:"status_code"`
	ResponseTime      int64       `json:"response_time_ms"`
	ContentLength     int64       `json:"content_length"`
	SSLValid          bool        `json:"ssl_valid"`
	SSLExpiry         *time.Time  `json:"ssl_expiry"`
	LastError         string      `json:"last_error"`
	UptimePercent     float64     `json:"uptime_percent"`
	TotalChecks       int         `json:"total_checks"`
	SuccessfulChecks  int         `json:"successful_checks"`
	LastChecked       time.Time   `json:"last_checked"`
	CreatedAt         time.Time   `json:"created_at"`
	Config             *SiteConfig `json:"config,omitempty"`
	
	DNSTime           int64       `json:"dns_time"`
	ConnectTime       int64       `json:"connect_time"`
	TLSTime           int64       `json:"tls_time"`
	TTFB              int64       `json:"ttfb"`
	ContentHash       string      `json:"content_hash"`
	RedirectCount     int         `json:"redirect_count"`
	FinalURL          string      `json:"final_url"`
	
	SSLKeyLength      int         `json:"ssl_key_length"`
	SSLAlgorithm      string      `json:"ssl_algorithm"`
	SSLIssuer         string      `json:"ssl_issuer"`
	
	ServerType        string      `json:"server_type"`
	PoweredBy         string      `json:"powered_by"`
	ContentType       string      `json:"content_type"`
	CacheControl      string      `json:"cache_control"`
}

type SiteConfig struct {
	SiteID           int                    `json:"site_id"`
	CheckInterval    int                    `json:"check_interval"`
	CronSchedule     string                 `json:"cron_schedule"`     // Новое поле для cron выражений
	ScheduleEnabled  bool                   `json:"schedule_enabled"`  // Включен ли планировщик
	Timeout          int                    `json:"timeout"`
	ExpectedStatus   int                    `json:"expected_status"`
	FollowRedirects  bool                   `json:"follow_redirects"`
	MaxRedirects     int                    `json:"max_redirects"`
	CheckSSL         bool                   `json:"check_ssl"`
	SSLAlertDays     int                    `json:"ssl_alert_days"`
	CheckKeywords    string                 `json:"check_keywords"`
	AvoidKeywords    string                 `json:"avoid_keywords"`
	Headers          map[string]interface{} `json:"headers"`
	UserAgent        string                 `json:"user_agent"`
	Enabled          bool                   `json:"enabled"`
	NotifyOnDown     bool                   `json:"notify_on_down"`
	NotifyOnUp       bool                   `json:"notify_on_up"`
	
	CollectDNSTime       bool `json:"collect_dns_time"`
	CollectConnectTime   bool `json:"collect_connect_time"`
	CollectTLSTime       bool `json:"collect_tls_time"`
	CollectTTFB          bool `json:"collect_ttfb"`
	CollectContentHash   bool `json:"collect_content_hash"`
	CollectRedirects     bool `json:"collect_redirects"`
	CollectSSLDetails    bool `json:"collect_ssl_details"`
	CollectServerInfo    bool `json:"collect_server_info"`
	CollectHeaders       bool `json:"collect_headers"`
	
	ShowResponseTime     bool `json:"show_response_time"`
	ShowContentLength    bool `json:"show_content_length"`
	ShowUptime          bool `json:"show_uptime"`
	ShowSSLInfo         bool `json:"show_ssl_info"`
	ShowServerInfo      bool `json:"show_server_info"`
	ShowPerformance     bool `json:"show_performance"`
	ShowRedirectInfo    bool `json:"show_redirect_info"`
	ShowContentInfo     bool `json:"show_content_info"`
	
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// GetEffectiveSchedule возвращает эффективное расписание (cron или интервал)
func (sc *SiteConfig) GetEffectiveSchedule() string {
	if sc.ScheduleEnabled && sc.CronSchedule != "" {
		return sc.CronSchedule
	}
	
	// Конвертируем интервал в cron-выражение
	if sc.CheckInterval <= 0 {
		sc.CheckInterval = 300 // 5 минут по умолчанию
	}
	
	if sc.CheckInterval < 60 {
		// Для интервалов менее минуты используем каждую минуту
		return "* * * * *"
	}
	
	minutes := sc.CheckInterval / 60
	if minutes >= 60 {
		hours := minutes / 60
		if hours >= 24 {
			// Раз в день
			return "0 0 * * *"
		}
		// Каждые N часов
		return fmt.Sprintf("0 */%d * * *", hours)
	}
	
	// Каждые N минут
	return fmt.Sprintf("*/%d * * * *", minutes)
}

// GetScheduleDescription возвращает человекочитаемое описание расписания
func (sc *SiteConfig) GetScheduleDescription() string {
	if sc.ScheduleEnabled && sc.CronSchedule != "" {
		return fmt.Sprintf("По расписанию: %s", sc.CronSchedule)
	}
	
	if sc.CheckInterval <= 0 {
		return "Не настроено"
	}
	
	if sc.CheckInterval < 60 {
		return fmt.Sprintf("Каждые %d секунд", sc.CheckInterval)
	}
	
	minutes := sc.CheckInterval / 60
	if minutes < 60 {
		return fmt.Sprintf("Каждые %d минут", minutes)
	}
	
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("Каждые %d часов", hours)
	}
	
	days := hours / 24
	return fmt.Sprintf("Каждые %d дней", days)
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