# Реализация логики чатов и управления сессиями

## Обзор

Документация описывает реализацию логики работы с активными чатами и управления сессиями интервью для платформы TensorTalks. Архитектура включает систему управления сессиями с кэшированием, CRUD сервисы для чатов и результатов, а также пайплайн создания программы интервью.

## Архитектура

### Общая схема

```
Пользователь → Frontend → BFF → Session Manager (создание сессии)
                              ↓
                    Session CRUD (БД)
                              ↓
                    Redis (кэш активных сессий)
                              ↓
                    Kafka (interview.build.request)
                              ↓
                    Interview Builder Service
                              ↓
                    Kafka (interview.build.response)
                              ↓
                    Session Manager (сохранение программы)
                              ↓
                    Kafka (chat.events.out)
                              ↓
                    Mock Model Service
                              ↓
                    Chat CRUD (сохранение сообщений)
                    Results CRUD (сохранение результатов)
                              ↓
                    Kafka (chat.events.in)
                              ↓
                    BFF → Frontend → Пользователь
```

### Компоненты системы

#### 1. session-crud-service ✅
CRUD микросервис для сессий интервью в PostgreSQL.

**Эндпоинты:**
- `POST /sessions` — создание новой сессии
  - Request: `{ "user_id": "uuid", "params": { "topics": [...], "level": "...", "type": "..." } }`
  - Response: `{ "session": {...} }`
- `GET /sessions/:id` — получение сессии по ID
- `GET /sessions/user/:user_id` — получение всех сессий пользователя
- `PUT /sessions/:id/program` — обновление программы интервью
- `PUT /sessions/:id/close` — закрытие сессии (установка end_time)
- `DELETE /sessions/:id` — удаление сессии

**Таблица `sessions`:**
- session_id (UUID, PK)
- user_id (UUID, indexed)
- start_time, end_time (timestamps)
- params (JSONB) — topics, level, type
- interview_program (JSONB) — программа интервью

#### 2. session-service (session-manager) ✅
Микросервис управления сессиями с Redis кэшированием и интеграцией с interview builder.

**Реализованные возможности:**
- ✅ Redis кэш для активных сессий
- ✅ Интеграция с session-crud-service
- ✅ Kafka producer/consumer для работы с interview builder
- ✅ REST API:
  - `POST /sessions` — создание сессии с параметрами и проверкой лимита
  - `GET /sessions/:id/program` — получение программы (из Redis или CRUD)
  - `PUT /sessions/:id/close` — закрытие сессии
- ✅ Логика:
  - При создании: проверка лимита активных сессий в Redis → создание в CRUD → отправка запроса в Kafka → ожидание ответа (таймаут 30 сек) → сохранение программы в CRUD и Redis
  - При получении программы: сначала Redis, если нет — из CRUD (с кэшированием)
  - При закрытии: удаление из Redis, обновление в CRUD

#### 3. interview-builder-service ✅
Python FastAPI сервис для динамического создания программы интервью.

**Функциональность:**
- Слушает очередь `interview.build.request`
- Получает параметры интервью (topics, level, type)
- Запрашивает вопросы из questions-crud-service по фильтрам
- Запрашивает знания из knowledge-base-crud-service для каждого вопроса
- Собирает программу интервью (5 вопросов по умолчанию)
- Упорядочивает вопросы по логике (связанные вопросы рядом)
- Отправляет программу в очередь `interview.build.response`

**Формат события interview.build.request:**
```json
{
  "event_id": "uuid",
  "event_type": "interview.build.request",
  "timestamp": "ISO8601",
  "service": "session-manager-service",
  "version": "1.0.0",
  "payload": {
    "session_id": "uuid",
    "params": {
      "topics": ["ml", "nlp"],
      "level": "middle",
      "type": "interview"
    }
  }
}
```

**Формат события interview.build.response:**
```json
{
  "event_id": "uuid",
  "event_type": "interview.build.response",
  "timestamp": "ISO8601",
  "service": "interview-builder-service",
  "version": "1.0.0",
  "payload": {
    "session_id": "uuid",
    "program": {
      "questions": [
        {
          "question": "...",
          "theory": "...",
          "order": 1
        }
      ]
    }
  }
}
```

