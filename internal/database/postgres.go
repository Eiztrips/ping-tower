package database

import (
    "database/sql"
    "fmt"
    "log"
    "site-monitor/internal/models"

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

    if err := dbInstance.runMigrations(); err != nil {
        return nil, fmt.Errorf("ошибка выполнения миграций: %w", err)
    }

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

func (db *DB) GetAllSites() ([]models.Site, error) {
    query := `SELECT 
                id, url, status, 
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
              FROM sites 
              ORDER BY created_at DESC`
    
    rows, err := db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("Ошибка получения списка сайтов: %w", err)
    }
    defer rows.Close()

    var sites []models.Site
    for rows.Next() {
        var site models.Site
        var sslExpiry sql.NullTime
        err := rows.Scan(
            &site.ID, &site.URL, &site.Status, &site.StatusCode, &site.ResponseTime,
            &site.ContentLength, &site.SSLValid, &sslExpiry, &site.LastError,
            &site.TotalChecks, &site.SuccessfulChecks, &site.UptimePercent,
            &site.LastChecked, &site.CreatedAt,
            &site.DNSTime, &site.ConnectTime, &site.TLSTime, &site.TTFB,
            &site.ContentHash, &site.RedirectCount, &site.FinalURL,
            &site.SSLKeyLength, &site.SSLAlgorithm, &site.SSLIssuer,
            &site.ServerType, &site.PoweredBy, &site.ContentType, &site.CacheControl)
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

func (db *DB) runMigrations() error {
    log.Println("🔄 Выполнение миграций базы данных...")
    
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return fmt.Errorf("ошибка создания таблицы миграций: %w", err)
    }

    if !db.isMigrationApplied(1) {
        err = db.applyMigration1()
        if err != nil {
            return fmt.Errorf("ошибка применения миграции 1: %w", err)
        }
        db.markMigrationApplied(1)
    }

    if !db.isMigrationApplied(2) {
        err = db.applyMigration2()
        if err != nil {
            return fmt.Errorf("ошибка применения миграции 2: %w", err)
        }
        db.markMigrationApplied(2)
    }

    if !db.isMigrationApplied(3) {
        err = db.applyMigration3()
        if err != nil {
            return fmt.Errorf("ошибка применения миграции 3: %w", err)
        }
        db.markMigrationApplied(3)
    }

    log.Println("✅ Миграции выполнены успешно")
    return nil
}

func (db *DB) isMigrationApplied(version int) bool {
    var count int
    err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count)
    return err == nil && count > 0
}

func (db *DB) markMigrationApplied(version int) {
    db.Exec("INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING", version)
}

func (db *DB) applyMigration1() error {
    query := `
    -- Create basic sites table
    CREATE TABLE IF NOT EXISTS sites (
        id SERIAL PRIMARY KEY,
        url VARCHAR(255) NOT NULL UNIQUE,
        status VARCHAR(20) NOT NULL DEFAULT 'unknown',
        last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    );

    -- Insert sample data
    INSERT INTO sites (url) VALUES 
        ('https://google.com'),
        ('https://github.com') 
    ON CONFLICT (url) DO NOTHING;
    `
    
    _, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("ошибка выполнения миграции 1: %w", err)
    }
    
    log.Println("Миграция 1 выполнена: базовая таблица sites создана")
    return nil
}

func (db *DB) applyMigration2() error {
    query := `
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
    `

    _, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("ошибка выполнения миграции 2: %w", err)
    }
    
    log.Println("Миграция 2 выполнена: добавлены поля для расширенного мониторинга")
    return nil
}

func (db *DB) applyMigration3() error {
    query := `
    -- Add all the detailed monitoring columns that might be missing
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
    `

    _, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("ошибка выполнения миграции 3: %w", err)
    }
    
    log.Println("Миграция 3 выполнена: добавлены поля для детального мониторинга")
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