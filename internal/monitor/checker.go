package monitor

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"site-monitor/internal/models"
	"site-monitor/internal/database"
	_ "github.com/lib/pq"
)

type Checker struct {
	db       *database.DB
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
		
	DNSTime       int64     `json:"dns_time"`
	ConnectTime   int64     `json:"connect_time"`
	TLSTime       int64     `json:"tls_time"`
	TTFB          int64     `json:"ttfb"`
	ContentHash   string    `json:"content_hash"`
	RedirectCount int       `json:"redirect_count"`
	FinalURL      string    `json:"final_url"`
	Headers       map[string]string `json:"headers"`
	Keywords      []string  `json:"keywords_found"`
	
	SSLKeyLength  int       `json:"ssl_key_length"`
	SSLAlgorithm  string    `json:"ssl_algorithm"`
	SSLIssuer     string    `json:"ssl_issuer"`
	
	ServerType    string    `json:"server_type"`
	PoweredBy     string    `json:"powered_by"`
	ContentType   string    `json:"content_type"`
	CacheControl  string    `json:"cache_control"`
	Cookies       []string  `json:"cookies"`
}

func NewChecker(db *database.DB, interval time.Duration) *Checker {
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
		db:     db,
		client: client,
	}
}

// CheckAllSitesOnDemand - проверка всех сайтов по требованию
func (c *Checker) CheckAllSitesOnDemand() {
	log.Println("🔍 Запуск проверки по требованию...")
	c.checkAllSites()
}

func (c *Checker) checkAllSites() {
	log.Println("📋 Получение списка сайтов для проверки...")
	
	rows, err := c.db.Query(`SELECT s.id, s.url, c.enabled, c.check_interval 
							 FROM sites s 
							 LEFT JOIN site_configs c ON s.id = c.site_id 
							 WHERE COALESCE(c.enabled, true) = true`)
	if err != nil {
		log.Printf("❌ Ошибка получения списка сайтов: %v", err)
		return
	}
	defer rows.Close()

	sitesCount := 0
	for rows.Next() {
		var site models.Site
		var enabled bool
		var checkInterval int
		
		if err := rows.Scan(&site.ID, &site.URL, &enabled, &checkInterval); err != nil {
			log.Printf("❌ Ошибка чтения данных сайта: %v", err)
			continue
		}
		
		if !enabled {
			continue
		}
		
		sitesCount++
		log.Printf("🔍 Проверяем сайт: %s (ID: %d)", site.URL, site.ID)
		
		config, err := c.db.GetSiteConfig(site.ID)
		if err != nil {
			log.Printf("❌ Ошибка получения конфигурации для сайта %d: %v", site.ID, err)
			config = &models.SiteConfig{
				SiteID: site.ID,
				CheckInterval: 30,
				Timeout: 30,
				ExpectedStatus: 200,
				FollowRedirects: true,
				MaxRedirects: 10,
				CheckSSL: true,
				UserAgent: "Site-Monitor/1.0",
				// Conservative defaults - only basic metrics
				CollectDNSTime: false,
				CollectConnectTime: false,
				CollectTLSTime: false,
				CollectTTFB: false,
				CollectContentHash: false,
				CollectRedirects: false,
				CollectSSLDetails: true,
				CollectServerInfo: false,
				CollectHeaders: false,
			}
		}
		
		result := c.checkSiteWithConfig(site.URL, config)
		c.updateSiteStatus(&site, result)
		c.saveCheckHistory(site.ID, result)
		
		time.Sleep(500 * time.Millisecond)
	}
	
	log.Printf("✅ Проверено активных сайтов: %d", sitesCount)
}

