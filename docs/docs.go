package docs

var SwaggerInfo = struct {
	Version     string
	Host        string
	BasePath    string
	Schemes     []string
	Title       string
	Description string
}{
	Version:     "1.0.0",
	Host:        "localhost:8080",
	BasePath:    "/api",
	Schemes:     []string{"http", "https"},
	Title:       "Site Monitor API",
	Description: `## 🚀 Полнофункциональный API для мониторинга сайтов

Site Monitor предоставляет мощный REST API для автоматического мониторинга веб-сайтов с детальной аналитикой, планировщиком заданий и SSL мониторингом.

### ✨ Основные возможности:
- 🌐 **Мониторинг сайтов** - проверка доступности, времени отклика и статус кодов
- 📊 **Детальная аналитика** - DNS время, TCP соединение, TLS handshake, TTFB
- 🔒 **SSL мониторинг** - проверка валидности сертификатов и уведомления об истечении
- ⏰ **Cron планировщик** - гибкие расписания проверок для каждого сайта
- 📈 **ClickHouse метрики** - высокопроизводительная аналитическая база данных
- 🔄 **Real-time обновления** - Server-Sent Events для живых обновлений
- 🎨 **Современный веб-интерфейс** - адаптивный дизайн с интерактивными графиками`,
}

const SwaggerTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "termsOfService": "https://sitemonitor.com/terms",
        "contact": {
            "name": "Site Monitor Support",
            "url": "https://github.com/your-repo/site-monitor",
            "email": "support@sitemonitor.com"
        },
        "license": {
            "name": "MIT",
            "url": "https://opensource.org/licenses/MIT"
        },
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}"
}`

func init() {
}
