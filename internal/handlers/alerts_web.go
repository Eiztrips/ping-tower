package handlers

import (
	"html/template"
	"net/http"
)

const alertConfigTemplate = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Site Monitor - Конфигурация оповещений</title>
    <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css" rel="stylesheet">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
            min-height: 100vh;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .navigation {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(15px);
            border-radius: 15px;
            padding: 15px 30px;
            margin-bottom: 30px;
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 10px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .nav-brand {
            display: flex;
            align-items: center;
            gap: 10px;
            font-weight: bold;
            color: white;
            font-size: 1.1em;
        }

        .nav-links {
            display: flex;
            gap: 20px;
        }

        .nav-link {
            color: rgba(255, 255, 255, 0.8);
            text-decoration: none;
            padding: 8px 16px;
            border-radius: 8px;
            transition: all 0.3s ease;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .nav-link.active {
            background: rgba(255, 255, 255, 0.2);
            color: white;
        }

        .nav-link:hover {
            background: rgba(255, 255, 255, 0.15);
            color: white;
        }

        .header {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(15px);
            border-radius: 20px;
            padding: 30px;
            margin-bottom: 30px;
            text-align: center;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .header h1 {
            color: white;
            font-size: 2.5em;
            margin-bottom: 10px;
        }

        .header p {
            color: rgba(255, 255, 255, 0.8);
            font-size: 1.2em;
        }

        .alerts-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
            gap: 30px;
            margin-bottom: 30px;
        }

        .alert-card {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(15px);
            border-radius: 15px;
            padding: 25px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .alert-card h3 {
            color: white;
            margin-bottom: 20px;
            display: flex;
            align-items: center;
            gap: 10px;
        }

        .form-group {
            margin-bottom: 20px;
        }

        .form-label {
            display: block;
            color: rgba(255, 255, 255, 0.9);
            margin-bottom: 8px;
            font-weight: 500;
        }

        .form-input {
            width: 100%;
            padding: 12px 16px;
            border: 1px solid rgba(255, 255, 255, 0.2);
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.1);
            color: white;
            font-size: 14px;
            transition: all 0.3s ease;
        }

        .form-input::placeholder {
            color: rgba(255, 255, 255, 0.6);
        }

        .form-input:focus {
            outline: none;
            border-color: #3498db;
            background: rgba(255, 255, 255, 0.15);
        }

        .form-checkbox {
            display: flex;
            align-items: center;
            gap: 10px;
            margin-bottom: 15px;
        }

        .form-checkbox input[type="checkbox"] {
            width: 18px;
            height: 18px;
            accent-color: #3498db;
        }

        .form-checkbox label {
            color: rgba(255, 255, 255, 0.9);
            cursor: pointer;
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
            background: linear-gradient(45deg, #3498db, #2980b9);
            color: white;
        }

        .btn-primary:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(52, 152, 219, 0.4);
        }

        .btn-success {
            background: linear-gradient(45deg, #27ae60, #2ecc71);
            color: white;
            margin-right: 10px;
        }

        .btn-success:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(39, 174, 96, 0.4);
        }

        .btn-warning {
            background: linear-gradient(45deg, #f39c12, #e67e22);
            color: white;
            margin-right: 10px;
        }

        .btn-warning:hover {
            transform: translateY(-2px);
            box-shadow: 0 5px 15px rgba(243, 156, 18, 0.4);
        }

        .status-indicator {
            display: inline-block;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            margin-right: 8px;
        }

        .status-enabled {
            background-color: #27ae60;
        }

        .status-disabled {
            background-color: #e74c3c;
        }

        .config-list {
            background: rgba(255, 255, 255, 0.1);
            backdrop-filter: blur(15px);
            border-radius: 15px;
            padding: 25px;
            box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
            border: 1px solid rgba(255, 255, 255, 0.1);
            margin-bottom: 30px;
        }

        .config-item {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }

        .config-item:last-child {
            border-bottom: none;
        }

        .config-info h4 {
            color: white;
            margin-bottom: 5px;
        }

        .config-info p {
            color: rgba(255, 255, 255, 0.7);
            font-size: 14px;
        }

        .config-actions {
            display: flex;
            gap: 10px;
        }

        .modal {
            display: none;
            position: fixed;
            z-index: 1000;
            left: 0;
            top: 0;
            width: 100%;
            height: 100%;
            background-color: rgba(0, 0, 0, 0.5);
        }

        .modal-content {
            background: linear-gradient(135deg, #2c3e50 0%, #34495e 100%);
            margin: 5% auto;
            padding: 30px;
            border-radius: 15px;
            width: 90%;
            max-width: 600px;
            max-height: 80vh;
            overflow-y: auto;
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .close {
            color: rgba(255, 255, 255, 0.8);
            float: right;
            font-size: 28px;
            font-weight: bold;
            cursor: pointer;
        }

        .close:hover {
            color: white;
        }

        .tabs {
            display: flex;
            margin-bottom: 20px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.2);
        }

        .tab {
            padding: 10px 20px;
            background: none;
            border: none;
            color: rgba(255, 255, 255, 0.7);
            cursor: pointer;
            border-bottom: 2px solid transparent;
            transition: all 0.3s ease;
        }

        .tab.active {
            color: white;
            border-bottom-color: #3498db;
        }

        .tab-content {
            display: none;
        }

        .tab-content.active {
            display: block;
        }

        .textarea {
            width: 100%;
            min-height: 100px;
            padding: 12px 16px;
            border: 1px solid rgba(255, 255, 255, 0.2);
            border-radius: 8px;
            background: rgba(255, 255, 255, 0.1);
            color: white;
            font-size: 14px;
            resize: vertical;
            font-family: inherit;
        }

        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
        }

        @media (max-width: 768px) {
            .form-row {
                grid-template-columns: 1fr;
            }

            .nav-links {
                flex-direction: column;
                gap: 10px;
            }

            .alerts-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Навигационная панель -->
        <nav class="navigation">
            <div class="nav-brand">
                <i class="fas fa-globe"></i>
                Site Monitor
            </div>
            <div class="nav-links">
                <a href="/" class="nav-link">
                    <i class="fas fa-tachometer-alt"></i> Дашборд
                </a>
                <a href="/demo" class="nav-link">
                    <i class="fas fa-rocket"></i> Live Demo
                </a>
                <a href="/metrics" class="nav-link">
                    <i class="fas fa-chart-line"></i> Метрики
                </a>
                <a href="/alerts" class="nav-link active">
                    <i class="fas fa-bell"></i> Оповещения
                </a>
                <a href="/api/sites" class="nav-link">
                    <i class="fas fa-code"></i> API
                </a>
            </div>
        </nav>

        <!-- Заголовок -->
        <div class="header">
            <h1><i class="fas fa-bell"></i> Конфигурация оповещений</h1>
            <p>Настройка системы уведомлений: Email, Webhook, Telegram</p>
        </div>

        <!-- Список существующих конфигураций -->
        <div class="config-list">
            <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px;">
                <h3 style="color: white; margin: 0;">
                    <i class="fas fa-cog"></i> Конфигурации оповещений
                </h3>
                <button class="btn btn-primary" onclick="openCreateModal()">
                    <i class="fas fa-plus"></i> Добавить конфигурацию
                </button>
            </div>

            <div id="config-items">
                <!-- Конфигурации будут загружены через JavaScript -->
            </div>
        </div>

        <!-- Модальное окно для создания/редактирования конфигурации -->
        <div id="configModal" class="modal">
            <div class="modal-content">
                <span class="close" onclick="closeModal()">&times;</span>
                <h2 style="color: white; margin-bottom: 20px;" id="modal-title">
                    <i class="fas fa-plus"></i> Новая конфигурация оповещений
                </h2>

                <form id="alertConfigForm">
                    <!-- Основные настройки -->
                    <div class="form-group">
                        <label class="form-label">Название конфигурации</label>
                        <input type="text" class="form-input" id="configName" placeholder="Например: production-alerts" required>
                    </div>

                    <div class="form-checkbox">
                        <input type="checkbox" id="configEnabled" checked>
                        <label for="configEnabled">Включить конфигурацию</label>
                    </div>

                    <!-- Вкладки для типов оповещений -->
                    <div class="tabs">
                        <button type="button" class="tab active" data-tab="email">
                            <i class="fas fa-envelope"></i> Email
                        </button>
                        <button type="button" class="tab" data-tab="webhook">
                            <i class="fas fa-link"></i> Webhook
                        </button>
                        <button type="button" class="tab" data-tab="telegram">
                            <i class="fas fa-paper-plane"></i> Telegram
                        </button>
                        <button type="button" class="tab" data-tab="conditions">
                            <i class="fas fa-filter"></i> Условия
                        </button>
                    </div>

                    <!-- Email настройки -->
                    <div id="email-tab" class="tab-content active">
                        <div class="form-checkbox">
                            <input type="checkbox" id="emailEnabled">
                            <label for="emailEnabled">Включить Email оповещения</label>
                        </div>

                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">SMTP сервер</label>
                                <input type="text" class="form-input" id="smtpServer" placeholder="smtp.gmail.com">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Порт</label>
                                <input type="text" class="form-input" id="smtpPort" placeholder="587" value="587">
                            </div>
                        </div>

                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Имя пользователя</label>
                                <input type="text" class="form-input" id="smtpUsername" placeholder="your-email@gmail.com">
                            </div>
                            <div class="form-group">
                                <label class="form-label">Пароль</label>
                                <input type="password" class="form-input" id="smtpPassword" placeholder="App Password">
                            </div>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Отправитель</label>
                            <input type="email" class="form-input" id="emailFrom" placeholder="alerts@yourcompany.com">
                        </div>

                        <div class="form-group">
                            <label class="form-label">Получатели (через запятую)</label>
                            <textarea class="textarea" id="emailTo" placeholder="admin1@company.com, admin2@company.com"></textarea>
                        </div>
                    </div>

                    <!-- Webhook настройки -->
                    <div id="webhook-tab" class="tab-content">
                        <div class="form-checkbox">
                            <input type="checkbox" id="webhookEnabled">
                            <label for="webhookEnabled">Включить Webhook оповещения</label>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Webhook URL</label>
                            <input type="url" class="form-input" id="webhookUrl" placeholder="https://your-webhook-endpoint.com/alerts">
                        </div>

                        <div class="form-row">
                            <div class="form-group">
                                <label class="form-label">Таймаут (сек)</label>
                                <input type="number" class="form-input" id="webhookTimeout" placeholder="10" value="10" min="1" max="60">
                            </div>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Дополнительные заголовки (key:value, по одному на строку)</label>
                            <textarea class="textarea" id="webhookHeaders" placeholder="Authorization:Bearer your-token&#10;Content-Type:application/json"></textarea>
                        </div>
                    </div>

                    <!-- Telegram настройки -->
                    <div id="telegram-tab" class="tab-content">
                        <div class="form-checkbox">
                            <input type="checkbox" id="telegramEnabled">
                            <label for="telegramEnabled">Включить Telegram оповещения</label>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Bot Token</label>
                            <input type="text" class="form-input" id="telegramBotToken" placeholder="1234567890:ABCdefGHIjklMNOpqrsTUVwxyz">
                        </div>

                        <div class="form-group">
                            <label class="form-label">Chat ID</label>
                            <input type="text" class="form-input" id="telegramChatId" placeholder="-1001234567890">
                        </div>

                        <div style="background: rgba(255, 255, 255, 0.1); padding: 15px; border-radius: 8px; margin-top: 15px;">
                            <p style="color: rgba(255, 255, 255, 0.8); font-size: 14px; margin: 0;">
                                <i class="fas fa-info-circle"></i>
                                Для получения Bot Token создайте бота через @BotFather в Telegram.
                                Chat ID можно получить через @userinfobot или добавив бота в группу.
                            </p>
                        </div>
                    </div>

                    <!-- Условия оповещений -->
                    <div id="conditions-tab" class="tab-content">
                        <h4 style="color: white; margin-bottom: 15px;">Когда отправлять оповещения:</h4>

                        <div class="form-checkbox">
                            <input type="checkbox" id="alertOnDown" checked>
                            <label for="alertOnDown">При падении сайта</label>
                        </div>

                        <div class="form-checkbox">
                            <input type="checkbox" id="alertOnUp">
                            <label for="alertOnUp">При восстановлении сайта</label>
                        </div>

                        <div class="form-checkbox">
                            <input type="checkbox" id="alertOnSslExpiry" checked>
                            <label for="alertOnSslExpiry">При истечении SSL сертификата</label>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Дней до истечения SSL</label>
                            <input type="number" class="form-input" id="sslExpiryDays" value="30" min="1" max="365">
                        </div>

                        <div class="form-checkbox">
                            <input type="checkbox" id="alertOnResponseTime">
                            <label for="alertOnResponseTime">При превышении времени отклика</label>
                        </div>

                        <div class="form-group">
                            <label class="form-label">Максимальное время отклика (мс)</label>
                            <input type="number" class="form-input" id="responseTimeThreshold" value="5000" min="100">
                        </div>
                    </div>

                    <div style="margin-top: 30px; display: flex; gap: 10px;">
                        <button type="submit" class="btn btn-success">
                            <i class="fas fa-save"></i> Сохранить
                        </button>
                        <button type="button" class="btn btn-warning" onclick="testAlert()" id="testBtn" style="display: none;">
                            <i class="fas fa-paper-plane"></i> Тест
                        </button>
                        <button type="button" class="btn btn-primary" onclick="closeModal()">
                            Отмена
                        </button>
                    </div>
                </form>
            </div>
        </div>
    </div>

    <script>
        let currentConfigName = null;
        let configs = [];

        // Переключение вкладок
        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', () => {
                const tabName = tab.dataset.tab;

                // Убрать активный класс со всех вкладок и контента
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));

                // Добавить активный класс к выбранной вкладке и контенту
                tab.classList.add('active');
                document.getElementById(tabName + '-tab').classList.add('active');
            });
        });

        // Загрузка конфигураций при загрузке страницы
        window.addEventListener('load', loadConfigs);

        function loadConfigs() {
            fetch('/api/alerts/configs')
                .then(response => response.json())
                .then(data => {
                    configs = data || [];
                    renderConfigs();
                })
                .catch(error => {
                    console.error('Error loading configs:', error);
                    document.getElementById('config-items').innerHTML =
                        '<p style="color: rgba(255, 255, 255, 0.7);">Ошибка загрузки конфигураций</p>';
                });
        }

        function renderConfigs() {
            const container = document.getElementById('config-items');

            if (configs.length === 0) {
                container.innerHTML = '<p style="color: rgba(255, 255, 255, 0.7);">Нет настроенных конфигураций</p>';
                return;
            }

            container.innerHTML = configs.map(config => {
                const enabledChannels = [];
                if (config.email_enabled) enabledChannels.push('Email');
                if (config.webhook_enabled) enabledChannels.push('Webhook');
                if (config.telegram_enabled) enabledChannels.push('Telegram');

                const channelsText = enabledChannels.length > 0 ? enabledChannels.join(', ') : 'Нет активных каналов';
                const statusClass = config.enabled ? 'status-enabled' : 'status-disabled';
                const deleteButton = config.name !== 'global' ?
                    "<button class='btn btn-danger' onclick='deleteConfig(\"" + config.name + "\")'>" +
                        "<i class='fas fa-trash'></i> Удалить" +
                    "</button>" : '';

                return "<div class='config-item'>" +
                    "<div class='config-info'>" +
                        "<h4>" +
                            "<span class='status-indicator " + statusClass + "'></span>" +
                            config.name +
                        "</h4>" +
                        "<p>" + channelsText + "</p>" +
                    "</div>" +
                    "<div class='config-actions'>" +
                        "<button class='btn btn-primary' onclick='editConfig(\"" + config.name + "\")'>" +
                            "<i class='fas fa-edit'></i> Редактировать" +
                        "</button>" +
                        deleteButton +
                    "</div>" +
                "</div>";
            }).join('');
        }

        function openCreateModal() {
            currentConfigName = null;
            document.getElementById('modal-title').innerHTML = '<i class="fas fa-plus"></i> Новая конфигурация оповещений';
            document.getElementById('alertConfigForm').reset();
            document.getElementById('configEnabled').checked = true;
            document.getElementById('configName').disabled = false;
            document.getElementById('testBtn').style.display = 'none';
            document.getElementById('configModal').style.display = 'block';
        }

        function editConfig(name) {
            const config = configs.find(c => c.name === name);
            if (!config) return;

            currentConfigName = name;
            document.getElementById('modal-title').innerHTML = '<i class="fas fa-edit"></i> Редактирование: ' + name;

            // Заполнить форму данными конфигурации
            document.getElementById('configName').value = config.name;
            document.getElementById('configName').disabled = true;
            document.getElementById('configEnabled').checked = config.enabled;

            // Email настройки
            document.getElementById('emailEnabled').checked = config.email_enabled;
            document.getElementById('smtpServer').value = config.smtp_server || '';
            document.getElementById('smtpPort').value = config.smtp_port || '587';
            document.getElementById('smtpUsername').value = config.smtp_username || '';
            document.getElementById('smtpPassword').value = config.smtp_password || '';
            document.getElementById('emailFrom').value = config.email_from || '';
            document.getElementById('emailTo').value = config.email_to || '';

            // Webhook настройки
            document.getElementById('webhookEnabled').checked = config.webhook_enabled;
            document.getElementById('webhookUrl').value = config.webhook_url || '';
            document.getElementById('webhookTimeout').value = config.webhook_timeout || 10;

            // Преобразовать headers обратно в текст
            const headersText = Object.entries(config.webhook_headers || {})
                .map(([key, value]) => key + ':' + value)
                .join('\n');
            document.getElementById('webhookHeaders').value = headersText;

            // Telegram настройки
            document.getElementById('telegramEnabled').checked = config.telegram_enabled;
            document.getElementById('telegramBotToken').value = config.telegram_bot_token || '';
            document.getElementById('telegramChatId').value = config.telegram_chat_id || '';

            // Условия
            document.getElementById('alertOnDown').checked = config.alert_on_down;
            document.getElementById('alertOnUp').checked = config.alert_on_up;
            document.getElementById('alertOnSslExpiry').checked = config.alert_on_ssl_expiry;
            document.getElementById('sslExpiryDays').value = config.ssl_expiry_days || 30;
            document.getElementById('alertOnResponseTime').checked = config.alert_on_response_time_threshold;
            document.getElementById('responseTimeThreshold').value = config.response_time_threshold || 5000;

            document.getElementById('testBtn').style.display = 'inline-flex';
            document.getElementById('configModal').style.display = 'block';
        }

        function closeModal() {
            document.getElementById('configModal').style.display = 'none';
            currentConfigName = null;
        }

        function deleteConfig(name) {
            if (!confirm('Вы уверены, что хотите удалить конфигурацию "' + name + '"?')) {
                return;
            }

            fetch('/api/alerts/configs/' + encodeURIComponent(name), {
                method: 'DELETE'
            })
            .then(response => {
                if (response.ok) {
                    loadConfigs();
                } else {
                    throw new Error('Ошибка удаления');
                }
            })
            .catch(error => {
                console.error('Error deleting config:', error);
                alert('Ошибка удаления конфигурации');
            });
        }

        function testAlert() {
            if (!currentConfigName) return;

            const testData = {
                config_name: currentConfigName,
                test_url: 'https://example.com'
            };

            fetch('/api/alerts/test', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(testData)
            })
            .then(response => {
                if (response.ok) {
                    alert('Тестовое оповещение отправлено!');
                } else {
                    throw new Error('Ошибка отправки тестового оповещения');
                }
            })
            .catch(error => {
                console.error('Error sending test alert:', error);
                alert('Ошибка отправки тестового оповещения');
            });
        }

        // Обработка формы
        document.getElementById('alertConfigForm').addEventListener('submit', function(e) {
            e.preventDefault();

            // Парсинг webhook headers
            const headersText = document.getElementById('webhookHeaders').value;
            const webhookHeaders = {};
            if (headersText.trim()) {
                headersText.split('\n').forEach(line => {
                    const [key, ...valueParts] = line.split(':');
                    if (key && valueParts.length > 0) {
                        webhookHeaders[key.trim()] = valueParts.join(':').trim();
                    }
                });
            }

            const configData = {
                name: document.getElementById('configName').value,
                enabled: document.getElementById('configEnabled').checked,
                email_enabled: document.getElementById('emailEnabled').checked,
                webhook_enabled: document.getElementById('webhookEnabled').checked,
                telegram_enabled: document.getElementById('telegramEnabled').checked,

                // Email settings
                smtp_server: document.getElementById('smtpServer').value,
                smtp_port: document.getElementById('smtpPort').value,
                smtp_username: document.getElementById('smtpUsername').value,
                smtp_password: document.getElementById('smtpPassword').value,
                email_from: document.getElementById('emailFrom').value,
                email_to: document.getElementById('emailTo').value,

                // Webhook settings
                webhook_url: document.getElementById('webhookUrl').value,
                webhook_headers: webhookHeaders,
                webhook_timeout: parseInt(document.getElementById('webhookTimeout').value) || 10,

                // Telegram settings
                telegram_bot_token: document.getElementById('telegramBotToken').value,
                telegram_chat_id: document.getElementById('telegramChatId').value,

                // Alert conditions
                alert_on_down: document.getElementById('alertOnDown').checked,
                alert_on_up: document.getElementById('alertOnUp').checked,
                alert_on_ssl_expiry: document.getElementById('alertOnSslExpiry').checked,
                ssl_expiry_days: parseInt(document.getElementById('sslExpiryDays').value) || 30,
                alert_on_response_time_threshold: document.getElementById('alertOnResponseTime').checked,
                response_time_threshold: parseInt(document.getElementById('responseTimeThreshold').value) || 5000
            };

            const url = currentConfigName ?
                '/api/alerts/configs/' + encodeURIComponent(currentConfigName) :
                '/api/alerts/configs';
            const method = currentConfigName ? 'PUT' : 'POST';

            fetch(url, {
                method: method,
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(configData)
            })
            .then(response => {
                if (response.ok) {
                    closeModal();
                    loadConfigs();
                } else {
                    throw new Error('Ошибка сохранения');
                }
            })
            .catch(error => {
                console.error('Error saving config:', error);
                alert('Ошибка сохранения конфигурации');
            });
        });

        // Закрытие модального окна при клике вне его
        window.onclick = function(event) {
            const modal = document.getElementById('configModal');
            if (event.target === modal) {
                closeModal();
            }
        }
    </script>
</body>
</html>`

func AlertsWebHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl, err := template.New("alerts").Parse(alertConfigTemplate)
		if err != nil {
			http.Error(w, "Error parsing template", http.StatusInternalServerError)
			return
		}

		tmpl.Execute(w, nil)
	}
}