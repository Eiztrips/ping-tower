package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"site-monitor/internal/config"
	"site-monitor/internal/monitor"
)

type AlertManager struct {
	config *config.AlertsConfig
}

type AlertData struct {
	SiteURL      string                `json:"site_url"`
	SiteID       int                   `json:"site_id"`
	Status       string                `json:"status"`
	StatusCode   int                   `json:"status_code"`
	ResponseTime int64                 `json:"response_time"`
	Error        string                `json:"error,omitempty"`
	Timestamp    time.Time             `json:"timestamp"`
	AlertType    string                `json:"alert_type"`
	CheckResult  *monitor.CheckResult  `json:"check_result,omitempty"`
}

func NewAlertManager(alertsConfig *config.AlertsConfig) *AlertManager {
	return &AlertManager{
		config: alertsConfig,
	}
}

func (am *AlertManager) SendAlert(siteID int, siteURL string, result monitor.CheckResult, alertType string) error {
	if !am.config.Enabled {
		log.Println("🔕 Алерты отключены в конфигурации")
		return nil
	}

	alertData := AlertData{
		SiteURL:      siteURL,
		SiteID:       siteID,
		Status:       result.Status,
		StatusCode:   result.StatusCode,
		ResponseTime: result.ResponseTime,
		Error:        result.Error,
		Timestamp:    time.Now(),
		AlertType:    alertType,
		CheckResult:  &result,
	}

	var errors []string

	// Send email alert
	if am.config.Email.Enabled {
		if err := am.sendEmailAlert(alertData); err != nil {
			errors = append(errors, fmt.Sprintf("email: %v", err))
			log.Printf("❌ Ошибка отправки email алерта: %v", err)
		} else {
			log.Printf("✅ Email алерт отправлен для %s", siteURL)
		}
	}

	// Send webhook alert
	if am.config.Webhook.Enabled {
		if err := am.sendWebhookAlert(alertData); err != nil {
			errors = append(errors, fmt.Sprintf("webhook: %v", err))
			log.Printf("❌ Ошибка отправки webhook алерта: %v", err)
		} else {
			log.Printf("✅ Webhook алерт отправлен для %s", siteURL)
		}
	}

	// Send Telegram alert
	if am.config.Telegram.Enabled {
		if err := am.sendTelegramAlert(alertData); err != nil {
			errors = append(errors, fmt.Sprintf("telegram: %v", err))
			log.Printf("❌ Ошибка отправки Telegram алерта: %v", err)
		} else {
			log.Printf("✅ Telegram алерт отправлен для %s", siteURL)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to send some alerts: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (am *AlertManager) sendEmailAlert(alertData AlertData) error {
	if am.config.Email.SMTPServer == "" || len(am.config.Email.To) == 0 {
		return fmt.Errorf("email configuration incomplete")
	}

	subject := fmt.Sprintf("🔔 Site Alert: %s - %s", alertData.SiteURL, strings.ToUpper(alertData.Status))

	body := fmt.Sprintf(`
Site Monitor Alert

🌐 Site: %s
📊 Status: %s
⏱️ Response Time: %dms
🔢 Status Code: %d
⏰ Time: %s
🔍 Check Type: %s

`, alertData.SiteURL, strings.ToUpper(alertData.Status), alertData.ResponseTime,
		alertData.StatusCode, alertData.Timestamp.Format("2006-01-02 15:04:05"), alertData.AlertType)

	if alertData.Error != "" {
		body += fmt.Sprintf("❌ Error: %s\n", alertData.Error)
	}

	if alertData.CheckResult != nil {
		body += fmt.Sprintf(`
📈 Additional Details:
• Content Length: %d bytes
• DNS Time: %dms
• Connect Time: %dms
• TLS Time: %dms
• TTFB: %dms
• SSL Valid: %t
`, alertData.CheckResult.ContentLength, alertData.CheckResult.DNSTime,
			alertData.CheckResult.ConnectTime, alertData.CheckResult.TLSTime,
			alertData.CheckResult.TTFB, alertData.CheckResult.SSLValid)
	}

	message := []byte(fmt.Sprintf("Subject: %s\r\n"+
		"From: %s\r\n"+
		"To: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", subject, am.config.Email.From, strings.Join(am.config.Email.To, ","), body))

	auth := smtp.PlainAuth("", am.config.Email.Username, am.config.Email.Password, am.config.Email.SMTPServer)
	err := smtp.SendMail(am.config.Email.SMTPServer+":"+am.config.Email.Port, auth,
		am.config.Email.From, am.config.Email.To, message)

	return err
}

func (am *AlertManager) sendWebhookAlert(alertData AlertData) error {
	if am.config.Webhook.URL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	jsonData, err := json.Marshal(alertData)
	if err != nil {
		return fmt.Errorf("failed to marshal alert data: %v", err)
	}

	client := &http.Client{
		Timeout: time.Duration(am.config.Webhook.Timeout) * time.Second,
	}

	req, err := http.NewRequest("POST", am.config.Webhook.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Site-Monitor/1.0")

	// Add custom headers
	for key, value := range am.config.Webhook.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

func (am *AlertManager) sendTelegramAlert(alertData AlertData) error {
	if am.config.Telegram.BotToken == "" || am.config.Telegram.ChatID == "" {
		return fmt.Errorf("telegram configuration incomplete")
	}

	statusEmoji := "🔴"
	if alertData.Status == "up" {
		statusEmoji = "🟢"
	}

	message := fmt.Sprintf(`
%s *Site Monitor Alert*

🌐 *Site:* %s
📊 *Status:* %s
⏱️ *Response Time:* %dms
🔢 *Status Code:* %d
⏰ *Time:* %s
🔍 *Check Type:* %s
`, statusEmoji, alertData.SiteURL, strings.ToUpper(alertData.Status),
		alertData.ResponseTime, alertData.StatusCode,
		alertData.Timestamp.Format("2006-01-02 15:04:05"), alertData.AlertType)

	if alertData.Error != "" {
		message += fmt.Sprintf("\n❌ *Error:* %s", alertData.Error)
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", am.config.Telegram.BotToken)

	payload := map[string]interface{}{
		"chat_id":    am.config.Telegram.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(telegramURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("telegram API returned status code: %d", resp.StatusCode)
	}

	return nil
}

// Legacy support for existing code
type Notifier struct {
	alertManager *AlertManager
}

func NewNotifier(smtpServer, port, username, password, from string, to []string) *Notifier {
	emailConfig := &config.EmailAlertConfig{
		Enabled:    true,
		SMTPServer: smtpServer,
		Port:       port,
		Username:   username,
		Password:   password,
		From:       from,
		To:         to,
	}

	alertsConfig := &config.AlertsConfig{
		Enabled: true,
		Email:   *emailConfig,
	}

	return &Notifier{
		alertManager: NewAlertManager(alertsConfig),
	}
}

func (n *Notifier) SendNotification(siteURL string) error {
	// Convert legacy call to new alert system
	result := monitor.CheckResult{
		Status:     "down",
		StatusCode: 0,
		Error:      "Site is down",
	}

	return n.alertManager.SendAlert(0, siteURL, result, "legacy")
}