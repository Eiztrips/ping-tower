package monitor

import (
    "database/sql"
    "log"
    "net/http"
    "time"

    _ "github.com/lib/pq"
)

type Site struct {
    ID        int
    URL       string
    Status    string
    LastChecked time.Time
}

type Checker struct {
    db      *sql.DB
    interval time.Duration
}

func NewChecker(db *sql.DB, interval time.Duration) *Checker {
    return &Checker{
        db:      db,
        interval: interval,
    }
}

func (c *Checker) Start() {
    ticker := time.NewTicker(c.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            c.checkSites()
        }
    }
}

func (c *Checker) checkSites() {
    rows, err := c.db.Query("SELECT id, url FROM sites")
    if err != nil {
        log.Println("Ошибка получения списка сайтов:", err)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var site Site
        if err := rows.Scan(&site.ID, &site.URL); err != nil {
            log.Println("Ошибка чтения данных сайта:", err)
            continue
        }
        c.checkSite(&site)
    }
}

func (c *Checker) checkSite(site *Site) {
    resp, err := http.Get(site.URL)
    if err != nil || resp.StatusCode != http.StatusOK {
        site.Status = "down"
        log.Printf("Сайт %s недоступен", site.URL)
    } else {
        site.Status = "up"
        log.Printf("Сайт %s доступен", site.URL)
    }
    site.LastChecked = time.Now()

    _, err = c.db.Exec("UPDATE sites SET status = $1, last_checked = $2 WHERE id = $3", site.Status, site.LastChecked, site.ID)
    if err != nil {
        log.Println("Ошибка обновления статуса сайта:", err)
    }
}

func StartMonitoring(db *sql.DB, interval time.Duration) {
    checker := NewChecker(db, interval)
    checker.Start()
}