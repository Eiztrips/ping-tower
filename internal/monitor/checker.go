package monitor

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
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
		
	// New detailed metrics
	DNSTime       int64     `json:"dns_time"`
	ConnectTime   int64     `json:"connect_time"`
	TLSTime       int64     `json:"tls_time"`
	TTFB          int64     `json:"ttfb"`
	ContentHash   string    `json:"content_hash"`
	RedirectCount int       `json:"redirect_count"`
	FinalURL      string    `json:"final_url"`
	Headers       map[string]string `json:"headers"`
	Keywords      []string  `json:"keywords_found"`
	
	// SSL Details
	SSLKeyLength  int       `json:"ssl_key_length"`
	SSLAlgorithm  string    `json:"ssl_algorithm"`
	SSLIssuer     string    `json:"ssl_issuer"`
	
	// Server Info
	ServerType    string    `json:"server_type"`
	PoweredBy     string    `json:"powered_by"`
	ContentType   string    `json:"content_type"`
	CacheControl  string    `json:"cache_control"`
	Cookies       []string  `json:"cookies"`
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
		
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("✅ Проверено сайтов: %d", sitesCount)
}

func (c *Checker) checkSite(siteURL string) CheckResult {
	log.Printf("🌐 Начинаем детальную проверку: %s", siteURL)
	start := time.Now()
	result := CheckResult{
		Status:     "down",
		StatusCode: 0,
		SSLValid:   false,
		Headers:    make(map[string]string),
		Keywords:   []string{},
		Cookies:    []string{},
	}

	// Парсим URL для детального анализа
	parsedURL, err := url.Parse(siteURL)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid URL: %v", err)
		return result
	}

	// Измеряем время DNS lookup
	dnsStart := time.Now()
	ips, err := net.LookupIP(parsedURL.Hostname())
	if err != nil {
		result.Error = fmt.Sprintf("DNS lookup failed: %v", err)
		return result
	}
	result.DNSTime = time.Since(dnsStart).Milliseconds()
	log.Printf("🔍 DNS lookup для %s: %dмс, IP: %v", siteURL, result.DNSTime, ips[0])

	// SSL проверка с детальной информацией
	if strings.HasPrefix(siteURL, "https://") {
		log.Printf("🔒 Детальная SSL проверка для: %s", siteURL)
		sslValid, sslExpiry, sslDetails := c.checkSSLDetailed(siteURL)
		result.SSLValid = sslValid
		result.SSLExpiry = sslExpiry
		result.SSLKeyLength = sslDetails.KeyLength
		result.SSLAlgorithm = sslDetails.Algorithm
		result.SSLIssuer = sslDetails.Issuer
	}

	// Создаем клиент с детальным трекингом времени
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	// Подсчет редиректов
	redirectCount := 0
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Выполняем запрос с замером времени
	connectStart := time.Now()
	resp, err := client.Get(siteURL)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(start).Milliseconds()
		log.Printf("❌ Ошибка при проверке %s: %v", siteURL, err)
		return result
	}
	defer resp.Body.Close()

	result.ConnectTime = time.Since(connectStart).Milliseconds()
	result.RedirectCount = redirectCount
	result.FinalURL = resp.Request.URL.String()
	result.ResponseTime = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	// Time to First Byte (время до получения первого байта)
	ttfbStart := time.Now()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err == nil {
		result.TTFB = time.Since(ttfbStart).Milliseconds()
		result.ContentLength = int64(len(bodyBytes))
		
		// Вычисляем хэш контента для отслеживания изменений
		hash := sha256.Sum256(bodyBytes)
		result.ContentHash = fmt.Sprintf("%x", hash[:8]) // Первые 8 байт хэша
		
		// Ищем ключевые слова в контенте
		result.Keywords = c.findKeywords(string(bodyBytes))
		
		log.Printf("📄 Контент %s: размер %d байт, хэш %s, ключевых слов найдено: %d", 
			siteURL, result.ContentLength, result.ContentHash, len(result.Keywords))
	}

	// Собираем заголовки ответа
	result.Headers = make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			result.Headers[key] = values[0]
		}
	}

	// Извлекаем важную информацию из заголовков
	result.ServerType = resp.Header.Get("Server")
	result.PoweredBy = resp.Header.Get("X-Powered-By")
	result.ContentType = resp.Header.Get("Content-Type")
	result.CacheControl = resp.Header.Get("Cache-Control")

	// Собираем информацию о куках
	for _, cookie := range resp.Cookies() {
		result.Cookies = append(result.Cookies, cookie.Name+"="+cookie.Value)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
		log.Printf("✅ Детальная проверка %s завершена успешно (код: %d, время: %dмс, редиректов: %d)", 
			siteURL, resp.StatusCode, result.ResponseTime, result.RedirectCount)
	} else {
		result.Status = "down"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		log.Printf("❌ Сайт %s недоступен (код: %d)", siteURL, resp.StatusCode)
	}

	return result
}