#### 4. chat-crud-service ✅
CRUD микросервис для чатов и сообщений в PostgreSQL.

**Эндпоинты:**
- `POST /messages` — сохранение нового сообщения
  - Request: `{ "session_id": "uuid", "type": "system"|"user", "content": "..." }`
- `GET /messages/:session_id` — получение всех сообщений сессии
- `GET /chat-dumps/:session_id` — получение дампа завершенного чата
- `POST /chat-dumps/:session_id` — создание дампа чата из сообщений

**Таблицы:**
- `messages` — id, session_id, type, content, created_at
- `chat_dumps` — id, session_id (unique), chat (JSONB), created_at, updated_at

#### 5. results-crud-service ✅
CRUD микросервис для результатов интервью в PostgreSQL.

**Эндпоинты:**
- `POST /results` — создание нового результата
  - Request: `{ "session_id": "uuid", "score": 85, "feedback": "..." }`
- `GET /results/:session_id` — получение результата по session_id
- `GET /results?session_ids=uuid1,uuid2,...` — получение результатов по списку session_id

**Таблица `results`:**
- id, session_id (unique), score, feedback, created_at, updated_at

#### 6. Kafka очереди

**Существующие топики:**

**chat.events.out** (BFF → Model):
- `chat.started` — начало нового чата
- `chat.resumed` — восстановление активной сессии чата
- `chat.user_message` — сообщение от пользователя
- `chat.terminated` — досрочное завершение чата пользователем

**chat.events.in** (Model → BFF):
- `chat.model_question` — вопрос от модели
- `chat.completed` — завершение чата с результатами

**Новые топики:**
- `interview.build.request` — запрос на создание программы интервью
- `interview.build.response` — ответ с программой интервью

#### 7. Redis
Кэширование активных сессий:
- Ключ: `session:{session_id}`
- Значение: JSON с программой интервью и метаданными
- TTL: настраиваемый (например, 24 часа)

#### 8. Mock Model Service ✅ (будущий marking-service)
**Реализованные возможности:**
- ✅ Убрана внутренняя логика вопросов (удалён model/questions.go)
- ✅ Добавлен HTTP клиент к session-manager для получения программы интервью
- ✅ Добавлен HTTP клиент к chat-crud для сохранения сообщений
- ✅ Добавлен HTTP клиент к results-crud для сохранения результатов
- ✅ Модифицирована логика:
  - При `chat.started`: запрос программы у session-manager, использование программы для вопросов (новая сессия)
  - При `chat.resumed`: запрос программы у session-manager, получение истории из chat-crud, восстановление состояния сессии
  - При `chat.user_message`: обработка сообщения пользователя и отправка следующего вопроса
  - При `chat.terminated`: досрочное завершение интервью пользователем
  - При каждом сообщении (system/user): сначала сохранение в chat-crud, потом отправка в Kafka
  - При завершении: сохранение финального сообщения, создание дампа чата, сохранение результатов, закрытие сессии в session-manager

#### 9. BFF Service ✅
**Реализованные возможности:**
- ✅ Обновлено создание сессии для передачи параметров интервью (topics, level, type)
- ✅ Добавлены клиенты к новым сервисам:
  - session-manager-client (обновлен для работы с параметрами)
  - session-crud-client (получение списка сессий по user_id)
  - chat-crud-client (получение сообщений и дампов чатов)
  - results-crud-client (получение результатов)
- ✅ Новые endpoints:
  - `GET /api/interviews?user_id=uuid` — список всех интервью пользователя
  - `GET /api/interviews/:session_id/chat` — получение истории чата
  - `GET /api/interviews/:session_id/result` — получение результата

## Пайплайн работы

### 1. Создание и подготовка сессии

```
Frontend → BFF → Session Manager (POST /sessions с параметрами)
                              ↓
                    Проверка лимита активных сессий в Redis
                              ↓
                    Session CRUD (создание записи)
                              ↓
                    Kafka Producer (interview.build.request)
                              ↓
                    Interview Builder Service (обработка)
                              ↓
                    Kafka Producer (interview.build.response)
                              ↓
                    Session Manager Consumer (получение программы)
                              ↓
                    Session CRUD (обновление программы)
                              ↓
                    Redis (сохранение в кэш)
                              ↓
                    BFF → Frontend (возврат session_id)
```

