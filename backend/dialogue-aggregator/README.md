# Dialogue Aggregator Service

Центральный сервис системы TensorTalks для агрегации диалогов, управления их состоянием и обогащения событий метаданными.

## Назначение

Dialogue Aggregator отвечает за:
- **Агрегацию диалогов** из событий Kafka
- **Управление состоянием** диалогов в Redis (KV-хранилище)
- **Обогащение сообщений** метаданными и ролями (user/assistant/system)
- **Fan-out событий** в различные Kafka-топики для последующей обработки

## Архитектура

Сервис построен на **event-driven архитектуре**:
- Обработка событий из Kafka в реальном времени
- Stateless обработка, stateful хранилище (Redis)
- At-least-once delivery гарантии
- Идемпотентная обработка событий

### Потоки данных

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│  BFF Service │────▶│ publication. │────▶│  Dialogue    │
│              │     │    events    │     │  Aggregator  │
└──────────────┘     └──────────────┘     └──────┬───────┘
                                                   │
┌──────────────┐     ┌──────────────┐             │
│Agent Service │────▶│ generated.   │─────────────┤
│              │     │   phrases    │             │
└──────────────┘     └──────────────┘             │
                                                   ▼
                              ┌─────────────────────────┐
                              │     Redis Storage       │
                              │  - Dialogue State       │
                              │  - Message Cache        │
                              └─────────────────────────┘
                                                   │
                                                   ▼
                              ┌─────────────────────────┐
                              │   Kafka Topics (out)    │
                              │  - messages.full.data   │
                              │  - history.full.events  │
                              └─────────────────────────┘
```

## Kafka Топики

### Входящие топики (Consumer)

| Топик | Consumer Group | Описание |
|-------|---------------|----------|
| `publication.events` | `dialogue-aggregator-publication-group` | Команды для запуска процессов (dialogue.started, user.message.new) |
| `messages.events` | `dialogue-aggregator-messages-group` | События сообщений для восстановления состояния |
| `generated.phrases` | `dialogue-aggregator-generated-group` | Сгенерированные ответы от Agent Service |

### Исходящие топики (Producer)

| Топик | Описание | Подписчики |
|-------|----------|------------|
| `messages.full.data` | Полные сообщения с метаданными для LLM-агента | Agent Service, Analytics |
| `history.full.events` | Immutable event store для истории диалогов | Marking Service, восстановление состояния |

## Обрабатываемые события

### 1. `dialogue.started`

Начало нового диалога.

**Payload:**
```json
{
  "chat_id": "chat-xyz789",
  "user_id": "user-123456",
  "session_id": "session-abc123",
  "dialogue_type": "ml_interview",
  "started_at": "2025-01-15T10:30:00.123Z"
}
```

**Обработка:**
- Создание состояния диалога в Redis
- Добавление в индекс активных диалогов
- Публикация в `history.full.events`

### 2. `user.message.new`

Новое сообщение от пользователя.

**Payload:**
```json
{
  "chat_id": "chat-xyz789",
  "user_id": "user-123456",
  "message_id": "msg-789",
  "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
  "timestamp": "2025-01-15T10:31:00.123Z"
}
```

**Обработка:**
- Добавление сообщения в кэш Redis
- Обновление состояния диалога (флаги, версия)
- Обогащение метаданными (role, message_index, dialogue_context)
- Публикация в `messages.full.data` и `history.full.events`

### 3. `phrase.agent.generated`

Сгенерированный ответ агента.

**Payload:**
```json
{
  "chat_id": "chat-xyz789",
  "message_id": "msg-790",
  "generated_text": "Верно! Теперь объясните разницу между L1 и L2?",
  "confidence": 0.95,
  "timestamp": "2025-01-15T10:32:00.123Z"
}
```

**Обработка:**
- Добавление сообщения ассистента в диалог
- Обновление флагов (awaiting_llm → false, awaiting_user → true)
- Публикация в `history.full.events`

## Структура данных в Redis

### Состояние диалога

**Key:** `dialogue:{chat_id}:state`  
**TTL:** 24 часа

```json
{
  "chat_id": "chat-xyz789",
  "user_id": "user-123456",
  "session_id": "session-abc123",
  "dialogue_type": "ml_interview",
  "status": "active",
  "started_at": "2025-01-15T10:30:00.123Z",
  "last_activity": "2025-01-15T10:31:00.123Z",
  "messages_count": 5,
  "state_version": 5,
  "flags": {
    "is_finished": false,
    "awaiting_llm": true,
    "awaiting_user": false,
    "has_error": false
  },
  "metadata": {
    "topic": "regularization",
    "difficulty": "medium"
  }
}
```

### Кэш сообщений

**Key:** `dialogue:{chat_id}:messages`  
**Тип:** Redis List  
**TTL:** 24 часа  
**Лимит:** 50 последних сообщений

### Индекс активных диалогов

**Key:** `dialogue:active:index`  
**Тип:** Redis Set  
**Элементы:** `chat_id`

## Установка и запуск

### Требования

- Python 3.11+
- Docker и Docker Compose
- Kafka (для обработки событий)
- Redis (для хранения состояния)

### Запуск через Docker Compose

```bash
# Запуск всех зависимостей (Kafka, Redis, Zookeeper)
docker-compose up -d redis zookeeper kafka

