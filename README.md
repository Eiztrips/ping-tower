# 🌐 Site Monitor - Ваш Надежный Страж Интернета

<div align="center">

![Site Monitor Demo](https://raw.githubusercontent.com/yourusername/site-monitor/main/assets/demo.gif)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Postgres](https://img.shields.io/badge/postgres-%23316192.svg?style=for-the-badge&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

**Никогда больше не теряйте клиентов из-за недоступности сайта!** 🚀

*Круглосуточный мониторинг • Мгновенные уведомления • Простая настройка*

</div>

---

## 🎯 Что это такое?

**Site Monitor** - это ваш личный помощник, который **24/7** следит за доступностью ваших сайтов и сразу же сообщает, если что-то пошло не так. Представьте, что у вас есть верный пес, который лает каждый раз, когда к дому подходит незваный гость - только вместо гостей он следит за падением сайтов! 🐕‍🦺

### 🌟 Почему это круто?

- **⚡ Мгновенные уведомления** - узнавайте о проблемах раньше ваших клиентов
- **📊 Красивый веб-интерфейс** - управляйте мониторингом прямо из браузера  
- **🔄 Автоматическая проверка** - настройте и забудьте
- **💾 История мониторинга** - отслеживайте статистику доступности
- **🐳 Docker-ready** - разворачивается одной командой

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

## 🎮 Как пользоваться

### 📱 Веб-интерфейс

Откройте `http://localhost:8080` и наслаждайтесь интуитивным интерфейсом:

- **➕ Добавление сайтов** - просто введите URL и нажмите "Добавить"
- **📋 Список сайтов** - все ваши сайты с актуальными статусами
- **🗑️ Управление** - удаляйте ненужные сайты одним кликом
- **🔄 Автообновление** - статусы обновляются каждые 30 секунд

### 🔌 REST API

Для разработчиков и интеграций:

#### Добавить сайт для мониторинга
```bash
curl -X POST http://localhost:8080/api/sites \
  -H "Content-Type: application/json" \
  -d '{"url":"https://your-awesome-site.com"}'
```

#### Получить все отслеживаемые сайты
```bash
curl http://localhost:8080/api/sites
```

#### Проверить статус конкретного сайта
```bash
curl http://localhost:8080/api/sites/https%3A%2F%2Fgoogle.com/status
```

#### Удалить сайт из мониторинга  
```bash
curl -X DELETE http://localhost:8080/api/sites/https%3A%2F%2Fgoogle.com
```

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
│   ├── 💾 database/                  # Работа с БД
│   │   └── migrations/               # Миграции
│   ├── 🌐 handlers/                  # HTTP обработчики
│   ├── 📊 models/                    # Модели данных  
│   ├── 🔍 monitor/                   # Логика мониторинга
│   └── 📨 notifications/             # Уведомления
├── 📝 pkg/logger/                    # Логирование
├── 🐳 docker-compose.yml             # Docker конфигурация
└── 📖 README.md                      # Эта документация
```

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

- [ ] 📊 **Дашборд с графиками** uptime статистики
- [ ] 🔔 **Slack/Discord уведомления**
- [ ] 📱 **Мобильное приложение**
- [ ] 🤖 **Telegram бот** для управления
- [ ] 📈 **Метрики производительности** (время отклика)
- [ ] 🌍 **Multi-region мониторинг**

---

## 📄 Лицензия

Этот проект лицензирован под **MIT License** - см. файл [LICENSE](LICENSE) для подробностей.

---

<div align="center">

**Сделано с ❤️ для разработчиков и сисадминов**

*Если этот проект помог вам - поставьте ⭐!*

[Сообщить о баге](https://github.com/yourusername/site-monitor/issues) • 
[Предложить функцию](https://github.com/yourusername/site-monitor/issues) • 
[Задать вопрос](https://github.com/yourusername/site-monitor/discussions)

</div>