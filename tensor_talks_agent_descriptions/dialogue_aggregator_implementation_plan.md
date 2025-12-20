# План реализации Dialogue Aggregator

## Обзор

Этот документ описывает пошаговый план реализации Dialogue Aggregator сервиса для TensorTalks. Реализация будет выполняться на Python с использованием современных практик разработки.

---

## Этап 1: Подготовка инфраструктуры и настройка проекта

### 1.1 Структура проекта

Создать структуру каталогов:

```
backend/dialogue-aggregator/
├── src/
│   ├── __init__.py
│   ├── main.py                 # Точка входа приложения
│   ├── config.py               # Конфигурация из env
│   ├── models/                 # Pydantic модели
│   │   ├── __init__.py
│   │   ├── events.py           # Модели событий Kafka
│   │   ├── dialogue.py         # Модели диалога и состояния
│   │   └── messages.py         # Модели сообщений
│   ├── services/               # Бизнес-логика
│   │   ├── __init__.py
│   │   ├── event_router.py    # Маршрутизация событий
│   │   ├── state_manager.py   # Управление состоянием диалогов
│   │   └── enricher.py        # Обогащение сообщений
│   ├── kafka/                  # Kafka клиенты
│   │   ├── __init__.py
│   │   ├── consumer.py        # Kafka consumer
│   │   ├── producer.py        # Kafka producer
│   │   └── schemas.py         # Схемы событий
│   ├── redis_client/           # Redis клиент
│   │   ├── __init__.py
│   │   ├── client.py          # Redis клиент
│   │   └── models.py          # Модели для Redis
│   ├── logger/                 # Логирование
│   │   ├── __init__.py
│   │   └── setup.py           # Настройка логирования
│   ├── metrics/                # Метрики Prometheus
│   │   ├── __init__.py
│   │   └── collector.py       # Сбор метрик
│   └── health/                 # Health checks
│       ├── __init__.py
│       └── checks.py          # Проверки здоровья
├── tests/
│   ├── __init__.py
│   ├── unit/
│   │   ├── test_models.py
│   │   ├── test_services.py
│   │   └── test_enricher.py
│   ├── integration/
│   │   ├── test_kafka.py
│   │   └── test_redis.py
│   └── fixtures/
│       └── events.py
├── Dockerfile
├── .dockerignore
├── requirements.txt
├── pyproject.toml
└── README.md
```

### 1.2 Зависимости (requirements.txt)

```txt
# Core
fastapi==0.104.1
uvicorn[standard]==0.24.0
pydantic==2.5.0
pydantic-settings==2.1.0

# Kafka
confluent-kafka==2.3.0
avro-python3==1.11.3

# Redis
redis==5.0.1
hiredis==2.2.3

# Logging
structlog==23.2.0
python-json-logger==2.0.7

# Metrics
prometheus-client==0.19.0

# Testing
pytest==7.4.3
pytest-asyncio==0.21.1
pytest-cov==4.1.0
testcontainers==3.7.1
faker==20.1.0

# Utilities
python-dateutil==2.8.2
```

### 1.3 Настройка Docker Compose

Добавить в `docker-compose.yml`:

```yaml
redis:
  image: redis:7-alpine
  ports:
    - "6379:6379"
  command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
  volumes:
    - redis_data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 3s
    retries: 5

dialogue-aggregator:
  build:
    context: ./backend/dialogue-aggregator
  environment:
    # Kafka
    KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    KAFKA_CONSUMER_GROUP_PUBLICATION: dialogue-aggregator-publication-group
    KAFKA_CONSUMER_GROUP_MESSAGES: dialogue-aggregator-messages-group
    KAFKA_CONSUMER_GROUP_GENERATED: dialogue-aggregator-generated-group
    KAFKA_TOPIC_PUBLICATION: publication.events
    KAFKA_TOPIC_MESSAGES: messages.events
    KAFKA_TOPIC_GENERATED: generated.phrases
    KAFKA_TOPIC_FULL_DATA: messages.full.data
    KAFKA_TOPIC_HISTORY: history.full.events
    KAFKA_TOPIC_DLQ: dialogue-aggregator.dlq
    KAFKA_CONSUMER_AUTO_OFFSET_RESET: earliest
    KAFKA_CONSUMER_ENABLE_AUTO_COMMIT: "false"
    KAFKA_CONSUMER_MAX_POLL_RECORDS: 100
    KAFKA_PRODUCER_ACKS: all
    KAFKA_PRODUCER_RETRIES: 3
    KAFKA_PRODUCER_COMPRESSION_TYPE: snappy
    # Redis
    REDIS_HOST: redis
    REDIS_PORT: 6379
    REDIS_DB: 0
    REDIS_PASSWORD: ""
    REDIS_MAX_CONNECTIONS: 50
    REDIS_DIALOGUE_TTL_SECONDS: 86400
    REDIS_MESSAGES_CACHE_SIZE: 50
    # Service
    SERVICE_NAME: dialogue-aggregator
    SERVICE_VERSION: 1.0.0
    LOG_LEVEL: INFO
    LOG_FORMAT: json
    MAX_MESSAGES_CACHE_SIZE: 50
    # Metrics
    METRICS_PORT: 9091
    ENABLE_PROMETHEUS: "true"
  depends_on:
    kafka:
      condition: service_started
    redis:
      condition: service_healthy
  ports:
    - "9091:9091"
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9091/health"]
    interval: 10s
    timeout: 5s
    retries: 3

volumes:
  redis_data:
```

