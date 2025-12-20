# Архитектура Dialogue Aggregator

## 1. Обзор

**Dialogue Aggregator** — центральный сервис системы TensorTalks, ответственный за:
- Агрегацию диалогов из событий Kafka
- Поддержание состояния диалогов в KV-хранилище
- Обогащение сообщений метаданными и ролями
- Fan-out событий в различные Kafka-топики для последующей обработки

### Основные принципы:
- **Event-driven**: вся логика построена на обработке событий из Kafka
- **Stateless processing, stateful storage**: обработка без состояния, состояние хранится в Redis
- **At-least-once delivery**: гарантированная обработка каждого события
- **Идемпотентность**: повторная обработка события не меняет результат

---

## 2. Kafka Топики

### 2.1 Входящие топики (Consumer)

#### 2.1.1 `publication.events` (Publication Topic)
**Назначение**: Топик команд для запуска асинхронных процессов.

**Типы событий**:
- `dialogue.started` — начало нового диалога
- `user.message.new` — новое сообщение от пользователя
- `dialogue.continued` — продолжение существующего диалога
- `user.action` — действие пользователя (например, пропуск вопроса)

**Пример события `dialogue.started`**:
```json
{
  "event_id": "evt-abc123",
  "event_type": "dialogue.started",
  "timestamp": "2025-01-15T10:30:00.123Z",
  "service": "bff-service",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "user_id": "user-123456",
    "session_id": "session-abc123",
    "dialogue_type": "ml_interview",
    "started_at": "2025-01-15T10:30:00.123Z"
  },
  "metadata": {
    "request_id": "req-123",
    "correlation_id": "corr-456"
  }
}
```

**Пример события `user.message.new`**:
```json
{
  "event_id": "evt-def456",
  "event_type": "user.message.new",
  "timestamp": "2025-01-15T10:31:00.123Z",
  "service": "bff-service",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "user_id": "user-123456",
    "message_id": "msg-789",
    "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
    "timestamp": "2025-01-15T10:31:00.123Z"
  },
  "metadata": {
    "request_id": "req-456",
    "correlation_id": "corr-456"
  }
}
```

**Consumer Group**: `dialogue-aggregator-publication-group`
**Partitioning**: По `chat_id` (key-based partitioning для гарантии порядка)

---

#### 2.1.2 `messages.events` (Messages Events Topic)
**Назначение**: Топик событий сообщений (event stream) для восстановления состояния.

**Типы событий**:
- `message.user.sent` — пользователь отправил сообщение
- `message.assistant.generated` — ассистент сгенерировал ответ
- `message.system.created` — системное сообщение

**Пример события `message.user.sent`**:
```json
{
  "event_id": "evt-ghi789",
  "event_type": "message.user.sent",
  "timestamp": "2025-01-15T10:31:00.123Z",
  "service": "bff-service",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "message_id": "msg-789",
    "user_id": "user-123456",
    "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
    "timestamp": "2025-01-15T10:31:00.123Z"
  },
  "metadata": {
    "request_id": "req-456"
  }
}
```

**Consumer Group**: `dialogue-aggregator-messages-group`
**Partitioning**: По `chat_id`

---

#### 2.1.3 `generated.phrases` (Generated Phrase Topic)
**Назначение**: Топик сгенерированных LLM-ответов от Agent Service.

**Типы событий**:
- `phrase.agent.generated` — агент сгенерировал ответ

**Пример события `phrase.agent.generated`**:
```json
{
  "event_id": "evt-jkl012",
  "event_type": "phrase.agent.generated",
  "timestamp": "2025-01-15T10:32:00.123Z",
  "service": "agent-service",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "message_id": "msg-790",
    "generated_text": "Верно! L1 регуляризация (Lasso) добавляет к функции потерь сумму абсолютных значений параметров. Теперь объясните, в чем разница между L1 и L2 регуляризацией?",
    "confidence": 0.95,
    "intermediate_steps": [
      {
        "step": "reasoning",
        "content": "Пользователь дал правильный ответ про L1"
      }
    ],
    "metadata": {
      "model": "gpt-4",
      "temperature": 0.7
    },
    "timestamp": "2025-01-15T10:32:00.123Z"
  },
  "metadata": {
    "request_id": "req-789",
    "correlation_id": "corr-456"
  }
}
```

**Consumer Group**: `dialogue-aggregator-generated-group`
**Partitioning**: По `chat_id`

