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
        
        .config-modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0,0,0,0.5);
        }
        
        .config-content {
            background-color: #fff;
            margin: 5% auto;
            padding: 30px;
            border-radius: 20px;
            width: 90%;
            max-width: 600px;
            max-height: 80vh;
            overflow-y: auto;
        }
        
        .config-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
            padding-bottom: 15px;
            border-bottom: 2px solid #ecf0f1;
        }
        
        .config-form {
            display: grid;
            gap: 20px;
        }
        
        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }
        
        .form-field {
            display: flex;
            flex-direction: column;
        }
        
        .form-field.full-width {
            grid-column: 1 / -1;
        }
        
        .form-label {
            font-weight: bold;
            margin-bottom: 5px;
            color: #2c3e50;
            font-size: 0.9em;
        }
        
        .form-control {
            padding: 10px;
            border: 2px solid #e9ecef;
            border-radius: 8px;
            font-size: 14px;
        }
        
        .form-control:focus {
            outline: none;
            border-color: #667eea;
        }
        
        .checkbox-field {
            display: flex;
            align-items: center;
            gap: 8px;
            margin-top: 20px;
        }
        
        .checkbox-field input[type="checkbox"] {
            width: 18px;
            height: 18px;
        }
        
        .config-actions {
            display: flex;
            justify-content: flex-end;
            gap: 10px;
            margin-top: 25px;
            padding-top: 20px;
            border-top: 2px solid #ecf0f1;
        }
        
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        
        .btn-secondary:hover {
            background: #5a6268;
        }
        
        .close-btn {
            background: none;
            border: none;
            font-size: 24px;
            cursor: pointer;
            color: #aaa;
        }
        
        .close-btn:hover {
            color: #000;
        }
        
        @media (max-width: 768px) {
            .form-row {
                grid-template-columns: 1fr;
            }
            
            .config-content {
                margin: 10% auto;
                width: 95%;
                padding: 20px;
            }
        }
        
        .config-section {
            margin-bottom: 25px;
            padding: 20px;
            background: #f8f9fa;
            border-radius: 10px;
            border-left: 4px solid #667eea;
        }
        
        .config-section h4 {
            margin-bottom: 15px;
            color: #2c3e50;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        
        .checkbox-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 15px;
        }
        
        .checkbox-field {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px;
            background: white;
            border-radius: 5px;
            border: 1px solid #e9ecef;
        }
        
        .checkbox-field input[type="checkbox"] {
            width: 18px;
            height: 18px;
        }
        
        @media (max-width: 768px) {
            .checkbox-grid {
                grid-template-columns: 1fr;
            }
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

    <!-- Configuration Modal -->
    <div id="configModal" class="config-modal">
        <div class="config-content">
            <div class="config-header">
                <h3><i class="fas fa-cogs"></i> Расширенные настройки сайта</h3>
                <button class="close-btn" onclick="closeConfigModal()">&times;</button>
            </div>
            <form id="configForm" class="config-form">
                <input type="hidden" id="configSiteId" name="siteId">
                
                <!-- Basic Settings -->
                <div class="config-section">
                    <h4><i class="fas fa-cog"></i> Основные настройки</h4>
                    <div class="form-row">
                        <div class="form-field">
                            <label class="form-label">Интервал проверки (сек)</label>
                            <input type="number" class="form-control" id="checkInterval" name="checkInterval" min="10" max="3600">
                        </div>
                        <div class="form-field">
                            <label class="form-label">Таймаут (сек)</label>
                            <input type="number" class="form-control" id="timeout" name="timeout" min="5" max="300">
                        </div>
                    </div>
                    
                    <div class="form-row">
                        <div class="form-field">
                            <label class="form-label">Ожидаемый статус</label>
                            <select class="form-control" id="expectedStatus" name="expectedStatus">
                                <option value="200">200 OK</option>
                                <option value="301">301 Moved Permanently</option>
                                <option value="302">302 Found</option>
                                <option value="0">Любой успешный</option>
                            </select>
                        </div>
                        <div class="form-field">
                            <label class="form-label">Макс. редиректов</label>
                            <input type="number" class="form-control" id="maxRedirects" name="maxRedirects" min="0" max="20">
                        </div>
                    </div>
                </div>

                <!-- Metric Collection Settings -->
                <div class="config-section">
                    <h4><i class="fas fa-chart-line"></i> Сбор метрик</h4>
                    <div class="checkbox-grid">
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectDNSTime" name="collectDNSTime">
                            <label for="collectDNSTime">DNS время</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectConnectTime" name="collectConnectTime">
                            <label for="collectConnectTime">TCP соединение</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectTLSTime" name="collectTLSTime">
                            <label for="collectTLSTime">TLS рукопожатие</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectTTFB" name="collectTTFB">
                            <label for="collectTTFB">Time to First Byte</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectContentHash" name="collectContentHash">
                            <label for="collectContentHash">Хэш контента</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectRedirects" name="collectRedirects">
                            <label for="collectRedirects">Информация о редиректах</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectSSLDetails" name="collectSSLDetails">
                            <label for="collectSSLDetails">Детали SSL</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectServerInfo" name="collectServerInfo">
                            <label for="collectServerInfo">Информация о сервере</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="collectHeaders" name="collectHeaders">
                            <label for="collectHeaders">HTTP заголовки</label>
                        </div>
                    </div>
                </div>

                <!-- Display Settings -->
                <div class="config-section">
                    <h4><i class="fas fa-eye"></i> Отображение метрик</h4>
                    <div class="checkbox-grid">
                        <div class="checkbox-field">
                            <input type="checkbox" id="showResponseTime" name="showResponseTime">
                            <label for="showResponseTime">Время отклика</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showContentLength" name="showContentLength">
                            <label for="showContentLength">Размер контента</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showUptime" name="showUptime">
                            <label for="showUptime">Процент аптайма</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showSSLInfo" name="showSSLInfo">
                            <label for="showSSLInfo">SSL информация</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showServerInfo" name="showServerInfo">
                            <label for="showServerInfo">Информация о сервере</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showPerformance" name="showPerformance">
                            <label for="showPerformance">Детали производительности</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showRedirectInfo" name="showRedirectInfo">
                            <label for="showRedirectInfo">Информация о редиректах</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="showContentInfo" name="showContentInfo">
                            <label for="showContentInfo">Анализ контента</label>
                        </div>
                    </div>
                </div>

                <!-- Keywords and Advanced -->
                <div class="config-section">
                    <h4><i class="fas fa-search"></i> Расширенные настройки</h4>
                    <div class="form-field full-width">
                        <label class="form-label">User Agent</label>
                        <input type="text" class="form-control" id="userAgent" name="userAgent" placeholder="Site-Monitor/1.0">
                    </div>
                    
                    <div class="form-row">
                        <div class="form-field">
                            <label class="form-label">Ключевые слова (через запятую)</label>
                            <input type="text" class="form-control" id="checkKeywords" name="checkKeywords" placeholder="welcome, success">
                        </div>
                        <div class="form-field">
                            <label class="form-label">Исключить слова (через запятую)</label>
                            <input type="text" class="form-control" id="avoidKeywords" name="avoidKeywords" placeholder="error, 404">
                        </div>
                    </div>
                    
                    <div class="form-field">
                        <label class="form-label">SSL предупреждение за (дней)</label>
                        <input type="number" class="form-control" id="sslAlertDays" name="sslAlertDays" min="1" max="365">
                    </div>
                </div>

                <!-- Notification Settings -->
                <div class="config-section">
                    <h4><i class="fas fa-bell"></i> Уведомления</h4>
                    <div class="checkbox-grid">
                        <div class="checkbox-field">
                            <input type="checkbox" id="enabled" name="enabled">
                            <label for="enabled">Включить мониторинг</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="followRedirects" name="followRedirects">
                            <label for="followRedirects">Следовать редиректам</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="checkSSL" name="checkSSL">
                            <label for="checkSSL">Проверять SSL сертификат</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="notifyOnDown" name="notifyOnDown">
                            <label for="notifyOnDown">Уведомления при недоступности</label>
                        </div>
                        <div class="checkbox-field">
                            <input type="checkbox" id="notifyOnUp" name="notifyOnUp">
                            <label for="notifyOnUp">Уведомления при восстановлении</label>
                        </div>
                    </div>
                </div>
                
                <div class="config-actions">
                    <button type="button" class="btn btn-secondary" onclick="closeConfigModal()">Отмена</button>
                    <button type="submit" class="btn btn-primary">Сохранить настройки</button>
                </div>
            </form>
        </div>
    </div>

    <script>
        let statusChart = null;
        let eventSource = null;
        
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
            
            setTimeout(function() {
                notification.style.opacity = '1';
            }, 100);
            
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
            if (!dateString) return 'Не определено';
            
            try {
                const date = new Date(dateString);
                if (isNaN(date.getTime())) {
                    return 'Неверная дата';
                }
                
                return date.toLocaleString('ru-RU', {
                    year: 'numeric',
                    month: '2-digit',
                    day: '2-digit',
                    hour: '2-digit',
                    minute: '2-digit',
                    second: '2-digit'
                });
            } catch (error) {
                console.error('Error formatting date:', error);
                return 'Ошибка даты';
            }
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

        function triggerCheck() {
            document.getElementById('sitesList').innerHTML = '<div class="loading"><div class="spinner"></div>Проверка сайтов...</div>';
            document.getElementById('sitesUp').textContent = '-';
            document.getElementById('sitesDown').textContent = '-';
            document.getElementById('avgUptime').textContent = '-';
            document.getElementById('avgResponse').textContent = '-';
            
            fetch('/api/check', {
                method: 'POST'
            })
            .then(function(response) {
                return response.json();
            })
            .then(function(data) {
                if (data.error) {
                    showNotification('Ошибка: ' + data.error, 'error');
                } else {
                    setTimeout(function() {
                        loadSites();
                        loadDashboardStats();
                    }, 2000);
                }
            })
            .catch(function(error) {
                console.error('Ошибка запуска проверки:', error);
                showNotification('Ошибка запуска проверки', 'error');
                loadSites();
                loadDashboardStats();
            });
        }

        function deleteSite(url) {
            if (confirm('Вы уверены, что хотите удалить этот сайт из мониторинга?')) {
                fetch('/api/sites/delete', {
                    method: 'DELETE',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({ url: url })
                })
                .then(function(response) {
                    if (!response.ok) {
                        throw new Error('Network response was not ok');
                    }
                    return response.json();
                })
                .then(function(data) {
                    if (data.message) {
                        showNotification(data.message, 'success');
                    } else if (data.error) {
                        showNotification('Ошибка: ' + data.error, 'error');
                    }
                })
                .catch(function(error) {
                    console.error('Ошибка удаления сайта:', error);
                    showNotification('Ошибка удаления сайта', 'error');
                });
            }
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

        function openConfigModal(siteId) {
            document.getElementById('configSiteId').value = siteId;
            
            fetch('/api/sites/' + siteId + '/config')
                .then(response => response.json())
                .then(config => {
                    document.getElementById('checkInterval').value = config.check_interval || 30;
                    document.getElementById('timeout').value = config.timeout || 30;
                    document.getElementById('expectedStatus').value = config.expected_status || 200;
                    document.getElementById('maxRedirects').value = config.max_redirects || 10;
                    document.getElementById('userAgent').value = config.user_agent || 'Site-Monitor/1.0';
                    document.getElementById('checkKeywords').value = config.check_keywords || '';
                    document.getElementById('avoidKeywords').value = config.avoid_keywords || '';
                    document.getElementById('sslAlertDays').value = config.ssl_alert_days || 30;
                    
                    document.getElementById('collectDNSTime').checked = config.collect_dns_time === true;
                    document.getElementById('collectConnectTime').checked = config.collect_connect_time === true;
                    document.getElementById('collectTLSTime').checked = config.collect_tls_time === true;
                    document.getElementById('collectTTFB').checked = config.collect_ttfb === true;
                    document.getElementById('collectContentHash').checked = config.collect_content_hash === true;
                    document.getElementById('collectRedirects').checked = config.collect_redirects === true;
                    document.getElementById('collectSSLDetails').checked = config.collect_ssl_details !== false;
                    document.getElementById('collectServerInfo').checked = config.collect_server_info === true;
                    document.getElementById('collectHeaders').checked = config.collect_headers === true;
                    document.getElementById('showResponseTime').checked = config.show_response_time !== false;
                    document.getElementById('showContentLength').checked = config.show_content_length !== false;
                    document.getElementById('showUptime').checked = config.show_uptime !== false;
                    document.getElementById('showSSLInfo').checked = config.show_ssl_info !== false;
                    document.getElementById('showServerInfo').checked = config.show_server_info === true;
                    document.getElementById('showPerformance').checked = config.show_performance === true;
                    document.getElementById('showRedirectInfo').checked = config.show_redirect_info === true;
                    document.getElementById('showContentInfo').checked = config.show_content_info === true;
                    
                    document.getElementById('enabled').checked = config.enabled !== false;
                    document.getElementById('followRedirects').checked = config.follow_redirects !== false;
                    document.getElementById('checkSSL').checked = config.check_ssl !== false;
                    document.getElementById('notifyOnDown').checked = config.notify_on_down !== false;
                    document.getElementById('notifyOnUp').checked = config.notify_on_up !== false;
                    
                    document.getElementById('configModal').style.display = 'block';
                })
                .catch(error => {
                    console.error('Ошибка загрузки конфигурации:', error);
                    showNotification('Ошибка загрузки настроек', 'error');
                });
        }
        
        function closeConfigModal() {
            document.getElementById('configModal').style.display = 'none';
        }

        function generateSiteCard(site, index) {
            const config = site.config || {};
            const sslIndicator = site.url.startsWith('https://') && config.show_ssl_info !== false ? 
                (site.ssl_valid ? 
                    '<span class="ssl-indicator ssl-valid"><i class="fas fa-lock"></i> SSL OK</span>' : 
                    '<span class="ssl-indicator ssl-invalid"><i class="fas fa-lock-open"></i> SSL Ошибка</span>') 
                : '';
            
            let detailsHtml = '';
            
            if (config.show_response_time !== false && site.response_time_ms) {
                detailsHtml += '<div class="detail-item"><i class="fas fa-clock"></i> ' + formatTime(site.response_time_ms) + '</div>';
            }
            if (config.show_content_length !== false && site.content_length) {
                detailsHtml += '<div class="detail-item"><i class="fas fa-file-alt"></i> ' + formatBytes(site.content_length) + '</div>';
            }
            if (config.show_uptime !== false) {
                detailsHtml += '<div class="detail-item"><i class="fas fa-chart-line"></i> ' + (site.uptime_percent || 0).toFixed(1) + '% аптайм</div>';
            }
            
            detailsHtml += '<div class="detail-item"><i class="fas fa-code"></i> ' + (site.status_code || 'N/A') + '</div>';
            detailsHtml += '<div class="detail-item"><i class="fas fa-calendar"></i> ' + formatDate(site.last_checked) + '</div>';
            
            if (sslIndicator) {
                detailsHtml += '<div class="detail-item">' + sslIndicator + '</div>';
            }
            
            return '<div class="site-card ' + site.status + '">' +
                '<div class="site-header">' +
                    '<div class="site-url">' + site.url + '</div>' +
                    '<div class="site-status ' + site.status + '">' +
                        '<i class="fas fa-' + (site.status === 'up' ? 'check' : 'times') + '"></i>' +
                        site.status.toUpperCase() +
                    '</div>' +
                '</div>' +
                '<div class="site-details">' + detailsHtml + '</div>' +
                (site.last_error ? '<div style="color: #e74c3c; font-size: 0.9em; margin-bottom: 10px;"><i class="fas fa-exclamation-triangle"></i> ' + site.last_error + '</div>' : '') +
                '<button class="site-details-toggle" onclick="toggleDetails(' + index + ')">' +
                    '<i class="fas fa-info-circle"></i> Подробная информация' +
                '</button>' +
                '<div id="details-' + index + '" class="site-details-expanded">' +
                    generateDetailedInfo(site, config) +
                '</div>' +
                '<div class="site-actions">' +
                    '<button class="btn btn-primary" onclick="openConfigModal(' + site.id + ')">' +
                        '<i class="fas fa-cog"></i> Настроить' +
                    '</button>' +
                    '<button class="btn btn-danger" onclick="deleteSite(\'' + site.url + '\')">' +
                        '<i class="fas fa-trash"></i> Удалить' +
                    '</button>' +
                '</div>' +
            '</div>';
        }

        function generateDetailedInfo(site, config) {
            var detailsHtml = '';
            
            if (config.show_performance !== false) {
                detailsHtml += 
                    '<div class="details-section">' +
                        '<h4><i class="fas fa-stopwatch"></i> Время отклика</h4>' +
                        '<div class="details-grid">';
                        
                if (config.collect_dns_time !== false && site.dns_time !== undefined) {
                    detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">DNS Lookup</div>' +
                            '<div class="metric-value">' + formatTime(site.dns_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.dns_time) + '" style="width: ' + getPerformanceWidth(site.dns_time) + '"></div></div>' +
                        '</div>';
                }
                
                if (config.collect_connect_time !== false && site.connect_time !== undefined) {
                    detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">TCP Connect</div>' +
                            '<div class="metric-value">' + formatTime(site.connect_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.connect_time) + '" style="width: ' + getPerformanceWidth(site.connect_time) + '"></div></div>' +
                        '</div>';
                }
                
                if (site.url.startsWith('https://') && config.collect_tls_time !== false && site.tls_time !== undefined) {
                    detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">TLS Handshake</div>' +
                            '<div class="metric-value">' + formatTime(site.tls_time || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.tls_time) + '" style="width: ' + getPerformanceWidth(site.tls_time) + '"></div></div>' +
                        '</div>';
                }
                
                if (config.collect_ttfb !== false && site.ttfb !== undefined) {
                    detailsHtml += 
                        '<div class="detail-metric">' +
                            '<div class="metric-label">Time to First Byte</div>' +
                            '<div class="metric-value">' + formatTime(site.ttfb || 0) + '</div>' +
                            '<div class="performance-bar"><div class="performance-fill ' + getPerformanceClass(site.ttfb) + '" style="width: ' + getPerformanceWidth(site.ttfb) + '"></div></div>' +
                        '</div>';
                }
                
                detailsHtml += '</div></div>';
            }
            
            if (site.url.startsWith('https://') && config.show_ssl_info !== false && config.collect_ssl_details !== false) {
                detailsHtml += 
                '<div class="details-section">' +
                    '<h4><i class="fas fa-shield-alt"></i> SSL/TLS Сертификат</h4>' +
                    '<div class="details-grid">';
                    
                if (site.ssl_algorithm) {
                    detailsHtml += 
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Алгоритм</div>' +
                            '<div class="metric-value">' + site.ssl_algorithm + '</div>' +
                        '</div>';
                }
                
                if (site.ssl_key_length) {
                    detailsHtml += 
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Длина ключа</div>' +
                            '<div class="metric-value">' + site.ssl_key_length + ' бит</div>' +
                        '</div>';
                }
                
                if (site.ssl_issuer) {
                    detailsHtml += 
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Издатель</div>' +
                            '<div class="metric-value">' + site.ssl_issuer + '</div>' +
                        '</div>';
                }
                
                if (site.ssl_expiry) {
                    detailsHtml += 
                        '<div class="detail-metric ssl-details ' + (site.ssl_valid ? '' : 'invalid') + '">' +
                            '<div class="metric-label">Действителен до</div>' +
                            '<div class="metric-value">' + formatDate(site.ssl_expiry) + '</div>' +
                        '</div>';
                }
                
                detailsHtml += '</div></div>';
            }
            
            if (config.show_server_info !== false && config.collect_server_info !== false) {
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
                
                if (config.collect_headers !== false) {
                    if (site.content_type) {
                        detailsHtml += 
                            '<div class="detail-metric">' +
                                '<div class="metric-label">Content-Type</div>' +
                                '<div class="metric-value">' + site.content_type + '</div>' +
                            '</div>';
                    }
                    
                    if (site.cache_control) {
                        detailsHtml += 
                            '<div class="detail-metric">' +
                                '<div class="metric-label">Cache-Control</div>' +
                                '<div class="metric-value">' + site.cache_control + '</div>' +
                            '</div>';
                    }
                }
                
                detailsHtml += '</div></div>';
            }
            
            if (config.show_redirect_info !== false && config.collect_redirects !== false) {
                detailsHtml += 
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
                
                detailsHtml += '</div></div>';
            }
            
            if (config.show_content_info !== false) {
                detailsHtml += 
                    '<div class="details-section">' +
                        '<h4><i class="fas fa-file-code"></i> Анализ контента</h4>' +
                        '<div class="content-info">' +
                            'Размер контента: ' + formatBytes(site.content_length || 0) + '<br>';
                            
                if (config.collect_content_hash !== false && site.content_hash) {
                    detailsHtml += 'Хэш контента: ' + site.content_hash + ' (для отслеживания изменений)<br>';
                }
                
                detailsHtml += 'Последняя проверка: ' + formatDate(site.last_checked) + '</div></div>';
            }
            
            return detailsHtml;
        }

        function loadSites() {
            fetch('/api/sites')
                .then(response => response.json())
                .then(sites => {
                    const sitesList = document.getElementById('sitesList');
                    if (sites && sites.length > 0) {
                        sitesList.innerHTML = sites.map((site, index) => {
                            return generateSiteCard(site, index);
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
                } else if (data.error) {
                    showNotification('Ошибка: ' + data.error, 'error');
                }
            })
            .catch(function(error) {
                console.error('Ошибка добавления сайта:', error);
                showNotification('Ошибка добавления сайта', 'error');
            });
        });

        document.getElementById('configForm').addEventListener('submit', function(e) {
            e.preventDefault();
            const siteId = document.getElementById('configSiteId').value;
            
            const config = {
                check_interval: parseInt(document.getElementById('checkInterval').value),
                timeout: parseInt(document.getElementById('timeout').value),
                expected_status: parseInt(document.getElementById('expectedStatus').value),
                follow_redirects: document.getElementById('followRedirects').checked,
                max_redirects: parseInt(document.getElementById('maxRedirects').value),
                check_ssl: document.getElementById('checkSSL').checked,
                ssl_alert_days: parseInt(document.getElementById('sslAlertDays').value),
                check_keywords: document.getElementById('checkKeywords').value,
                avoid_keywords: document.getElementById('avoidKeywords').value,
                headers: {},
                user_agent: document.getElementById('userAgent').value,
                enabled: document.getElementById('enabled').checked,
                notify_on_down: document.getElementById('notifyOnDown').checked,
                notify_on_up: document.getElementById('notifyOnUp').checked,
                collect_dns_time: document.getElementById('collectDNSTime').checked,
                collect_connect_time: document.getElementById('collectConnectTime').checked,
                collect_tls_time: document.getElementById('collectTLSTime').checked,
                collect_ttfb: document.getElementById('collectTTFB').checked,
                collect_content_hash: document.getElementById('collectContentHash').checked,
                collect_redirects: document.getElementById('collectRedirects').checked,
                collect_ssl_details: document.getElementById('collectSSLDetails').checked,
                collect_server_info: document.getElementById('collectServerInfo').checked,
                collect_headers: document.getElementById('collectHeaders').checked,
                show_response_time: document.getElementById('showResponseTime').checked,
                show_content_length: document.getElementById('showContentLength').checked,
                show_uptime: document.getElementById('showUptime').checked,
                show_ssl_info: document.getElementById('showSSLInfo').checked,
                show_server_info: document.getElementById('showServerInfo').checked,
                show_performance: document.getElementById('showPerformance').checked,
                show_redirect_info: document.getElementById('showRedirectInfo').checked,
                show_content_info: document.getElementById('showContentInfo').checked
            };
            
            fetch('/api/sites/' + siteId + '/config', {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(config)
            })
            .then(response => response.json())
            .then(data => {
                if (data.status === 'ok') {
                    showNotification('Настройки сохранены успешно', 'success');
                    closeConfigModal();
                    loadSites();
                } else {
                    showNotification('Ошибка: ' + (data.error || 'Неизвестная ошибка'), 'error');
                }
            })
            .catch(error => {
                console.error('Ошибка сохранения настроек:', error);
                showNotification('Ошибка сохранения настроек', 'error');
            });
        });
        
        window.onclick = function(event) {
            const modal = document.getElementById('configModal');
            if (event.target === modal) {
                closeConfigModal();
            }
        }

        connectSSE();
        loadDashboardStats();
        loadSites();
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