---

## Этап 2: Базовая инфраструктура

### 2.1 Конфигурация (config.py)

**Задачи**:
- Создать класс конфигурации на базе Pydantic Settings
- Загрузить все настройки из environment variables
- Валидация конфигурации при старте

**Критерии готовности**:
- ✅ Все настройки загружаются из env
- ✅ Типизация настроек через Pydantic
- ✅ Валидация обязательных параметров

### 2.2 Логирование (logger/setup.py)

**Задачи**:
- Настроить structured logging через structlog
- JSON формат для продакшена
- Контекстное логирование (chat_id, event_id, correlation_id)

**Критерии готовности**:
- ✅ Логи в JSON формате
- ✅ Автоматическое добавление контекста
- ✅ Уровни логирования (DEBUG, INFO, WARNING, ERROR)

### 2.3 Метрики (metrics/collector.py)

**Задачи**:
- Интеграция Prometheus клиента
- Определение всех метрик из архитектуры
- HTTP endpoint для Prometheus

**Критерии готовности**:
- ✅ Все метрики определены
- ✅ Endpoint `/metrics` работает
- ✅ Метрики обновляются корректно

### 2.4 Health Checks (health/checks.py)

**Задачи**:
- Endpoint `/health` для общих проверок
- Endpoint `/health/kafka` для проверки Kafka
- Endpoint `/health/redis` для проверки Redis
- Endpoint `/ready` для readiness probe

**Критерии готовности**:
- ✅ Все endpoints работают
- ✅ Проверки реально тестируют подключения
- ✅ Возвращают правильные HTTP коды

---

## Этап 3: Pydantic модели

### 3.1 Модели событий (models/events.py)

**Задачи**:
- Реализовать базовую модель `KafkaEvent`
- Реализовать модели payload для всех типов событий
- Валидация и сериализация/десериализация

**Типы событий для реализации**:
- `dialogue.started`
- `user.message.new`
- `phrase.agent.generated`
- `message.user.sent`
- `message.assistant.generated`

**Критерии готовности**:
- ✅ Все модели определены
- ✅ Валидация работает корректно
- ✅ JSON сериализация/десериализация
- ✅ Unit тесты для моделей

### 3.2 Модели диалога (models/dialogue.py)

**Задачи**:
- Реализовать `DialogueState`
- Реализовать `DialogueFlags`
- Реализовать `DialogueStatus` enum

**Критерии готовности**:
- ✅ Все модели определены
- ✅ Валидация полей
- ✅ Сериализация для Redis

### 3.3 Модели сообщений (models/messages.py)

**Задачи**:
- Реализовать `Message`
- Реализовать `MessageRole` enum
- Реализовать модели для публикации (`MessageFullPayload`, `HistoryEventPayload`)

**Критерии готовности**:
- ✅ Все модели определены
- ✅ Поддержка всех ролей
- ✅ Unit тесты

---

## Этап 4: Redis клиент

### 4.1 Redis Client (redis_client/client.py)

**Задачи**:
- Создать Redis клиент с connection pooling
- Реализовать методы для работы с состоянием диалога
- Реализовать методы для работы с кэшем сообщений
- Обработка ошибок и retry логика

**Методы для реализации**:
- `get_dialogue_state(chat_id) -> Optional[DialogueState]`
- `save_dialogue_state(chat_id, state) -> None`
- `add_message(chat_id, message) -> None`
- `get_messages(chat_id, limit) -> List[Message]`
- `add_to_active_index(chat_id) -> None`
- `remove_from_active_index(chat_id) -> None`
- `get_active_dialogues() -> Set[str]`

