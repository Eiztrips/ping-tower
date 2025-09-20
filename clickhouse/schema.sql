-- Main metrics table for storing all site monitoring data
CREATE TABLE IF NOT EXISTS site_metrics (
    timestamp DateTime64(3) DEFAULT now64(),
    timestamp_date DateTime DEFAULT now(),
    site_id UInt32,
    site_url String,
    status String,
    status_code UInt16,
    response_time_ms UInt64,
    content_length UInt64,

    -- Performance metrics
    dns_time_ms UInt64 DEFAULT 0,
    connect_time_ms UInt64 DEFAULT 0,
    tls_time_ms UInt64 DEFAULT 0,
    ttfb_ms UInt64 DEFAULT 0,

    -- SSL metrics
    ssl_valid UInt8 DEFAULT 0,
    ssl_expiry Nullable(DateTime),
    ssl_key_length UInt16 DEFAULT 0,
    ssl_algorithm String DEFAULT '',
    ssl_issuer String DEFAULT '',

    -- Content metrics
    content_hash String DEFAULT '',
    content_type String DEFAULT '',

    -- Redirect metrics
    redirect_count UInt8 DEFAULT 0,
    final_url String DEFAULT '',

    -- Server info
    server_type String DEFAULT '',
    powered_by String DEFAULT '',
    cache_control String DEFAULT '',

    -- Error tracking
    error_message String DEFAULT '',

    -- Metadata
    check_type Enum8('manual' = 1, 'automatic' = 2) DEFAULT 'automatic',
    config_version UInt32 DEFAULT 1
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp_date)
ORDER BY (site_id, timestamp)
TTL timestamp_date + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;

-- Aggregated hourly metrics for faster queries
CREATE MATERIALIZED VIEW IF NOT EXISTS site_metrics_hourly
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(hour)
ORDER BY (site_id, hour)
AS SELECT
    toStartOfHour(timestamp_date) as hour,
    site_id,
    site_url,
    count() as total_checks,
    countIf(status = 'up') as successful_checks,
    avg(response_time_ms) as avg_response_time,
    quantile(0.95)(response_time_ms) as p95_response_time,
    quantile(0.99)(response_time_ms) as p99_response_time,
    min(response_time_ms) as min_response_time,
    max(response_time_ms) as max_response_time,
    avg(content_length) as avg_content_length,
    avg(dns_time_ms) as avg_dns_time,
    avg(connect_time_ms) as avg_connect_time,
    avg(tls_time_ms) as avg_tls_time,
    avg(ttfb_ms) as avg_ttfb,
    countIf(ssl_valid = 1) as ssl_valid_count,
    uniq(status_code) as unique_status_codes
FROM site_metrics
GROUP BY hour, site_id, site_url;

-- Daily aggregated metrics
CREATE MATERIALIZED VIEW IF NOT EXISTS site_metrics_daily
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(day)
ORDER BY (site_id, day)
AS SELECT
    toDate(timestamp_date) as day,
    site_id,
    site_url,
    count() as total_checks,
    countIf(status = 'up') as successful_checks,
    (countIf(status = 'up') * 100.0 / count()) as uptime_percent,
    avg(response_time_ms) as avg_response_time,
    quantile(0.95)(response_time_ms) as p95_response_time,
    quantile(0.99)(response_time_ms) as p99_response_time,
    min(response_time_ms) as min_response_time,
    max(response_time_ms) as max_response_time,
    sum(content_length) as total_content_length,
    avg(dns_time_ms) as avg_dns_time,
    avg(connect_time_ms) as avg_connect_time,
    avg(tls_time_ms) as avg_tls_time,
    avg(ttfb_ms) as avg_ttfb
FROM site_metrics
GROUP BY day, site_id, site_url;

-- Downtime events tracking
CREATE TABLE IF NOT EXISTS downtime_events (
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
SETTINGS index_granularity = 8192;

-- Performance anomalies detection
CREATE TABLE IF NOT EXISTS performance_anomalies (
    anomaly_id UUID DEFAULT generateUUIDv4(),
    site_id UInt32,
    site_url String,
    timestamp DateTime64(3),
    metric_name String,
    metric_value Float64,
    baseline_value Float64,
    deviation_percent Float32,
    severity Enum8('low' = 1, 'medium' = 2, 'high' = 3, 'critical' = 4),
    created_at DateTime64(3) DEFAULT now()
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (site_id, timestamp)
SETTINGS index_granularity = 8192;

-- SSL certificate expiration tracking
CREATE TABLE IF NOT EXISTS ssl_certificates (
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
SETTINGS index_granularity = 8192;

-- Create indexes for better query performance
-- Note: ClickHouse doesn't have traditional indexes like PostgreSQL,
-- but we can create additional sorting keys and projections for optimization

-- Projection for time-based queries
ALTER TABLE site_metrics ADD PROJECTION time_status_projection (
    SELECT *
    ORDER BY (timestamp, status)
);

-- Projection for site-based queries
ALTER TABLE site_metrics ADD PROJECTION site_performance_projection (
    SELECT site_id, timestamp, response_time_ms, status
    ORDER BY (site_id, timestamp)
);