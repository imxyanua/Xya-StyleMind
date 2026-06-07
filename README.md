## Xya-StyleMind

AI-powered fashion ecommerce platform.

### Tech Stack

- Frontend: Next.js, TypeScript, TailwindCSS, shadcn/ui
- Backend: Go, Gin
- Database: PostgreSQL
- Infra: Docker Compose, Nginx, VPS
- AI: Gemini API

### Core Features

- Authentication and role-based access control
- Product catalog management
- Category management
- Product filtering and detail view
- Shopping cart
- Checkout and order workflow
- Wishlist / favorite products
- Product reviews and ratings
- Admin routes for management operations
- Future: AI outfit recommendation
- Future: Smart search

### Architecture Overview

The project follows a modular architecture with clear feature boundaries across backend modules:

- `auth`
- `category`
- `product`
- `cart`
- `order`

Backend uses a layered pattern:

- Handler layer: HTTP request/response
- Service layer: business logic
- Repository layer: data access via `pgx`

Database migrations are auto-applied at startup by the migration runner.

### API Overview

Current API groups are versioned under `/api/v1`:

- Auth: register, login, logout, current user
- Categories: public list + admin create
- Products: public list/detail + admin CRUD
- Cart: authenticated user cart operations
- Orders: checkout, user order history/detail, admin status update
- Wishlist: authenticated user favorite products
- Reviews: verified-purchase product reviews, review list, rating summary
- Health: `/healthz`, `/livez`, `/readyz`, and legacy `/api/v1/health`

### API Documentation

The backend API contract is documented in OpenAPI format:

- [`docs/openapi.yaml`](docs/openapi.yaml)

Preview locally with Redoc:

```bash
npx @redocly/cli preview-docs docs/openapi.yaml
```

Or preview with Swagger UI:

```bash
docker run --rm -p 8081:8080 -e SWAGGER_JSON=/docs/openapi.yaml -v ${PWD}/docs:/docs swaggerapi/swagger-ui
```

Notes:

- Examples use placeholder tokens only.
- Protected routes require `Authorization: Bearer <access_token>`.
- Admin routes require an authenticated user with role `admin`.
- `/metrics` uses Prometheus text format and should be protected by internal networking or reverse proxy rules in production.

### Development Status

The backend foundation is implemented and ready for feature expansion.

Implemented:

- JWT authentication and role middleware
- JWT registered claims validation and Redis-backed token revocation
- Product/catalog/cart/order flow
- Wishlist/favorite product flow
- Verified-purchase product review and rating flow
- Advanced product search/filter/sort with rating-aware listing
- Auto migration runner
- Dockerized local development support with PostgreSQL and Redis

Planned next:

- Frontend product search/filter integration
- Frontend UI polish
- AI recommendation

### How To Run Locally

1. Start infrastructure:

```bash
docker compose up --build -d
```

2. Start backend directly (optional local dev mode):

```bash
cd backend
go run ./cmd/server
```

3. Seed development data (idempotent, non-destructive):

```bash
cd backend
go run ./cmd/seed
```

4. Build backend:

```bash
cd backend
go build ./...
```

5. Health checks:

```bash
curl http://localhost:8080/livez
curl http://localhost:8080/readyz
```

### Seed Data Notes

- Development seed inserts 6 categories and 36 products.
- Product data includes diverse styles/colors, varied stock, and VND pricing.
- Seed is idempotent via deterministic IDs and upsert logic, so running multiple times will not duplicate records.
- Seed does not reset or delete existing database data.

### Backend Testing

Run unit/API tests:

```bash
cd backend
go test ./...
```

Run backend build verification:

```bash
cd backend
go build ./...
```

Run integration tests with PostgreSQL:

```bash
cd backend
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/stylemind?sslmode=disable" go test -tags=integration ./internal/order ./internal/auth ./internal/product ./internal/cart ./internal/wishlist ./internal/review
```

