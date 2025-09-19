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
    <title>Мониторинг сайтов</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            text-align: center;
        }
        h2 {
            color: #555;
            border-bottom: 2px solid #007bff;
            padding-bottom: 10px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            font-weight: bold;
        }
        input[type="url"] {
            width: 100%;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            box-sizing: border-box;
        }
        button {
            background-color: #007bff;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        button:hover {
            background-color: #0056b3;
        }
        .sites-list {
            margin-top: 30px;
        }
        .site-item {
            background-color: #f8f9fa;
            padding: 15px;
            margin-bottom: 10px;
            border-radius: 4px;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .site-info {
            flex: 1;
        }
        .site-url {
            font-weight: bold;
            margin-bottom: 5px;
        }
        .site-status {
            font-size: 12px;
        }
        .status-up {
            color: #28a745;
        }
        .status-down {
            color: #dc3545;
        }
        .status-unknown {
            color: #6c757d;
        }
        .delete-btn {
            background-color: #dc3545;
            font-size: 12px;
            padding: 5px 10px;
        }
        .delete-btn:hover {
            background-color: #c82333;
        }
        .api-section {
            margin-top: 40px;
            background-color: #f8f9fa;
            padding: 20px;
            border-radius: 4px;
        }
        pre {
            background-color: #e9ecef;
            padding: 15px;
            border-radius: 4px;
            overflow-x: auto;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🌐 Сервис мониторинга сайтов</h1>
        
        <h2>Добавить новый сайт</h2>
        <form id="addSiteForm">
            <div class="form-group">
                <label for="url">URL сайта:</label>
                <input type="url" id="url" name="url" placeholder="https://example.com" required>
            </div>
            <button type="submit">Добавить сайт</button>
        </form>

        <div class="sites-list">
            <h2>Отслеживаемые сайты</h2>
            <div id="sitesList">
                <!-- Сайты будут загружены через JavaScript -->
            </div>
        </div>

        <div class="api-section">
            <h2>API Эндпоинты</h2>
            <ul>
                <li><strong>POST /api/sites</strong> - добавить сайт для мониторинга</li>
                <li><strong>GET /api/sites</strong> - получить все отслеживаемые сайты</li>
                <li><strong>GET /api/sites/{url}/status</strong> - получить статус конкретного сайта</li>
                <li><strong>DELETE /api/sites/{url}</strong> - удалить сайт из мониторинга</li>
            </ul>
        </div>
    </div>

    <script>
        function loadSites() {
            fetch('/api/sites')
                .then(response => response.json())
                .then(sites => {
                    const sitesList = document.getElementById('sitesList');
                    if (sites && sites.length > 0) {
                        sitesList.innerHTML = sites.map(site => 
                            '<div class="site-item">' +
                                '<div class="site-info">' +
                                    '<div class="site-url">' + site.url + '</div>' +
                                    '<div class="site-status status-' + site.status + '">' +
                                        'Статус: ' + site.status + ' | Последняя проверка: ' + site.last_checked +
                                    '</div>' +
                                '</div>' +
                                '<button class="delete-btn" onclick="deleteSite(\'' + site.url + '\')">Удалить</button>' +
                            '</div>'
                        ).join('');
                    } else {
                        sitesList.innerHTML = '<p>Нет добавленных сайтов для мониторинга</p>';
                    }
                })
                .catch(error => {
                    console.error('Ошибка загрузки сайтов:', error);
                    document.getElementById('sitesList').innerHTML = '<p>Ошибка загрузки данных</p>';
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
            .then(response => response.json())
            .then(data => {
                if (data.message) {
                    alert(data.message);
                    document.getElementById('url').value = '';
                    loadSites();
                } else if (data.error) {
                    alert('Ошибка: ' + data.error);
                }
            })
            .catch(error => {
                console.error('Ошибка добавления сайта:', error);
                alert('Ошибка добавления сайта');
            });
        });

        function deleteSite(url) {
            if (confirm('Вы уверены, что хотите удалить этот сайт из мониторинга?')) {
                fetch('/api/sites/' + encodeURIComponent(url), {
                    method: 'DELETE'
                })
                .then(response => response.json())
                .then(data => {
                    if (data.message) {
                        alert(data.message);
                        loadSites();
                    } else if (data.error) {
                        alert('Ошибка: ' + data.error);
                    }
                })
                .catch(error => {
                    console.error('Ошибка удаления сайта:', error);
                    alert('Ошибка удаления сайта');
                });
            }
        }

        loadSites();
        setInterval(loadSites, 30000);
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