package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"site-monitor/internal/database"
	"site-monitor/internal/models"
	"time"
)

type DemoData struct {
	Stats DashboardStats
	Sites []models.Site
	Metrics map[string]interface{}
	Jobs []CronJob
}

type CronJob struct {
	SiteURL     string
	Schedule    string
	Status      string
	RunCount    int
	ErrorCount  int
	NextRun     string
	LastRun     string
	Description string
}

var demoDatabase *database.DB

func SetDemoDatabase(db *database.DB) {
	demoDatabase = db
}

const demoTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Site Monitor Demo - Полный функционал</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; }
        .container { max-width: 1400px; margin: 0 auto; padding: 20px; }
        .header { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 20px; padding: 30px; margin-bottom: 30px; text-align: center; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
        .header h1 { color: #2c3e50; font-size: 2.5em; margin-bottom: 10px; }
        .header p { color: #7f8c8d; font-size: 1.2em; }
        .demo-badge { display: inline-block; background: #e74c3c; color: white; padding: 5px 15px; border-radius: 20px; font-size: 0.9em; margin-left: 10px; animation: pulse 2s infinite; }
        @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.7; } }
        .stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin-bottom: 30px; }
        .stat-card { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 15px; padding: 25px; text-align: center; box-shadow: 0 8px 32px rgba(0,0,0,0.1); transition: transform 0.3s ease; }
        .stat-card:hover { transform: translateY(-5px); }
        .stat-icon { font-size: 2.5em; margin-bottom: 15px; }
        .stat-value { font-size: 2.2em; font-weight: bold; margin-bottom: 5px; }
        .stat-label { color: #7f8c8d; font-size: 0.9em; }
        .success { color: #27ae60; }
        .danger { color: #e74c3c; }
        .info { color: #3498db; }
        .warning { color: #f39c12; }
        .dashboard-content { display: grid; grid-template-columns: 2fr 1fr; gap: 30px; margin-bottom: 30px; }
        .panel { background: rgba(255,255,255,0.95); backdrop-filter: blur(10px); border-radius: 20px; padding: 30px; box-shadow: 0 8px 32px rgba(0,0,0,0.1); }
        .panel-title { font-size: 1.8em; margin-bottom: 20px; color: #2c3e50; display: flex; align-items: center; gap: 10px; }
        .site-card { background: #f8f9fa; border-radius: 15px; padding: 20px; margin-bottom: 15px; transition: all 0.3s ease; border-left: 5px solid #27ae60; }
        .site-card.down { border-left-color: #e74c3c; }
        .site-card.unknown { border-left-color: #f39c12; }
        .site-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
        .site-url { font-weight: bold; font-size: 1.1em; color: #2c3e50; word-break: break-all; }
        .site-status { display: flex; align-items: center; gap: 5px; padding: 5px 12px; border-radius: 20px; font-size: 0.9em; font-weight: bold; }
        .site-status.up { background: rgba(39, 174, 96, 0.1); color: #27ae60; }
        .site-status.down { background: rgba(231, 76, 60, 0.1); color: #e74c3c; }
        .site-status.unknown { background: rgba(243, 156, 18, 0.1); color: #f39c12; }
        .site-details { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 10px; margin: 15px 0; font-size: 0.9em; color: #7f8c8d; }
        .detail-item { display: flex; align-items: center; gap: 5px; }
        .btn { padding: 8px 16px; border: none; border-radius: 10px; font-size: 14px; font-weight: bold; cursor: pointer; transition: all 0.3s ease; text-decoration: none; display: inline-flex; align-items: center; gap: 5px; }
        .btn-primary { background: linear-gradient(45deg, #667eea, #764ba2); color: white; }
        .btn-danger { background: linear-gradient(45deg, #e74c3c, #c0392b); color: white; }
        .btn-success { background: linear-gradient(45deg, #27ae60, #2ecc71); color: white; }
        .btn:hover { transform: translateY(-2px); box-shadow: 0 5px 15px rgba(0,0,0,0.3); }
        .scheduler-panel { background: rgba(52, 152, 219, 0.1); border: 2px solid #3498db; }
        .scheduler-title { color: #3498db; }
        .job-item { background: white; padding: 15px; border-radius: 10px; margin-bottom: 10px; border-left: 4px solid #3498db; }
        .job-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
        .job-name { font-weight: bold; color: #2c3e50; }
        .job-status { padding: 3px 8px; border-radius: 15px; font-size: 0.8em; font-weight: bold; }
        .job-status.enabled { background: rgba(39, 174, 96, 0.1); color: #27ae60; }
        .job-status.disabled { background: rgba(149, 165, 166, 0.1); color: #95a5a6; }
        .job-details { font-size: 0.9em; color: #7f8c8d; }
        .cron-expression { font-family: monospace; background: #ecf0f1; padding: 3px 6px; border-radius: 3px; }
        .metrics-panel { background: rgba(39, 174, 96, 0.05); border: 2px solid #27ae60; }
        .metrics-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; }
        .metric-card { background: white; padding: 15px; border-radius: 10px; text-align: center; }
        .metric-value { font-size: 1.5em; font-weight: bold; margin-bottom: 5px; }
        .metric-label { font-size: 0.9em; color: #7f8c8d; }
        .ssl-indicator { display: inline-flex; align-items: center; gap: 5px; padding: 2px 8px; border-radius: 10px; font-size: 0.8em; }
        .ssl-valid { background: rgba(39, 174, 96, 0.1); color: #27ae60; }
        .ssl-invalid { background: rgba(231, 76, 60, 0.1); color: #e74c3c; }
        .ssl-unknown { background: rgba(149, 165, 166, 0.1); color: #95a5a6; }
        .site-error { color: #e74c3c; font-size: 0.9em; margin: 10px 0; background: rgba(231, 76, 60, 0.1); padding: 8px; border-radius: 5px; }
        .performance-details { margin-top: 15px; padding: 15px; background: rgba(52, 152, 219, 0.05); border-radius: 10px; }
        .performance-title { font-weight: bold; color: #3498db; margin-bottom: 10px; }
        .performance-metrics { display: grid; grid-template-columns: repeat(auto-fit, minmax(120px, 1fr)); gap: 10px; }
        .performance-metric { text-align: center; padding: 8px; background: white; border-radius: 5px; }
        .performance-metric-value { font-weight: bold; color: #2c3e50; }
        .performance-metric-label { font-size: 0.8em; color: #7f8c8d; margin-top: 3px; }
        .chart-container { height: 200px; margin: 20px 0; }
        .demo-actions { margin-top: 20px; text-align: center; }
        .demo-actions .btn { margin: 0 10px; }
        @media (max-width: 1200px) { .dashboard-content { grid-template-columns: 1fr; } }
        .realtime-indicator { 
            display: inline-block; 
            width: 8px; 
            height: 8px; 
            background: #27ae60; 
            border-radius: 50%; 
            animation: pulse 1s infinite; 
            margin-left: 10px;
        }
        .feature-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-top: 20px; }
        .feature-panel { background: rgba(255,255,255,0.9); border-radius: 15px; padding: 20px; }
        .feature-panel h4 { color: #2c3e50; margin-bottom: 15px; display: flex; align-items: center; gap: 10px; }
        .feature-list { list-style: none; }
        .feature-list li { margin-bottom: 8px; color: #7f8c8d; }
        .feature-list li:before { content: "✅ "; margin-right: 8px; }
    </style>
</head>
<body>
    <div class="container">
        <!-- Навигационная панель -->
        <nav class="navigation">
            <div class="nav-brand">
                <i class="fas fa-rocket"></i>
                Site Monitor Demo
            </div>
            <div class="nav-links">
                <a href="/" class="nav-link">
                    <i class="fas fa-tachometer-alt"></i> Основной дашборд
                </a>
                <a href="/demo" class="nav-link active demo">
                    <i class="fas fa-rocket"></i> Live Demo
                </a>
                <a href="/metrics" class="nav-link metrics">
                    <i class="fas fa-chart-bar"></i> Детальные метрики
                </a>
                <a href="/api/sites" class="nav-link">
                    <i class="fas fa-code"></i> API Endpoints
                </a>
            </div>
            <div class="nav-status">
                <div class="status-dot"></div>
                <span>Demo режим активен</span>
            </div>
        </nav>

        <div class="header">
            <h1><i class="fas fa-globe"></i> Site Monitor <span class="demo-badge">LIVE DEMO</span></h1>
            <p>Полная демонстрация функционала с реальными данными <span class="realtime-indicator"></span></p>
            <div style="margin-top: 15px;">
                <a href="/" class="btn btn-primary">
                    <i class="fas fa-arrow-left"></i> К основному интерфейсу
                </a>
                <a href="/metrics" class="btn btn-success">
                    <i class="fas fa-chart-line"></i> Детальная аналитика
                </a>
            </div>
        </div>

        <!-- Статистика -->
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-icon success"><i class="fas fa-check-circle"></i></div>
                <div class="stat-value success">{{.Stats.SitesUp}}</div>
                <div class="stat-label">Сайтов онлайн</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon danger"><i class="fas fa-times-circle"></i></div>
                <div class="stat-value danger">{{.Stats.SitesDown}}</div>
                <div class="stat-label">Сайтов оффлайн</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon info"><i class="fas fa-chart-line"></i></div>
                <div class="stat-value info">{{printf "%.1f%%" .Stats.AvgUptime}}</div>
                <div class="stat-label">Средний аптайм</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon warning"><i class="fas fa-clock"></i></div>
                <div class="stat-value warning">{{printf "%.0fмс" .Stats.AvgResponseTime}}</div>
                <div class="stat-label">Среднее время</div>
            </div>
        </div>

        <div class="dashboard-content">
            <!-- Левая панель - Сайты -->
            <div class="panel">
                <div class="panel-title">
                    <i class="fas fa-server"></i> Мониторинг сайтов ({{len .Sites}} сайтов)
                </div>
                
                {{range $index, $site := .Sites}}
                <div class="site-card {{$site.Status}}">
                    <div class="site-header">
                        <div class="site-url">{{$site.URL}}</div>
                        <div class="site-status {{$site.Status}}">
                            <i class="fas fa-{{if eq $site.Status "up"}}check{{else if eq $site.Status "down"}}times{{else}}question{{end}}"></i> 
                            {{if eq $site.Status "up"}}UP{{else if eq $site.Status "down"}}DOWN{{else}}UNKNOWN{{end}}
                        </div>
                    </div>
                    <div class="site-details">
                        {{if gt $site.ResponseTime 0}}
                        <div class="detail-item"><i class="fas fa-clock"></i> {{$site.ResponseTime}}мс</div>
                        {{end}}
                        {{if gt $site.ContentLength 0}}
                        <div class="detail-item"><i class="fas fa-file-alt"></i> 
                            {{if lt $site.ContentLength 1024}}{{$site.ContentLength}} Б
                            {{else if lt $site.ContentLength 1048576}}{{printf "%.1f КБ" (divFloat $site.ContentLength 1024)}}
                            {{else}}{{printf "%.1f МБ" (divFloat $site.ContentLength 1048576)}}{{end}}
                        </div>
                        {{end}}
                        <div class="detail-item"><i class="fas fa-chart-line"></i> {{printf "%.1f%%" $site.UptimePercent}} аптайм</div>
                        {{if gt $site.StatusCode 0}}
                        <div class="detail-item"><i class="fas fa-code"></i> {{$site.StatusCode}}</div>
                        {{end}}
                        <div class="detail-item"><i class="fas fa-calendar"></i> {{timeAgo $site.LastChecked}}</div>
                        {{if hasPrefix $site.URL "https://"}}
                        <div class="detail-item">
                            <span class="ssl-indicator {{if $site.SSLValid}}ssl-valid{{else}}ssl-invalid{{end}}">
                                <i class="fas fa-{{if $site.SSLValid}}lock{{else}}lock-open{{end}}"></i> 
                                SSL {{if $site.SSLValid}}OK{{else}}Ошибка{{end}}
                            </span>
                        </div>
                        {{end}}
                    </div>
                    
                    {{if ne $site.LastError ""}}
                    <div class="site-error">
                        <i class="fas fa-exclamation-triangle"></i> {{$site.LastError}}
                    </div>
                    {{end}}

                    {{if $site.Config}}
                    <div style="margin-top: 10px;">
                        <strong>🕐 Расписание:</strong> 
                        <span class="cron-expression">{{$site.Config.GetEffectiveSchedule}}</span> 
                        ({{$site.Config.GetScheduleDescription}})
                    </div>
                    {{end}}

                    <!-- Детальные метрики производительности -->
                    {{if or (gt $site.DNSTime 0) (gt $site.ConnectTime 0) (gt $site.TLSTime 0) (gt $site.TTFB 0)}}
                    <div class="performance-details">
                        <div class="performance-title"><i class="fas fa-stopwatch"></i> Детальная производительность</div>
                        <div class="performance-metrics">
                            {{if gt $site.DNSTime 0}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.DNSTime}}мс</div>
                                <div class="performance-metric-label">DNS Lookup</div>
                            </div>
                            {{end}}
                            {{if gt $site.ConnectTime 0}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.ConnectTime}}мс</div>
                                <div class="performance-metric-label">TCP Connect</div>
                            </div>
                            {{end}}
                            {{if and (hasPrefix $site.URL "https://") (gt $site.TLSTime 0)}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.TLSTime}}мс</div>
                                <div class="performance-metric-label">TLS Handshake</div>
                            </div>
                            {{end}}
                            {{if gt $site.TTFB 0}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.TTFB}}мс</div>
                                <div class="performance-metric-label">Time to First Byte</div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}

                    <!-- SSL детали -->
                    {{if and (hasPrefix $site.URL "https://") (ne $site.SSLAlgorithm "")}}
                    <div class="performance-details">
                        <div class="performance-title"><i class="fas fa-shield-alt"></i> SSL Сертификат</div>
                        <div class="performance-metrics">
                            {{if ne $site.SSLAlgorithm ""}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.SSLAlgorithm}}</div>
                                <div class="performance-metric-label">Алгоритм</div>
                            </div>
                            {{end}}
                            {{if gt $site.SSLKeyLength 0}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{$site.SSLKeyLength}} бит</div>
                                <div class="performance-metric-label">Длина ключа</div>
                            </div>
                            {{end}}
                            {{if ne $site.SSLIssuer ""}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{truncate $site.SSLIssuer 15}}</div>
                                <div class="performance-metric-label">Издатель</div>
                            </div>
                            {{end}}
                            {{if $site.SSLExpiry}}
                            <div class="performance-metric">
                                <div class="performance-metric-value">{{daysUntil $site.SSLExpiry}} дн.</div>
                                <div class="performance-metric-label">До истечения</div>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}

                    <div style="margin-top: 15px;">
                        <button class="btn btn-primary" onclick="window.location.href='/'">
                            <i class="fas fa-cog"></i> Настроить
                        </button>
                        <button class="btn btn-danger" onclick="deleteSite('{{$site.URL}}')">
                            <i class="fas fa-trash"></i> Удалить
                        </button>
                        <button class="btn btn-success" onclick="checkSite('{{$site.URL}}')">
                            <i class="fas fa-sync"></i> Проверить
                        </button>
                    </div>
                </div>
                {{else}}
                <div class="site-card">
                    <div class="site-header">
                        <div class="site-url">Нет сайтов для мониторинга</div>
                    </div>
                    <div class="demo-actions">
                        <button class="btn btn-primary" onclick="addDemoSites()">
                            <i class="fas fa-plus"></i> Добавить демо-сайты
                        </button>
                        <button class="btn btn-primary" onclick="window.location.href='/'">
                            <i class="fas fa-home"></i> Перейти к основному интерфейсу
                        </button>
                    </div>
                </div>
                {{end}}
            </div>

            <!-- Правая панель - Планировщик заданий -->
            <div class="panel scheduler-panel">
                <div class="panel-title scheduler-title">
                    <i class="fas fa-calendar-alt"></i> Cron Планировщик
                </div>
                
                {{range .Jobs}}
                <div class="job-item">
                    <div class="job-header">
                        <div class="job-name">{{.SiteURL}}</div>
                        <div class="job-status {{.Status}}">{{if eq .Status "enabled"}}Активно{{else}}Отключено{{end}}</div>
                    </div>
                    <div class="job-details">
                        <div><strong>Cron:</strong> <span class="cron-expression">{{.Schedule}}</span></div>
                        <div><strong>Описание:</strong> {{.Description}}</div>
                        <div><strong>Запусков:</strong> {{.RunCount}} ({{.ErrorCount}} ошибок)</div>
                        <div><strong>Следующий:</strong> {{.NextRun}}</div>
                        <div><strong>Последний:</strong> {{.LastRun}}</div>
                    </div>
                </div>
                {{else}}
                <div class="job-item">
                    <div class="job-header">
                        <div class="job-name">Нет активных заданий</div>
                        <div class="job-status disabled">Пусто</div>
                    </div>
                    <div class="job-details">
                        <div>Добавьте сайты для создания cron-заданий мониторинга</div>
                    </div>
                </div>
                {{end}}

                <div style="margin-top: 20px; text-align: center;">
                    <button class="btn btn-primary" onclick="triggerCheck()">
                        <i class="fas fa-play"></i> Запустить проверку
                    </button>
                </div>
            </div>
        </div>

        <!-- Панель метрик ClickHouse -->
        <div class="panel metrics-panel">
            <div class="panel-title">
                <i class="fas fa-chart-bar"></i> Метрики системы
            </div>
            <div class="metrics-grid">
                <div class="metric-card">
                    <div class="metric-value success">{{.Metrics.total_checks}}</div>
                    <div class="metric-label">Всего проверок</div>
                </div>
                <div class="metric-card">
                    <div class="metric-value info">{{.Metrics.avg_dns}}мс</div>
                    <div class="metric-label">Средний DNS</div>
                </div>
                <div class="metric-card">
                    <div class="metric-value warning">{{.Metrics.avg_connect}}мс</div>
                    <div class="metric-label">Средний TCP</div>
                </div>
                <div class="metric-card">
                    <div class="metric-value danger">{{.Metrics.avg_tls}}мс</div>
                    <div class="metric-label">Средний TLS</div>
                </div>
                <div class="metric-card">
                    <div class="metric-value info">{{.Metrics.avg_ttfb}}мс</div>
                    <div class="metric-label">Средний TTFB</div>
                </div>
                <div class="metric-card">
                    <div class="metric-value success">{{.Metrics.ssl_valid_percent}}%</div>
                    <div class="metric-label">Валидный SSL</div>
                </div>
            </div>
        </div>

        <!-- Возможности системы -->
        <div class="feature-grid">
            <div class="feature-panel">
                <h4><i class="fas fa-calendar-alt"></i> Cron-подобный планировщик</h4>
                <ul class="feature-list">
                    <li>Поддержка полных cron-выражений (*/5 * * * *)</li>
                    <li>Индивидуальные расписания для каждого сайта</li>
                    <li>Статистика выполнения заданий</li>
                    <li>Автоматическое восстановление при ошибках</li>
                    <li>Управление заданиями через веб-интерфейс</li>
                </ul>
            </div>
            
            <div class="feature-panel">
                <h4><i class="fas fa-stopwatch"></i> Детальная диагностика</h4>
                <ul class="feature-list">
                    <li>DNS Lookup Time - время разрешения домена</li>
                    <li>TCP Connect Time - время установления соединения</li>
                    <li>TLS Handshake Time - время SSL рукопожатия</li>
                    <li>Time to First Byte (TTFB) - время до первого байта</li>
                    <li>Полное время отклика с разбивкой по этапам</li>
                </ul>
            </div>
            
            <div class="feature-panel">
                <h4><i class="fas fa-shield-alt"></i> SSL/TLS мониторинг</h4>
                <ul class="feature-list">
                    <li>Проверка валидности SSL сертификатов</li>
                    <li>Отслеживание дат истечения сертификатов</li>
                    <li>Анализ алгоритмов шифрования (RSA, ECDSA)</li>
                    <li>Информация о длине ключа и издателе</li>
                    <li>Предупреждения об истекающих сертификатах</li>
                </ul>
            </div>
            
            <div class="feature-panel">
                <h4><i class="fas fa-database"></i> ClickHouse метрики</h4>
                <ul class="feature-list">
                    <li>Высокопроизводительная аналитическая БД</li>
                    <li>Агрегированные почасовые и дневные метрики</li>
                    <li>История производительности и доступности</li>
                    <li>Батчевая загрузка для оптимальной производительности</li>
                    <li>Автоматическая очистка старых данных (TTL)</li>
                </ul>
            </div>
            
            <div class="feature-panel">
                <h4><i class="fas fa-mobile-alt"></i> Современный интерфейс</h4>
                <ul class="feature-list">
                    <li>Адаптивный дизайн для всех устройств</li>
                    <li>Real-time обновления через Server-Sent Events</li>
                    <li>Интерактивные графики и диаграммы</li>
                    <li>Детальные настройки для каждого сайта</li>
                    <li>Темная и светлая темы оформления</li>
                </ul>
            </div>
            
            <div class="feature-panel">
                <h4><i class="fas fa-server"></i> Enterprise возможности</h4>
                <ul class="feature-list">
                    <li>Docker контейнеризация</li>
                    <li>PostgreSQL для конфигурации</li>
                    <li>REST API для интеграций</li>
                    <li>Система уведомлений (email, webhook)</li>
                    <li>Масштабируемая архитектура</li>
                </ul>
            </div>
        </div>

        <div class="demo-actions">
            <button class="btn btn-success" onclick="addDemoSites()">
                <i class="fas fa-plus"></i> Добавить демо-сайты
            </button>
            <button class="btn btn-primary" onclick="window.location.href='/'">
                <i class="fas fa-home"></i> Основной интерфейс
            </button>
            <button class="btn btn-info" onclick="window.location.href='/metrics'">
                <i class="fas fa-chart-bar"></i> Детальные метрики
            </button>
            <button class="btn btn-warning" onclick="triggerCheck()">
                <i class="fas fa-sync"></i> Проверить все сайты
            </button>
            <button class="btn btn-info" onclick="window.location.reload()">
                <i class="fas fa-redo"></i> Обновить данные
            </button>
        </div>
    </div>

    <script>
        // Простая демонстрация обновлений
        function updateDemo() {
            const values = document.querySelectorAll('.stat-value');
            values.forEach(value => {
                if (value.textContent.includes('мс')) {
                    const currentMs = parseInt(value.textContent);
                    const newMs = currentMs + Math.floor(Math.random() * 20) - 10;
                    value.textContent = Math.max(100, newMs) + 'мс';
                }
            });
        }
        
        // Обновляем значения каждые 3 секунды для демонстрации
        setInterval(updateDemo, 3000);
        
        function addDemoSites() {
            const demoSites = [
                'https://google.com',
                'https://github.com',
                'https://stackoverflow.com',
                'https://cloudflare.com',
                'https://httpbin.org/status/500'
            ];

            let added = 0;
            demoSites.forEach(url => {
                fetch('/api/sites', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: url })
                })
                .then(response => response.json())
                .then(data => {
                    if (data.message) {
                        added++;
                        if (added === demoSites.length) {
                            showNotification('✅ Все демо-сайты добавлены! Обновляем страницу...', 'success');
                            setTimeout(() => window.location.reload(), 2000);
                        }
                    }
                })
                .catch(error => console.error('Error adding demo site:', error));
            });
        }

        function triggerCheck() {
            showNotification('🔄 Запущена проверка всех сайтов...', 'info');
            fetch('/api/check', { method: 'POST' })
                .then(response => response.json())
                .then(data => {
                    showNotification('✅ Проверка запущена успешно', 'success');
                    setTimeout(() => window.location.reload(), 3000);
                })
                .catch(error => {
                    showNotification('❌ Ошибка запуска проверки', 'error');
                });
        }

        function checkSite(url) {
            showNotification('🔍 Проверяем сайт: ' + url, 'info');
        }

        function deleteSite(url) {
            if (confirm('Удалить сайт из мониторинга: ' + url + '?')) {
                fetch('/api/sites/delete', {
                    method: 'DELETE',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ url: url })
                })
                .then(response => response.json())
                .then(data => {
                    showNotification('🗑️ Сайт удален: ' + url, 'success');
                    setTimeout(() => window.location.reload(), 1000);
                })
                .catch(error => {
                    showNotification('❌ Ошибка удаления сайта', 'error');
                });
            }
        }

        function showNotification(message, type) {
            type = type || 'info';
            const notification = document.createElement('div');
            const colors = {
                success: '#27ae60',
                error: '#e74c3c', 
                warning: '#f39c12',
                info: '#3498db'
            };
            
            notification.style.cssText = 'position: fixed; top: 20px; right: 20px; padding: 15px 20px; border-radius: 10px; color: white; font-weight: bold; z-index: 9999; opacity: 0; transition: opacity 0.3s ease; max-width: 300px; word-wrap: break-word; background: ' + (colors[type] || colors.info) + ';';
            notification.textContent = message;
            document.body.appendChild(notification);
            
            setTimeout(function() { notification.style.opacity = '1'; }, 100);
            setTimeout(function() {
                notification.style.opacity = '0';
                setTimeout(function() { 
                    if (document.body.contains(notification)) {
                        document.body.removeChild(notification); 
                    }
                }, 300);
            }, 3000);
        }
        
        // Auto-refresh every 30 seconds
        setInterval(function() {
            window.location.reload();
        }, 30000);
    </script>
</body>
</html>`

func DemoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Собираем реальные данные из базы
		data := DemoData{
			Stats: DashboardStats{},
			Sites: []models.Site{},
			Metrics: make(map[string]interface{}),
			Jobs: []CronJob{},
		}

		if demoDatabase != nil {
			// Получаем статистику
			countQuery := `SELECT COUNT(*) FROM sites`
			demoDatabase.QueryRow(countQuery).Scan(&data.Stats.TotalSites)
			
			if data.Stats.TotalSites > 0 {
				statsQuery := `SELECT 
								COUNT(CASE WHEN status = 'up' THEN 1 END) as up,
								COUNT(CASE WHEN status = 'down' THEN 1 END) as down,
								COALESCE(AVG(CASE WHEN COALESCE(total_checks, 0) > 0 THEN (COALESCE(successful_checks, 0)::float / COALESCE(total_checks, 1)::float * 100) ELSE 0 END), 0) as avg_uptime,
								COALESCE(AVG(COALESCE(response_time, 0)::float), 0) as avg_response_time
							  FROM sites`
				
				demoDatabase.QueryRow(statsQuery).Scan(&data.Stats.SitesUp, &data.Stats.SitesDown, &data.Stats.AvgUptime, &data.Stats.AvgResponseTime)
			}

			// Получаем сайты с конфигурациями
			sites, err := demoDatabase.GetAllSites()
			if err == nil {
				for i, site := range sites {
					config, err := demoDatabase.GetSiteConfig(site.ID)
					if err == nil {
						sites[i].Config = config
					}
				}
				data.Sites = sites
			}

			// Создаем демо cron jobs на основе реальных сайтов
			for _, site := range data.Sites {
				job := CronJob{
					SiteURL:     site.URL,
					Schedule:    "*/5 * * * *",
					Status:      "enabled",
					RunCount:    int(site.TotalChecks),
					ErrorCount:  int(site.TotalChecks - site.SuccessfulChecks),
					NextRun:     "через 2-3 минуты",
					LastRun:     timeAgo(site.LastChecked),
					Description: "Автоматическая проверка каждые 5 минут",
				}

				if site.Config != nil {
					job.Schedule = site.Config.GetEffectiveSchedule()
					job.Description = site.Config.GetScheduleDescription()
					if !site.Config.Enabled {
						job.Status = "disabled"
					}
				}

				data.Jobs = append(data.Jobs, job)
			}
		}

		// Создаем демо-метрики
		data.Metrics = map[string]interface{}{
			"total_checks":      calculateTotalChecks(data.Sites),
			"avg_dns":          calculateAvgDNS(data.Sites),
			"avg_connect":      calculateAvgConnect(data.Sites),
			"avg_tls":          calculateAvgTLS(data.Sites),
			"avg_ttfb":         calculateAvgTTFB(data.Sites),
			"ssl_valid_percent": calculateSSLValidPercent(data.Sites),
		}

		// Создаем функции для шаблона
		funcMap := template.FuncMap{
			"timeAgo": timeAgo,
			"divFloat": func(a, b int64) float64 {
				if b == 0 {
					return 0
				}
				return float64(a) / float64(b)
			},
			"hasPrefix": func(s, prefix string) bool {
				return len(s) >= len(prefix) && s[:len(prefix)] == prefix
			},
			"truncate": func(s string, length int) string {
				if len(s) <= length {
					return s
				}
				return s[:length] + "..."
			},
			"daysUntil": func(t *time.Time) int {
				if t == nil {
					return 0
				}
				return int(time.Until(*t).Hours() / 24)
			},
		}

		tmpl := template.Must(template.New("demo").Funcs(funcMap).Parse(demoTemplate))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
	}
}

// Вспомогательные функции для расчета метрик
func calculateTotalChecks(sites []models.Site) int {
	total := 0
	for _, site := range sites {
		total += site.TotalChecks
	}
	return total
}

func calculateAvgDNS(sites []models.Site) int {
	if len(sites) == 0 {
		return 0
	}
	total := int64(0)
	count := 0
	for _, site := range sites {
		if site.DNSTime > 0 {
			total += site.DNSTime
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(total / int64(count))
}

func calculateAvgConnect(sites []models.Site) int {
	if len(sites) == 0 {
		return 0
	}
	total := int64(0)
	count := 0
	for _, site := range sites {
		if site.ConnectTime > 0 {
			total += site.ConnectTime
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(total / int64(count))
}

func calculateAvgTLS(sites []models.Site) int {
	if len(sites) == 0 {
		return 0
	}
	total := int64(0)
	count := 0
	for _, site := range sites {
		if site.TLSTime > 0 {
			total += site.TLSTime
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(total / int64(count))
}

func calculateAvgTTFB(sites []models.Site) int {
	if len(sites) == 0 {
		return 0
	}
	total := int64(0)
	count := 0
	for _, site := range sites {
		if site.TTFB > 0 {
			total += site.TTFB
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return int(total / int64(count))
}

func calculateSSLValidPercent(sites []models.Site) float64 {
	if len(sites) == 0 {
		return 0
	}
	httpsCount := 0
	validCount := 0
	for _, site := range sites {
		if len(site.URL) >= 8 && site.URL[:8] == "https://" {
			httpsCount++
			if site.SSLValid {
				validCount++
			}
		}
	}
	if httpsCount == 0 {
		return 0
	}
	return float64(validCount) / float64(httpsCount) * 100
}

func timeAgo(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "только что"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%d мин назад", minutes)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d ч назад", hours)
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d дн назад", days)
	}
}