---

### 2.2 Исходящие топики (Producer)

#### 2.2.1 `messages.full.data` (Messages Full Data Topic)
**Назначение**: Топик полных сообщений с максимальным payload для LLM-агента.

**Структура сообщения**:
```json
{
  "event_id": "evt-mno345",
  "event_type": "message.full",
  "timestamp": "2025-01-15T10:31:00.123Z",
  "service": "dialogue-aggregator",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "message_id": "msg-789",
    "role": "user",
    "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
    "metadata": {
      "user_id": "user-123456",
      "message_index": 3,
      "dialogue_context": {
        "total_messages": 5,
        "topic": "regularization",
        "difficulty": "medium"
      }
    },
    "embeddings": null,
    "source": "user_input",
    "timestamp": "2025-01-15T10:31:00.123Z",
    "processed_at": "2025-01-15T10:31:01.456Z"
  },
  "metadata": {
    "request_id": "req-456",
    "correlation_id": "corr-456"
  }
}
```

**Используется**:
- LLM Agent Service (для генерации ответов)
- Analytics (для анализа диалогов)
- Offline processing (для обучения моделей)

**Partitioning**: По `chat_id`

---

#### 2.2.2 `history.full.events` (Full History Events Topic)
**Назначение**: Immutable event store для исторического лога диалогов.

**Особенности**:
- Никогда не перезаписывается
- Используется для восстановления состояния
- Поддерживает replay событий

**Структура сообщения** (аналогична `messages.full.data`, но с дополнительными полями):
```json
{
  "event_id": "evt-pqr678",
  "event_type": "history.event",
  "timestamp": "2025-01-15T10:31:00.123Z",
  "service": "dialogue-aggregator",
  "version": "1.0.0",
  "payload": {
    "chat_id": "chat-xyz789",
    "dialogue_state": {
      "status": "active",
      "messages_count": 5,
      "last_activity": "2025-01-15T10:31:00.123Z"
    },
    "message": {
      "message_id": "msg-789",
      "role": "user",
      "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
      "timestamp": "2025-01-15T10:31:00.123Z"
    },
    "snapshot": {
      "chat_id": "chat-xyz789",
      "state_version": 5,
      "all_messages": [...]
    }
  },
  "metadata": {
    "request_id": "req-456",
    "correlation_id": "corr-456"
  }
}
```

**Подписчики**:
- Dialogue Aggregator (для восстановления состояния)
- Marking Service (для анализа и разметки)

**Partitioning**: По `chat_id`

---

## 3. Key-Value хранилище (Redis)

### 3.1 Назначение
- Быстрый доступ к текущему состоянию диалога
- Кэширование последних N сообщений для контекста LLM
- Хранение флагов и метаданных диалога

### 3.2 Структура данных

#### 3.2.1 Диалог (Dialogue State)
**Key**: `dialogue:{chat_id}:state`
**TTL**: 24 часа (для активных диалогов)