# Создание Kafka топиков (опционально, если auto-creation выключен)
docker exec tensortalks_kafka_1 kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic publication.events --partitions 3 --replication-factor 1

# Запуск сервиса
docker-compose up -d dialogue-aggregator

# Проверка статуса
docker-compose ps dialogue-aggregator
curl http://localhost:8085/health
```

### Локальная разработка

```bash
# Установка зависимостей
pip install -r requirements.txt

# Настройка переменных окружения (опционально)
export KAFKA_BOOTSTRAP_SERVERS=localhost:9092
export REDIS_HOST=localhost
export REDIS_PORT=6379

# Запуск
python -m src.main
```

## Конфигурация

Все настройки задаются через environment variables. Полный список доступен в `src/config.py`.

### Основные параметры

#### Kafka
```env
KAFKA_BOOTSTRAP_SERVERS=kafka:9092
KAFKA_TOPIC_PUBLICATION=publication.events
KAFKA_TOPIC_MESSAGES=messages.events
KAFKA_TOPIC_GENERATED=generated.phrases
KAFKA_TOPIC_FULL_DATA=messages.full.data
KAFKA_TOPIC_HISTORY=history.full.events
KAFKA_CONSUMER_ENABLE_AUTO_COMMIT=false
```

#### Redis
```env
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
REDIS_DIALOGUE_TTL_SECONDS=86400
REDIS_MESSAGES_CACHE_SIZE=50
```

#### Сервис
```env
SERVICE_NAME=dialogue-aggregator
SERVICE_VERSION=1.0.0
LOG_LEVEL=INFO
LOG_FORMAT=json
METRICS_PORT=9091
```

## API Endpoints

### Health Checks

Все endpoints возвращают JSON.

#### `GET /health`

Общий health check сервиса.

**Response:**
```json
{
  "status": "healthy",
  "kafka": {
    "status": "healthy"
  },
  "redis": {
    "status": "healthy"
  }
}
```

#### `GET /health/kafka`

Проверка подключения к Kafka.

#### `GET /health/redis`

Проверка подключения к Redis.

#### `GET /ready`

Readiness probe для Kubernetes/Docker.

**Response:**
```json
{
  "status": "ready"
}
```

### Метрики

#### `GET /metrics` (port 9091)

Prometheus метрики в формате Prometheus exposition format.

**Доступ через:** `http://localhost:9091/metrics`

## Метрики Prometheus

### Бизнес-метрики

| Метрика | Тип | Описание |
|---------|-----|----------|
| `dialogue_events_processed_total{event_type, status}` | Counter | Количество обработанных событий |
| `dialogue_state_updates_total{chat_id}` | Counter | Обновления состояния диалогов |
| `dialogue_active_count` | Gauge | Количество активных диалогов |
| `dialogue_messages_total{chat_id}` | Counter | Количество сообщений в диалоге |

### Технические метрики

| Метрика | Тип | Описание |
|---------|-----|----------|
| `event_processing_duration_seconds{event_type}` | Histogram | Время обработки событий |
| `redis_operation_duration_seconds{operation}` | Histogram | Время операций с Redis |
| `kafka_producer_duration_seconds` | Histogram | Время публикации в Kafka |
| `error_count{error_type, service}` | Counter | Количество ошибок по типам |

## Логирование

Сервис использует структурированное логирование в JSON формате через `structlog`.

