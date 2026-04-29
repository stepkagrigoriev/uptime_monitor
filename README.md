# uptime_monitor
Небольшой пет-проект, распределенная система для мониторинга доступности сайтов, состоящая из 2ух сервисов: API для управления списком сайтов и Worker для фонового мониторинга

## Стек
* Golang, GORM, Gorilla Mux, Zap
* PostgreSQL, Docker/Docker Compose

## Структура
*   `cmd/api` - API для управления списком сайтов
*   `cmd/worker` - сервис, который проверяет доступность сайтов
*   `internal/models` - ORM модели
*   `internal/storage` - логика подключения к БД и миграции

## Запуск
1. Запустите базу данных:
   ```bash
   docker-compose up -d
   ```
2. Запустите API (в отдельном терминале):
   ```bash
   cd cmd/api
   go run main.go
   ```
3. Запустите Worker (в отдельном терминале):
   ```bash
   cd cmd/worker
   go run main.go
   ```


## API

### Добавить сайт
`POST /sites`
**Request Body:**
```json
{
  "url": "https://youtube.com"  не работает :(
}
```

**Response:**
*   `201 Created` — сайт успешно добавлен.


### Получить список сайтов
`GET /sites`

**Response:**
```json
[
  {
    "id": 1,
    "url": "https://youtube.com",
    "created_at": "2026-01-01T00:00:00Z"
  }
]
```



### Получить статистику пингов сайта
`GET /sites/{id}/stats`

**Response:**
```json
[
  {
    "id": 105,
    "website_id": 1,
    "status_code": 200,
    "response_time_ms": 145,
    "checked_at": "2026-01-01T00:00:00Z"
  }
]
```

## Как работает
1. API принимает запросы от пользователя и сохраняет URL в таблицу `websites` БД 
2. Worker раз в 30 секунд считывает список всех сайтов из базы
3. Для каждого сайта Worker параллельно (через горутины) выполняет HTTP GET запрос
4. Результат (статус-код и время ответа) сохраняется в таблицу `ping_results`

Ура!! Работает с кайфом