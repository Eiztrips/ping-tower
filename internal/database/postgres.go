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
    query := "SELECT id, url, status, last_checked FROM sites WHERE url = $1"
    err := db.QueryRow(query, url).Scan(&site.ID, &site.URL, &site.Status, &site.LastChecked)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("Сайт не найден")
        }
        return nil, fmt.Errorf("Ошибка запроса к сайту: %w", err)
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
    query := "SELECT id, url, status, last_checked FROM sites ORDER BY created_at DESC"
    rows, err := db.Query(query)
    if err != nil {
        return nil, fmt.Errorf("Ошибка получения списка сайтов: %w", err)
    }
    defer rows.Close()

    var sites []models.Site
    for rows.Next() {
        var site models.Site
        err := rows.Scan(&site.ID, &site.URL, &site.Status, &site.LastChecked)
        if err != nil {
            return nil, fmt.Errorf("Ошибка чтения данных сайта: %w", err)
        }
        sites = append(sites, site)
    }

    if err = rows.Err(); err != nil {
        return nil, fmt.Errorf("Ошибка при обработке результатов: %w", err)
    }

    return sites, nil
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

// USE DOCKER DB INIT
// func (db *DB) runMigrations() error {
//     createTableSQL := `
//     -- Create sites table for monitoring websites
//     CREATE TABLE IF NOT EXISTS sites (
//         id SERIAL PRIMARY KEY,
//         url VARCHAR(255) NOT NULL UNIQUE,
//         status VARCHAR(20) NOT NULL DEFAULT 'unknown',
//         last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
//         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
//         updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
//     );

//     -- Create index for faster URL lookups
//     CREATE INDEX IF NOT EXISTS idx_sites_url ON sites(url);

//     -- Insert some sample data
//     INSERT INTO sites (url) VALUES 
//         ('https://google.com'),
//         ('https://github.com') 
//     ON CONFLICT (url) DO NOTHING;`

//     _, err := db.Exec(createTableSQL)
//     if err != nil {
//         return fmt.Errorf("ошибка выполнения миграции: %w", err)
//     }

//     log.Println("Миграции выполнены успешно")
//     return nil
// }

func (db *DB) UpdateSiteStatus(id int, status string) error {
    query := "UPDATE sites SET status = $1, last_checked = CURRENT_TIMESTAMP WHERE id = $2"
    _, err := db.Exec(query, status, id)
    if err != nil {
        return fmt.Errorf("Ошибка обновления статуса сайта: %w", err)
    }
    return nil
}