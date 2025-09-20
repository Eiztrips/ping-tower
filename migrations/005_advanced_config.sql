-- Add advanced configuration columns for metric collection and display control
DO $$ 
BEGIN
    -- Metric collection flags - only basic metrics enabled by default
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_dns_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_dns_time BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_connect_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_connect_time BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_tls_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_tls_time BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_ttfb') THEN
        ALTER TABLE site_configs ADD COLUMN collect_ttfb BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_content_hash') THEN
        ALTER TABLE site_configs ADD COLUMN collect_content_hash BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_redirects') THEN
        ALTER TABLE site_configs ADD COLUMN collect_redirects BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_ssl_details') THEN
        ALTER TABLE site_configs ADD COLUMN collect_ssl_details BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_server_info') THEN
        ALTER TABLE site_configs ADD COLUMN collect_server_info BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_headers') THEN
        ALTER TABLE site_configs ADD COLUMN collect_headers BOOLEAN DEFAULT FALSE;
    END IF;
    
    -- Display control flags - only basic metrics shown by default
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_response_time') THEN
        ALTER TABLE site_configs ADD COLUMN show_response_time BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_content_length') THEN
        ALTER TABLE site_configs ADD COLUMN show_content_length BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_uptime') THEN
        ALTER TABLE site_configs ADD COLUMN show_uptime BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_ssl_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_ssl_info BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_server_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_server_info BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_performance') THEN
        ALTER TABLE site_configs ADD COLUMN show_performance BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_redirect_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_redirect_info BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_content_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_content_info BOOLEAN DEFAULT FALSE;
    END IF;
END $$;

-- Set default values for existing configs - only basic metrics enabled
UPDATE site_configs SET 
    collect_dns_time = FALSE,
    collect_connect_time = FALSE,
    collect_tls_time = FALSE,
    collect_ttfb = FALSE,
    collect_content_hash = FALSE,
    collect_redirects = FALSE,
    collect_ssl_details = TRUE,
    collect_server_info = FALSE,
    collect_headers = FALSE,
    show_response_time = TRUE,
    show_content_length = TRUE,
    show_uptime = TRUE,
    show_ssl_info = TRUE,
    show_server_info = FALSE,
    show_performance = FALSE,
    show_redirect_info = FALSE,
    show_content_info = FALSE,
    updated_at = CURRENT_TIMESTAMP
WHERE collect_dns_time IS NULL;
