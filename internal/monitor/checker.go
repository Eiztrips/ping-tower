package monitor

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"site-monitor/internal/models"
	_ "github.com/lib/pq"
)

type Checker struct {
	db       *sql.DB
	interval time.Duration
	client   *http.Client
}

type CheckResult struct {
	Status        string
	StatusCode    int
	ResponseTime  int64
	ContentLength int64
	SSLValid      bool
	SSLExpiry     *time.Time
	Error         string
}

func NewChecker(db *sql.DB, interval time.Duration) *Checker {
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	return &Checker{
		db:       db,
		interval: interval,
		client:   client,
	}
}

func (c *Checker) Start() {
	log.Println("🔍 Запуск мониторинга сайтов...")
	
	// Выполняем первую проверку сразу
	log.Println("▶️ Выполняем первичную проверку всех сайтов...")
	c.checkAllSites()
	
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	
	log.Printf("⏰ Настроен интервал проверки: %v", c.interval)

	for {
		select {
		case <-ticker.C:
			log.Println("🔄 Запуск периодической проверки...")
			c.checkAllSites()
		}
	}
}

func (c *Checker) checkAllSites() {
	log.Println("📋 Получение списка сайтов для проверки...")
	
	rows, err := c.db.Query("SELECT id, url FROM sites")
	if err != nil {
		log.Printf("❌ Ошибка получения списка сайтов: %v", err)
		return
	}
	defer rows.Close()

	sitesCount := 0
	for rows.Next() {
		var site models.Site
		if err := rows.Scan(&site.ID, &site.URL); err != nil {
			log.Printf("❌ Ошибка чтения данных сайта: %v", err)
			continue
		}
		
		sitesCount++
		log.Printf("🔍 Проверяем сайт: %s (ID: %d)", site.URL, site.ID)
		
		result := c.checkSite(site.URL)
		c.updateSiteStatus(&site, result)
		c.saveCheckHistory(site.ID, result)
		
		// Небольшая пауза между проверками
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("✅ Проверено сайтов: %d", sitesCount)
}

func (c *Checker) checkSite(siteURL string) CheckResult {
	log.Printf("🌐 Начинаем проверку: %s", siteURL)
	start := time.Now()
	result := CheckResult{
		Status:     "down",
		StatusCode: 0,
		SSLValid:   false,
	}

	// Проверяем SSL сертификат для HTTPS сайтов
	if strings.HasPrefix(siteURL, "https://") {
		log.Printf("🔒 Проверяем SSL для: %s", siteURL)
		result.SSLValid, result.SSLExpiry = c.checkSSL(siteURL)
		if result.SSLValid {
			log.Printf("✅ SSL сертификат валиден для: %s", siteURL)
		} else {
			log.Printf("⚠️ Проблемы с SSL сертификатом для: %s", siteURL)
		}
	}

	// Выполняем HTTP запрос
	log.Printf("📡 Выполняем HTTP запрос к: %s", siteURL)
	resp, err := c.client.Get(siteURL)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(start).Milliseconds()
		log.Printf("❌ Ошибка при проверке %s: %v (время: %dмс)", siteURL, err, result.ResponseTime)
		return result
	}
	defer resp.Body.Close()

	result.ResponseTime = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	// Читаем содержимое для определения размера
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		result.ContentLength = int64(len(body))
		log.Printf("📄 Размер контента %s: %d байт", siteURL, result.ContentLength)
	}

	// Определяем статус на основе кода ответа
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
		log.Printf("✅ Сайт %s доступен (код: %d, время: %dмс, размер: %d байт)", 
			siteURL, resp.StatusCode, result.ResponseTime, result.ContentLength)
	} else {
		result.Status = "down"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		log.Printf("❌ Сайт %s недоступен (код: %d, время: %dмс)", 
			siteURL, resp.StatusCode, result.ResponseTime)
	}

	return result
}

func (c *Checker) checkSSL(siteURL string) (bool, *time.Time) {
	u, err := url.Parse(siteURL)
	if err != nil {
		log.Printf("❌ Ошибка парсинга URL для SSL проверки %s: %v", siteURL, err)
		return false, nil
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	log.Printf("🔐 Подключаемся к %s для проверки SSL", host)
	
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second}, 
		"tcp", 
		host, 
		&tls.Config{ServerName: u.Hostname()})
	if err != nil {
		log.Printf("❌ Ошибка SSL соединения с %s: %v", host, err)
		return false, nil
	}
	defer conn.Close()

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		log.Printf("❌ SSL сертификаты не найдены для %s", siteURL)
		return false, nil
	}

	cert := certs[0]
	now := time.Now()
	
	log.Printf("🔍 SSL сертификат для %s: выдан до %v", siteURL, cert.NotAfter)
	
	// Проверяем валидность сертификата
	if now.After(cert.NotAfter) || now.Before(cert.NotBefore) {
		log.Printf("⚠️ SSL сертификат для %s истек или еще не действует", siteURL)
		return false, &cert.NotAfter
	}

	log.Printf("✅ SSL сертификат для %s валиден до %v", siteURL, cert.NotAfter)
	return true, &cert.NotAfter
}

func (c *Checker) updateSiteStatus(site *models.Site, result CheckResult) {
	log.Printf("💾 Обновляем статус сайта %s: %s", site.URL, result.Status)
	
	// Обновляем основную информацию о сайте - исправляем SQL запрос
	query := `UPDATE sites SET 
                status = $1::varchar, 
                status_code = $2, 
                response_time = $3, 
                content_length = $4, 
                ssl_valid = $5, 
                ssl_expiry = $6, 
                last_error = $7, 
                last_checked = CURRENT_TIMESTAMP,
                total_checks = COALESCE(total_checks, 0) + 1,
                successful_checks = COALESCE(successful_checks, 0) + CASE WHEN $1::varchar = 'up' THEN 1 ELSE 0 END
              WHERE id = $8`

	_, err := c.db.Exec(query,
		result.Status,
		result.StatusCode,
		result.ResponseTime,
		result.ContentLength,
		result.SSLValid,
		result.SSLExpiry,
		result.Error,
		site.ID)

	if err != nil {
		log.Printf("❌ Ошибка обновления статуса сайта %s: %v", site.URL, err)
	} else {
		log.Printf("✅ Статус сайта %s успешно обновлен", site.URL)
	}
}

func (c *Checker) saveCheckHistory(siteID int, result CheckResult) {
	query := `INSERT INTO site_history (site_id, status, status_code, response_time, error, checked_at) 
              VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)`

	_, err := c.db.Exec(query, siteID, result.Status, result.StatusCode, result.ResponseTime, result.Error)
	if err != nil {
		log.Printf("❌ Ошибка сохранения истории проверки для сайта ID %d: %v", siteID, err)
	} else {
		log.Printf("✅ История проверки сохранена для сайта ID %d", siteID)
	}
}

func StartMonitoring(db *sql.DB, interval time.Duration) {
	checker := NewChecker(db, interval)
	checker.Start()
}