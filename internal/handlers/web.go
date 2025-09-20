package handlers

import (
	"html/template"
	"net/http"
)

const webTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Site Monitor - Мониторинг сайтов</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css" rel="stylesheet">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            margin-bottom: 30px;
            text-align: center;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .header h1 {
            color: #2c3e50;
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .header p {
            color: #7f8c8d;
            font-size: 1.2em;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }

        .stat-card {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 15px;
            padding: 25px;
            text-align: center;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            transition: transform 0.3s ease;
        }

        .stat-card:hover {
            transform: translateY(-5px);
        }

        .stat-icon {
            font-size: 2.5em;
            margin-bottom: 15px;
        }

        .stat-value {
            font-size: 2.2em;
            font-weight: bold;
            margin-bottom: 5px;
        }

        .stat-label {
            color: #7f8c8d;
            font-size: 0.9em;
        }

        .success { color: #27ae60; }
        .danger { color: #e74c3c; }
        .info { color: #3498db; }
        .warning { color: #f39c12; }

        .dashboard-content {
            display: grid;
            grid-template-columns: 1fr 400px;
            gap: 30px;
            margin-bottom: 30px;
        }

        .sites-panel, .chart-panel {
            background: rgba(255, 255, 255, 0.95);
            backdrop-filter: blur(10px);
            border-radius: 20px;
            padding: 30px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
        }

        .panel-title {
            font-size: 1.8em;
            margin-bottom: 20px;
            color: #2c3e50;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .add-site-form {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 15px;
            margin-bottom: 20px;
        }

        .form-group {
            display: flex;
            gap: 10px;
        }

        .form-input {
            flex: 1;
            padding: 12px;
            border: 2px solid #e9ecef;
            border-radius: 10px;
            font-size: 16px;
            transition: border-color 0.3s ease;
        }

        .form-input:focus {
            outline: none;
            border-color: #667eea;
        }

        .btn {
            padding: 12px 24px;
            border: none;
            border-radius: 10px;
            font-size: 16px;
            font-weight: bold;
            cursor: pointer;
            transition: all 0.3s ease;
            text-decoration: none;
            display: inline-flex;
            align-items: center;
            gap: 8px;
        }

        .btn-primary {
            background: linear-gradient(45deg, #667eea, #764ba2);
            color: white;
        }

        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(102, 126, 234, 0.4);
        }

        .btn-danger {
            background: linear-gradient(45deg, #e74c3c, #c0392b);
            color: white;
            padding: 8px 16px;
            font-size: 14px;
        }

        .btn-danger:hover {
            transform: translateY(-1px);
        }

        .sites-list {
            max-height: 600px;
            overflow-y: auto;
        }

        .site-card {
            background: #f8f9fa;
            border-radius: 15px;
            padding: 20px;
            margin-bottom: 15px;
            transition: all 0.3s ease;
            border-left: 5px solid #e9ecef;
        }

        .site-card.up {
            border-left-color: #27ae60;
        }

        .site-card.down {
            border-left-color: #e74c3c;
        }

        .site-card:hover {
            transform: translateX(5px);
            box-shadow: 0 5px 15px rgba(0, 0, 0, 0.1);
        }

        .site-header {
            display: flex;
            justify-content: between;
            align-items: center;
            margin-bottom: 10px;
        }

        .site-url {
            font-weight: bold;
            font-size: 1.1em;
            color: #2c3e50;
            flex: 1;
        }

        .site-status {
            display: flex;
            align-items: center;
            gap: 5px;
            padding: 5px 12px;
            border-radius: 20px;
            font-size: 0.9em;
            font-weight: bold;
        }

        .site-status.up {
            background: rgba(39, 174, 96, 0.1);
            color: #27ae60;
        }

        .site-status.down {
            background: rgba(231, 76, 60, 0.1);
            color: #e74c3c;
        }

        .site-details {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 10px;
            margin-bottom: 15px;
            font-size: 0.9em;
            color: #7f8c8d;
        }

        .detail-item {
            display: flex;
            align-items: center;
            gap: 5px;
        }

        .site-actions {
            display: flex;
            gap: 10px;
            justify-content: flex-end;
        }

        .chart-container {
            height: 300px;
            margin: 20px 0;
        }

        @media (max-width: 1200px) {
            .dashboard-content {
                grid-template-columns: 1fr;
            }
        }

        @media (max-width: 768px) {
            .stats-grid {
                grid-template-columns: repeat(2, 1fr);
            }
            
            .container {
                padding: 10px;
            }
            
            .header {
                padding: 20px;
            }
            
            .header h1 {
                font-size: 2em;
            }
        }

        .loading {
            text-align: center;
            padding: 40px;
            color: #7f8c8d;
        }

        .spinner {
            border: 4px solid #f3f3f3;
            border-top: 4px solid #667eea;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto 20px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        .ssl-indicator {
            display: inline-flex;
            align-items: center;
            gap: 5px;
            padding: 2px 8px;
            border-radius: 10px;
            font-size: 0.8em;
        }

        .ssl-valid {
            background: rgba(39, 174, 96, 0.1);
            color: #27ae60;
        }

        .ssl-invalid {
            background: rgba(231, 76, 60, 0.1);
            color: #e74c3c;
        }

        .site-details-toggle {
            background: #667eea;
            color: white;
            border: none;
            padding: 8px 16px;
            border-radius: 20px;
            cursor: pointer;
            font-size: 0.9em;
            margin-top: 10px;
            transition: all 0.3s ease;
        }
        
        .site-details-toggle:hover {
            background: #5a6fd8;
            transform: translateY(-1px);
        }
        
        .site-details-expanded {
            margin-top: 15px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 15px;
            border-left: 4px solid #667eea;
            display: none;
        }
        
        .details-section {
            margin-bottom: 20px;
        }
        
        .details-section h4 {
            color: #2c3e50;
            margin-bottom: 10px;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .details-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 10px;
            margin-bottom: 15px;
        }
        
        .detail-metric {
            background: white;
            padding: 12px;
            border-radius: 8px;
            border-left: 3px solid #3498db;
        }
        
        .metric-label {
            font-size: 0.8em;
            color: #7f8c8d;
            margin-bottom: 4px;
        }
        
        .metric-value {
            font-weight: bold;
            color: #2c3e50;
        }
        
        .ssl-details {
            background: rgba(39, 174, 96, 0.1);
            border-left-color: #27ae60;
        }
        
        .ssl-details.invalid {
            background: rgba(231, 76, 60, 0.1);
            border-left-color: #e74c3c;
        }
        
        .performance-bar {
            background: #ecf0f1;
            height: 8px;
            border-radius: 4px;
            overflow: hidden;
            margin-top: 4px;
        }
        
        .performance-fill {
            height: 100%;
            transition: width 0.3s ease;
        }
        
        .perf-excellent { background: #27ae60; }
        .perf-good { background: #f39c12; }
        .perf-poor { background: #e74c3c; }
        
        .content-info {
            background: #fff;
            padding: 10px;
            border-radius: 8px;
            font-family: monospace;
            font-size: 0.9em;
            color: #555;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1><i class="fas fa-globe"></i> Site Monitor</h1>
            <p>Профессиональный мониторинг доступности веб-сайтов</p>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-icon success"><i class="fas fa-check-circle"></i></div>
                <div class="stat-value success" id="sitesUp">-</div>
                <div class="stat-label">Сайтов онлайн</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon danger"><i class="fas fa-times-circle"></i></div>
                <div class="stat-value danger" id="sitesDown">-</div>
                <div class="stat-label">Сайтов оффлайн</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon info"><i class="fas fa-chart-line"></i></div>
                <div class="stat-value info" id="avgUptime">-</div>
                <div class="stat-label">Средний аптайм</div>
            </div>
            <div class="stat-card">
                <div class="stat-icon warning"><i class="fas fa-clock"></i></div>
                <div class="stat-value warning" id="avgResponse">-</div>
                <div class="stat-label">Среднее время отклика</div>
            </div>
        </div>

        <div class="dashboard-content">
            <div class="sites-panel">
                <div class="panel-title">
                    <i class="fas fa-server"></i>
                    Мониторинг сайтов
                </div>
                
                <div class="add-site-form">
                    <form id="addSiteForm">
                        <div class="form-group">
                            <input type="url" class="form-input" id="url" name="url" placeholder="https://example.com" required>
                            <button type="submit" class="btn btn-primary">
                                <i class="fas fa-plus"></i>
                                Добавить
                            </button>
                            <button type="button" class="btn btn-primary" onclick="triggerCheck()">
                                <i class="fas fa-sync"></i>
                                Проверить сейчас
                            </button>
                        </div>
                    </form>
                </div>

                <div class="sites-list" id="sitesList">
                    <div class="loading">
                        <div class="spinner"></div>
                        Загрузка данных...
                    </div>
                </div>
            </div>

            <div class="chart-panel">
                <div class="panel-title">
                    <i class="fas fa-chart-pie"></i>
                    Статистика
                </div>
                <div class="chart-container">
                    <canvas id="statusChart"></canvas>
                </div>
            </div>
        </div>
    </div>

    <script>
        let statusChart = null;
        let eventSource = null;
        
        // Подключаемся к Server-Sent Events
        function connectSSE() {
            if (eventSource) {
                eventSource.close();
            }
            
            eventSource = new EventSource('/api/sse');
            
            eventSource.onmessage = function(event) {
                try {
                    const message = JSON.parse(event.data);
                    handleSSEMessage(message);
                } catch (error) {
                    console.error('Ошибка парсинга SSE сообщения:', error);
                }
            };
            
            eventSource.onerror = function(event) {
                console.error('SSE connection error:', event);
                // Переподключаемся через 5 секунд
                setTimeout(connectSSE, 5000);
            };
            
            eventSource.onopen = function(event) {
                console.log('SSE подключение установлено');
            };
        }
        
        function handleSSEMessage(message) {
            console.log('SSE сообщение:', message);
            
            switch (message.type) {
                case 'site_checked':
                    console.log('Сайт проверен:', message.data);
                    loadSites();
                    loadDashboardStats();
                    showNotification('Проверен сайт: ' + message.data.url + ' - ' + message.data.status.toUpperCase(), 
                        message.data.status === 'up' ? 'success' : 'error');
                    break;
                case 'site_added':
                    console.log('Сайт добавлен:', message.data);
                    loadSites();
                    loadDashboardStats();
                    showNotification('Добавлен сайт: ' + message.data.url, 'success');
                    break;
                case 'site_deleted':
                    console.log('Сайт удален:', message.data);
                    loadSites();
                    loadDashboardStats();
                    showNotification('Удален сайт: ' + message.data.url, 'warning');
                    break;
                case 'check_started':
                    console.log('Проверка запущена');
                    showNotification('Проверка всех сайтов запущена', 'info');
                    break;
            }
        }
        
        function showNotification(message, type) {
            type = type || 'info';
            const notification = document.createElement('div');
            notification.style.cssText = 
                'position: fixed;' +
                'top: 20px;' +
                'right: 20px;' +
                'padding: 15px 20px;' +
                'border-radius: 10px;' +
                'color: white;' +
                'font-weight: bold;' +
                'z-index: 9999;' +
                'opacity: 0;' +
                'transition: opacity 0.3s ease;' +
                'max-width: 300px;' +
                'word-wrap: break-word;';
            
            const colors = {
                success: '#27ae60',
                error: '#e74c3c',
                warning: '#f39c12',
                info: '#3498db'
            };
            
            notification.style.backgroundColor = colors[type] || colors.info;
            notification.textContent = message;
            
            document.body.appendChild(notification);
            
            // Анимация появления
            setTimeout(function() {
                notification.style.opacity = '1';
            }, 100);
            
            // Автоматическое удаление
            setTimeout(function() {
                notification.style.opacity = '0';
                setTimeout(function() {
                    if (document.body.contains(notification)) {
                        document.body.removeChild(notification);
                    }
                }, 300);
            }, 4000);
        }
        
        function formatTime(ms) {
            if (ms < 1000) return ms + 'мс';
            return (ms / 1000).toFixed(1) + 'с';
        }

        function formatBytes(bytes) {
            if (bytes === 0) return '0 Б';
            const k = 1024;
            const sizes = ['Б', 'КБ', 'МБ', 'ГБ'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }

        function formatDate(dateString) {
            return new Date(dateString).toLocaleString('ru-RU');
        }

        function getPerformanceClass(time) {
            if (time < 500) return 'perf-excellent';
            if (time < 2000) return 'perf-good';
            return 'perf-poor';
        }
        
        function getPerformanceWidth(time) {
            if (time < 100) return '20%';
            if (time < 500) return '40%';
            if (time < 1000) return '60%';
            if (time < 2000) return '80%';
            return '100%';
        }

        function loadDashboardStats() {
            fetch('/api/dashboard/stats')
                .then(response => response.json())
                .then(stats => {
                    document.getElementById('sitesUp').textContent = stats.sites_up || 0;
                    document.getElementById('sitesDown').textContent = stats.sites_down || 0;
                    document.getElementById('avgUptime').textContent = (stats.avg_uptime || 0).toFixed(1) + '%';
                    document.getElementById('avgResponse').textContent = formatTime(Math.round(stats.avg_response_time || 0));
                    
                    updateStatusChart(stats);
                })
                .catch(error => {
                    console.error('Ошибка загрузки статистики:', error);
                    document.getElementById('sitesUp').textContent = '0';
                    document.getElementById('sitesDown').textContent = '0';
                    document.getElementById('avgUptime').textContent = '0.0%';
                    document.getElementById('avgResponse').textContent = '0мс';
                });
        }

        function updateStatusChart(stats) {
            const ctx = document.getElementById('statusChart').getContext('2d');
            
            if (statusChart) {
                statusChart.destroy();
            }
            
            const sitesUp = stats.sites_up || 0;
            const sitesDown = stats.sites_down || 0;
            
            statusChart = new Chart(ctx, {
                type: 'doughnut',
                data: {
                    labels: ['Онлайн', 'Оффлайн'],
                    datasets: [{
                        data: [sitesUp, sitesDown],
                        backgroundColor: ['#27ae60', '#e74c3c'],
                        borderWidth: 0
                    }]
                },
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                        legend: {
                            position: 'bottom',
                            labels: {
                                padding: 20,
                                usePointStyle: true
                            }
                        }
                    }
                }
            });
        }

        function loadSites() {
            fetch('/api/sites')
                .then(response => response.json())
                .then(sites => {
                    const sitesList = document.getElementById('sitesList');
                    if (sites && sites.length > 0) {
                        sitesList.innerHTML = sites.map((site, index) => {
                            const sslIndicator = site.url.startsWith('https://') ? 
                                (site.ssl_valid ? 
                                    '<span class="ssl-indicator ssl-valid"><i class="fas fa-lock"></i> SSL OK</span>' : 
                                    '<span class="ssl-indicator ssl-invalid"><i class="fas fa-lock-open"></i> SSL Ошибка</span>') 
                                : '';
                            
                            return '<div class="site-card ' + site.status + '">' +
                                '<div class="site-header">' +
                                    '<div class="site-url">' + site.url + '</div>' +
                                    '<div class="site-status ' + site.status + '">' +
                                        '<i class="fas fa-' + (site.status === 'up' ? 'check' : 'times') + '"></i>' +
                                        site.status.toUpperCase() +
                                    '</div>' +
                                '</div>' +
                                '<div class="site-details">' +
                                    '<div class="detail-item"><i class="fas fa-code"></i> ' + (site.status_code || 'N/A') + '</div>' +
                                    '<div class="detail-item"><i class="fas fa-clock"></i> ' + formatTime(site.response_time_ms || 0) + '</div>' +
                                    '<div class="detail-item"><i class="fas fa-file-alt"></i> ' + formatBytes(site.content_length || 0) + '</div>' +
                                    '<div class="detail-item"><i class="fas fa-chart-line"></i> ' + (site.uptime_percent || 0).toFixed(1) + '% аптайм</div>' +
                                    '<div class="detail-item"><i class="fas fa-calendar"></i> ' + formatDate(site.last_checked) + '</div>' +
                                    '<div class="detail-item">' + sslIndicator + '</div>' +
                                '</div>' +
                                (site.last_error ? '<div style="color: #e74c3c; font-size: 0.9em; margin-bottom: 10px;"><i class="fas fa-exclamation-triangle"></i> ' + site.last_error + '</div>' : '') +
                                '<button class="site-details-toggle" onclick="toggleDetails(' + index + ')">' +
                                    '<i class="fas fa-info-circle"></i> Подробная информация' +
                                '</button>' +
                                '<div id="details-' + index + '" class="site-details-expanded">' +
                                    generateDetailedInfo(site) +
                                '</div>' +
                                '<div class="site-actions">' +
                                    '<button class="btn btn-danger" onclick="deleteSite(\'' + site.url + '\')">' +
                                        '<i class="fas fa-trash"></i> Удалить' +
                                    '</button>' +
                                '</div>' +
                            '</div>';
                        }).join('');
                    } else {
                        sitesList.innerHTML = '<div class="loading">Нет добавленных сайтов для мониторинга</div>';
                    }
                })
                .catch(error => {
                    console.error('Ошибка загрузки сайтов:', error);
                    document.getElementById('sitesList').innerHTML = '<div class="loading">Ошибка загрузки данных</div>';
                });
        }
        
        function generateDetailedInfo(site) {
            var detailsHtml = 
                '<div class="details-section">' +
                    '<h4><i class="fas fa-stopwatch"></i> Время отклика</h4>' +
                    '<div class="details-grid">' +
                        '<div class="detail-metric">' +
                            '<div class="metric-label">DNS Lookup</div>' +
                            '<div class="metric-value">' + formatTime(site.dns_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.dns_time) + '" style="width: ' + getPerformanceWidth(site.dns_time) + '"></div></div>' +
                        '</div>' +
                        '<div class="detail-metric">' +
                            '<div class="metric-label">TCP Connect</div>' +
                            '<div class="metric-value">' + formatTime(site.connect_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.connect_time) + '" style="width: ' + getPerformanceWidth(site.connect_time) + '"></div></div>' +
                        '</div>';
            
            if (site.url.startsWith('https://')) {
                detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">TLS Handshake</div>' +
                            '<div class="metric-value">' + formatTime(site.tls_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.tls_time) + '" style="width: ' + getPerformanceWidth(site.tls_time) + '"></div></div>' +
                        '</div>';
            }
            
            detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Time to First Byte</div>' +
                            '<div class="metric-value">' + formatTime(site.ttfb || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.ttfb) + '" style="width: ' + getPerformanceWidth(site.ttfb) + '"></div></div>' +
                        '</div>' +
                    '</div>' +
                '</div>';
                
            if (site.url.startsWith('https://')) {
                detailsHtml += 
                '<div class="details-section">' +
                    '<h4><i class="fas fa-shield-alt"></i> SSL/TLS Сертификат</h4>' +
                    '<div class="details-grid">' +
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Алгоритм</div>' +
                            '<div class="metric-value">' + (site.ssl_algorithm || 'N/A') + '</div>' +
                        '</div>' +
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Длина ключа</div>' +
                            '<div class="metric-value">' + (site.ssl_key_length || 0) + ' бит</div>' +
                        '</div>' +
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Издатель</div>' +
                            '<div class="metric-value">' + (site.ssl_issuer || 'N/A') + '</div>' +
                        '</div>';
                
                if (site.ssl_expiry) {
                    detailsHtml += 
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Действителен до</div>' +
                            '<div class="metric-value">' + formatDate(site.ssl_expiry) + '</div>' +
                        '</div>';
                }
                
                detailsHtml += 
                    '</div>' +
                '</div>';
            }
                
            detailsHtml += 
                '<div class="details-section">' +
                    '<h4><i class="fas fa-server"></i> Информация о сервере</h4>' +
                    '<div class="details-grid">';
                    
            if (site.server_type) {
                detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Сервер</div>' +
                            '<div class="metric-value">' + site.server_type + '</div>' +
                        '</div>';
            }
            
            if (site.powered_by) {
                detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Powered By</div>' +
                            '<div class="metric-value">' + site.powered_by + '</div>' +
                        '</div>';
            }
            
            detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Content-Type</div>' +
                            '<div class="metric-value">' + (site.content_type || 'N/A') + '</div>' +
                        '</div>';
                        
            if (site.cache_control) {
                detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Cache-Control</div>' +
                            '<div class="metric-value">' + site.cache_control + '</div>' +
                        '</div>';
            }
            
            detailsHtml += 
                    '</div>' +
                '</div>' +
                
                '<div class="details-section">' +
                    '<h4><i class="fas fa-exchange-alt"></i> Редиректы и навигация</h4>' +
                    '<div class="details-grid">' +
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Количество редиректов</div>' +
                            '<div class="metric-value">' + (site.redirect_count || 0) + '</div>' +
                        '</div>';
                        
            if (site.final_url && site.final_url !== site.url) {
                detailsHtml += 
                        '<div class="detail-metric" style="grid-column: 1 / -1;">' +
                            '<div class="metric-label">Финальный URL</div>' +
                            '<div class="metric-value" style="word-break: break-all;">' + site.final_url + '</div>' +
                        '</div>';
            }
            
            detailsHtml += 
                    '</div>' +
                '</div>' +
                
                '<div class="details-section">' +
                    '<h4><i class="fas fa-file-code"></i> Анализ контента</h4>' +
                    '<div class="content-info">' +
                        'Размер контента: ' + formatBytes(site.content_length || 0) + '<br>' +
                        'Хэш контента: ' + (site.content_hash || 'N/A') + ' (для отслеживания изменений)<br>' +
                        'Последняя проверка: ' + formatDate(site.last_checked) +
                    '</div>' +
                '</div>';
                
            return detailsHtml;
        }
        
        function toggleDetails(index) {
            const details = document.getElementById('details-' + index);
            const button = details.previousElementSibling;
            
            if (details.style.display === 'block') {
                details.style.display = 'none';
                button.innerHTML = '<i class="fas fa-info-circle"></i> Подробная информация';
            } else {
                details.style.display = 'block';
                button.innerHTML = '<i class="fas fa-times"></i> Скрыть подробности';
            }
        }
        
        document.getElementById('addSiteForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const url = document.getElementById('url').value;
            
            fetch('/api/sites', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ url: url })
            })
            .then(function(response) {
                return response.json();
            })
            .then(function(data) {
                if (data.message) {
                    document.getElementById('url').value = '';
                    // Не вызываем loadSites() и loadDashboardStats() здесь,
                    // потому что они будут вызваны через SSE
                } else if (data.error) {
                    showNotification('Ошибка: ' + data.error, 'error');
                }
            })
            .catch(function(error) {
                console.error('Ошибка добавления сайта:', error);
                showNotification('Ошибка добавления сайта', 'error');
            });
        });

        function deleteSite(url) {
            if (confirm('Вы уверены, что хотите удалить этот сайт из мониторинга?')) {
                fetch('/api/sites/' + encodeURIComponent(url), {
                    method: 'DELETE'
                })
                .then(function(response) {
                    return response.json();
                })
                .then(function(data) {
                    if (data.error) {
                        showNotification('Ошибка: ' + data.error, 'error');
                    }
                    // Успешное удаление будет обработано через SSE
                })
                .catch(function(error) {
                    console.error('Ошибка удаления сайта:', error);
                    showNotification('Ошибка удаления сайта', 'error');
                });
            }
        }

        function triggerCheck() {
            fetch('/api/check', {
                method: 'POST'
            })
            .then(function(response) {
                return response.json();
            })
            .then(function(data) {
                if (data.error) {
                    showNotification('Ошибка: ' + data.error, 'error');
                }
                // Уведомление о запуске будет показано через SSE
            })
            .catch(function(error) {
                console.error('Ошибка запуска проверки:', error);
                showNotification('Ошибка запуска проверки', 'error');
            });
        }

        // Подключаемся к SSE при загрузке страницы
        connectSSE();
        
        loadDashboardStats();
        loadSites();
        
        // Убираем автоматическое обновление по таймеру, так как теперь используем SSE
        // setInterval(function() {
        //     loadDashboardStats();
        //     loadSites();
        // }, 30000);
    </script>
</body>
</html>`

func WebInterfaceHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.Must(template.New("web").Parse(webTemplate))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, nil)
	}
}