### 2. Старт интервью

```
Frontend → BFF → Kafka (chat.events.out: chat.started)
                              ↓
                    Mock Model Service (получение события)
                              ↓
                    Session Manager (GET /sessions/:id/program)
                              ↓ (из Redis или CRUD)
                    Mock Model Service (использование программы)
                              ↓
                    Chat CRUD (POST /messages - сохранение вопроса)
                              ↓
                    Kafka (chat.events.in: chat.model_question)
                              ↓
                    BFF Consumer → Frontend (отображение вопроса)
```

### 3. Отправка ответа пользователя

```
Frontend → BFF → Kafka (chat.events.out: chat.user_message)
                              ↓
                    Mock Model Service (обработка ответа)
                              ↓
                    Проверка программы интервью (следующий вопрос)
                              ↓
                    Chat CRUD (POST /messages - сохранение вопроса)
                              ↓
                    Kafka (chat.events.in: chat.model_question)
                              ↓
                    BFF Consumer → Frontend (отображение вопроса)
```

### 4. Завершение интервью

```
Mock Model Service (последний вопрос задан)
                              ↓
                    Chat CRUD (POST /messages - последнее сообщение)
                              ↓
                    Chat CRUD (POST /chat-dumps/:session_id - создание дампа)
                              ↓
                    Results CRUD (POST /results - сохранение результатов)
                              ↓
                    Session Manager (PUT /sessions/:id/close)
                              ↓
                    Kafka (chat.events.in: chat.completed)
                              ↓
                    BFF Consumer → Frontend (отображение результатов)
```

### 5. Просмотр истории интервью

```
Frontend → BFF (GET /api/interviews)
                              ↓
                    Session Manager → Session CRUD (GET /sessions/user/:user_id)
                              ↓
                    Session Manager → Results CRUD (GET /results?session_ids=...)
                              ↓
                    BFF → Frontend (список интервью)
                              ↓
Frontend → BFF (GET /api/interviews/:session_id/chat)
                              ↓
                    Chat CRUD (GET /messages/:session_id)
                              ↓
                    BFF → Frontend (история чата)
                              ↓
Frontend → BFF (GET /api/interviews/:session_id/result)
                              ↓
                    Results CRUD (GET /results/:session_id)
                              ↓
                    BFF → Frontend (результат)
```

## Статус реализации

### ✅ Завершено

1. **session-crud-service** — полностью реализован
2. **chat-crud-service** — полностью реализован
3. **results-crud-service** — полностью реализован
4. **interview-builder-service** — полностью реализован (Python FastAPI, Kafka producer/consumer, динамическое создание программы интервью)
5. **session-service (session-manager)** — полностью реализован (Redis кэш, Kafka интеграция, CRUD клиент)
6. **mock-model-service (будущий marking-service)** — полностью реализован (интеграция с новыми сервисами)
7. **bff-service** — полностью обновлен для работы с новыми API
8. **docker-compose.yml** — обновлен со всеми новыми сервисами и Redis
9. **Фронтенд** — обновлен для работы с реальными данными:
   - Обновлен `startChat` для передачи параметров интервью (topics, level, type)
   - Добавлена загрузка списка интервью из API
   - Обновлена страница Results для показа реальных данных и истории чата
   - Убраны моковые данные из Dashboard

## Метрики

Добавлены метрики для мониторинга:
- `tensortalks_kafka_messages_produced_total` — количество отправленных сообщений в Kafka
- `tensortalks_kafka_messages_consumed_total` — количество полученных сообщений из Kafka
- `tensortalks_business_sessions_created_total` — количество созданных сессий
- `tensortalks_business_chat_operations_total` — операции с чатами
- `tensortalks_business_result_operations_total` — операции с результатами

## Логирование

Все операции логируются:
- Создание сессий
- Сохранение сообщений и результатов
- Отправка/получение сообщений в Kafka
- Обработка событий
- Ошибки

## Следующие шаги (будущие улучшения)

1. **WebSocket интеграция** — для real-time обновлений вместо polling
2. **Расширение интервью-билдера** — улучшение логики выбора вопросов и генерации программ интервью
3. **Восстановление сессий** — возможность продолжить незавершенное интервью
4. **Дополнительные метрики** — расширенная аналитика и мониторинг
