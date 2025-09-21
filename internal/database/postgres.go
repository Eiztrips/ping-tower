package database

import (
	"encoding/json"
	"database/sql"
	"fmt"
	"log"
	"ping-tower/internal/models"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func NewDB(dataSourceName string) (*DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("ошибка при открытии базы данных: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка при подключении к базе данных: %w", err)
	}

	dbInstance := &DB{db}

	log.Println("Успешное подключение к базе данных")
	return dbInstance, nil
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.DB.Exec(query, args...)
}

func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) GetSiteByURL(url string) (*models.Site, error) {
	var site models.Site
	var sslExpiry sql.NullTime
	query := `SELECT id, url, status, 
              COALESCE(status_code, 0) as status_code,
              COALESCE(response_time, 0) as response_time,
              COALESCE(content_length, 0) as content_length,
              COALESCE(ssl_valid, false) as ssl_valid,
              ssl_expiry,
              COALESCE(last_error, '') as last_error,
              COALESCE(total_checks, 0) as total_checks,
              COALESCE(successful_checks, 0) as successful_checks,
              CASE 
                  WHEN COALESCE(total_checks, 0) > 0 THEN (COALESCE(successful_checks, 0)::float / COALESCE(total_checks, 1)::float * 100)
                  ELSE 0 
              END as uptime_percent,
              COALESCE(last_checked, created_at) as last_checked,
              created_at,
              COALESCE(dns_time, 0) as dns_time,
              COALESCE(connect_time, 0) as connect_time,
              COALESCE(tls_time, 0) as tls_time,
              COALESCE(ttfb, 0) as ttfb,
              COALESCE(content_hash, '') as content_hash,
              COALESCE(redirect_count, 0) as redirect_count,
              COALESCE(final_url, url) as final_url,
              COALESCE(ssl_key_length, 0) as ssl_key_length,
              COALESCE(ssl_algorithm, '') as ssl_algorithm,
              COALESCE(ssl_issuer, '') as ssl_issuer,
              COALESCE(server_type, '') as server_type,
              COALESCE(powered_by, '') as powered_by,
              COALESCE(content_type, '') as content_type,
              COALESCE(cache_control, '') as cache_control
              FROM sites WHERE url = $1`
    
    err := db.QueryRow(query, url).Scan(
        &site.ID, &site.URL, &site.Status, &site.StatusCode, &site.ResponseTime,
        &site.ContentLength, &site.SSLValid, &sslExpiry, &site.LastError,
        &site.TotalChecks, &site.SuccessfulChecks, &site.UptimePercent,
        &site.LastChecked, &site.CreatedAt,
        &site.DNSTime, &site.ConnectTime, &site.TLSTime, &site.TTFB,
        &site.ContentHash, &site.RedirectCount, &site.FinalURL,
        &site.SSLKeyLength, &site.SSLAlgorithm, &site.SSLIssuer,
        &site.ServerType, &site.PoweredBy, &site.ContentType, &site.CacheControl)
    
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("Сайт не найден")
        }
        return nil, fmt.Errorf("Ошибка запроса к сайту: %w", err)
    }
    
    if sslExpiry.Valid {
        site.SSLExpiry = &sslExpiry.Time
    }

    return &site, nil
}

func (db *DB) AddSite(url string) error {
    query := "INSERT INTO sites (url) VALUES ($1)"
    _, err := db.Exec(query, url)
    if err != nil {
        return fmt.Errorf("Ошибка добавления сайта: %w", err)
    }
    
    log.Printf("Сайт %s добавлен для мониторинга", url)
    return nil
}

