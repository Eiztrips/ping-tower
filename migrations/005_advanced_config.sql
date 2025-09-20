-- Add advanced configuration columns for metric collection and display control
DO $$ 
BEGIN
    -- Metric collection flags - собираем ВСЕ метрики по умолчанию
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_dns_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_dns_time BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_connect_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_connect_time BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_tls_time') THEN
        ALTER TABLE site_configs ADD COLUMN collect_tls_time BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_ttfb') THEN
        ALTER TABLE site_configs ADD COLUMN collect_ttfb BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_content_hash') THEN
        ALTER TABLE site_configs ADD COLUMN collect_content_hash BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_redirects') THEN
        ALTER TABLE site_configs ADD COLUMN collect_redirects BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_ssl_details') THEN
        ALTER TABLE site_configs ADD COLUMN collect_ssl_details BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_server_info') THEN
        ALTER TABLE site_configs ADD COLUMN collect_server_info BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'collect_headers') THEN
        ALTER TABLE site_configs ADD COLUMN collect_headers BOOLEAN DEFAULT TRUE;
    END IF;
    
    -- Display control flags - показываем ВСЕ метрики по умолчанию
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
        ALTER TABLE site_configs ADD COLUMN show_server_info BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_performance') THEN
        ALTER TABLE site_configs ADD COLUMN show_performance BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_redirect_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_redirect_info BOOLEAN DEFAULT TRUE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'site_configs' AND column_name = 'show_content_info') THEN
        ALTER TABLE site_configs ADD COLUMN show_content_info BOOLEAN DEFAULT TRUE;
    END IF;
END $$;

-- Set default values for existing configs - собираем и показываем ВСЕ метрики, интервал 5 минут
UPDATE site_configs SET 
    check_interval = 300, -- 5 минут
    collect_dns_time = TRUE,
    collect_connect_time = TRUE,
    collect_tls_time = TRUE,
    collect_ttfb = TRUE,
    collect_content_hash = TRUE,
    collect_redirects = TRUE,
    collect_ssl_details = TRUE,
    collect_server_info = TRUE,
    collect_headers = TRUE,
    show_response_time = TRUE,
    show_content_length = TRUE,
    show_uptime = TRUE,
    show_ssl_info = TRUE,
    show_server_info = TRUE,
    show_performance = TRUE,
    show_redirect_info = TRUE,
    show_content_info = TRUE,
    updated_at = CURRENT_TIMESTAMP;
