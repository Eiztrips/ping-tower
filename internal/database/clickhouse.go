package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseDB struct {
	conn driver.Conn
}

type ClickHouseConfig struct {
	Host     string
	Port     int
	Database string
	Username string
	Password string
	Debug    bool
}

func NewClickHouseDB(config ClickHouseConfig) (*ClickHouseDB, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", config.Host, config.Port)},
		Auth: clickhouse.Auth{
			Database: config.Database,
			Username: config.Username,
			Password: config.Password,
		},
		Debug: config.Debug,
		Debugf: func(format string, v ...interface{}) {
			if config.Debug {
				log.Printf("[ClickHouse Debug] "+format, v...)
			}
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		DialTimeout:      time.Second * 30,
		MaxOpenConns:     10,
		MaxIdleConns:     5,
		ConnMaxLifetime:  time.Hour,
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	chDB := &ClickHouseDB{conn: conn}

	if err := chDB.initializeSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Println("✅ Successfully connected to ClickHouse")
	return chDB, nil
}

func (ch *ClickHouseDB) initializeSchema() error {
	ctx := context.Background()

	schemas := []string{
		`CREATE TABLE IF NOT EXISTS site_metrics (
			timestamp DateTime64(3) DEFAULT now64(),
			timestamp_date DateTime DEFAULT now(),
			site_id UInt32,
			site_url String,
			status String,
			status_code UInt16,
			response_time_ms UInt64,
			content_length UInt64 DEFAULT 0,
			dns_time_ms UInt64 DEFAULT 0,
			connect_time_ms UInt64 DEFAULT 0,
			tls_time_ms UInt64 DEFAULT 0,
			ttfb_ms UInt64 DEFAULT 0,
			ssl_valid UInt8 DEFAULT 0,
			ssl_expiry Nullable(DateTime),
			ssl_key_length UInt16 DEFAULT 0,
			ssl_algorithm String DEFAULT '',
			ssl_issuer String DEFAULT '',
			content_hash String DEFAULT '',
			content_type String DEFAULT '',
			redirect_count UInt8 DEFAULT 0,
			final_url String DEFAULT '',
			server_type String DEFAULT '',
			powered_by String DEFAULT '',
			cache_control String DEFAULT '',
			error_message String DEFAULT '',
			check_type Enum8('manual' = 1, 'automatic' = 2) DEFAULT 'automatic',
			config_version UInt32 DEFAULT 1
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp_date)
		ORDER BY (site_id, timestamp)
		TTL timestamp_date + INTERVAL 3 MONTH  -- Уменьшено с 1 года до 3 месяцев
		SETTINGS index_granularity = 8192`,

		`CREATE MATERIALIZED VIEW IF NOT EXISTS site_metrics_hourly
		ENGINE = SummingMergeTree()
		PARTITION BY toYYYYMM(hour)
		ORDER BY (site_id, hour)
		TTL hour + INTERVAL 1 MONTH  -- TTL для hourly данных
		AS SELECT
			toStartOfHour(timestamp_date) as hour,
			site_id,
			any(site_url) as site_url,  -- Используем any() вместо site_url для экономии
			count() as total_checks,
			countIf(status = 'up') as successful_checks,
			avg(response_time_ms) as avg_response_time,
			quantile(0.95)(response_time_ms) as p95_response_time,
			min(response_time_ms) as min_response_time,
			max(response_time_ms) as max_response_time,
			avgIf(dns_time_ms, dns_time_ms > 0) as avg_dns_time,
			avgIf(connect_time_ms, connect_time_ms > 0) as avg_connect_time,
			avgIf(tls_time_ms, tls_time_ms > 0) as avg_tls_time,
			avgIf(ttfb_ms, ttfb_ms > 0) as avg_ttfb,
			countIf(ssl_valid = 1) as ssl_valid_count
		FROM site_metrics
		WHERE timestamp_date >= subtractDays(now(), 7)  -- Только последние 7 дней для hourly
		GROUP BY hour, site_id`,

		`CREATE MATERIALIZED VIEW IF NOT EXISTS site_metrics_daily
		ENGINE = SummingMergeTree()
		PARTITION BY toYYYYMM(day)
		ORDER BY (site_id, day)
		TTL day + INTERVAL 6 MONTH  -- TTL для daily данных
		AS SELECT
			toDate(timestamp_date) as day,
			site_id,
			any(site_url) as site_url,
			count() as total_checks,
			countIf(status = 'up') as successful_checks,
			(countIf(status = 'up') * 100.0 / count()) as uptime_percent,
			avg(response_time_ms) as avg_response_time,
			quantile(0.95)(response_time_ms) as p95_response_time,
			min(response_time_ms) as min_response_time,
			max(response_time_ms) as max_response_time
		FROM site_metrics
		WHERE timestamp_date >= subtractMonths(now(), 3)  -- Только последние 3 месяца
		GROUP BY day, site_id`,

		`CREATE TABLE IF NOT EXISTS downtime_events (
			event_id UUID DEFAULT generateUUIDv4(),
			site_id UInt32,
			site_url String,
			start_time DateTime64(3),
			start_time_date DateTime DEFAULT now(),
			end_time Nullable(DateTime64(3)),
			duration_seconds Nullable(UInt64),
			error_message String DEFAULT '',
			status_code UInt16 DEFAULT 0,
			is_resolved UInt8 DEFAULT 0,
			created_at DateTime64(3) DEFAULT now64()
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(start_time_date)
		ORDER BY (site_id, start_time)
		TTL start_time_date + INTERVAL 6 MONTH  -- 6 месяцев для downtime events
		SETTINGS index_granularity = 8192`,

		`CREATE TABLE IF NOT EXISTS ssl_certificates (
			site_id UInt32,
			site_url String,
			ssl_issuer String,
			ssl_algorithm String,
			ssl_key_length UInt16,
			ssl_expiry DateTime,
			days_until_expiry Int32,
			last_checked DateTime64(3) DEFAULT now64(),
			is_valid UInt8 DEFAULT 1
		) ENGINE = ReplacingMergeTree(last_checked)
		PARTITION BY toYYYYMM(ssl_expiry)
		ORDER BY (site_id, ssl_expiry)
		TTL ssl_expiry + INTERVAL 1 MONTH  -- Убираем старые SSL записи через месяц после истечения
		SETTINGS index_granularity = 8192`,
	}

	for i, schema := range schemas {
		if err := ch.conn.Exec(ctx, schema); err != nil {
			return fmt.Errorf("failed to execute schema %d: %w", i+1, err)
		}
	}

	log.Println("✅ ClickHouse schema initialized with optimized TTL and storage settings")
	return nil
}

func (ch *ClickHouseDB) Close() error {
	if ch.conn != nil {
		return ch.conn.Close()
	}
	return nil
}

func (ch *ClickHouseDB) Ping() error {
	return ch.conn.Ping(context.Background())
}

type SiteMetric struct {
	Timestamp       time.Time
	TimestampDate   time.Time
	SiteID          uint32
	SiteURL         string
	Status          string
	StatusCode      uint16
	ResponseTimeMs  uint64
	ContentLength   uint64
	DNSTimeMs       uint64
	ConnectTimeMs   uint64
	TLSTimeMs       uint64
	TTFBMs          uint64
	SSLValid        uint8
	SSLExpiry       *time.Time
	SSLKeyLength    uint16
	SSLAlgorithm    string
	SSLIssuer       string
	ContentHash     string
	ContentType     string
	RedirectCount   uint8
	FinalURL        string
	ServerType      string
	PoweredBy       string
	CacheControl    string
	ErrorMessage    string
	CheckType       string
	ConfigVersion   uint32
}

func (ch *ClickHouseDB) InsertMetric(metric SiteMetric) error {
	ctx := context.Background()

	query := `INSERT INTO site_metrics (
		timestamp, timestamp_date, site_id, site_url, status, status_code, response_time_ms,
		content_length, dns_time_ms, connect_time_ms, tls_time_ms, ttfb_ms,
		ssl_valid, ssl_expiry, ssl_key_length, ssl_algorithm, ssl_issuer,
		content_hash, content_type, redirect_count, final_url,
		server_type, powered_by, cache_control, error_message, check_type, config_version
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	return ch.conn.Exec(ctx, query,
		metric.Timestamp, metric.TimestampDate, metric.SiteID, metric.SiteURL, metric.Status,
		metric.StatusCode, metric.ResponseTimeMs, metric.ContentLength,
		metric.DNSTimeMs, metric.ConnectTimeMs, metric.TLSTimeMs, metric.TTFBMs,
		metric.SSLValid, metric.SSLExpiry, metric.SSLKeyLength,
		metric.SSLAlgorithm, metric.SSLIssuer, metric.ContentHash,
		metric.ContentType, metric.RedirectCount, metric.FinalURL,
		metric.ServerType, metric.PoweredBy, metric.CacheControl,
		metric.ErrorMessage, metric.CheckType, metric.ConfigVersion,
	)
}

func (ch *ClickHouseDB) InsertMetricsBatch(metrics []SiteMetric) error {
	if len(metrics) == 0 {
		return nil
	}

	ctx := context.Background()
	batch, err := ch.conn.PrepareBatch(ctx, `INSERT INTO site_metrics (
		timestamp, timestamp_date, site_id, site_url, status, status_code, response_time_ms,
		content_length, dns_time_ms, connect_time_ms, tls_time_ms, ttfb_ms,
		ssl_valid, ssl_expiry, ssl_key_length, ssl_algorithm, ssl_issuer,
		content_hash, content_type, redirect_count, final_url,
		server_type, powered_by, cache_control, error_message, check_type, config_version
	)`)

	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, metric := range metrics {
		err := batch.Append(
			metric.Timestamp, metric.TimestampDate, metric.SiteID, metric.SiteURL, metric.Status,
			metric.StatusCode, metric.ResponseTimeMs, metric.ContentLength,
			metric.DNSTimeMs, metric.ConnectTimeMs, metric.TLSTimeMs, metric.TTFBMs,
			metric.SSLValid, metric.SSLExpiry, metric.SSLKeyLength,
			metric.SSLAlgorithm, metric.SSLIssuer, metric.ContentHash,
			metric.ContentType, metric.RedirectCount, metric.FinalURL,
			metric.ServerType, metric.PoweredBy, metric.CacheControl,
			metric.ErrorMessage, metric.CheckType, metric.ConfigVersion,
		)
		if err != nil {
			return fmt.Errorf("failed to append to batch: %w", err)
		}
	}

	return batch.Send()
}

type HourlyMetrics struct {
	Hour              time.Time
	SiteID            uint32
	SiteURL           string
	TotalChecks       uint64
	SuccessfulChecks  uint64
	AvgResponseTime   float64
	P95ResponseTime   float64
	P99ResponseTime   float64
	MinResponseTime   uint64
	MaxResponseTime   uint64
	AvgContentLength  float64
	AvgDNSTime        float64
	AvgConnectTime    float64
	AvgTLSTime        float64
	AvgTTFB           float64
	SSLValidCount     uint64
	UniqueStatusCodes uint64
}

func (ch *ClickHouseDB) GetHourlyMetrics(siteID uint32, hours int) ([]HourlyMetrics, error) {
	ctx := context.Background()

	query := `SELECT
		hour, site_id, site_url, total_checks, successful_checks,
		avg_response_time, p95_response_time, p99_response_time,
		min_response_time, max_response_time, avg_content_length,
		avg_dns_time, avg_connect_time, avg_tls_time, avg_ttfb,
		ssl_valid_count, unique_status_codes
	FROM site_metrics_hourly
	WHERE site_id = ? AND hour >= subtractHours(now(), ?)
	ORDER BY hour DESC`

	rows, err := ch.conn.Query(ctx, query, siteID, hours)
	if err != nil {
		return nil, fmt.Errorf("failed to query hourly metrics: %w", err)
	}
	defer rows.Close()

	var metrics []HourlyMetrics
	for rows.Next() {
		var m HourlyMetrics
		err := rows.Scan(
			&m.Hour, &m.SiteID, &m.SiteURL, &m.TotalChecks, &m.SuccessfulChecks,
			&m.AvgResponseTime, &m.P95ResponseTime, &m.P99ResponseTime,
			&m.MinResponseTime, &m.MaxResponseTime, &m.AvgContentLength,
			&m.AvgDNSTime, &m.AvgConnectTime, &m.AvgTLSTime, &m.AvgTTFB,
			&m.SSLValidCount, &m.UniqueStatusCodes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan hourly metrics: %w", err)
		}
		metrics = append(metrics, m)
	}

	return metrics, nil
}

func (ch *ClickHouseDB) RecordDowntimeEvent(siteID uint32, siteURL, errorMessage string, statusCode uint16) error {
	ctx := context.Background()

	now := time.Now()
	query := `INSERT INTO downtime_events (site_id, site_url, start_time, start_time_date, error_message, status_code)
			  VALUES (?, ?, ?, ?, ?, ?)`

	return ch.conn.Exec(ctx, query, siteID, siteURL, now, now, errorMessage, statusCode)
}

func (ch *ClickHouseDB) ResolveDowntimeEvent(siteID uint32) error {
	ctx := context.Background()

	query := `ALTER TABLE downtime_events UPDATE
			  end_time = now(),
			  duration_seconds = dateDiff('second', start_time, now()),
			  is_resolved = 1
			  WHERE site_id = ? AND is_resolved = 0`

	return ch.conn.Exec(ctx, query, siteID)
}

func (ch *ClickHouseDB) UpdateSSLCertificate(siteID uint32, siteURL, issuer, algorithm string, keyLength uint16, expiry time.Time) error {
	ctx := context.Background()

	daysUntilExpiry := int32(time.Until(expiry).Hours() / 24)
	isValid := uint8(0)
	if time.Now().Before(expiry) {
		isValid = 1
	}

	query := `INSERT INTO ssl_certificates (
		site_id, site_url, ssl_issuer, ssl_algorithm, ssl_key_length,
		ssl_expiry, days_until_expiry, is_valid
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	return ch.conn.Exec(ctx, query, siteID, siteURL, issuer, algorithm, keyLength, expiry, daysUntilExpiry, isValid)
}

func (ch *ClickHouseDB) GetExpiringSSLCertificates(days int) ([]map[string]interface{}, error) {
	ctx := context.Background()

	query := `SELECT site_id, site_url, ssl_issuer, ssl_expiry, days_until_expiry
			  FROM ssl_certificates
			  WHERE days_until_expiry <= ? AND is_valid = 1
			  ORDER BY days_until_expiry ASC`

	rows, err := ch.conn.Query(ctx, query, days)
	if err != nil {
		return nil, fmt.Errorf("failed to query expiring SSL certificates: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var siteID uint32
		var siteURL, issuer string
		var expiry time.Time
		var daysUntil int32

		err := rows.Scan(&siteID, &siteURL, &issuer, &expiry, &daysUntil)
		if err != nil {
			return nil, fmt.Errorf("failed to scan SSL certificate: %w", err)
		}

		results = append(results, map[string]interface{}{
			"site_id":           siteID,
			"site_url":          siteURL,
			"ssl_issuer":        issuer,
			"ssl_expiry":        expiry,
			"days_until_expiry": daysUntil,
		})
	}

	return results, nil
}

func (ch *ClickHouseDB) CleanupOldData() error {
	ctx := context.Background()
	
	queries := []string{
		`OPTIMIZE TABLE site_metrics FINAL`,
		`OPTIMIZE TABLE site_metrics_hourly FINAL`, 
		`OPTIMIZE TABLE site_metrics_daily FINAL`,
		`OPTIMIZE TABLE downtime_events FINAL`,
		`OPTIMIZE TABLE ssl_certificates FINAL`,
	}
	
	for _, query := range queries {
		if err := ch.conn.Exec(ctx, query); err != nil {
			log.Printf("⚠️ Failed to optimize table: %v", err)
		}
	}
	
	log.Println("✅ ClickHouse tables optimized")
	return nil
}