**Формат лога:**
```json
{
  "timestamp": "2025-01-15T10:31:01.456Z",
  "level": "INFO",
  "service": "dialogue-aggregator",
  "version": "1.0.0",
  "message": "Event processed successfully",
  "event_id": "evt-def456",
  "event_type": "user.message.new",
  "chat_id": "chat-xyz789",
  "processing_time_ms": 45,
  "correlation_id": "corr-456"
}
```

**Уровни:** DEBUG, INFO, WARNING, ERROR

**Конфигурация:** `LOG_LEVEL` (по умолчанию: INFO)

## Структура проекта

```
backend/dialogue-aggregator/
├── src/
│   ├── main.py                 # Точка входа приложения
│   ├── config.py               # Конфигурация (Pydantic Settings)
│   ├── models/                 # Pydantic модели
│   │   ├── events.py           # Модели событий Kafka
│   │   ├── dialogue.py         # Модели состояния диалога
│   │   └── messages.py         # Модели сообщений
│   ├── services/               # Бизнес-логика
│   │   ├── event_router.py     # Маршрутизация событий
│   │   ├── state_manager.py    # Управление состоянием
│   │   ├── enricher.py         # Обогащение сообщений
│   │   └── handlers.py         # Обработчики событий
│   ├── kafka/                  # Kafka клиенты
│   │   ├── producer.py         # Producer для публикации
│   │   └── consumer.py         # Consumer для чтения
│   ├── redis_client/           # Redis клиент
│   │   └── client.py           # Операции с Redis
│   ├── logger/                 # Логирование
│   │   └── setup.py            # Настройка structlog
│   ├── metrics/                # Метрики Prometheus
│   │   └── collector.py        # Сбор метрик
│   └── health/                 # Health checks
│       └── checks.py           # FastAPI endpoints
├── tests/                      # Тесты
├── Dockerfile                  # Docker образ
├── requirements.txt            # Python зависимости
└── README.md                   # Документация
```

## Разработка

### Запуск тестов

```bash
# Unit тесты
pytest tests/unit/

# С покрытием
pytest tests/ --cov=src --cov-report=html
```

### Локальный запуск с зависимостями

```bash
# Запуск Kafka и Redis через docker-compose
docker-compose up -d kafka redis zookeeper

# Запуск сервиса локально
python -m src.main
```

## Интеграция с другими сервисами

### Входящие интеграции

- **BFF Service** → публикует события в `publication.events`
- **Agent Service** → публикует ответы в `generated.phrases`

### Исходящие интеграции

- **Agent Service** ← читает из `messages.full.data`
- **Marking Service** ← читает из `history.full.events`
- **BFF Service** ← читает состояние из Redis (опционально)

## Troubleshooting

### Сервис не запускается

**Проблема:** Port already in use  
**Решение:** Проверьте, что порты 8085 (FastAPI) и 9091 (metrics) свободны

### Kafka consumer не работает

**Проблема:** Unknown topic or partition  
**Решение:** Убедитесь, что топики созданы:
```bash
docker exec tensortalks_kafka_1 kafka-topics --list --bootstrap-server localhost:9092
```

### Redis подключение не работает

**Проблема:** Connection refused  
**Решение:** Проверьте, что Redis запущен и доступен:
```bash
docker-compose ps redis
docker exec tensortalks_redis_1 redis-cli ping
```

### Высокий lag в Kafka consumer

**Проблема:** Медленная обработка событий  
**Решение:** 
- Проверьте метрики `event_processing_duration_seconds`
- Увеличьте количество партиций в топиках
- Добавьте больше инстансов сервиса (горизонтальное масштабирование)

## Масштабирование

Сервис поддерживает горизонтальное масштабирование:

- **Kafka Consumer Groups** автоматически распределяют партиции между инстансами
- **Redis** - shared state storage (в продакшене используйте Redis Cluster)
- **Partitioning** - события партиционируются по `chat_id` для гарантии порядка

**Пример масштабирования:**
```bash
docker-compose up -d --scale dialogue-aggregator=3
```

## Мониторинг

### Рекомендуемые алерты

- `kafka_consumer_lag > 1000` - критический lag
- `error_count > 10 per minute` - высокий уровень ошибок
- `redis_operation_duration_seconds > 1` - медленные операции Redis
- `dialogue_active_count` - мониторинг роста активных диалогов

### Grafana Dashboards

Рекомендуется создать дашборды для:
- Обзора обработки событий
- Производительности Redis
- Kafka consumer lag
- Ошибки по типам

## Лицензия

Внутренний сервис TensorTalks.
