-- Add detailed monitoring columns
DO $$ 
BEGIN
    -- Add dns_time column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'dns_time') THEN
        ALTER TABLE sites ADD COLUMN dns_time BIGINT DEFAULT 0;
    END IF;
    
    -- Add connect_time column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'connect_time') THEN
        ALTER TABLE sites ADD COLUMN connect_time BIGINT DEFAULT 0;
    END IF;
    
    -- Add tls_time column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'tls_time') THEN
        ALTER TABLE sites ADD COLUMN tls_time BIGINT DEFAULT 0;
    END IF;
    
    -- Add ttfb column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ttfb') THEN
        ALTER TABLE sites ADD COLUMN ttfb BIGINT DEFAULT 0;
    END IF;
    
    -- Add content_hash column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'content_hash') THEN
        ALTER TABLE sites ADD COLUMN content_hash VARCHAR(255) DEFAULT '';
    END IF;
    
    -- Add redirect_count column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'redirect_count') THEN
        ALTER TABLE sites ADD COLUMN redirect_count INTEGER DEFAULT 0;
    END IF;
    
    -- Add final_url column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'final_url') THEN
        ALTER TABLE sites ADD COLUMN final_url TEXT DEFAULT '';
    END IF;
    
    -- Add ssl_key_length column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ssl_key_length') THEN
        ALTER TABLE sites ADD COLUMN ssl_key_length INTEGER DEFAULT 0;
    END IF;
    
    -- Add ssl_algorithm column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ssl_algorithm') THEN
        ALTER TABLE sites ADD COLUMN ssl_algorithm VARCHAR(50) DEFAULT '';
    END IF;
    
    -- Add ssl_issuer column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ssl_issuer') THEN
        ALTER TABLE sites ADD COLUMN ssl_issuer TEXT DEFAULT '';
    END IF;
    
    -- Add server_type column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'server_type') THEN
        ALTER TABLE sites ADD COLUMN server_type VARCHAR(255) DEFAULT '';
    END IF;
    
    -- Add powered_by column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'powered_by') THEN
        ALTER TABLE sites ADD COLUMN powered_by VARCHAR(255) DEFAULT '';
    END IF;
    
    -- Add content_type column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'content_type') THEN
        ALTER TABLE sites ADD COLUMN content_type VARCHAR(255) DEFAULT '';
    END IF;
    
    -- Add cache_control column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'cache_control') THEN
        ALTER TABLE sites ADD COLUMN cache_control VARCHAR(255) DEFAULT '';
    END IF;
END $$;
