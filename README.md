# 🌐 Site Monitor - Профессиональный Мониторинг Веб-Ресурсов

<div align="center">

![Site Monitor Demo](https://raw.githubusercontent.com/yourusername/site-monitor/main/assets/demo.gif)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

**Никогда больше не теряйте клиентов из-за недоступности сайта!** 🚀

*Детальная диагностика • Мгновенные уведомления • Real-time мониторинг*

</div>

---

## 🎯 Что это такое?

**Site Monitor** - это профессиональная система мониторинга, которая **24/7** отслеживает доступность и производительность ваших веб-ресурсов с детальной диагностикой каждого компонента. Получайте полную картину состояния ваших сайтов - от DNS до SSL сертификатов! 🔍

### 🌟 Ключевые возможности

#### 📊 **Детальная диагностика производительности**
- **⚡ DNS Lookup Time** - время разрешения доменного имени
- **🔌 TCP Connect Time** - время установления соединения  
- **🔒 TLS Handshake Time** - время SSL/TLS рукопожатия
- **📡 Time to First Byte (TTFB)** - время до получения первого байта
- **📈 Полное время отклика** - общее время выполнения запроса

#### 🛡️ **SSL/TLS мониторинг**
- **🔐 Валидация сертификатов** - проверка действительности
- **📅 Даты истечения** - предупреждения об истекающих сертификатах
- **🔑 Алгоритмы шифрования** - RSA, ECDSA и длина ключей
- **🏢 Информация об издателе** - кто выдал сертификат

#### 🌍 **Анализ HTTP ответов**
- **📋 HTTP статус коды** - 200, 301, 404, 500 и другие
- **🔄 Отслеживание редиректов** - количество и финальный URL
- **📄 Анализ заголовков** - Server, X-Powered-By, Cache-Control
- **🍪 Cookie анализ** - отслеживание установки cookies
- **📝 Контент-анализ** - размер, хэш для отслеживания изменений

#### 💻 **Современный интерфейс**
- **📱 Адаптивный дизайн** - работает на всех устройствах
- **⚡ Real-time обновления** - Server-Sent Events для мгновенных уведомлений
- **📊 Интерактивные графики** - визуализация статистики
- **🔍 Подробная информация** - развернутые метрики по клику

---

## 🚀 Быстрый старт

### Вариант 1: Docker (рекомендуется) 🐳

```bash
# Склонируйте репозиторий
git clone https://github.com/yourusername/site-monitor.git
cd site-monitor

# Запустите все сервисы
docker-compose up -d

# 🎉 Готово! Откройте http://localhost:8080
```

### Вариант 2: Ручная установка 🛠️

```bash
# 1. Убедитесь, что у вас есть Go 1.18+ и PostgreSQL
go version
psql --version

# 2. Склонируйте и подготовьте проект
git clone https://github.com/yourusername/site-monitor.git
cd site-monitor
go mod tidy

# 3. Настройте базу данных
createdb site_monitor

# 4. Создайте .env файл (или используйте переменные окружения)
cp .env.example .env
# Отредактируйте .env под ваши настройки

# 5. Запустите приложение
go run cmd/main.go

# 🎉 Готово! Откройте http://localhost:8080
```

---

## 🎮 Интерфейс и возможности

### 📱 Веб-дашборд

Откройте `http://localhost:8080` и получите доступ к профессиональному интерфейсу:

#### **Основная информация (всегда видна):**
- 🌐 **URL сайта** и текущий статус
- 📊 **HTTP код ответа** (200, 404, 500...)
- ⏱️ **Время отклика** в миллисекундах  
- 📄 **Размер контента** в байтах
- 📈 **Процент аптайма** за все время
- 📅 **Время последней проверки**
- 🔒 **SSL статус** для HTTPS сайтов

#### **Детальная диагностика (по клику):**

**🔍 Время отклика:**
```
DNS Lookup:     45мс    ████████░░ Отлично
TCP Connect:    120мс   ████████░░ Хорошо  
TLS Handshake:  89мс    ████████░░ Отлично
TTFB:          340мс   ██████░░░░ Хорошо
```

**🛡️ SSL/TLS Сертификат:**
- Алгоритм: RSA
- Длина ключа: 2048 бит
- Издатель: Let's Encrypt Authority X3
- Действителен до: 15.12.2024 23:59:59

**🌐 Информация о сервере:**
- Сервер: nginx/1.18.0
- Powered By: PHP/8.1
- Content-Type: text/html; charset=UTF-8
- Cache-Control: max-age=3600

**🔄 Редиректы и навигация:**
- Количество редиректов: 2
- Финальный URL: https://www.example.com/

**📊 Анализ контента:**
- Размер: 16.48 КБ
- Хэш: a1b2c3d4 (для отслеживания изменений)
- Последняя проверка: 20.09.2025, 11:42:26

### 🔌 REST API

Для разработчиков и интеграций:

#### Добавить сайт для мониторинга
```bash
curl -X POST http://localhost:8080/api/sites \
  -H "Content-Type: application/json" \
  -d '{"url":"https://your-awesome-site.com"}'
```

#### Получить детальную информацию о сайтах
```bash
curl http://localhost:8080/api/sites
```

Ответ содержит все метрики:
```json
{
  "id": 1,
  "url": "https://example.com",
  "status": "up",
  "status_code": 200,
  "response_time_ms": 1156,
  "dns_time": 45,
  "connect_time": 120,
  "tls_time": 89,
  "ttfb": 340,
  "ssl_valid": true,
  "ssl_algorithm": "RSA",
  "ssl_key_length": 2048,
  "ssl_issuer": "Let's Encrypt",
  "server_type": "nginx/1.18.0",
  "content_hash": "a1b2c3d4",
  "redirect_count": 0,
  "uptime_percent": 99.8
}
```

#### Статистика дашборда
```bash
curl http://localhost:8080/api/dashboard/stats
```

#### Принудительная проверка всех сайтов
```bash
curl -X POST http://localhost:8080/api/check
```

### 📡 Real-time обновления

Система использует **Server-Sent Events (SSE)** для мгновенных обновлений:
- Автоматическое обновление статусов при проверке
- Уведомления о добавлении/удалении сайтов
- Индикаторы запуска проверок

---

## ⚙️ Конфигурация

Создайте файл `.env` или установите переменные окружения:

```env
# 🗄️ База данных
DATABASE_URL=postgres://user:password@localhost:5432/site_monitor?sslmode=disable

# 🌐 Сервер
PORT=8080

# ⏱️ Интервал проверки (в секундах)
CHECK_INTERVAL=30

# 📧 Email уведомления (опционально)
SMTP_SERVER=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
NOTIFICATION_FROM=your-email@gmail.com
NOTIFICATION_TO=admin@yourcompany.com
```

---

## 🏗️ Архитектура проекта

```
site-monitor/
├── 🎯 cmd/main.go                    # Точка входа
├── ⚙️ internal/
│   ├── 🔧 config/                    # Конфигурация
│   ├── 💾 database/                  # Работа с БД и миграции
│   ├── 🌐 handlers/                  # HTTP обработчики и SSE
│   ├── 📊 models/                    # Модели данных с метриками
│   ├── 🔍 monitor/                   # Детальная диагностика
│   └── 📨 notifications/             # Email уведомления
├── 📝 pkg/logger/                    # Логирование
├── 🗄️ migrations/                    # SQL миграции
├── 🐳 docker-compose.yml             # Docker конфигурация
└── 📖 README.md                      # Эта документация
```

---

## 📈 Собираемые метрики

### 🔹 **Доступность и аптайм**
- HTTP коды ответа (200, 301, 404, 500, etc.)
- Процент успешных запросов
- Время недоступности

### 🔹 **Время отклика**  
- **DNS lookup time** - время разрешения DNS
- **TCP connect time** - время установления соединения
- **TLS handshake time** - время SSL рукопожатия (для HTTPS)
- **Time to first byte (TTFB)** - время до первого байта
- **Полное время ответа** - общее время запроса

### 🔹 **Содержимое ответа**
- Размер ответа в байтах
- SHA256 хэш контента (отслеживание изменений)
- Поиск ключевых слов (error, welcome, login, etc.)

### 🔹 **Поведение сайта**
- Количество редиректов и финальный URL
- HTTP заголовки (Server, X-Powered-By, Cache-Control, Content-Type)
- Анализ cookies

### 🔹 **HTTPS сертификаты**
- Дата истечения сертификата
- Алгоритм шифрования (RSA/ECDSA)
- Длина ключа
- Информация об издателе
- Валидность цепочки сертификатов

---

## 🤝 Участие в разработке

Мы рады вашему участию! Вот как можно помочь:

1. **🍴 Fork** репозиторий
2. **🌿 Создайте** ветку для новой функции (`git checkout -b feature/amazing-feature`)
3. **💾 Зафиксируйте** изменения (`git commit -m 'Add amazing feature'`)
4. **📤 Отправьте** в ветку (`git push origin feature/amazing-feature`)
5. **🔄 Создайте** Pull Request

### 🐛 Нашли баг?

Создайте [Issue](https://github.com/yourusername/site-monitor/issues) с подробным описанием:
- Что произошло
- Что вы ожидали
- Шаги для воспроизведения
- Версия ОС и Go

---

## 🎁 Планы на будущее

- [ ] 📊 **Исторические графики** производительности
- [ ] 🔔 **Slack/Discord/Telegram уведомления**
- [ ] 📱 **Мобильное приложение**
- [ ] 🤖 **Telegram бот** для управления
- [ ] 🌍 **Multi-region мониторинг** из разных точек мира
- [ ] 📈 **SLA отчеты** и экспорт данных
- [ ] 🔍 **Keyword мониторинг** с настраиваемыми правилами
- [ ] ⚡ **Webhooks** для интеграции с внешними системами
- [ ] 📧 **Smart алерты** с настраиваемыми условиями
- [ ] 🏢 **Multi-tenant** поддержка для сервис-провайдеров

---

## 📄 Лицензия

Этот проект лицензирован под **MIT License** - см. файл [LICENSE](LICENSE) для подробностей.

---

<div align="center">

**Создано с ❤️ для DevOps инженеров и сисадминов**

*Если этот проект помог вам - поставьте ⭐!*

[Сообщить о баге](https://github.com/yourusername/site-monitor/issues) • 
[Предложить функцию](https://github.com/yourusername/site-monitor/issues) • 
[Задать вопрос](https://github.com/yourusername/site-monitor/discussions)

### 🏆 Профессиональный мониторинг для профессиональных команд

*Site Monitor - когда каждая миллисекунда имеет значение*

</div>