**Критерии готовности**:
- ✅ Все методы реализованы
- ✅ Connection pooling работает
- ✅ TTL устанавливается корректно
- ✅ Обработка ошибок и retry
- ✅ Integration тесты с Redis

### 4.2 Redis операции (redis_client/operations.py)

**Задачи**:
- Оптимизация операций (batch операции)
- Реализация транзакций где необходимо
- Кэширование часто используемых данных

**Критерии готовности**:
- ✅ Batch операции реализованы
- ✅ Производительность соответствует требованиям

---

## Этап 5: Kafka клиенты

### 5.1 Kafka Producer (kafka/producer.py)

**Задачи**:
- Создать Kafka producer с правильной конфигурацией
- Реализовать метод публикации событий
- Обработка ошибок публикации
- Retry логика
- Метрики для публикации

**Методы для реализации**:
- `publish(topic, event, key=None) -> None`
- `publish_batch(topic, events, key=None) -> None`
- `close() -> None`

**Критерии готовности**:
- ✅ Producer работает корректно
- ✅ Retry логика реализована
- ✅ Метрики собираются
- ✅ Обработка ошибок

### 5.2 Kafka Consumer (kafka/consumer.py)

**Задачи**:
- Создать Kafka consumers для трех топиков
- Реализовать обработку сообщений
- Manual commit offsets
- Обработка ошибок
- Метрики lag

**Методы для реализации**:
- `subscribe(topic, group_id, handler) -> None`
- `poll(timeout) -> List[KafkaEvent]`
- `commit() -> None`
- `close() -> None`

**Критерии готовности**:
- ✅ Consumers для всех топиков
- ✅ Manual commit работает
- ✅ Lag метрики собираются
- ✅ Обработка ошибок и DLQ

---

## Этап 6: Бизнес-логика

### 6.1 Event Router (services/event_router.py)

**Задачи**:
- Маршрутизация событий по типу
- Вызов соответствующих обработчиков
- Валидация входящих событий
- Обработка неизвестных типов событий

**Методы для реализации**:
- `route_event(event: KafkaEvent) -> None`
- `register_handler(event_type, handler) -> None`

**Критерии готовности**:
- ✅ Маршрутизация работает
- ✅ Все типы событий обрабатываются
- ✅ Валидация выполняется
- ✅ Unit тесты

### 6.2 State Manager (services/state_manager.py)

**Задачи**:
- Управление жизненным циклом состояния диалога
- Обновление состояния при обработке событий
- Версионирование состояния
- Проверка консистентности

**Методы для реализации**:
- `initialize_dialogue(chat_id, payload) -> DialogueState`
- `update_dialogue_state(chat_id, updates) -> DialogueState`
- `add_message_to_dialogue(chat_id, message) -> None`
- `get_dialogue_state(chat_id) -> Optional[DialogueState]`
- `finish_dialogue(chat_id) -> None`

**Критерии готовности**:
- ✅ Все методы работают
- ✅ Версионирование реализовано
- ✅ Консистентность проверяется
- ✅ Unit и integration тесты

### 6.3 Event Enricher (services/enricher.py)

**Задачи**:
- Обогащение сообщений метаданными
- Проставление ролей сообщений
- Добавление dialogue_context
- Добавление message_index

**Методы для реализации**:
- `enrich_message(message, dialogue_state) -> MessageFullPayload`
- `add_dialogue_context(message, state) -> Dict[str, Any]`
- `calculate_message_index(chat_id, role) -> int`

**Критерии готовности**:
- ✅ Обогащение работает корректно
- ✅ Все метаданные добавляются
- ✅ Unit тесты

---

## Этап 7: Обработчики событий

### 7.1 Обработчик dialogue.started

**Задачи**:
- Обработка события начала диалога
- Создание нового состояния в Redis
- Добавление в индекс активных диалогов
- Публикация в history.full.events

**Логика**:
1. Проверить, не существует ли диалог
2. Создать DialogueState
3. Сохранить в Redis
4. Добавить в активный индекс
5. Создать системное сообщение
6. Публиковать в history
7. Логировать

**Критерии готовности**:
- ✅ Полная обработка события
- ✅ Обработка ошибок
- ✅ Логирование
- ✅ Метрики
- ✅ Тесты

### 7.2 Обработчик user.message.new

