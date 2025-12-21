## mock-interview-builder-service — сервис создания программы интервью

`mock-interview-builder-service` — сервис для создания программы интервью, работающий с Kafka очередями.

### Функциональность

- Слушает очередь `interview.build.request` на запросы создания программы интервью
- Отправляет созданную программу в очередь `interview.build.response`
- Пока что возвращает статичную программу интервью (5 вопросов с теорией)
- В будущем: динамическая генерация программы на основе параметров (topics, level, type)

### Архитектура

```
Session Manager → Kafka (interview.build.request)
                              ↓
                    Interview Builder Service
                              ↓
                    Kafka (interview.build.response)
                              ↓
                    Session Manager (сохранение программы)
```

### Конфигурация

- `MOCK_INTERVIEW_BUILDER_SERVER_HOST` — хост сервера (по умолчанию "0.0.0.0")
- `MOCK_INTERVIEW_BUILDER_SERVER_PORT` — порт сервера (по умолчанию 8089)
- `MOCK_INTERVIEW_BUILDER_KAFKA_BROKERS` — адреса Kafka брокеров
- `MOCK_INTERVIEW_BUILDER_KAFKA_TOPIC_REQUEST` — топик для запросов (по умолчанию "interview.build.request")
- `MOCK_INTERVIEW_BUILDER_KAFKA_TOPIC_RESPONSE` — топик для ответов (по умолчанию "interview.build.response")
- `MOCK_INTERVIEW_BUILDER_KAFKA_CONSUMER_GROUP` — группа consumer

### Формат программы интервью

Программа содержит упорядоченный список вопросов, каждый с текстом вопроса и теорией:

```json
{
  "questions": [
    {
      "question": "Объясните разницу между L1 и L2 регуляризацией.",
      "theory": "L1 регуляризация (Lasso) добавляет сумму абсолютных значений...",
      "order": 1
    }
  ]
}
```

### Метрики

- `tensortalks_http_requests_total` — HTTP запросы
- `tensortalks_http_request_duration_seconds` — длительность HTTP запросов

