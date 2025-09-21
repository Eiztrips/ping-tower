package metrics

import (
	"fmt"
	"log"
	"site-monitor/internal/database"
	"site-monitor/internal/monitor"
	"sync"
	"time"
)

type Service struct {
	clickhouse    *database.ClickHouseDB
	postgres      *database.DB
	batchSize     int
	flushInterval time.Duration
	buffer        []database.SiteMetric
	bufferMutex   sync.Mutex
	stopChan      chan struct{}
	wg            sync.WaitGroup
	siteStates    map[int]SiteState
	statesMutex   sync.RWMutex
	
	lastMetrics   map[int]database.SiteMetric  
	metricsMutex  sync.RWMutex
	maxDailyRows  int64                        
	dailyRowCount int64                        
	lastResetDate time.Time                    
}

type SiteState struct {
	LastStatus       string
	DownSince        *time.Time
	LastResponseTime int64
	LastSSLExpiry    *time.Time
	LastMetricSent   time.Time 
}

type Config struct {
	ClickHouse      database.ClickHouseConfig
	BatchSize       int
	FlushInterval   time.Duration
	MaxDailyRows    int64 
	MinMetricGap    time.Duration  
}

func NewService(config Config, postgresDB *database.DB) (*Service, error) {
	clickhouseDB, err := database.NewClickHouseDB(config.ClickHouse)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	service := &Service{
		clickhouse:    clickhouseDB,
		postgres:      postgresDB,
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		buffer:        make([]database.SiteMetric, 0, config.BatchSize),
		stopChan:      make(chan struct{}),
		siteStates:    make(map[int]SiteState),
		lastMetrics:   make(map[int]database.SiteMetric),
		maxDailyRows:  config.MaxDailyRows,
		lastResetDate: time.Now().Truncate(24 * time.Hour),
	}

	if config.BatchSize <= 0 {
		service.batchSize = 50
	}
	if config.FlushInterval <= 0 {
		service.flushInterval = 30 * time.Second
	}
	if config.MaxDailyRows <= 0 {
		service.maxDailyRows = 50000
	}

	service.startBatchProcessor()
	log.Printf("✅ Metrics service started with batch size: %d, flush interval: %v, max daily rows: %d",
		service.batchSize, service.flushInterval, service.maxDailyRows)

	return service, nil
}

func (s *Service) startBatchProcessor() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.flushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.flushBuffer()
			case <-s.stopChan:
				s.flushBuffer()
				return
			}
		}
	}()
}

func (s *Service) RecordCheckResult(siteID int, siteURL string, result monitor.CheckResult, checkType string) error {
	if !s.checkDailyLimit() {
		log.Printf("⚠️ Daily row limit (%d) reached, skipping metric recording for site %d", s.maxDailyRows, siteID)
		return nil
	}

	if !s.shouldRecordMetric(siteID, result) {
		log.Printf("🔄 Skipping duplicate/unchanged metric for site %d (%s)", siteID, siteURL)
		return nil
	}

	s.handleSiteStateChange(siteID, siteURL, result)

	metric := s.convertCheckResultToMetric(siteID, siteURL, result, checkType)
	
	s.metricsMutex.Lock()
	s.lastMetrics[siteID] = metric
	s.metricsMutex.Unlock()

	s.bufferMutex.Lock()
	s.buffer = append(s.buffer, metric)
	s.dailyRowCount++
	shouldFlush := len(s.buffer) >= s.batchSize
	s.bufferMutex.Unlock()

	if shouldFlush {
		s.flushBuffer()
	}

	if result.Status == "down" {
		if err := s.recordDowntimeEvent(uint32(siteID), siteURL, result.Error, uint16(result.StatusCode)); err != nil {
			log.Printf("⚠️ Failed to record downtime event: %v", err)
		}
	} else if result.Status == "up" {
		s.statesMutex.RLock()
		state := s.siteStates[siteID]
		wasDown := state.LastStatus == "down"
		s.statesMutex.RUnlock()
		
		if wasDown {
			if err := s.resolveDowntimeEvent(uint32(siteID)); err != nil {
				log.Printf("⚠️ Failed to resolve downtime event: %v", err)
			}
		}
	}

	if result.SSLExpiry != nil && result.SSLAlgorithm != "" {
		s.statesMutex.RLock()
		state := s.siteStates[siteID]
		sslChanged := state.LastSSLExpiry == nil || !state.LastSSLExpiry.Equal(*result.SSLExpiry)
		s.statesMutex.RUnlock()
		
		if sslChanged {
			if err := s.updateSSLCertificate(uint32(siteID), siteURL, result); err != nil {
				log.Printf("⚠️ Failed to update SSL certificate info: %v", err)
			}
		}
	}

	return nil
}