**Задачи**:
- Обработка нового сообщения пользователя
- Обновление состояния диалога
- Добавление сообщения в кэш
- Публикация в messages.full.data и history.full.events

**Логика**:
1. Загрузить состояние диалога
2. Проверить валидность (существует, активен)
3. Создать Message объект
4. Добавить в кэш Redis
5. Обновить состояние (flags, version, count)
6. Обогатить сообщение
7. Публиковать в messages.full.data
8. Публиковать в history.full.events
9. Сохранить обновленное состояние
10. Логировать

**Критерии готовности**:
- ✅ Полная обработка события
- ✅ Валидация
- ✅ Обогащение
- ✅ Публикация в оба топика
- ✅ Обработка ошибок
- ✅ Тесты

### 7.3 Обработчик phrase.agent.generated

**Задачи**:
- Обработка сгенерированного ответа агента
- Добавление сообщения ассистента в диалог
- Обновление состояния (awaiting_llm -> false, awaiting_user -> true)
- Публикация в history.full.events

**Логика**:
1. Загрузить состояние диалога
2. Проверить валидность
3. Создать Message с role=assistant
4. Добавить в кэш
5. Обновить состояние и флаги
6. Обогатить сообщение
7. Публиковать в history
8. Сохранить состояние
9. Логировать

**Критерии готовности**:
- ✅ Полная обработка события
- ✅ Обновление флагов
- ✅ Публикация
- ✅ Тесты

### 7.4 Обработчик message.user.sent (из messages.events)

**Задачи**:
- Обработка события из messages.events (replay)
- Идемпотентная обработка
- Проверка дубликатов по message_id

**Логика**:
1. Проверить, не обработано ли уже сообщение (по message_id)
2. Если не обработано, выполнить логику аналогичную user.message.new
3. Логировать

**Критерии готовности**:
- ✅ Идемпотентность
- ✅ Проверка дубликатов
- ✅ Тесты

---

## Этап 8: Интеграция компонентов

### 8.1 Главное приложение (main.py)

**Задачи**:
- Инициализация всех компонентов
- Запуск Kafka consumers в отдельных потоках/процессах
- Запуск FastAPI для health checks и metrics
- Graceful shutdown
- Обработка сигналов

**Структура**:
```python
async def main():
    # Инициализация
    config = load_config()
    logger = setup_logger(config)
    redis_client = create_redis_client(config)
    kafka_producer = create_kafka_producer(config)
    state_manager = StateManager(redis_client, logger)
    enricher = EventEnricher(logger)
    event_router = EventRouter(state_manager, enricher, kafka_producer, logger)
    
    # Создание consumers
    publication_consumer = create_consumer(config, "publication")
    messages_consumer = create_consumer(config, "messages")
    generated_consumer = create_consumer(config, "generated")
    
    # Запуск обработчиков
    # ...
    
    # Запуск FastAPI
    # ...
    
    # Graceful shutdown
    # ...
```

**Критерии готовности**:
- ✅ Все компоненты инициализируются
- ✅ Consumers работают параллельно
- ✅ FastAPI запускается
- ✅ Graceful shutdown работает
- ✅ Все ресурсы освобождаются

### 8.2 Интеграция обработчиков

**Задачи**:
- Подключить все обработчики к event_router
- Настроить маршрутизацию
- Протестировать полный цикл обработки

**Критерии готовности**:
- ✅ Все обработчики подключены
- ✅ События маршрутизируются корректно
- ✅ End-to-end тесты проходят

---

## Этап 9: Docker и деплой

### 9.1 Dockerfile

**Задачи**:
- Создать оптимизированный Dockerfile
- Multi-stage build
- Минимальный размер образа

```dockerfile
FROM python:3.11-slim as builder

WORKDIR /app

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

FROM python:3.11-slim

WORKDIR /app

COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages
COPY --from=builder /usr/local/bin /usr/local/bin

COPY src/ ./src/

CMD ["python", "-m", "src.main"]
```

**Критерии готовности**:
- ✅ Dockerfile создан
- ✅ Образ собирается
- ✅ Размер образа оптимизирован

### 9.2 Интеграция в docker-compose

**Задачи**:
- Добавить сервис в docker-compose.yml
- Настроить зависимости
- Протестировать запуск

**Критерии готовности**:
- ✅ Сервис запускается в docker-compose
- ✅ Все зависимости настроены
- ✅ Health checks работают

---

## Этап 10: Тестирование

### 10.1 Unit тесты