**Структура**:
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
    "difficulty": "medium",
    "current_question_index": 2
  }
}
```

#### 3.2.2 Последние сообщения (Message Cache)
**Key**: `dialogue:{chat_id}:messages`
**TTL**: 24 часа
**Тип**: Redis List (используется для хранения последних N сообщений)

**Структура каждого элемента списка**:
```json
{
  "message_id": "msg-789",
  "role": "user",
  "content": "L1 регуляризация добавляет сумму абсолютных значений весов",
  "timestamp": "2025-01-15T10:31:00.123Z",
  "processed_at": "2025-01-15T10:31:01.456Z"
}
```

**Ограничение**: Максимум 50 последних сообщений на диалог (LRU eviction)

#### 3.2.3 Индекс активных диалогов
**Key**: `dialogue:active:index`
**Тип**: Redis Set
**Назначение**: Быстрый поиск всех активных диалогов

**Элементы**: `chat_id` (например, "chat-xyz789")

---

### 3.3 Операции с Redis

#### 3.3.1 Получение состояния диалога
```
GET dialogue:{chat_id}:state
```

#### 3.3.2 Обновление состояния диалога
```
SET dialogue:{chat_id}:state {json} EX 86400
```

#### 3.3.3 Добавление сообщения
```
RPUSH dialogue:{chat_id}:messages {json}
LTRIM dialogue:{chat_id}:messages -50 -1
EXPIRE dialogue:{chat_id}:messages 86400
```

#### 3.3.4 Получение последних N сообщений
```
LRANGE dialogue:{chat_id}:messages -N -1
```

#### 3.3.5 Отметка диалога как активного
```
SADD dialogue:active:index {chat_id}
```

#### 3.3.6 Завершение диалога
```
SREM dialogue:active:index {chat_id}
```

---

## 4. Внутренняя архитектура Dialogue Aggregator

### 4.1 Компоненты сервиса

```
┌─────────────────────────────────────────────────────┐
│           Dialogue Aggregator Service               │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  Kafka Consumers (3 параллельных консьюмера) │  │
│  │  - publication.events                         │  │
│  │  - messages.events                            │  │
│  │  - generated.phrases                          │  │
│  └──────────────┬───────────────────────────────┘  │
│                 │                                    │
│                 v                                    │
│  ┌──────────────────────────────────────────────┐  │
│  │  Event Router                                │  │
│  │  - Маршрутизация по event_type               │  │
│  │  - Валидация событий                         │  │
│  └──────────────┬───────────────────────────────┘  │
│                 │                                    │
│                 v                                    │
│  ┌──────────────────────────────────────────────┐  │
│  │  Dialogue State Manager                      │  │
│  │  - Агрегация диалога                         │  │
│  │  - Обогащение метаданными                    │  │
│  │  - Управление состоянием в Redis             │  │
│  └──────────────┬───────────────────────────────┘  │
│                 │                                    │
│                 v                                    │
│  ┌──────────────────────────────────────────────┐  │
│  │  Event Enricher                              │  │
│  │  - Добавление ролей (user/assistant/system)  │  │
│  │  - Проставление message_index                │  │
│  │  - Добавление dialogue_context               │  │
│  └──────────────┬───────────────────────────────┘  │
│                 │                                    │
│                 v                                    │
│  ┌──────────────────────────────────────────────┐  │
│  │  Kafka Producers (2 продюсера)               │  │
│  │  - messages.full.data                        │  │
│  │  - history.full.events                       │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  Redis Client                                │  │
│  │  - Операции с KV-хранилищем                  │  │
│  │  - Connection pooling                        │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
│  ┌──────────────────────────────────────────────┐  │
│  │  Logger                                      │  │
│  │  - Structured logging                        │  │
│  │  - Metrics для Prometheus                    │  │
│  └──────────────────────────────────────────────┘  │
│                                                      │
└─────────────────────────────────────────────────────┘
```

---

### 4.2 Обработка событий

#### 4.2.1 Обработка `dialogue.started`
1. Получить событие из `publication.events`
2. Проверить, не существует ли уже диалог с таким `chat_id`
3. Создать новое состояние диалога в Redis:
   - Сохранить базовую информацию
   - Установить статус "active"
   - Добавить в индекс активных диалогов
4. Создать системное сообщение о начале диалога
5. Публиковать в `history.full.events`
6. Логировать событие

#### 4.2.2 Обработка `user.message.new`
1. Получить событие из `publication.events`
2. Загрузить текущее состояние диалога из Redis
3. Проверить валидность (диалог существует, статус активен)
4. Добавить сообщение в кэш Redis (список сообщений)
5. Обновить состояние диалога:
   - Увеличить `messages_count`
   - Обновить `last_activity`
   - Установить флаг `awaiting_llm: true`
   - Увеличить `state_version`
6. Обогатить сообщение:
   - Проставить `role: "user"`
   - Добавить `message_index`
   - Добавить `dialogue_context` из состояния
7. Публиковать в `messages.full.data` (для LLM Agent)
8. Публиковать в `history.full.events` (для истории)
9. Сохранить обновленное состояние в Redis
10. Логировать событие

#### 4.2.3 Обработка `phrase.agent.generated`
1. Получить событие из `generated.phrases`
2. Загрузить текущее состояние диалога из Redis
3. Проверить валидность
4. Добавить сгенерированное сообщение в кэш Redis
5. Обновить состояние диалога:
   - Увеличить `messages_count`
   - Обновить `last_activity`
   - Установить флаг `awaiting_llm: false`
   - Установить флаг `awaiting_user: true`
   - Увеличить `state_version`
6. Обогатить сообщение:
   - Проставить `role: "assistant"`
   - Добавить `message_index`
   - Добавить `dialogue_context`
7. Публиковать в `history.full.events`
8. Сохранить обновленное состояние в Redis
9. Логировать событие

#### 4.2.4 Обработка `message.user.sent` из `messages.events`
1. Получить событие из `messages.events`
2. Использовать для восстановления состояния (replay)
3. Аналогична обработке `user.message.new`, но с проверкой идемпотентности

---

### 4.3 Обработка ошибок

#### 4.3.1 Ошибки обработки событий
- Логирование ошибки с полным контекстом события
- Отправка события в Dead Letter Queue (DLQ) топик
- Продолжение обработки следующих событий
- Метрики для мониторинга ошибок

#### 4.3.2 Ошибки Redis
- Retry с экспоненциальным backoff
- Circuit breaker для защиты от каскадных отказов
- Fallback: публикация события в DLQ для последующей обработки

#### 4.3.3 Ошибки Kafka
- Автоматический retry для transient ошибок
- Метрики для отслеживания lag консьюмеров
- Алерты при критическом lag

---

## 5. Модели данных (Pydantic)

### 5.1 Базовые модели

```python
from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, Dict, Any, List
from enum import Enum