func (s *Service) checkDailyLimit() bool {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	
	if today.After(s.lastResetDate) {
		s.dailyRowCount = 0
		s.lastResetDate = today
		log.Printf("📊 Daily metrics counter reset for %s", today.Format("2006-01-02"))
	}
	
	return s.dailyRowCount < s.maxDailyRows
}

func (s *Service) shouldRecordMetric(siteID int, result monitor.CheckResult) bool {
	s.metricsMutex.RLock()
	lastMetric, exists := s.lastMetrics[siteID]
	s.metricsMutex.RUnlock()
	
	if !exists {
		return true 
	}
	
	now := time.Now()
	
	s.statesMutex.RLock()
	state := s.siteStates[siteID]
	timeSinceLastMetric := now.Sub(state.LastMetricSent)
	s.statesMutex.RUnlock()
	
	if timeSinceLastMetric > 30*time.Minute {
		return true
	}
	
	if result.Status != lastMetric.Status {
		return true
	}
	
	if lastMetric.ResponseTimeMs > 0 {
		responseTimeDiff := float64(abs(int64(result.ResponseTime) - int64(lastMetric.ResponseTimeMs))) / float64(lastMetric.ResponseTimeMs)
		if responseTimeDiff > 0.2 { 
			return true
		}
	}
	
	if uint16(result.StatusCode) != lastMetric.StatusCode {
		return true
	}
	
	return false
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (s *Service) handleSiteStateChange(siteID int, siteURL string, result monitor.CheckResult) {
	s.statesMutex.Lock()
	defer s.statesMutex.Unlock()

	currentState, exists := s.siteStates[siteID]
	if !exists {
		currentState = SiteState{LastStatus: "unknown"}
	}

	now := time.Now()
	newState := currentState
	newState.LastResponseTime = result.ResponseTime
	newState.LastMetricSent = now 

	if currentState.LastStatus != result.Status {
		if result.Status == "down" {
			newState.LastStatus = "down"
			newState.DownSince = &now
			log.Printf("🔴 Site %s went DOWN", siteURL)
		} else if result.Status == "up" && currentState.LastStatus == "down" {
			newState.LastStatus = "up"
			newState.DownSince = nil
			log.Printf("🟢 Site %s is back UP", siteURL)
		} else if result.Status == "up" {
			newState.LastStatus = "up"
			newState.DownSince = nil
		}
	}

	if result.Status == "up" && result.ResponseTime > 10000 {
		if currentState.LastResponseTime == 0 || float64(abs(result.ResponseTime-currentState.LastResponseTime))/float64(currentState.LastResponseTime) > 0.5 {
			log.Printf("⚠️ Very slow response time for %s: %dms", siteURL, result.ResponseTime)
		}
	}

	if result.SSLExpiry != nil {
		newState.LastSSLExpiry = result.SSLExpiry
	}

	s.siteStates[siteID] = newState
}

func (s *Service) convertCheckResultToMetric(siteID int, siteURL string, result monitor.CheckResult, checkType string) database.SiteMetric {
	now := time.Now()
	metric := database.SiteMetric{
		Timestamp:     now,
		TimestampDate: now.Truncate(time.Hour),
		SiteID:        uint32(siteID),
		SiteURL:       siteURL,
		Status:        result.Status,
		StatusCode:    uint16(result.StatusCode),
		ResponseTimeMs: uint64(result.ResponseTime),
		CheckType:     checkType,
		ConfigVersion: 1,
	}

	s.metricsMutex.RLock()
	lastMetric, hasLast := s.lastMetrics[siteID]
	s.metricsMutex.RUnlock()
	
	if !hasLast || s.hasSignificantChange(lastMetric, result) {
		metric.ContentLength = uint64(result.ContentLength)
		metric.DNSTimeMs = uint64(result.DNSTime)
		metric.ConnectTimeMs = uint64(result.ConnectTime)
		metric.TLSTimeMs = uint64(result.TLSTime)
		metric.TTFBMs = uint64(result.TTFB)
		metric.ContentHash = result.ContentHash
		metric.ContentType = result.ContentType
		metric.RedirectCount = uint8(result.RedirectCount)
		metric.FinalURL = result.FinalURL
		metric.ServerType = result.ServerType
		metric.PoweredBy = result.PoweredBy
		metric.CacheControl = result.CacheControl
		metric.SSLKeyLength = uint16(result.SSLKeyLength)
		metric.SSLAlgorithm = result.SSLAlgorithm
		metric.SSLIssuer = result.SSLIssuer
		
		if result.SSLValid {
			metric.SSLValid = 1
		} else {
			metric.SSLValid = 0
		}
		
		if result.SSLExpiry != nil {
			metric.SSLExpiry = result.SSLExpiry
		}
	}

	if !hasLast || lastMetric.Status != result.Status {
		metric.ErrorMessage = result.Error
	}

	return metric
}

func (s *Service) hasSignificantChange(lastMetric database.SiteMetric, result monitor.CheckResult) bool {
	if lastMetric.Status != result.Status {
		return true
	}
	
	if lastMetric.ResponseTimeMs > 0 {
		diff := float64(abs(int64(result.ResponseTime) - int64(lastMetric.ResponseTimeMs))) / float64(lastMetric.ResponseTimeMs)
		if diff > 0.3 {
			return true
		}
	}
	
	if lastMetric.ContentLength > 0 && result.ContentLength > 0 {
		diff := float64(abs(int64(result.ContentLength) - int64(lastMetric.ContentLength))) / float64(lastMetric.ContentLength)
		if diff > 0.1 {
			return true
		}
	}
	
	return false
}

func (s *Service) flushBuffer() {
	s.bufferMutex.Lock()
	if len(s.buffer) == 0 {
		s.bufferMutex.Unlock()
		return
	}

	toFlush := make([]database.SiteMetric, len(s.buffer))
	copy(toFlush, s.buffer)
	s.buffer = s.buffer[:0]
	s.bufferMutex.Unlock()

	if err := s.clickhouse.InsertMetricsBatch(toFlush); err != nil {
		log.Printf("❌ Failed to flush metrics batch (%d items): %v", len(toFlush), err)

		s.bufferMutex.Lock()
		s.buffer = append(toFlush, s.buffer...)
		s.bufferMutex.Unlock()
	} else {
		log.Printf("✅ Successfully flushed %d metrics to ClickHouse", len(toFlush))
	}
}

func (s *Service) recordDowntimeEvent(siteID uint32, siteURL, errorMessage string, statusCode uint16) error {
	return s.clickhouse.RecordDowntimeEvent(siteID, siteURL, errorMessage, statusCode)
}

func (s *Service) resolveDowntimeEvent(siteID uint32) error {
	return s.clickhouse.ResolveDowntimeEvent(siteID)
}

func (s *Service) updateSSLCertificate(siteID uint32, siteURL string, result monitor.CheckResult) error {
	if result.SSLExpiry == nil {
		return nil
	}

	return s.clickhouse.UpdateSSLCertificate(
		siteID, siteURL, result.SSLIssuer, result.SSLAlgorithm,
		uint16(result.SSLKeyLength), *result.SSLExpiry,
	)
}

func (s *Service) GetHourlyMetrics(siteID int, hours int) ([]database.HourlyMetrics, error) {
	return s.clickhouse.GetHourlyMetrics(uint32(siteID), hours)
}

func (s *Service) GetExpiringSSLCertificates(days int) ([]map[string]interface{}, error) {
	return s.clickhouse.GetExpiringSSLCertificates(days)
}

func (s *Service) GetSitePerformanceSummary(siteID int, hours int) (*PerformanceSummary, error) {
	metrics, err := s.GetHourlyMetrics(siteID, hours)
	if err != nil {
		return nil, err
	}

	if len(metrics) == 0 {
		return &PerformanceSummary{}, nil
	}

	summary := &PerformanceSummary{
		SiteID:   uint32(siteID),
		SiteURL:  metrics[0].SiteURL,
		Period:   fmt.Sprintf("Last %d hours", hours),
		DataFrom: metrics[len(metrics)-1].Hour,
		DataTo:   metrics[0].Hour,
	}

	var totalChecks, successfulChecks uint64
	var responseTimes, dnsTime, connectTime, tlsTime, ttfb []float64

	for _, m := range metrics {
		totalChecks += m.TotalChecks
		successfulChecks += m.SuccessfulChecks

		if m.TotalChecks > 0 {
			responseTimes = append(responseTimes, m.AvgResponseTime)
			dnsTime = append(dnsTime, m.AvgDNSTime)
			connectTime = append(connectTime, m.AvgConnectTime)
			tlsTime = append(tlsTime, m.AvgTLSTime)
			ttfb = append(ttfb, m.AvgTTFB)
		}
	}

	if totalChecks > 0 {
		summary.TotalChecks = totalChecks
		summary.SuccessfulChecks = successfulChecks
		summary.UptimePercent = float64(successfulChecks) / float64(totalChecks) * 100

		if len(responseTimes) > 0 {
			summary.AvgResponseTime = average(responseTimes)
			summary.AvgDNSTime = average(dnsTime)
			summary.AvgConnectTime = average(connectTime)
			summary.AvgTLSTime = average(tlsTime)
			summary.AvgTTFB = average(ttfb)
		}
	}

	return summary, nil
}

type PerformanceSummary struct {
	SiteID            uint32    `json:"site_id"`
	SiteURL           string    `json:"site_url"`
	Period            string    `json:"period"`
	DataFrom          time.Time `json:"data_from"`
	DataTo            time.Time `json:"data_to"`
	TotalChecks       uint64    `json:"total_checks"`
	SuccessfulChecks  uint64    `json:"successful_checks"`
	UptimePercent     float64   `json:"uptime_percent"`
	AvgResponseTime   float64   `json:"avg_response_time_ms"`
	AvgDNSTime        float64   `json:"avg_dns_time_ms"`
	AvgConnectTime    float64   `json:"avg_connect_time_ms"`
	AvgTLSTime        float64   `json:"avg_tls_time_ms"`
	AvgTTFB           float64   `json:"avg_ttfb_ms"`
}

func average(numbers []float64) float64 {
	if len(numbers) == 0 {
		return 0
	}

	var sum float64
	for _, n := range numbers {
		sum += n
	}
	return sum / float64(len(numbers))
}

func (s *Service) GetSystemHealth() map[string]interface{} {
	health := map[string]interface{}{
		"clickhouse_connected": false,
		"postgres_connected":   false,
		"buffer_size":          0,
		"batch_size":           s.batchSize,
		"flush_interval":       s.flushInterval.String(),
		"daily_rows_used":      s.dailyRowCount,
		"daily_rows_limit":     s.maxDailyRows,
		"daily_usage_percent":  float64(s.dailyRowCount) / float64(s.maxDailyRows) * 100,
	}

	if s.clickhouse != nil {
		if err := s.clickhouse.Ping(); err == nil {
			health["clickhouse_connected"] = true
		}
	}

	if s.postgres != nil {
		if err := s.postgres.DB.Ping(); err == nil {
			health["postgres_connected"] = true
		}
	}

	s.bufferMutex.Lock()
	health["buffer_size"] = len(s.buffer)
	s.bufferMutex.Unlock()

	return health
}

func (s *Service) Close() error {
	close(s.stopChan)
	s.wg.Wait()

	s.flushBuffer()

	if s.clickhouse != nil {
		return s.clickhouse.Close()
	}

	return nil
}