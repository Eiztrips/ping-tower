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
              created_at
              FROM sites WHERE url = $1`
    
    err := db.QueryRow(query, url).Scan(
        &site.ID, &site.URL, &site.Status, &site.StatusCode, &site.ResponseTime,
        &site.ContentLength, &site.SSLValid, &sslExpiry, &site.LastError,
        &site.TotalChecks, &site.SuccessfulChecks, &site.UptimePercent,
        &site.LastChecked, &site.CreatedAt)
    
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
                created_at 
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
            &site.LastChecked, &site.CreatedAt)
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