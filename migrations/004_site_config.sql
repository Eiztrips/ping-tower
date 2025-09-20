-- Add site configuration table
CREATE TABLE IF NOT EXISTS site_configs (
    site_id INTEGER PRIMARY KEY REFERENCES sites(id) ON DELETE CASCADE,
    check_interval INTEGER DEFAULT 30,
    timeout INTEGER DEFAULT 30,
    expected_status INTEGER DEFAULT 200,
    follow_redirects BOOLEAN DEFAULT TRUE,
    max_redirects INTEGER DEFAULT 10,
    check_ssl BOOLEAN DEFAULT TRUE,
    ssl_alert_days INTEGER DEFAULT 30,
    check_keywords TEXT DEFAULT '',
    avoid_keywords TEXT DEFAULT '',
    headers JSONB DEFAULT '{}',
    user_agent VARCHAR(500) DEFAULT 'Site-Monitor/1.0',
    enabled BOOLEAN DEFAULT TRUE,
    notify_on_down BOOLEAN DEFAULT TRUE,
    notify_on_up BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert default configs for existing sites
INSERT INTO site_configs (site_id) 
SELECT id FROM sites 
ON CONFLICT (site_id) DO NOTHING;

-- Create trigger to auto-create config for new sites
CREATE OR REPLACE FUNCTION create_site_config()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO site_configs (site_id) VALUES (NEW.id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_site_config
    AFTER INSERT ON sites
    FOR EACH ROW
    EXECUTE FUNCTION create_site_config();
