# Реализация MVP логики чатов

## Обзор

Реализована базовая MVP логика работы с активными чатами для платформы TensorTalks.

## Архитектура

```
Пользователь → Frontend → BFF → Session Service (создание сессии)
                              ↓
                         Kafka Producer (chat.events.out)
                              ↓
                         Kafka Broker
                              ↓
                         Model Service (будущее)
                              ↓
                         Kafka Consumer (chat.events.in)
                              ↓
                         BFF → Frontend → Пользователь
```

## Компоненты

### 1. session-service

Микросервис-заглушка для управления сессиями чатов.

**Эндпоинты:**
- `POST /sessions` — создание новой сессии
  - Request: `{ "user_id": "uuid" }`
  - Response: `{ "session_id": "uuid" }`

**Реализация:**
- Пока просто генерирует UUID для сессии
- В будущем будет хранить сессии в БД и управлять их жизненным циклом

### 2. Kafka очереди

#### Топик `chat.events.out` (BFF → Модель)

**События:**
1. `chat.started` — начало нового чата
   - Payload: `session_id`, `user_id`, `started_at`

2. `chat.user_message` — сообщение от пользователя
   - Payload: `session_id`, `user_id`, `content`, `message_id`, `timestamp`

#### Топик `chat.events.in` (Модель → BFF)

**События:**
1. `chat.model_question` — вопрос от модели
   - Payload: `session_id`, `user_id`, `question`, `question_id`, `timestamp`

2. `chat.completed` — завершение чата с результатами
   - Payload: `session_id`, `user_id`, `results` (score, feedback, recommendations), `completed_at`

### 3. BFF Service

**Новые эндпоинты:**
- `POST /api/chat/start` — начать новый чат
  - Request: `{ "user_id": "uuid" }`
  - Response: `{ "session_id": "uuid" }`

- `POST /api/chat/message` — отправить сообщение
  - Request: `{ "session_id": "uuid", "user_id": "uuid", "content": "текст" }`
  - Response: `{ "status": "ok" }`

**Логика:**
1. При старте чата:
   - BFF запрашивает сессию у `session-service`
   - BFF отправляет событие `chat.started` в Kafka

2. При отправке сообщения:
   - BFF отправляет событие `chat.user_message` в Kafka

3. При получении события от модели:
   - BFF читает из Kafka топика `chat.events.in`
   - Обрабатывает события `chat.model_question` и `chat.completed`
   - В будущем будет отправлять через WebSocket клиенту

### 4. Frontend

**Обновления:**
- `src/services/chat.ts` — новый сервис для работы с чатами
- `src/pages/Chat.tsx` — обновлен для работы с реальным API
- `src/pages/Dashboard.tsx` — добавлена кнопка "Начать новое интервью"

**Функциональность:**
- Автоматическое создание сессии при открытии чата
- Отправка сообщений через API
- Отображение сообщений в реальном времени (пока заглушка)
- Ожидание вопросов от модели (пока заглушка)

## Пайплайн работы

### Старт чата

1. Пользователь нажимает "Начать новое интервью" в Dashboard
2. Frontend вызывает `POST /api/chat/start` с `user_id`
3. BFF запрашивает сессию у `session-service`
4. BFF отправляет событие `chat.started` в Kafka (`chat.events.out`)
5. Frontend получает `session_id` и переходит на `/chat/{session_id}`

### Отправка сообщения

1. Пользователь вводит ответ и нажимает "Отправить"
2. Frontend вызывает `POST /api/chat/message` с `session_id`, `user_id`, `content`
3. BFF отправляет событие `chat.user_message` в Kafka (`chat.events.out`)
4. Frontend отображает сообщение в чате

### Получение вопроса от модели

1. Model Service (в будущем) обрабатывает событие и генерирует вопрос
2. Model Service отправляет событие `chat.model_question` в Kafka (`chat.events.in`)
3. BFF Consumer получает событие и обрабатывает его
4. В будущем: BFF отправляет вопрос через WebSocket клиенту
5. Frontend отображает вопрос в чате

### Завершение чата

1. Model Service отправляет событие `chat.completed` в Kafka (`chat.events.in`)
2. BFF Consumer получает событие с результатами
3. В будущем: BFF отправляет результаты через WebSocket клиенту
4. Frontend отображает результаты и перенаправляет на страницу результатов

## Метрики

Добавлены метрики для мониторинга:

- `tensortalks_kafka_messages_produced_total` — количество отправленных сообщений в Kafka
- `tensortalks_kafka_messages_consumed_total` — количество полученных сообщений из Kafka
- `tensortalks_kafka_message_processing_duration_seconds` — длительность обработки сообщений
- `tensortalks_business_sessions_created_total` — количество созданных сессий

## Логирование

Все операции логируются:
- Создание сессий
- Отправка/получение сообщений в Kafka
- Обработка событий
- Ошибки

## Mock Model Service

Реализована умная заглушка `mock-model-service`, которая:
- Читает события из `chat.events.out` (chat.started, chat.user_message)
- Отправляет события в `chat.events.in` (chat.model_question, chat.completed)
- Отслеживает состояние сессий и количество заданных вопросов
- Автоматически завершает чат после заданного количества вопросов (по умолчанию 5)
- Использует статичные вопросы из набора по ML
- Генерирует оценки, обратную связь и рекомендации

Подробнее см. [mock-model-service/README.md](./mock-model-service/README.md)

## Полный пайплайн работы

### 1. Старт чата
```
Frontend → BFF → Session Service (создание сессии)
         ↓
    Kafka (chat.events.out: chat.started)
         ↓
    Mock Model Service → Kafka (chat.events.in: chat.model_question)
         ↓
    BFF Consumer → ChatService (добавление вопроса в очередь)
         ↓
    Frontend (polling) → BFF → ChatService (получение вопроса)
         ↓
    Frontend отображает вопрос
```

### 2. Отправка ответа
```
Frontend → BFF → Kafka (chat.events.out: chat.user_message)
         ↓
    Mock Model Service (проверка счётчика)
         ↓
    Если < max_questions:
        → Kafka (chat.events.in: chat.model_question)
        → BFF Consumer → ChatService
        → Frontend (polling) получает вопрос
    Если >= max_questions:
        → Kafka (chat.events.in: chat.completed)
        → BFF Consumer → ChatService (сохранение результатов)
        → Frontend (polling) получает результаты
```

## Следующие шаги

1. **WebSocket интеграция** — для real-time обновлений вместо polling
2. **Хранение сессий** — добавить БД в `session-service`
3. **История чатов** — сохранять сообщения и результаты
4. **Реальная модель** — заменить mock-model-service на реальную AI-модель
5. **Обработка ошибок** — улучшить обработку ошибок Kafka и retry логику
