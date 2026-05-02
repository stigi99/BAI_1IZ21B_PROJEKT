# Security Lab: Vulnerable vs Secure Go App

University project for the course **Network Infrastructure Security**. The application is a small Go-based web service designed to demonstrate the contrast between an insecure implementation and a secure one.

## Tech Stack

- Go
- Gin Gonic
- SQLite
- Templ (server-side HTML views)
- HTMX (partial page interactions)
- Tailwind CSS (build pipeline + styling)

## Overview

The project exposes a minimal HTTP API with a SQLite-backed database and a server-rendered UI. It is intended as a security lab where the same application can be studied in two modes:

- insecure mode, where only basic checks are applied
- secure mode, where authentication and validation logic are expected to be enforced

## Run the App

### Prerequisites

- Go 1.25 or newer
- A working C toolchain, if required by the SQLite driver on your system
- Node.js + npm (for Tailwind build)

### Start the application

```bash
go mod tidy
npm install
npm run build:css
go run main.go
```

The server starts on `http://localhost:8080` and creates a local SQLite database file named `app.db` automatically if it does not already exist.

Optional environment variables:

- `PORT` (default: `:8080`)
- `DB_PATH` (default: `app.db`)
- `SECURITY_ENABLED` (`true` or `false`, default: `false`)
- `ADMIN_USERNAME` (default: `admin`)
- `ADMIN_PASSWORD` (default: `admin`)
- `ADMIN_EMAIL` (default: `admin@example.com`)

### Available endpoints

- `GET /ping` - health check, returns `pong`
- `GET /posts` - returns published blog posts
- `POST /login` - accepts JSON login data
- `POST /register` - creates a new regular user account

### Available UI routes

- `GET /` - redirects to `/ui/posts`
- `GET /ui/posts` - server-rendered posts view
- `GET /ui/login` - server-rendered login view
- `POST /ui/login` - login form submit
- `GET /ui/partials/posts` - HTMX partial for posts list refresh
- `POST /ui/partials/login` - HTMX partial for login result

Example login request:

```bash
curl -X POST http://localhost:8080/login \
	-H "Content-Type: application/json" \
	-d '{"username":"admin","password":"secret"}'
```

## Security Toggle Mechanism

The application uses a global flag named `SecurityEnabled` in [main.go](main.go) to represent the security mode.

- When `SecurityEnabled` is `false`, the app behaves in insecure mode.
- When `SecurityEnabled` is `true`, the code is meant to enforce secure handling such as authentication checks, password verification, and stronger input validation.

In the current codebase, this flag acts as the central switch for the lab scenario. It makes it easy to compare insecure and secure behaviour in one project without changing the API shape.

## Admin Account Bootstrap

On startup, the app ensures that an admin account exists in the `users` table.

- Default admin credentials: `admin` / `admin`
- You can override them with `ADMIN_USERNAME`, `ADMIN_PASSWORD`, and `ADMIN_EMAIL`
- The ensured admin account always has role `admin`

In secure mode, admins can delete any post. Regular users can delete only posts where they are the author.

## Notes

- The database schema is created automatically on startup.
- Sample blog posts and users are seeded when the tables are empty.
- The current login flow is intentionally minimal so that the difference between insecure and secure handling is easy to observe.

## Current Architecture (after refactor)

- `main.go` - app bootstrap + router wiring
- `internal/config` - app configuration loading
- `internal/db` - DB init/migration/seed
- `internal/service` - business/data access layer
- `internal/handlers` - JSON + UI handlers
- `internal/views/pages.templ` - Templ view definitions
- `assets/css/input.css` - Tailwind source styles
- `static/css/app.css` - generated Tailwind output served by app

## Templ Workflow

Views are authored in `.templ` files and compiled to Go code.

Generate code manually:

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./internal/views
```

Run integration tests:

```bash
go test -tags=integration -v .
```

## Tailwind Workflow

Build CSS once:

```bash
npm run build:css
```

Watch during development:

```bash
npm run watch:css
```

## What Changed Recently

1. Stage A completed (project split into `internal/*` packages).
2. Stage B completed (Templ UI implemented and separated to `.templ` file).
3. UI integration tests added for `/ui/posts` and `/ui/login`.
4. Service improved with `rows.Err()` check after iteration.
5. Rendering error handling in handlers unified and no longer ignored.
6. Stage C completed with HTMX partial routes for posts and login.
7. Stage D completed with Tailwind pipeline and styled Templ views.
8. Static assets route added (`/static`) and app CSS served from generated file.

## Next Steps

1. Stage E: implement first vulnerable/secure security scenarios.
2. Expand HTMX UX (loading/error states for more flows).
3. Harden current SQLite-based app and finalize defense materials.