**Задачи**:
- Покрытие всех моделей
- Покрытие всех сервисов
- Покрытие обработчиков (моки)

**Цель покрытия**: >80%

**Критерии готовности**:
- ✅ Все компоненты покрыты тестами
- ✅ Покрытие >80%
- ✅ Тесты проходят

### 10.2 Integration тесты

**Задачи**:
- Тесты с реальным Kafka (testcontainers)
- Тесты с реальным Redis (testcontainers)
- End-to-end тесты обработки событий

**Критерии готовности**:
- ✅ Integration тесты написаны
- ✅ Тесты используют testcontainers
- ✅ Полный цикл обработки протестирован

### 10.3 Performance тесты

**Задачи**:
- Нагрузочное тестирование
- Измерение throughput
- Измерение latency

**Критерии готовности**:
- ✅ Performance тесты написаны
- ✅ Результаты задокументированы
- ✅ Узкие места выявлены и оптимизированы

---

## Этап 11: Документация и финализация

### 11.1 README

**Задачи**:
- Описание сервиса
- Инструкции по запуску
- Описание конфигурации
- Примеры использования

**Критерии готовности**:
- ✅ README написан
- ✅ Все инструкции актуальны
- ✅ Примеры работают

### 11.2 API документация

**Задачи**:
- Документация health endpoints
- Документация metrics
- OpenAPI спецификация (если используется FastAPI)

**Критерии готовности**:
- ✅ Документация создана
- ✅ Endpoints задокументированы

### 11.3 Мониторинг и алерты

**Задачи**:
- Настройка Grafana dashboards
- Настройка алертов в Prometheus
- Документация метрик

**Критерии готовности**:
- ✅ Dashboards созданы
- ✅ Алерты настроены
- ✅ Метрики задокументированы

---

## Порядок реализации (Timeline)

### Неделя 1: Инфраструктура и модели
- День 1-2: Структура проекта, конфигурация, логирование
- День 3-4: Метрики, health checks
- День 5: Pydantic модели

### Неделя 2: Клиенты и базовые сервисы
- День 1-2: Redis клиент
- День 3-4: Kafka producer и consumer
- День 5: Event Router, State Manager, Enricher

### Неделя 3: Обработчики событий
- День 1: Обработчик dialogue.started
- День 2: Обработчик user.message.new
- День 3: Обработчик phrase.agent.generated
- День 4: Обработчик message.user.sent
- День 5: Интеграция и тестирование

### Неделя 4: Интеграция и деплой
- День 1-2: Главное приложение, интеграция
- День 3: Docker, docker-compose
- День 4: Тестирование (unit, integration, performance)
- День 5: Документация, финализация

---

## Критерии приемки (Definition of Done)

Сервис считается готовым, когда:

1. ✅ Все компоненты реализованы согласно архитектуре
2. ✅ Все тесты написаны и проходят (>80% покрытие)
3. ✅ Сервис запускается в docker-compose
4. ✅ Health checks работают
5. ✅ Метрики собираются и доступны
6. ✅ Логирование настроено и работает
7. ✅ Документация написана
8. ✅ Code review пройден
9. ✅ Интеграция с другими сервисами протестирована
10. ✅ Performance тесты пройдены

---

## Риски и митигация

### Риск 1: Проблемы с производительностью Redis
**Митигация**: 
- Использование connection pooling
- Оптимизация операций (batch)
- Мониторинг производительности
- Возможность масштабирования до Redis Cluster

### Риск 2: Проблемы с порядком обработки событий
**Митигация**: 
- Key-based partitioning по chat_id
- Версионирование состояния
- Проверка консистентности

### Риск 3: Потеря событий при сбоях
**Митигация**: 
- Manual commit offsets
- DLQ для проблемных событий
- Возможность replay из history.full.events

### Риск 4: Race conditions при параллельной обработке
**Митигация**: 
- Версионирование состояния
- Оптимистичная блокировка
- Проверка state_version перед обновлением

---

## Приоритизация

### MVP (Must Have)
1. Обработка dialogue.started
2. Обработка user.message.new
3. Обработка phrase.agent.generated
4. Redis для состояния
5. Публикация в messages.full.data и history.full.events
6. Health checks
7. Базовое логирование

### Nice to Have
1. Обработка message.user.sent (replay)
2. DLQ
3. Метрики Prometheus
4. Performance оптимизации
5. Расширенное логирование

---

Этот план обеспечивает пошаговую реализацию Dialogue Aggregator с четкими критериями готовности на каждом этапе.