class EventType(str, Enum):
    """Типы событий"""
    # Publication events
    DIALOGUE_STARTED = "dialogue.started"
    USER_MESSAGE_NEW = "user.message.new"
    DIALOGUE_CONTINUED = "dialogue.continued"
    USER_ACTION = "user.action"
    
    # Messages events
    MESSAGE_USER_SENT = "message.user.sent"
    MESSAGE_ASSISTANT_GENERATED = "message.assistant.generated"
    MESSAGE_SYSTEM_CREATED = "message.system.created"
    
    # Generated phrases
    PHRASE_AGENT_GENERATED = "phrase.agent.generated"
    
    # Full data events
    MESSAGE_FULL = "message.full"
    HISTORY_EVENT = "history.event"

class DialogueStatus(str, Enum):
    """Статусы диалога"""
    ACTIVE = "active"
    FINISHED = "finished"
    ERROR = "error"
    PAUSED = "paused"

class MessageRole(str, Enum):
    """Роли сообщений"""
    USER = "user"
    ASSISTANT = "assistant"
    SYSTEM = "system"
```

### 5.2 Модель события Kafka

```python
class KafkaEvent(BaseModel):
    """Базовая модель события Kafka"""
    event_id: str = Field(..., description="Уникальный ID события (UUID)")
    event_type: EventType = Field(..., description="Тип события")
    timestamp: datetime = Field(..., description="Время создания события (ISO 8601 UTC)")
    service: str = Field(..., description="Имя сервиса, создавшего событие")
    version: str = Field(..., description="Версия сервиса")
    payload: Dict[str, Any] = Field(..., description="Данные события")
    metadata: Optional[Dict[str, Any]] = Field(None, description="Дополнительные метаданные")
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
```

### 5.3 Модели payload для различных событий

```python
class DialogueStartedPayload(BaseModel):
    """Payload для dialogue.started"""
    chat_id: str
    user_id: str
    session_id: str
    dialogue_type: str = "ml_interview"
    started_at: datetime

class UserMessageNewPayload(BaseModel):
    """Payload для user.message.new"""
    chat_id: str
    user_id: str
    message_id: str
    content: str
    timestamp: datetime

class PhraseAgentGeneratedPayload(BaseModel):
    """Payload для phrase.agent.generated"""
    chat_id: str
    message_id: str
    generated_text: str
    confidence: Optional[float] = None
    intermediate_steps: Optional[List[Dict[str, Any]]] = None
    metadata: Optional[Dict[str, Any]] = None
    timestamp: datetime
```

### 5.4 Модель состояния диалога

```python
class DialogueFlags(BaseModel):
    """Флаги состояния диалога"""
    is_finished: bool = False
    awaiting_llm: bool = False
    awaiting_user: bool = False
    has_error: bool = False

class DialogueState(BaseModel):
    """Состояние диалога в Redis"""
    chat_id: str
    user_id: str
    session_id: str
    dialogue_type: str
    status: DialogueStatus
    started_at: datetime
    last_activity: datetime
    messages_count: int = 0
    state_version: int = 0
    flags: DialogueFlags = DialogueFlags()
    metadata: Dict[str, Any] = Field(default_factory=dict)
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }

class Message(BaseModel):
    """Модель сообщения"""
    message_id: str
    role: MessageRole
    content: str
    timestamp: datetime
    processed_at: Optional[datetime] = None
    
    class Config:
        json_encoders = {
            datetime: lambda v: v.isoformat()
        }
