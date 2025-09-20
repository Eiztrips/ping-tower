-- Add new columns for enhanced monitoring
DO $$ 
BEGIN
    -- Add status_code column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'status_code') THEN
        ALTER TABLE sites ADD COLUMN status_code INTEGER DEFAULT 0;
    END IF;
    
    -- Add response_time column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'response_time') THEN
        ALTER TABLE sites ADD COLUMN response_time BIGINT DEFAULT 0;
    END IF;
    
    -- Add content_length column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'content_length') THEN
        ALTER TABLE sites ADD COLUMN content_length BIGINT DEFAULT 0;
    END IF;
    
    -- Add ssl_valid column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ssl_valid') THEN
        ALTER TABLE sites ADD COLUMN ssl_valid BOOLEAN DEFAULT FALSE;
    END IF;
    
    -- Add ssl_expiry column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'ssl_expiry') THEN
        ALTER TABLE sites ADD COLUMN ssl_expiry TIMESTAMP NULL;
    END IF;
    
    -- Add last_error column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'last_error') THEN
        ALTER TABLE sites ADD COLUMN last_error TEXT DEFAULT '';
    END IF;
    
    -- Add total_checks column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'total_checks') THEN
        ALTER TABLE sites ADD COLUMN total_checks INTEGER DEFAULT 0;
    END IF;
    
    -- Add successful_checks column
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'sites' AND column_name = 'successful_checks') THEN
        ALTER TABLE sites ADD COLUMN successful_checks INTEGER DEFAULT 0;
    END IF;
END $$;

-- Create history table for tracking all checks
CREATE TABLE IF NOT EXISTS site_history (
    id SERIAL PRIMARY KEY,
    site_id INTEGER REFERENCES sites(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    status_code INTEGER DEFAULT 0,
    response_time BIGINT DEFAULT 0,
    error TEXT DEFAULT '',
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create additional indexes for better performance
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites(status);
CREATE INDEX IF NOT EXISTS idx_history_site_id ON site_history(site_id);
CREATE INDEX IF NOT EXISTS idx_history_checked_at ON site_history(checked_at);