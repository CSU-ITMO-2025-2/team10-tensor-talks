## auth-service — микросервис аутентификации

`auth-service` отвечает за регистрацию, логин и работу с JWT-токенами для пользователей TensorTalks.
Сервис не имеет прямого доступа к базе данных и общается с хранилищем пользователей только
через HTTP-микросервис `user-store-service`.

### Архитектура (общая схема)

```text
        +---------------------------+
        |        bff-service        |
        |   /api/auth/... (HTTP)    |
        +-------------+-------------+
                      |
                      | HTTP /auth/...
                      v
              +-------+--------+
              |   auth-service |
              |  Gin handlers  |
              +---+-------+----+
                  |       |
     JWT (HS256)  |       |  HTTP /users...
   (access/refresh)|       v
                  |  +----+----------------+
                  |  | user-store-service  |
                  |  |   (PostgreSQL)      |
                  |  +---------------------+
                  |
                  | (в будущем: другие микросервисы
                  |  смогут валидировать JWT и
                  |  использовать GUID пользователя)
```

### Архитектура модулей

- `cmd/auth-service/main.go`  
  Точка входа: загрузка конфигурации, инициализация сервера, graceful shutdown по сигналам ОС.

- `internal/config`  
  Загрузка конфигурации через Viper из `config/config.yaml` и переменных окружения `AUTH_*`:
  - `server.host`, `server.port` — адрес HTTP-сервера;
  - `user_store.base_url`, `user_store.timeout_seconds` — параметры подключения к `user-store-service`;
  - `jwt.issuer`, `jwt.audience`, `jwt.access_token_ttl`, `jwt.refresh_token_ttl`, `jwt.secret`.

- `internal/client`  
  HTTP-клиент для взаимодействия с `user-store-service`:
  - `CreateUser(login, passwordHash)` — создание пользователя с уже захешированным паролем;
  - `GetUserByLogin(login)` — получение пользователя по логину;
  - `GetUserByID(id)` — получение пользователя по внешнему GUID.

- `internal/tokens`  
  Менеджер JWT-токенов:
  - выпускает пару токенов (access/refresh);
  - вшивает в токен GUID пользователя, логин, issuer, audience и TTL;
  - валидирует токены по подписи и сроку действия.

- `internal/service`  
  Бизнес-логика аутентификации:
  - регистрация: валидация логина/пароля, хеширование через bcrypt, создание пользователя в `user-store-service`, выпуск токенов;
  - логин: поиск пользователя по логину, проверка пароля, выпуск токенов;
  - refresh: валидация refresh-токена, получение пользователя по GUID, выпуск новой пары токенов;
  - проверка access-токена и получение пользователя по GUID.

- `internal/handler`  
  HTTP-слой на Gin:
  - `POST /auth/register` — регистрация (login, password);
  - `POST /auth/login` — логин (login, password);
  - `POST /auth/refresh` — обновление токенов по refresh-токену;
  - `GET /auth/me` — информация о текущем пользователе по access-токену.

- `internal/server`  
  Сборка зависимостей, настройка Gin-роутера и запуск HTTP-сервера, хелсчек `GET /healthz`.

### Поток данных при регистрации

1. Клиент (через BFF) вызывает `POST /auth/register` с логином и паролем.
2. `auth-service` валидирует данные, хеширует пароль и вызывает `user-store-service /users` с полями `login` и `password_hash`.
3. `user-store-service` создаёт запись в PostgreSQL, возвращает пользователя с GUID.
4. `auth-service` создаёт пару JWT-токенов (access + refresh) и возвращает их клиенту.

### Поток данных при логине

1. Клиент отправляет `POST /auth/login`.
2. `auth-service` запрашивает пользователя по логину в `user-store-service`.
3. Сравнивает bcrypt-хеш и пароль, при успехе создаёт новую пару токенов.

### Работа с JWT-токенами

- Типы токенов:
  - **access-токен** — `subject = "access"`, короткий TTL (например, 15 минут), используется для авторизации в API.
  - **refresh-токен** — `subject = "refresh"`, длинный TTL (например, 30 дней), используется только для получения новой пары токенов.
- Поля claims (`tokens.Claims`):
  - `uid` — GUID пользователя (external_id из `user-store-service`);
  - `login` — нормализованный логин пользователя;
  - стандартные поля JWT: `iss` (issuer), `aud` (audience), `iat`, `exp`, `sub`.
- Подпись:
  - алгоритм: `HS256` (HMAC + secret);
  - секрет берётся из `jwt.secret` конфигурации и не должен храниться в открытом виде в репозитории.
- Использование:
  - фронтенд хранит пару токенов (например, в `localStorage`, см. фронтенд) и передаёт access-токен в заголовке:
    - `Authorization: Bearer <access_token>`;
  - для обновления вызывается `POST /auth/refresh` с JSON `{ "refresh_token": "<refresh_token>" }`;
  - другие микросервисы в будущем могут:
    - валидировать токены через отдельный сервис или общий пакет;
    - использовать `uid` как ключ для своих доменных сущностей (чаты, сессии и т.п.).

### Безопасность

- Пароли никогда не хранятся в открытом виде — только bcrypt-хеш.
- Все операции с пользователями проходят только через `user-store-service`.
- JWT подписываются общим секретом, настраиваемым через конфигурацию.
- Логин нормализуется (lowercase + trim) для предотвращения дубликатов с разным регистром.
