# PR Reviewer Assignment Service

Сервис для автоматического назначения ревьюеров на Pull Request'ы. Сервис управляет командами, пользователями и автоматически назначает ревьюеров согласно бизнес-правилам.


### Запуск через Docker Compose

Самый простой способ запустить сервис:

```bash
make up
```

Или напрямую:

```bash
docker compose up --build -d
```

Сервис будет доступен на `http://localhost:8080`.

### Остановка

```bash
make down
```

Или с очисткой данных:

```bash
make clean
```

## Структура проекта

```
reviewer/
├── internal/
│   ├── core/              
│   │   ├── errors.go      
│   │   ├── models.go      
│   │   ├── ports.go      
│   │   └── service.go     
│   ├── adapters/
│   │   ├── db/           
│   │   │   ├── storage.go
│   │   │   ├── team.go
│   │   │   ├── user.go
│   │   │   ├── pr.go
│   │   │   ├── mappers.go 
│   │   │   └── migrations/
│   │   └── rest/         
│   │       ├── http.go    
│   │       ├── dto.go     
│   │       ├── mappers.go 
│   │       ├── errors.go  
│   │       ├── middleware.go
│   │       └── http_test.go 
│   ├── config/            
│   └── closers/           
├── main.go               
└── configs/               
```

## Архитектура

Проект построен на принципах **Hexagonal Architecture (Ports & Adapters)** и **SOLID**.

### Принципы архитектуры

1. **Разделение на слои:**
   - **Core** (`internal/core/`) - бизнес-логика, не зависит от внешних библиотек
   - **Adapters** (`internal/adapters/`) - реализации портов (HTTP, БД)

2. **Dependency Inversion:**
   - Core определяет интерфейсы (порты) в `ports.go`
   - Adapters реализуют эти интерфейсы
   - Core не знает о деталях реализации БД или HTTP

3. **Чистые доменные модели:**
   - Модели в `core/models.go` не содержат db-тегов или JSON-тегов и служат для представления бизнес-сущностей без привязки к деталям реализации
   - Маппинг происходит в адаптерах через промежуточные структуры (`db/mappers.go`, `rest/mappers.go`) 

### Бизнес-правила

1. При создании PR автоматически назначаются до 2 активных ревьюверов из команды автора (исключая автора)
2. Переназначение заменяет одного ревьювера на случайного активного участника из команды заменяемого ревьювера
3. После `MERGED` менять список ревьюверов нельзя
4. Если доступных кандидатов меньше двух, назначается доступное количество (0/1)
5. Пользователь с `isActive = false` не назначается на ревью
6. Операция merge идемпотентна

## Middleware

Реализован middleware для логирования всех HTTP запросов (`internal/adapters/rest/middleware.go`).

Middleware логирует:
- HTTP метод
- Путь запроса
- Статус код ответа
- Время выполнения запроса
- IP адрес клиента

Пример лога:
```
INFO HTTP request method=POST path=/team/add status=201 duration=5.2ms remote_addr=127.0.0.1:12345
```

Middleware применяется ко всем эндпоинтам автоматически при запуске сервера.

## API

Сервис предоставляет следующие эндпоинты:

- `POST /team/add` - создание команды
- `GET /team/get?team_name=...` - получение команды
- `POST /users/setIsActive` - установка активности пользователя
- `POST /pullRequest/create` - создание PR
- `POST /pullRequest/merge` - merge PR
- `POST /pullRequest/reassign` - переназначение ревьювера
- `GET /users/getReview?user_id=...` - получение PR пользователя
- `GET /statistics` - статистика назначений

Полная спецификация API доступна в `.docs/openapi.yml`.

## Тестирование

### Интеграционные тесты

Реализованы интеграционные и E2E тесты в `internal/adapters/rest/http_test.go`:

1. **TestCreateTeam_Integration** - тест создания команды через HTTP
2. **TestCreatePR_Integration** - тест создания PR (сначала команда, потом PR)
3. **TestReassignReviewer_E2E** - E2E сценарий: команда → PR → переназначение (через service)
4. **TestMergePR_Integration** - тест merge PR через HTTP с проверкой идемпотентности
5. **TestReassignReviewer_Integration** - полный E2E через HTTP API: создание → переназначение
6. **TestGetStatistics_Integration** - тест эндпоинта статистики

### Запуск тестов

```bash
# Полный цикл (clean -> up -> tests -> clean)
make test

# Только интеграционные тесты (требует запущенную БД)
make test-integration
```