```

### 5.5 Модели для публикации в топики

```python
class MessageFullPayload(BaseModel):
    """Payload для messages.full.data"""
    chat_id: str
    message_id: str
    role: MessageRole
    content: str
    metadata: Dict[str, Any]
    embeddings: Optional[List[float]] = None
    source: str
    timestamp: datetime
    processed_at: datetime

class HistoryEventPayload(BaseModel):
    """Payload для history.full.events"""
    chat_id: str
    dialogue_state: DialogueState
    message: Optional[Message] = None
    snapshot: Optional[Dict[str, Any]] = None
```

---

## 6. Настройки (Environment Variables)

### 6.1 Kafka настройки

```env
# Kafka Brokers
KAFKA_BOOTSTRAP_SERVERS=kafka:9092

# Consumer settings
KAFKA_CONSUMER_GROUP_PUBLICATION=dialogue-aggregator-publication-group
KAFKA_CONSUMER_GROUP_MESSAGES=dialogue-aggregator-messages-group
KAFKA_CONSUMER_GROUP_GENERATED=dialogue-aggregator-generated-group

# Topics
KAFKA_TOPIC_PUBLICATION=publication.events
KAFKA_TOPIC_MESSAGES=messages.events
KAFKA_TOPIC_GENERATED=generated.phrases
KAFKA_TOPIC_FULL_DATA=messages.full.data
KAFKA_TOPIC_HISTORY=history.full.events
KAFKA_TOPIC_DLQ=dialogue-aggregator.dlq

# Consumer config
KAFKA_CONSUMER_AUTO_OFFSET_RESET=earliest
KAFKA_CONSUMER_ENABLE_AUTO_COMMIT=false
KAFKA_CONSUMER_MAX_POLL_RECORDS=100
KAFKA_CONSUMER_SESSION_TIMEOUT_MS=30000
KAFKA_CONSUMER_HEARTBEAT_INTERVAL_MS=10000

# Producer config
KAFKA_PRODUCER_ACKS=all
KAFKA_PRODUCER_RETRIES=3
KAFKA_PRODUCER_MAX_IN_FLIGHT_REQUESTS=5
KAFKA_PRODUCER_COMPRESSION_TYPE=snappy
```

### 6.2 Redis настройки

```env
# Redis connection
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_DB=0
REDIS_PASSWORD=
REDIS_MAX_CONNECTIONS=50

# Cache settings
REDIS_DIALOGUE_TTL_SECONDS=86400
REDIS_MESSAGES_CACHE_SIZE=50
REDIS_CONNECTION_TIMEOUT=5
REDIS_SOCKET_TIMEOUT=5
REDIS_RETRY_ON_TIMEOUT=true
```

### 6.3 Общие настройки

```env
# Service
SERVICE_NAME=dialogue-aggregator
SERVICE_VERSION=1.0.0
LOG_LEVEL=INFO
LOG_FORMAT=json

# Processing
MAX_MESSAGES_CACHE_SIZE=50
STATE_UPDATE_BATCH_SIZE=10
EVENT_PROCESSING_TIMEOUT_SECONDS=30

# Metrics
METRICS_PORT=9091
ENABLE_PROMETHEUS=true
```

---

## 7. Логирование

### 7.1 Структурированное логирование

Все логи в формате JSON для удобной обработки в Loki/Grafana:

```python
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

### 7.2 Типы логов

1. **Event Processing**:
   - Начало обработки события
   - Успешное завершение обработки
   - Ошибка обработки

2. **State Management**:
   - Загрузка состояния из Redis
   - Обновление состояния
   - Ошибки работы с Redis

3. **Kafka Operations**:
   - Публикация события
   - Ошибки публикации
   - Consumer lag

4. **Performance**:
   - Время обработки события
   - Время обращения к Redis
   - Размер кэша сообщений

---

## 8. Метрики (Prometheus)

### 8.1 Бизнес-метрики

- `dialogue_events_processed_total{event_type, status}` — общее количество обработанных событий
- `dialogue_state_updates_total{chat_id}` — обновления состояния диалога
- `dialogue_active_count` — количество активных диалогов
- `dialogue_messages_total{chat_id}` — общее количество сообщений в диалоге

### 8.2 Технические метрики

