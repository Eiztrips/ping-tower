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
}

type Config struct {
	ClickHouse    database.ClickHouseConfig
	BatchSize     int
	FlushInterval time.Duration
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
	}

	if config.BatchSize <= 0 {
		service.batchSize = 100
	}
	if config.FlushInterval <= 0 {
		service.flushInterval = 10 * time.Second
	}

	service.startBatchProcessor()
	log.Printf("✅ Metrics service started with batch size: %d, flush interval: %v",
		service.batchSize, service.flushInterval)

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
	metric := s.convertCheckResultToMetric(siteID, siteURL, result, checkType)

	s.bufferMutex.Lock()
	s.buffer = append(s.buffer, metric)
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
		if err := s.resolveDowntimeEvent(uint32(siteID)); err != nil {
			log.Printf("⚠️ Failed to resolve downtime event: %v", err)
		}
	}

	if result.SSLExpiry != nil && result.SSLAlgorithm != "" {
		if err := s.updateSSLCertificate(uint32(siteID), siteURL, result); err != nil {
			log.Printf("⚠️ Failed to update SSL certificate info: %v", err)
		}
	}

	return nil
}

func (s *Service) convertCheckResultToMetric(siteID int, siteURL string, result monitor.CheckResult, checkType string) database.SiteMetric {
	now := time.Now()
	metric := database.SiteMetric{
		Timestamp:     now,
		TimestampDate: now,
		SiteID:        uint32(siteID),
		SiteURL:       siteURL,
		Status:        result.Status,
		StatusCode:    uint16(result.StatusCode),
		ResponseTimeMs: uint64(result.ResponseTime),
		ContentLength: uint64(result.ContentLength),
		DNSTimeMs:     uint64(result.DNSTime),
		ConnectTimeMs: uint64(result.ConnectTime),
		TLSTimeMs:     uint64(result.TLSTime),
		TTFBMs:        uint64(result.TTFB),
		ContentHash:   result.ContentHash,
		ContentType:   result.ContentType,
		RedirectCount: uint8(result.RedirectCount),
		FinalURL:      result.FinalURL,
		ServerType:    result.ServerType,
		PoweredBy:     result.PoweredBy,
		CacheControl:  result.CacheControl,
		ErrorMessage:  result.Error,
		CheckType:     checkType,
		ConfigVersion: 1,
	}

	if result.SSLValid {
		metric.SSLValid = 1
	} else {
		metric.SSLValid = 0
	}

	if result.SSLExpiry != nil {
		metric.SSLExpiry = result.SSLExpiry
	}

	metric.SSLKeyLength = uint16(result.SSLKeyLength)
	metric.SSLAlgorithm = result.SSLAlgorithm
	metric.SSLIssuer = result.SSLIssuer

	return metric
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