func (db *DB) GetSiteConfig(siteID int) (*models.SiteConfig, error) {
	var config models.SiteConfig
	var headersJSON []byte
	
	query := `SELECT site_id, check_interval, timeout, expected_status, follow_redirects, 
			  max_redirects, check_ssl, ssl_alert_days, check_keywords, avoid_keywords,
			  COALESCE(headers, '{}'), user_agent, enabled, notify_on_down, notify_on_up,
			  COALESCE(collect_dns_time, FALSE), COALESCE(collect_connect_time, FALSE), 
			  COALESCE(collect_tls_time, FALSE), COALESCE(collect_ttfb, FALSE),
			  COALESCE(collect_content_hash, FALSE), COALESCE(collect_redirects, FALSE),
			  COALESCE(collect_ssl_details, TRUE), COALESCE(collect_server_info, FALSE),
			  COALESCE(collect_headers, FALSE), COALESCE(show_response_time, TRUE),
			  COALESCE(show_content_length, TRUE), COALESCE(show_uptime, TRUE),
			  COALESCE(show_ssl_info, TRUE), COALESCE(show_server_info, FALSE),
			  COALESCE(show_performance, FALSE), COALESCE(show_redirect_info, FALSE),
			  COALESCE(show_content_info, FALSE),
			  created_at, updated_at FROM site_configs WHERE site_id = $1`
	
	err := db.QueryRow(query, siteID).Scan(
		&config.SiteID, &config.CheckInterval, &config.Timeout, &config.ExpectedStatus,
		&config.FollowRedirects, &config.MaxRedirects, &config.CheckSSL, &config.SSLAlertDays,
		&config.CheckKeywords, &config.AvoidKeywords, &headersJSON, &config.UserAgent,
		&config.Enabled, &config.NotifyOnDown, &config.NotifyOnUp,
		&config.CollectDNSTime, &config.CollectConnectTime, &config.CollectTLSTime,
		&config.CollectTTFB, &config.CollectContentHash, &config.CollectRedirects,
		&config.CollectSSLDetails, &config.CollectServerInfo, &config.CollectHeaders,
		&config.ShowResponseTime, &config.ShowContentLength, &config.ShowUptime,
		&config.ShowSSLInfo, &config.ShowServerInfo, &config.ShowPerformance,
		&config.ShowRedirectInfo, &config.ShowContentInfo,
		&config.CreatedAt, &config.UpdatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("конфигурация не найдена")
		}
		return nil, err
	}
	
	config.Headers = make(map[string]interface{})
	if len(headersJSON) > 0 {
		json.Unmarshal(headersJSON, &config.Headers)
	}
	
	return &config, nil
}

func (db *DB) UpdateSiteConfig(config *models.SiteConfig) error {
	headersJSON, _ := json.Marshal(config.Headers)
	
	query := `UPDATE site_configs SET 
			  check_interval = $2, timeout = $3, expected_status = $4, follow_redirects = $5,
			  max_redirects = $6, check_ssl = $7, ssl_alert_days = $8, check_keywords = $9,
			  avoid_keywords = $10, headers = $11, user_agent = $12, enabled = $13,
			  notify_on_down = $14, notify_on_up = $15,
			  collect_dns_time = $16, collect_connect_time = $17, collect_tls_time = $18,
			  collect_ttfb = $19, collect_content_hash = $20, collect_redirects = $21,
			  collect_ssl_details = $22, collect_server_info = $23, collect_headers = $24,
			  show_response_time = $25, show_content_length = $26, show_uptime = $27,
			  show_ssl_info = $28, show_server_info = $29, show_performance = $30,
			  show_redirect_info = $31, show_content_info = $32,
			  updated_at = CURRENT_TIMESTAMP
			  WHERE site_id = $1`
	
	_, err := db.Exec(query, config.SiteID, config.CheckInterval, config.Timeout,
		config.ExpectedStatus, config.FollowRedirects, config.MaxRedirects, config.CheckSSL,
		config.SSLAlertDays, config.CheckKeywords, config.AvoidKeywords, headersJSON,
		config.UserAgent, config.Enabled, config.NotifyOnDown, config.NotifyOnUp,
		config.CollectDNSTime, config.CollectConnectTime, config.CollectTLSTime,
		config.CollectTTFB, config.CollectContentHash, config.CollectRedirects,
		config.CollectSSLDetails, config.CollectServerInfo, config.CollectHeaders,
		config.ShowResponseTime, config.ShowContentLength, config.ShowUptime,
		config.ShowSSLInfo, config.ShowServerInfo, config.ShowPerformance,
		config.ShowRedirectInfo, config.ShowContentInfo)
	
	return err
}