type SSLDetails struct {
	KeyLength int
	Algorithm string
	Issuer    string
}

func (c *Checker) checkSSLDetailed(siteURL string) (bool, *time.Time, SSLDetails) {
	details := SSLDetails{}
	
	u, err := url.Parse(siteURL)
	if err != nil {
		return false, nil, details
	}

	host := u.Host
	if !strings.Contains(host, ":") {
		host += ":443"
	}

	tlsStart := time.Now()
	conn, err := tls.DialWithDialer(
		&net.Dialer{Timeout: 10 * time.Second}, 
		"tcp", 
		host, 
		&tls.Config{ServerName: u.Hostname()})
	if err != nil {
		log.Printf("❌ Ошибка SSL соединения с %s: %v", host, err)
		return false, nil, details
	}
	defer conn.Close()

	tlsTime := time.Since(tlsStart).Milliseconds()
	log.Printf("🔐 TLS handshake для %s: %dмс", siteURL, tlsTime)

	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return false, nil, details
	}

	cert := certs[0]
	now := time.Now()
	
	// Извлекаем детальную информацию о сертификате
	if cert.PublicKey != nil {
		switch pub := cert.PublicKey.(type) {
		case *rsa.PublicKey:
			details.KeyLength = pub.N.BitLen()
			details.Algorithm = "RSA"
		case *ecdsa.PublicKey:
			details.KeyLength = pub.Params().BitSize
			details.Algorithm = "ECDSA"
		}
	}
	
	details.Issuer = cert.Issuer.CommonName
	
	log.Printf("🔍 SSL детали для %s: алгоритм %s, длина ключа %d бит, издатель %s", 
		siteURL, details.Algorithm, details.KeyLength, details.Issuer)
	
	valid := !now.After(cert.NotAfter) && !now.Before(cert.NotBefore)
	return valid, &cert.NotAfter, details
}

func (c *Checker) findKeywords(content string) []string {
	keywords := []string{}
	
	// Список ключевых слов для поиска
	searchWords := []string{
		"error", "ошибка", "404", "500", "503",
		"welcome", "добро пожаловать", "home", "главная",
		"login", "войти", "register", "регистрация",
		"success", "успешно", "completed", "завершено",
	}
	
	contentLower := strings.ToLower(content)
	
	for _, word := range searchWords {
		if strings.Contains(contentLower, strings.ToLower(word)) {
			keywords = append(keywords, word)
		}
	}
	
	return keywords
}

func (c *Checker) updateSiteStatus(site *models.Site, result CheckResult) {
	log.Printf("💾 Обновляем детальный статус сайта %s: %s", site.URL, result.Status)
	
	query := `UPDATE sites SET 
                status = $1::varchar, 
                status_code = $2, 
                response_time = $3, 
                content_length = $4, 
                ssl_valid = $5, 
                ssl_expiry = $6, 
                last_error = $7,
                dns_time = $8,
                connect_time = $9,
                tls_time = $10,
                ttfb = $11,
                content_hash = $12,
                redirect_count = $13,
                final_url = $14,
                server_type = $15,
                powered_by = $16,
                content_type = $17,
                cache_control = $18,
                ssl_key_length = $19,
                ssl_algorithm = $20,
                ssl_issuer = $21,
                last_checked = CURRENT_TIMESTAMP,
                total_checks = COALESCE(total_checks, 0) + 1,
                successful_checks = COALESCE(successful_checks, 0) + CASE WHEN $1::varchar = 'up' THEN 1 ELSE 0 END
              WHERE id = $22`

	_, err := c.db.Exec(query,
		result.Status, result.StatusCode, result.ResponseTime, result.ContentLength,
		result.SSLValid, result.SSLExpiry, result.Error,
		result.DNSTime, result.ConnectTime, result.TLSTime, result.TTFB,
		result.ContentHash, result.RedirectCount, result.FinalURL,
		result.ServerType, result.PoweredBy, result.ContentType, result.CacheControl,
		result.SSLKeyLength, result.SSLAlgorithm, result.SSLIssuer,
		site.ID)

	if err != nil {
		log.Printf("❌ Ошибка обновления детального статуса сайта %s: %v", site.URL, err)
	} else {
		log.Printf("✅ Детальный статус сайта %s успешно обновлен", site.URL)
		NotifySiteChecked(site.URL, result)
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

// Функция для отправки уведомлений (будет импортирована из handlers)
var NotifySiteChecked func(string, CheckResult)