func (c *Checker) checkSiteWithConfig(siteURL string, config *models.SiteConfig) CheckResult {
	log.Printf("🌐 Проверка с конфигурацией: %s (таймаут: %ds, ожидаемый статус: %d)", 
		siteURL, config.Timeout, config.ExpectedStatus)
	
	start := time.Now()
	result := CheckResult{
		Status:     "down",
		StatusCode: 0,
		SSLValid:   false,
		Headers:    make(map[string]string),
		Keywords:   []string{},
		Cookies:    []string{},
	}

	// Use config for timeout
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Duration(config.Timeout/3) * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: time.Duration(config.Timeout/3) * time.Second,
		},
	}

	// Handle redirects based on config
	redirectCount := 0
	if config.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			redirectCount++
			if redirectCount > config.MaxRedirects {
				return fmt.Errorf("too many redirects")
			}
			return nil
		}
	} else {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create request with custom User-Agent
	req, err := http.NewRequest("GET", siteURL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid request: %v", err)
		return result
	}
	
	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	}

	// Add custom headers
	for key, value := range config.Headers {
		if strValue, ok := value.(string); ok {
			req.Header.Set(key, strValue)
		}
	}

	parsedURL, err := url.Parse(siteURL)
	if err != nil {
		result.Error = fmt.Sprintf("Invalid URL: %v", err)
		return result
	}

	// DNS lookup only if enabled
	if config.CollectDNSTime {
		dnsStart := time.Now()
		ips, err := net.LookupIP(parsedURL.Hostname())
		if err != nil {
			result.Error = fmt.Sprintf("DNS lookup failed: %v", err)
			return result
		}
		result.DNSTime = time.Since(dnsStart).Milliseconds()
		log.Printf("🔍 DNS lookup для %s: %dмс, IP: %v", siteURL, result.DNSTime, ips[0])
	}

	// SSL check only if enabled
	if strings.HasPrefix(siteURL, "https://") && config.CollectSSLDetails {
		log.Printf("🔒 Детальная SSL проверка для: %s", siteURL)
		sslValid, sslExpiry, sslDetails := c.checkSSLDetailed(siteURL)
		result.SSLValid = sslValid
		result.SSLExpiry = sslExpiry
		if config.CollectSSLDetails {
			result.SSLKeyLength = sslDetails.KeyLength
			result.SSLAlgorithm = sslDetails.Algorithm
			result.SSLIssuer = sslDetails.Issuer
		}
	}

	// Collect performance metrics based on config
	requestStart := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		result.Error = err.Error()
		result.ResponseTime = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ResponseTime = time.Since(start).Milliseconds()

	// Collect timing metrics based on configuration
	if config.CollectTTFB {
		result.TTFB = time.Since(requestStart).Milliseconds() - result.ResponseTime
	}

	// Server info collection based on config
	if config.CollectServerInfo || config.CollectHeaders {
		if server := resp.Header.Get("Server"); server != "" && config.CollectServerInfo {
			result.ServerType = server
		}
		if powered := resp.Header.Get("X-Powered-By"); powered != "" && config.CollectServerInfo {
			result.PoweredBy = powered
		}
		if contentType := resp.Header.Get("Content-Type"); contentType != "" && config.CollectHeaders {
			result.ContentType = contentType
		}
		if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" && config.CollectHeaders {
			result.CacheControl = cacheControl
		}
	}

	// Redirect tracking based on config
	if config.CollectRedirects {
		result.RedirectCount = redirectCount
		result.FinalURL = resp.Request.URL.String()
	}

	// Check expected status
	statusValid := false
	if config.ExpectedStatus == 0 {
		statusValid = resp.StatusCode >= 200 && resp.StatusCode < 400
	} else {
		statusValid = resp.StatusCode == config.ExpectedStatus
	}

	// Read response body for content checks
	bodyBytes, err := io.ReadAll(resp.Body)
	if err == nil {
		result.ContentLength = int64(len(bodyBytes))
		
		// Content hash only if enabled
		if config.CollectContentHash {
			hash := sha256.Sum256(bodyBytes)
			result.ContentHash = fmt.Sprintf("%x", hash[:8])
		}
		
		// Check keywords if configured
		if config.CheckKeywords != "" || config.AvoidKeywords != "" {
			content := string(bodyBytes)
			contentLower := strings.ToLower(content)
			
			// Check required keywords
			if config.CheckKeywords != "" {
				keywords := strings.Split(config.CheckKeywords, ",")
				keywordFound := false
				for _, keyword := range keywords {
					keyword = strings.TrimSpace(keyword)
					if keyword != "" && strings.Contains(contentLower, strings.ToLower(keyword)) {
						keywordFound = true
						result.Keywords = append(result.Keywords, keyword)
					}
				}
				if !keywordFound && len(keywords) > 0 {
					statusValid = false
					result.Error = "Required keywords not found"
				}
			}
			
			// Check avoid keywords
			if config.AvoidKeywords != "" {
				avoidWords := strings.Split(config.AvoidKeywords, ",")
				for _, word := range avoidWords {
					word = strings.TrimSpace(word)
					if word != "" && strings.Contains(contentLower, strings.ToLower(word)) {
						statusValid = false
						result.Error = "Forbidden keyword found: " + word
						break
					}
				}
			}
		}
	}

	if statusValid {
		result.Status = "up"
		log.Printf("✅ Сайт %s доступен (код: %d, конфиг OK)", siteURL, resp.StatusCode)
	} else {
		result.Status = "down"
		if result.Error == "" {
			result.Error = fmt.Sprintf("Unexpected status: %d (expected: %d)", resp.StatusCode, config.ExpectedStatus)
		}
		log.Printf("❌ Сайт %s недоступен: %s", siteURL, result.Error)
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

// CheckOnDemand - создать инстанс checker и выполнить проверку
func CheckOnDemand(db *database.DB) {
	checker := NewChecker(db, 0) // interval не используется
	checker.CheckAllSitesOnDemand()
}

// StartPeriodicMonitoring - запуск фонового мониторинга с индивидуальными интервалами
func StartPeriodicMonitoring(db *database.DB) {
	checker := NewChecker(db, 0)
	
	go func() {
		log.Println("🔄 Запуск фонового мониторинга...")
		
		for {
			checker.checkSitesWithIntervals()
			time.Sleep(1 * time.Second) // проверяем каждую секунду, нужно ли кого-то проверить
		}
	}()
}

// checkSitesWithIntervals - проверка сайтов согласно их индивидуальным интервалам
func (c *Checker) checkSitesWithIntervals() {
	rows, err := c.db.Query(`
		SELECT s.id, s.url, s.last_checked, 
			   COALESCE(c.check_interval, 30) as check_interval,
			   COALESCE(c.enabled, true) as enabled
		FROM sites s 
		LEFT JOIN site_configs c ON s.id = c.site_id 
		WHERE COALESCE(c.enabled, true) = true
	`)
	if err != nil {
		log.Printf("❌ Ошибка получения списка сайтов для проверки: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now()
	
	for rows.Next() {
		var siteID int
		var url string
		var lastChecked time.Time
		var checkInterval int
		var enabled bool
		
		if err := rows.Scan(&siteID, &url, &lastChecked, &checkInterval, &enabled); err != nil {
			log.Printf("❌ Ошибка сканирования данных сайта: %v", err)
			continue
		}
		
		// Проверяем, нужно ли проверить сайт
		nextCheck := lastChecked.Add(time.Duration(checkInterval) * time.Second)
		if now.After(nextCheck) {
			log.Printf("🔍 Проверка сайта %s (интервал: %d сек)", url, checkInterval)
			
			result := c.checkSite(url, siteID)
			
			// Уведомляем через SSE
			if NotifySiteChecked != nil {
				NotifySiteChecked(url, result)
			}
		}
	}
}

// checkSite - проверка одного сайта с получением его конфигурации
func (c *Checker) checkSite(siteURL string, siteID int) CheckResult {
	config, err := c.db.GetSiteConfig(siteID)
	if err != nil {
		log.Printf("❌ Ошибка получения конфигурации для сайта %d: %v", siteID, err)
		config = &models.SiteConfig{
			SiteID: siteID,
			CheckInterval: 30,
			Timeout: 30,
			ExpectedStatus: 200,
			FollowRedirects: true,
			MaxRedirects: 10,
			CheckSSL: true,
			UserAgent: "Site-Monitor/1.0",
			// Conservative defaults - only basic metrics
			CollectDNSTime: false,
			CollectConnectTime: false,
			CollectTLSTime: false,
			CollectTTFB: false,
			CollectContentHash: false,
			CollectRedirects: false,
			CollectSSLDetails: true,
			CollectServerInfo: false,
			CollectHeaders: false,
		}
	}
	
	result := c.checkSiteWithConfig(siteURL, config)
	
	// Обновляем статус сайта в базе данных
	site := &models.Site{ID: siteID, URL: siteURL}
	c.updateSiteStatus(site, result)
	c.saveCheckHistory(siteID, result)
	
	return result
}

var NotifySiteChecked func(string, CheckResult)