func (db *DB) GetAllSites() ([]models.Site, error) {
	query := `SELECT 
				s.id, s.url, s.status, 
				COALESCE(s.status_code, 0) as status_code, 
				COALESCE(s.response_time, 0) as response_time, 
				COALESCE(s.content_length, 0) as content_length, 
				COALESCE(s.ssl_valid, false) as ssl_valid, 
				s.ssl_expiry, 
				COALESCE(s.last_error, '') as last_error, 
				COALESCE(s.total_checks, 0) as total_checks, 
				COALESCE(s.successful_checks, 0) as successful_checks,
				CASE 
					WHEN COALESCE(s.total_checks, 0) > 0 THEN (COALESCE(s.successful_checks, 0)::float / COALESCE(s.total_checks, 1)::float * 100)
					ELSE 0 
				END as uptime_percent,
				COALESCE(s.last_checked, s.created_at) as last_checked, 
				s.created_at,
				COALESCE(s.dns_time, 0) as dns_time,
				COALESCE(s.connect_time, 0) as connect_time,
				COALESCE(s.tls_time, 0) as tls_time,
				COALESCE(s.ttfb, 0) as ttfb,
				COALESCE(s.content_hash, '') as content_hash,
				COALESCE(s.redirect_count, 0) as redirect_count,
				COALESCE(s.final_url, s.url) as final_url,
				COALESCE(s.ssl_key_length, 0) as ssl_key_length,
				COALESCE(s.ssl_algorithm, '') as ssl_algorithm,
				COALESCE(s.ssl_issuer, '') as ssl_issuer,
				COALESCE(s.server_type, '') as server_type,
				COALESCE(s.powered_by, '') as powered_by,
				COALESCE(s.content_type, '') as content_type,
				COALESCE(s.cache_control, '') as cache_control,
				c.enabled
			  FROM sites s
			  LEFT JOIN site_configs c ON s.id = c.site_id
			  ORDER BY s.created_at DESC`
	
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("Ошибка получения списка сайтов: %w", err)
	}
	defer rows.Close()

	var sites []models.Site
	for rows.Next() {
		var site models.Site
		var sslExpiry sql.NullTime
		var enabled sql.NullBool
		err := rows.Scan(
			&site.ID, &site.URL, &site.Status, &site.StatusCode, &site.ResponseTime,
			&site.ContentLength, &site.SSLValid, &sslExpiry, &site.LastError,
			&site.TotalChecks, &site.SuccessfulChecks, &site.UptimePercent,
			&site.LastChecked, &site.CreatedAt,
			&site.DNSTime, &site.ConnectTime, &site.TLSTime, &site.TTFB,
			&site.ContentHash, &site.RedirectCount, &site.FinalURL,
			&site.SSLKeyLength, &site.SSLAlgorithm, &site.SSLIssuer,
			&site.ServerType, &site.PoweredBy, &site.ContentType, &site.CacheControl,
			&enabled)
		if err != nil {
			return nil, fmt.Errorf("Ошибка чтения данных сайта: %w", err)
		}
		
		if sslExpiry.Valid {
			site.SSLExpiry = &sslExpiry.Time
		}
		
		sites = append(sites, site)
	}

	return sites, nil
}

func (db *DB) GetSiteHistory(siteID int, limit int) ([]models.SiteHistory, error) {
    query := `SELECT id, site_id, status, status_code, response_time, error, checked_at 
              FROM site_history 
              WHERE site_id = $1 
              ORDER BY checked_at DESC 
              LIMIT $2`
    
    rows, err := db.Query(query, siteID, limit)
    if err != nil {
        return nil, fmt.Errorf("Ошибка получения истории сайта: %w", err)
    }
    defer rows.Close()

    var history []models.SiteHistory
    for rows.Next() {
        var h models.SiteHistory
        err := rows.Scan(&h.ID, &h.SiteID, &h.Status, &h.StatusCode, &h.ResponseTime, &h.Error, &h.CheckedAt)
        if err != nil {
            return nil, fmt.Errorf("Ошибка чтения истории: %w", err)
        }
        history = append(history, h)
    }

    return history, nil
}

func (db *DB) DeleteSite(url string) error {
    query := "DELETE FROM sites WHERE url = $1"
    result, err := db.Exec(query, url)
    if err != nil {
        return fmt.Errorf("Ошибка удаления сайта: %w", err)
    }

    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("Ошибка проверки результата удаления: %w", err)
    }

    if rowsAffected == 0 {
        return fmt.Errorf("Сайт не найден")
    }

    log.Printf("Сайт %s удален из мониторинга", url)
    return nil
}