Тесты используют реальную PostgreSQL из docker-compose. Перед каждым тестом БД очищается для изоляции.

### Тестирование через Postman

Сервис был протестирован вручную через Postman для проверки всех эндпоинтов и бизнес-логики.

**Типичный сценарий тестирования:**

1. **Создание команды:**
   - `POST /team/add` - создание команды "backend" с 3-4 пользователями
   - Проверка: команда создана, пользователи добавлены

2. **Создание PR:**
   - `POST /pullRequest/create` - создание PR от пользователя u1
   - Проверка: автоматически назначены 2 ревьювера из команды (исключая автора)

3. **Проверка статистики:**
   - `GET /statistics` - проверка количества назначений
   - Проверка: статистика отражает назначенных ревьюверов

4. **Переназначение ревьювера:**
   - `POST /pullRequest/reassign` - замена одного ревьювера
   - Проверка: новый ревьювер из команды заменяемого, старый удален из списка

5. **Деактивация пользователя:**
   - `POST /users/setIsActive` - установка is_active = false
   - Проверка: пользователь не назначается на новые PR

6. **Merge PR:**
   - `POST /pullRequest/merge` - merge PR
   - Проверка: статус изменен на MERGED, повторный merge идемпотентен

7. **Попытка переназначения после merge:**
   - `POST /pullRequest/reassign` на MERGED PR
   - Проверка: возвращается ошибка PR_MERGED

**Примеры запросов:**

- **Создать команду:** `POST /team/add` с телом:
  ```json
  {
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Alice", "is_active": true},
      {"user_id": "u2", "username": "Bob", "is_active": true},
      {"user_id": "u3", "username": "Charlie", "is_active": true}
    ]
  }
  ```

- **Создать PR:** `POST /pullRequest/create` с телом:
  ```json
  {
    "pull_request_id": "pr-1",
    "pull_request_name": "Add feature",
    "author_id": "u1"
  }
  ```

- **Получить статистику:** `GET /statistics`

Все эндпоинты протестированы и работают согласно спецификации OpenAPI.

## Линтер

Проект использует `golangci-lint` для проверки качества кода.

### Конфигурация

Конфигурация находится в `.golangci.yml` в корне проекта. Используются базовые линтеры:

- **errcheck** - проверка обработки ошибок
- **govet** - статический анализ (go vet)
- **staticcheck** - статический анализ
- **gosimple** - упрощение кода
- **ineffassign** - неиспользуемые присваивания
- **unused** - неиспользуемый код

Конфигурация простая и понятная, использует только стандартные проверки Go.

### Запуск линтера

```bash
make lint
```

Или напрямую:

```bash
cd reviewer && golangci-lint run ./...
```

## Makefile

Доступные команды:

```bash
make help              # Показать справку по командам
make up                # Запустить сервисы через Docker Compose
make down              # Остановить сервисы
make clean             # Остановить сервисы и удалить volumes
make test              # Полный цикл тестирования
make test-integration  # Запустить интеграционные тесты
make lint              # Запустить линтер
make build             # Собрать приложение
make logs              # Показать логи всех сервисов
make app-logs          # Показать логи приложения
make postgres-logs     # Показать логи PostgreSQL
```

## Конфигурация

Конфигурация загружается из YAML файлов в `reviewer/configs/`:

- `local.yml` - для локальной разработки
- `prod.yml` - для production (используется в Docker)

Также поддерживается переопределение через переменные окружения (через `cleanenv`).

Основные параметры:
- `DB_ADDRESS` - адрес PostgreSQL
- `HTTP_ADDRESS` - адрес HTTP сервера (по умолчанию `:8080`)
- `LOG_LEVEL` - уровень логирования (DEBUG, INFO, ERROR)

## База данных

Используется PostgreSQL 15. Миграции применяются автоматически при запуске приложения.

Схема БД:
- `teams` - команды
- `users` - пользователи
- `pull_requests` - Pull Request'ы
- `pull_request_reviewers` - связь PR и ревьюверов (many-to-many)

Миграции находятся в `reviewer/internal/adapters/db/migrations/`.

## Дополнительные задания

**Эндпоинт статистики** - реализован `GET /statistics`, возвращает количество назначений по пользователям и общее количество.

**Интеграционное/E2E тестирование** - реализовано 6 тестов, покрывающих основные сценарии.

**Конфигурация линтера** - настроен `.golangci.yml` с описанием правил.
