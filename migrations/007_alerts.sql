-- Add alert configuration table
	CREATE TABLE IF NOT EXISTS alert_configs (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		enabled BOOLEAN DEFAULT TRUE,
		email_enabled BOOLEAN DEFAULT FALSE,
		webhook_enabled BOOLEAN DEFAULT FALSE,
		telegram_enabled BOOLEAN DEFAULT FALSE,
		-- Email settings
		smtp_server VARCHAR(255) DEFAULT '',
		smtp_port VARCHAR(10) DEFAULT '587',
		smtp_username VARCHAR(255) DEFAULT '',
		smtp_password VARCHAR(255) DEFAULT '',
		email_from VARCHAR(255) DEFAULT '',
		email_to TEXT DEFAULT '',
		-- Webhook settings
		webhook_url TEXT DEFAULT '',
		webhook_headers JSONB DEFAULT '{}',
		webhook_timeout INTEGER DEFAULT 10,
		-- Telegram settings
		telegram_bot_token VARCHAR(500) DEFAULT '',
		telegram_chat_id VARCHAR(255) DEFAULT '',
		-- Alert conditions
		alert_on_down BOOLEAN DEFAULT TRUE,
		alert_on_up BOOLEAN DEFAULT FALSE,
		alert_on_ssl_expiry BOOLEAN DEFAULT TRUE,
		ssl_expiry_days INTEGER DEFAULT 30,
		alert_on_status_code_change BOOLEAN DEFAULT FALSE,
		alert_on_response_time_threshold BOOLEAN DEFAULT FALSE,
		response_time_threshold INTEGER DEFAULT 5000,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Insert default global alert configuration
	INSERT INTO alert_configs (name, enabled, alert_on_down, alert_on_up) VALUES ('global', true, true, true) ON CONFLICT (name) DO NOTHING;

	-- Add site-specific alert configuration mapping
	CREATE TABLE IF NOT EXISTS site_alert_configs (
		site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
		alert_config_id INTEGER REFERENCES alert_configs(id) ON DELETE CASCADE,
		PRIMARY KEY (site_id, alert_config_id)
	);

	-- Add alert history table for tracking sent alerts
	CREATE TABLE IF NOT EXISTS alert_history (
		id SERIAL PRIMARY KEY,
		site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
		alert_config_id INTEGER REFERENCES alert_configs(id) ON DELETE CASCADE,
		alert_type VARCHAR(50) NOT NULL,
		channel VARCHAR(20) NOT NULL, -- email, webhook, telegram
		status VARCHAR(20) NOT NULL, -- sent, failed
		message TEXT DEFAULT '',
		error_message TEXT DEFAULT '',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes for better performance
	CREATE INDEX IF NOT EXISTS idx_alert_configs_enabled ON alert_configs(enabled);
	CREATE INDEX IF NOT EXISTS idx_alert_history_site_id ON alert_history(site_id);
	CREATE INDEX IF NOT EXISTS idx_alert_history_created_at ON alert_history(created_at);