CREATE TABLE IF NOT EXISTS sites (
    id SERIAL PRIMARY KEY,
    url VARCHAR(255) NOT NULL UNIQUE,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sites_url ON sites(url);

INSERT INTO sites (url) VALUES 
    ('https://google.com'),
    ('https://github.com') 
ON CONFLICT (url) DO NOTHING;