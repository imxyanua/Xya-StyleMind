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

- Auth: register, login, current user
- Categories: public list + admin create
- Products: public list/detail + admin CRUD
- Cart: authenticated user cart operations
- Orders: checkout, user order history/detail, admin status update
- Health: service and DB health check

### Development Status

The backend foundation is implemented and ready for feature expansion.

Implemented:

- JWT authentication and role middleware
- Product/catalog/cart/order flow
- Auto migration runner
- Dockerized local development support

Planned next:

- Wishlist
- AI recommendation
- Smart search
- Frontend integration hardening

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

3. Build backend:

```bash
cd backend
go build ./...
```

4. Health check:

```bash
curl http://localhost:8080/api/v1/health
```

### Security & Secrets

- Never commit real `.env` files to the repository.
- Never commit real API keys, database passwords, JWT secrets, or cloud secrets.
- If any secret was ever committed, rotate it immediately.

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
