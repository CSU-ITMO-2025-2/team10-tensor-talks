## bff-service — Backend-for-Frontend

`bff-service` — это слой между браузером и внутренними микросервисами.  
Он предоставляет простой и стабильный HTTP API для фронтенда и проксирует
запросы аутентификации в `auth-service`.

### Архитектура (общая схема)

```text
   Браузер / Frontend           bff-service                 auth-service
+---------------------+   +------------------+   HTTP      +--------------------+
|  React SPA          |   |  /api/auth/...  |  /auth/...  | /auth/register     |
|  (Vite, TS, Router) +--->  Gin Handlers   +-------------> /auth/login        |
+---------------------+   |  CORS, JSON     |             | /auth/refresh      |
                          +------------------+             | /auth/me          |
                                   ^                       +---------+---------+
                                   |                                 |
                                   | HTTP /healthz                   |
                                   +---------------------------------+
```

### Роль в архитектуре

- единственная точка входа для фронтенда (`/api/...`);
- скрывает внутренние адреса и схемы auth/user-store сервисов;
- настраивает CORS и отвечает за “web-friendly” ответы и коды ошибок.

### Архитектура

- `cmd/bff-service/main.go`  
  Точка входа: загрузка конфигурации, инициализация сервера, graceful shutdown.

- `internal/config`  
  Конфигурация через Viper (`config/config.yaml` + `BFF_*` из окружения):
  - `server.host`, `server.port` — HTTP-сервер;
  - `auth_service.base_url`, `auth_service.timeout_seconds` — подключение к `auth-service`;
  - `session_service.base_url`, `session_service.timeout_seconds` — подключение к `session-service`;
  - `kafka.brokers`, `kafka.topic_chat_out`, `kafka.topic_chat_in`, `kafka.consumer_group` — настройки Kafka;
  - `cors.allow_origins`, `cors.allow_headers` — настройки CORS.

- `internal/client`  
  HTTP-клиенты к другим сервисам:
  - `AuthClient` — клиент к `auth-service`:
    - `Register(login, password)` — регистрация;
    - `Login(login, password)` — логин;
    - `Refresh(refreshToken)` — обновление токенов;
    - `Me(accessToken)` — получение текущего пользователя.
  - `SessionClient` — клиент к `session-service`:
    - `CreateSession(userID)` — создание новой сессии чата.

- `internal/service`  
  Бизнес-слой BFF:
  - `AuthService` — для аутентификации:
    - оборачивает ошибки из `auth-service` в доменные ошибки `ErrInvalidCredentials`, `ErrConflict`, `ErrBadRequest`;
    - предоставляет методы `Register`, `Login`, `Refresh`, `CurrentUser` для HTTP-слоя.
  - `ChatService` — для управления чатами:
    - создаёт сессии через `session-service`;
    - отправляет события в Kafka (`chat.events.out`);
    - читает события от модели из Kafka (`chat.events.in`);
    - управляет очередью вопросов и результатами чатов.

- `internal/handler`  
  HTTP-слой на Gin, маршруты под `/api`:
  - `POST /api/auth/register` — регистрация;
  - `POST /api/auth/login` — логин;
  - `POST /api/auth/refresh` — обновление токенов;
  - `GET /api/auth/me` — информация о текущем пользователе по access-токену;
  - `POST /api/chat/start` — начать новый чат;
  - `POST /api/chat/message` — отправить сообщение в чат;
  - `GET /api/chat/:session_id/question` — получить следующий вопрос (polling);
  - `GET /api/chat/:session_id/results` — получить результаты чата.

- `internal/middleware`  
  - CORS-мидлвара, сконфигурированная из `CORSConfig`.

- `internal/server`  
  Сборка зависимостей, регистрация маршрутов, health-check `GET /healthz` и запуск HTTP-сервера.

### Работа с токенами в BFF

- BFF **не создаёт и не валидирует** JWT самостоятельно — он делегирует эти задачи `auth-service`.
- Поток работы с токенами:
  - при регистрации/логине BFF вызывает соответствующие эндпоинты `auth-service` и просто отдаёт клиенту
    JSON с полями `user` и `tokens` (access/refresh);
  - при `GET /api/auth/me` BFF:
    - извлекает access-токен из заголовка `Authorization: Bearer ...`;
    - вызывает `auth-service /auth/me` и возвращает данные пользователя;
  - при `POST /api/auth/refresh` BFF:
    - принимает `refresh_token` от фронтенда;
    - вызывает `auth-service /auth/refresh` и отдаёт новую пару токенов.
- Валидация токенов и все проверки (срок действия, подпись, тип токена) происходят только в `auth-service`.

### Работа с чатами

BFF управляет чатами через:
- `session-service` — для создания сессий;
- Kafka — для асинхронной обработки событий чатов;
- `mock-model-service` — обрабатывает события и генерирует вопросы/результаты.

Подробнее см. [CHAT_IMPLEMENTATION.md](../CHAT_IMPLEMENTATION.md) и [KAFKA.md](../KAFKA.md).

### Ограничения

- BFF **не** обращается напрямую к `user-store-service` и не работает с базой данных.
- Любой доступ к учётным записям идёт через `auth-service`.
- Любые внутренние или отладочные эндпоинты других микросервисов (например, `/debug/users`) 
  не должны проксироваться через BFF во внешний мир.


