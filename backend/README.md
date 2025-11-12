## Backend Microservices Overview

This folder hosts the Go microservices that power the TensorTalks platform. The system follows a layered architecture that separates concerns between data persistence, authentication and frontend communication.

### Services

1. **user-store-service**
   - Acts as the only service with direct access to the credentials database.
   - Exposes a CRUD HTTP API for managing users.
   - Uses Gin for HTTP routing, GORM for PostgreSQL access, Viper for configuration, and Testify for unit tests.

2. **auth-service**
   - Handles registration, login and JWT issuance/validation.
   - Relies on the user-store-service to create and fetch users.
   - Hashes passwords with bcrypt before forwarding them to the store.

3. **bff-service**
   - Backend-for-frontend that exposes endpoints consumed by the React application.
   - Delegates authentication-related requests to the auth-service.

### Database

PostgreSQL stores credential data in a `users` table:

| Column        | Type      | Notes                               |
| ------------- | --------- | ----------------------------------- |
| id            | SERIAL PK | Internal numeric identifier         |
| external_id   | UUID      | Public GUID used across services    |
| login         | TEXT      | Unique username                     |
| password_hash | TEXT      | Bcrypt hash                         |
| created_at    | TIMESTAMP | Managed by GORM                     |
| updated_at    | TIMESTAMP | Managed by GORM                     |

### Configuration

Each service reads configuration via Viper from `config/config.yaml` and environment variables (`SERVICE_*`). Secrets such as JWT signing keys and database passwords should be supplied through environment variables in production.

### Tests

Unit tests use Testify for asserting service-specific business logic (password hashing helpers, validation, database adapters via mocks).

### Containers

- Dockerfiles exist for every service and the frontend.
- `docker-compose.yml` orchestrates:
  - React frontend
  - bff-service
  - auth-service
  - user-store-service
  - PostgreSQL