func (db *DB) UpdateSiteStatus(id int, status string) error {
    query := `UPDATE sites SET 
                status = $1::varchar, 
                last_checked = CURRENT_TIMESTAMP,
                total_checks = COALESCE(total_checks, 0) + 1,
                successful_checks = COALESCE(successful_checks, 0) + CASE WHEN $1::varchar = 'up' THEN 1 ELSE 0 END
              WHERE id = $2`
    _, err := db.Exec(query, status, id)
    if err != nil {
        return fmt.Errorf("Ошибка обновления статуса сайта: %w", err)
    }
    return nil
}

func (db *DB) TriggerCheck() error {
    log.Println("🔄 Принудительный запуск проверки всех сайтов")
    _, err := db.Exec("UPDATE sites SET last_checked = last_checked - INTERVAL '1 hour'")
    return err
}

// Alert configuration functions
func (db *DB) GetAlertConfig(name string) (*models.AlertConfig, error) {
	var config models.AlertConfig
	var webhookHeadersJSON []byte

	query := `SELECT id, name, enabled, email_enabled, webhook_enabled, telegram_enabled,
			  smtp_server, smtp_port, smtp_username, smtp_password, email_from, email_to,
			  webhook_url, COALESCE(webhook_headers, '{}'), webhook_timeout,
			  telegram_bot_token, telegram_chat_id,
			  alert_on_down, alert_on_up, alert_on_ssl_expiry, ssl_expiry_days,
			  alert_on_status_code_change, alert_on_response_time_threshold, response_time_threshold,
			  created_at, updated_at FROM alert_configs WHERE name = $1`

	err := db.QueryRow(query, name).Scan(
		&config.ID, &config.Name, &config.Enabled, &config.EmailEnabled, &config.WebhookEnabled, &config.TelegramEnabled,
		&config.SMTPServer, &config.SMTPPort, &config.SMTPUsername, &config.SMTPPassword, &config.EmailFrom, &config.EmailTo,
		&config.WebhookURL, &webhookHeadersJSON, &config.WebhookTimeout,
		&config.TelegramBotToken, &config.TelegramChatID,
		&config.AlertOnDown, &config.AlertOnUp, &config.AlertOnSSLExpiry, &config.SSLExpiryDays,
		&config.AlertOnStatusCodeChange, &config.AlertOnResponseTimeThreshold, &config.ResponseTimeThreshold,
		&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("конфигурация алертов не найдена")
		}
		return nil, err
	}

	config.WebhookHeaders = make(map[string]string)
	if len(webhookHeadersJSON) > 0 {
		json.Unmarshal(webhookHeadersJSON, &config.WebhookHeaders)
	}

	return &config, nil
}

func (db *DB) UpdateAlertConfig(config *models.AlertConfig) error {
	webhookHeadersJSON, _ := json.Marshal(config.WebhookHeaders)

	query := `UPDATE alert_configs SET
			  enabled = $2, email_enabled = $3, webhook_enabled = $4, telegram_enabled = $5,
			  smtp_server = $6, smtp_port = $7, smtp_username = $8, smtp_password = $9,
			  email_from = $10, email_to = $11,
			  webhook_url = $12, webhook_headers = $13, webhook_timeout = $14,
			  telegram_bot_token = $15, telegram_chat_id = $16,
			  alert_on_down = $17, alert_on_up = $18, alert_on_ssl_expiry = $19, ssl_expiry_days = $20,
			  alert_on_status_code_change = $21, alert_on_response_time_threshold = $22,
			  response_time_threshold = $23, updated_at = CURRENT_TIMESTAMP
			  WHERE name = $1`

	_, err := db.Exec(query, config.Name, config.Enabled, config.EmailEnabled, config.WebhookEnabled, config.TelegramEnabled,
		config.SMTPServer, config.SMTPPort, config.SMTPUsername, config.SMTPPassword,
		config.EmailFrom, config.EmailTo,
		config.WebhookURL, webhookHeadersJSON, config.WebhookTimeout,
		config.TelegramBotToken, config.TelegramChatID,
		config.AlertOnDown, config.AlertOnUp, config.AlertOnSSLExpiry, config.SSLExpiryDays,
		config.AlertOnStatusCodeChange, config.AlertOnResponseTimeThreshold, config.ResponseTimeThreshold)

	return err
}