- `event_processing_duration_seconds{event_type}` — время обработки события
- `redis_operation_duration_seconds{operation}` — время операций с Redis
- `kafka_producer_duration_seconds` — время публикации в Kafka
- `kafka_consumer_lag{topic, partition}` — lag консьюмера
- `redis_connection_pool_size` — размер пула соединений Redis
- `error_count{error_type, service}` — количество ошибок по типам

### 8.3 Health checks

- `/health` — общий health check
- `/health/kafka` — проверка подключения к Kafka
- `/health/redis` — проверка подключения к Redis
- `/ready` — readiness probe (готовность к работе)

---

## 9. Docker Compose интеграция

### 9.1 Redis сервис

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
```

### 9.2 Dialogue Aggregator сервис

```yaml
dialogue-aggregator:
  build:
    context: ./backend/dialogue-aggregator
  environment:
    KAFKA_BOOTSTRAP_SERVERS: kafka:9092
    KAFKA_TOPIC_PUBLICATION: publication.events
    KAFKA_TOPIC_MESSAGES: messages.events
    KAFKA_TOPIC_GENERATED: generated.phrases
    KAFKA_TOPIC_FULL_DATA: messages.full.data
    KAFKA_TOPIC_HISTORY: history.full.events
    REDIS_HOST: redis
    REDIS_PORT: 6379
    SERVICE_NAME: dialogue-aggregator
    LOG_LEVEL: INFO
  depends_on:
    kafka:
      condition: service_started
    redis:
      condition: service_healthy
  ports:
    - "9091:9091"  # Metrics port
  healthcheck:
    test: ["CMD", "curl", "-f", "http://localhost:9091/health"]
    interval: 10s
    timeout: 5s
    retries: 3
```

---

## 10. Масштабирование

### 10.1 Горизонтальное масштабирование

- **Consumer Groups**: Kafka автоматически распределяет партиции между консьюмерами в группе
- **Stateless Processing**: Каждый инстанс Dialogue Aggregator независим
- **Redis**: Shared state storage, все инстансы работают с одним Redis

### 10.2 Ограничения

- **Partitioning**: Количество партиций определяет максимальное параллелирование
- **Redis**: Может стать узким местом при высокой нагрузке (необходим Redis Cluster)
- **State Versioning**: Механизм версионирования состояния предотвращает race conditions

---

## 11. Безопасность

### 11.1 Аутентификация Kafka

- Использование SASL/SSL для подключения к Kafka (в продакшене)
- Настройка через environment variables

### 11.2 Аутентификация Redis

- Использование AUTH для Redis (в продакшене)
- Настройка через `REDIS_PASSWORD`

### 11.3 Валидация данных

- Строгая валидация всех входящих событий через Pydantic
- Проверка типов и обязательных полей
- Защита от инъекций через санитизацию входных данных

---

## 12. Восстановление после сбоев

### 12.1 Восстановление состояния из истории

- Чтение событий из `history.full.events` при старте сервиса (опционально)
- Восстановление состояния диалогов из Redis (если TTL не истек)
- Replay последних N событий для восстановления консистентности

### 12.2 Dead Letter Queue

- События, которые не удалось обработать, отправляются в DLQ
- Возможность ручного или автоматического replay из DLQ
- Мониторинг размера DLQ для выявления проблем

---

## 13. Тестирование

### 13.1 Unit тесты

- Тестирование обработчиков событий
- Тестирование логики обогащения сообщений
- Тестирование работы с Redis (mock)

### 13.2 Integration тесты

- Тестирование с реальным Kafka (testcontainers)
- Тестирование с реальным Redis (testcontainers)
- End-to-end тесты обработки полного цикла событий

### 13.3 Performance тесты

- Нагрузочное тестирование обработки событий
- Тестирование масштабирования
- Тестирование времени отклика Redis

---

## 14. Мониторинг и алертинг

### 14.1 Ключевые метрики для алертов

- `kafka_consumer_lag > 1000` — критический lag консьюмера
- `error_count > 10 per minute` — высокий уровень ошибок
- `redis_operation_duration_seconds > 1` — медленные операции Redis
- `dialogue_active_count` — мониторинг роста активных диалогов

### 14.2 Dashboards (Grafana)

- Обзор обработки событий
- Производительность Redis
- Kafka consumer lag
- Ошибки по типам

---

Это архитектура Dialogue Aggregator для TensorTalks. Сервис обеспечивает надежную агрегацию диалогов, управление состоянием и распределение событий для последующей обработки LLM-агентом и другими компонентами системы.