Integration tests are opt-in. If `TEST_DATABASE_URL` is not set, they skip safely.

### Frontend Verification

Run OpenAPI type generation, lint, typecheck, and production build:

```bash
cd frontend
npm ci
npm run generate:openapi
npm run lint
npx tsc --noEmit
npm run build
```

Run Playwright E2E locally after PostgreSQL and Redis are available:

```bash
cd backend
go run ./cmd/seed

cd ../frontend
npm run test:e2e
```

Playwright starts the backend and frontend dev servers automatically. For a custom test database or Redis instance, set:

- `E2E_DB_HOST`
- `E2E_DB_PORT`
- `E2E_DB_USER`
- `E2E_DB_PASSWORD`
- `E2E_DB_NAME`
- `E2E_REDIS_ADDR`

### CI

GitHub Actions runs the main quality gates in `.github/workflows/ci.yml`:

- Backend: `go test ./...`, `go build ./...`, and integration tests with PostgreSQL/Redis services.
- Frontend: `npm ci`, OpenAPI type generation, generated-type diff check, lint, `npx tsc --noEmit`, and `npm run build`.
- E2E: PostgreSQL + Redis services, backend seed runner, then Playwright against real backend/frontend servers.

CI uses only placeholder test environment values. No production secrets are required.

### Security & Secrets

- Never commit real `.env` files to the repository.
- Never commit real API keys, database passwords, JWT secrets, or cloud secrets.
- If any secret was ever committed, rotate it immediately.
- Redis is used in production-style deployments for auth rate limiting and JWT `jti` revocation.
- Logout stores only the token `jti` blacklist key with TTL; raw access tokens are never stored.
- API responses include baseline security headers, and request bodies are capped by `MAX_REQUEST_BODY_BYTES`.
- API request contexts have a configurable deadline via `REQUEST_TIMEOUT_SECONDS`.
- `/metrics` exposes Prometheus-compatible HTTP metrics. In production, keep it behind an internal network or reverse proxy allowlist.
- `/readyz` checks Postgres and configured Redis. Docker healthchecks use `/readyz` so the backend is healthy only when dependencies are ready.

### Public Repository Notes

- This public repository is source-code focused.
- Database schema migration SQL files are committed by design.
- Real database dumps containing real user/business data must not be committed.
- Internal/private working notes should stay out of public history.

### Environment Variables

Use `backend/.env.example` and `frontend/.env.example` as templates for local setup.

Backend key variables:

- `PORT`
- `DB_HOST`
- `DB_PORT`
- `DB_USER`
- `DB_PASSWORD`
- `DB_NAME`
- `JWT_SECRET`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `REQUEST_TIMEOUT_SECONDS`
- `MAX_REQUEST_BODY_BYTES`
- `REDIS_ADDR`
- `REDIS_PASSWORD`
- `REDIS_DB`
- `GEMINI_API_KEY`
- `OPENAI_API_KEY`
- `OPENAI_MODEL`
- `ANTHROPIC_API_KEY`
- `CLAUDE_MODEL`
- `CLOUDINARY_CLOUD_NAME`
- `CLOUDINARY_API_KEY`
- `CLOUDINARY_API_SECRET`

Frontend key variable:

- `NEXT_PUBLIC_API_BASE_URL`

### Database Notes

- Committing migration schema SQL is normal and expected.
- Do not commit database dumps with real data (`*.dump`, `*.sql.dump`, backups).
- For local development, initialize from migrations and local `.env` config only.

### Resume / CV Description

Built **Xya-StyleMind**, an AI-powered fashion ecommerce platform with a production-oriented backend foundation using **Go (Gin)** and **PostgreSQL**, featuring modular service/repository architecture, JWT-based authentication, role-based admin access, product catalog, cart, checkout/order workflows, automated database migrations, and Docker-based deployment workflow prepared for VPS + Nginx environments.