func (db *DB) GetAllAlertConfigs() ([]models.AlertConfig, error) {
	query := `SELECT id, name, enabled, email_enabled, webhook_enabled, telegram_enabled,
			  smtp_server, smtp_port, smtp_username, smtp_password, email_from, email_to,
			  webhook_url, COALESCE(webhook_headers, '{}'), webhook_timeout,
			  telegram_bot_token, telegram_chat_id,
			  alert_on_down, alert_on_up, alert_on_ssl_expiry, ssl_expiry_days,
			  alert_on_status_code_change, alert_on_response_time_threshold, response_time_threshold,
			  created_at, updated_at FROM alert_configs ORDER BY created_at`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения конфигураций алертов: %w", err)
	}
	defer rows.Close()

	var configs []models.AlertConfig
	for rows.Next() {
		var config models.AlertConfig
		var webhookHeadersJSON []byte

		err := rows.Scan(
			&config.ID, &config.Name, &config.Enabled, &config.EmailEnabled, &config.WebhookEnabled, &config.TelegramEnabled,
			&config.SMTPServer, &config.SMTPPort, &config.SMTPUsername, &config.SMTPPassword, &config.EmailFrom, &config.EmailTo,
			&config.WebhookURL, &webhookHeadersJSON, &config.WebhookTimeout,
			&config.TelegramBotToken, &config.TelegramChatID,
			&config.AlertOnDown, &config.AlertOnUp, &config.AlertOnSSLExpiry, &config.SSLExpiryDays,
			&config.AlertOnStatusCodeChange, &config.AlertOnResponseTimeThreshold, &config.ResponseTimeThreshold,
			&config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("ошибка чтения данных конфигурации алертов: %w", err)
		}

		config.WebhookHeaders = make(map[string]string)
		if len(webhookHeadersJSON) > 0 {
			json.Unmarshal(webhookHeadersJSON, &config.WebhookHeaders)
		}

		configs = append(configs, config)
	}

	return configs, nil
}

func (db *DB) CreateAlertConfig(config *models.AlertConfig) error {
	webhookHeadersJSON, _ := json.Marshal(config.WebhookHeaders)

	query := `INSERT INTO alert_configs
			  (name, enabled, email_enabled, webhook_enabled, telegram_enabled,
			   smtp_server, smtp_port, smtp_username, smtp_password, email_from, email_to,
			   webhook_url, webhook_headers, webhook_timeout,
			   telegram_bot_token, telegram_chat_id,
			   alert_on_down, alert_on_up, alert_on_ssl_expiry, ssl_expiry_days,
			   alert_on_status_code_change, alert_on_response_time_threshold, response_time_threshold)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
			  RETURNING id`

	err := db.QueryRow(query, config.Name, config.Enabled, config.EmailEnabled, config.WebhookEnabled, config.TelegramEnabled,
		config.SMTPServer, config.SMTPPort, config.SMTPUsername, config.SMTPPassword,
		config.EmailFrom, config.EmailTo,
		config.WebhookURL, webhookHeadersJSON, config.WebhookTimeout,
		config.TelegramBotToken, config.TelegramChatID,
		config.AlertOnDown, config.AlertOnUp, config.AlertOnSSLExpiry, config.SSLExpiryDays,
		config.AlertOnStatusCodeChange, config.AlertOnResponseTimeThreshold, config.ResponseTimeThreshold).Scan(&config.ID)

	return err
}

func (db *DB) DeleteAlertConfig(name string) error {
	query := "DELETE FROM alert_configs WHERE name = $1"
	result, err := db.Exec(query, name)
	if err != nil {
		return fmt.Errorf("ошибка удаления конфигурации алертов: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка проверки результата удаления: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("конфигурация алертов не найдена")
	}

	return nil
}

func (db *DB) LogAlert(siteID int, alertConfigID int, alertType, channel, status, message, errorMessage string) error {
	query := `INSERT INTO alert_history (site_id, alert_config_id, alert_type, channel, status, message, error_message)
			  VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := db.Exec(query, siteID, alertConfigID, alertType, channel, status, message, errorMessage)
	return err
}