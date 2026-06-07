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

A small Go web app with a SQLite-backed database and a server-rendered UI built with Templ + HTMX + Tailwind. The same application runs in two modes controlled by a single `SECURITY_ENABLED` flag:

- **vulnerable mode** — deliberate weaknesses for atak/obrona live demos
- **secure mode** — same code path with the proper security control enabled

Each scenario ships side-by-side: visit the page, fire a payload from the cheat-sheet drawer, watch it succeed in vuln, then flip the toggle and watch it fail in secure. The differences are confined to small `if h.securityEnabled { ... }` branches so the diff between modes is reviewable in seconds.

UI extras layered on top (from the claude.ai/design handoff): sakura petals, Burp-style Request Inspector, Attack Timeline that exports a Markdown PoC, mascot reacting to the current mode, and a 14-section payload cheat-sheet drawer with live filter.

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
go run .
```

> Use `go run .` (compiles every `.go` file in the package), not `go run main.go` — `main.go` references symbols defined in tests and other files.

The server starts on `http://localhost:8080` and creates a local SQLite database (`app.db`) automatically with seeded data: 3 demo posts (Welcome / SQL Injection 101 / Stored XSS demo) and 3 users (`admin`, `user1`, `alice`).

Optional environment variables:

- `PORT` (default: `:8080`)
- `DB_PATH` (default: `app.db`)
- `SECURITY_ENABLED` (`true` or `false`, default: `false`)
- `ADMIN_USERNAME` (default: `admin`)
- `ADMIN_PASSWORD` (default: `admin`)
- `ADMIN_EMAIL` (default: `admin@example.com`)

Run in **vulnerable mode** (default) for the live attack demos:

```bash
SECURITY_ENABLED=false go run .
```

Run in **secure mode** to prove the same attacks are blocked:

```bash
SECURITY_ENABLED=true go run .
```

> Tip: delete `app.db` between mode switches if you want a fresh seed (passwords are stored differently per mode).

### Available endpoints

JSON API:

- `GET /ping` — health check, returns `pong`
- `GET /posts` — published blog posts
- `POST /posts`, `PUT /posts/:id`, `DELETE /posts/:id` — CRUD (login required)
- `POST /login`, `POST /register`, `POST /logout`
- `GET /api/search?q=...` — SQL Injection demo, honors `SECURITY_ENABLED`
- `GET /api/search-vulnerable?q=...` — force-vulnerable SQLi (always concatenated, for side-by-side demo)
- `POST /api/comments-vulnerable` — force-vulnerable XSS (always stores raw HTML)
- `POST /api/comments-secure` — sanitized counterpart for the comments demo
- `GET/POST /csrf-vulnerable-form` — CSRF demo form (no token validation)
- `GET /debug/crash` — deliberate panic for Security Misconfiguration demo
- `GET /api/files-vulnerable?name=...`, `GET /api/files-secure?name=...` — Path Traversal / LFI vulnerable/secure comparison
- `GET /api/ping-vulnerable?host=...`, `GET /api/ping-secure?host=...` — Command Injection vulnerable/secure comparison

UI routes:

- `GET /` — redirects to `/ui/posts`
- `GET /ui/posts` — posts list with create/edit/delete
- `GET /ui/posts/view/:id` — single post + comments (Stored XSS demo)
- `POST /ui/posts/view/:id/comments` — submit comment (HTMX)
- `GET /ui/posts/edit/:id` — edit form
- `GET /ui/login`, `POST /ui/login`
- `GET /ui/register`, `POST /ui/register`
- `GET /ui/search` — SQL Injection demo with payload hints
- `GET /ui/csrf-demo`, `GET /ui/csrf-secure` — CSRF vulnerable/secure comparison
- `GET /ui/idor-demo` — Broken Access Control / IDOR demo
- `GET /ui/db-expose` — Sensitive Data Exposure demo
- `GET /ui/path-traversal` — Path Traversal / LFI demo
- `GET /ui/cmd-injection` — Command Injection demo
- `GET /ui/vuln-demos` — hub with all vulnerability scenarios (CWE/OWASP labelled)
- `GET /ui/csrf-demo` — CSRF vulnerable form (no token)
- `GET /ui/csrf-secure` — CSRF protected form with per-form token
- `GET /ui/idor-demo` — Broken Access Control demo (IDOR)
- `GET /ui/db-expose` — Sensitive Data Exposure demo (dumps user table)
- `GET /ui/path-traversal` — Path Traversal demo page
- `GET /ui/cmd-injection` — Command Injection demo page

HTMX partials:

- `POST /ui/partials/login`, `POST /ui/partials/register`
- `GET /ui/partials/posts`, `POST /ui/partials/posts/create`
- `POST /ui/partials/search`
- `POST /ui/partials/posts/view/:id/comments`

Example login request:

```bash
curl -X POST http://localhost:8080/login \
	-H "Content-Type: application/json" \
	-d '{"username":"admin","password":"admin"}'
```

Example SQL Injection (in vulnerable mode):

```bash
curl "http://localhost:8080/api/search?q=' OR 1=1 --"
# leaks every row including drafts

curl "http://localhost:8080/api/search-vulnerable?q=' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --"
# exfiltrates the users table
```

## Security Toggle Mechanism

The application uses a global flag named `SecurityEnabled` in [main.go](main.go) to represent the security mode.

- When `SecurityEnabled` is `false`, the app behaves in insecure mode.
- When `SecurityEnabled` is `true`, the code is meant to enforce secure handling such as authentication checks, password verification, and stronger input validation.

In the current codebase, this flag acts as the central switch for the lab scenario. It makes it easy to compare insecure and secure behaviour in one project without changing the API shape.

The UI header includes a runtime toggle button:

- vulnerable mode shows `Vulnerable` and a `↔ Secure` button
- secure mode shows `Secure` and a `↔ Vulnerable` button

The button posts to `POST /ui/mode/toggle`, updates the global handler/service mode, and redirects back to the current page. For storage-dependent demos such as plaintext-vs-bcrypt password storage, use a fresh database/restart when you need a clean seed comparison.

## Demo Screenshots

Evidence screenshots are stored under `docs/screenshots/`:

| File | What it shows |
|------|---------------|
| `01-vuln-demos-vulnerable.jpg` | Vulnerability hub in vulnerable mode |
| `02-sqli-vulnerable-results.jpg` | SQL Injection payload returning vulnerable results |
| `03-path-traversal-vulnerable.jpg` | Path Traversal reading outside `uploads` |
| `04-command-injection-vulnerable.jpg` | Command Injection executing extra shell input |
| `05-vuln-demos-secure-after-toggle.jpg` | Same hub after using the header toggle |
| `06-sqli-secure-blocked.jpg` | SQL Injection blocked by prepared statements |
| `07-path-traversal-secure-blocked.jpg` | Path Traversal blocked by path validation |
| `08-command-injection-secure-blocked.jpg` | Command Injection blocked by host validation/no shell |

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

## Implemented Vulnerabilities (Stage E — complete ✅, with bonuses)

**9 vulnerabilities** end-to-end with integration tests. An n=3 team needs 7 (2 mandatory + 5 additional); the last two are extras.

| # | Vulnerability | CWE / OWASP | Demo route |
|---|---------------|-------------|-----------|
| 1 | SQL Injection | CWE-89 / A03:2021 | `/ui/search`, `/api/search-vulnerable` |
| 2 | Stored XSS | CWE-79 / A03:2021 | `/ui/posts/view/1` (seeded XSS comment) |
| 3 | Broken Authentication | CWE-287 / A07:2021 | `/ui/login` (any password works in vuln mode) |
| 4 | Broken Access Control | CWE-639 / A01:2021 | `/ui/moderation`, `/ui/idor-demo` |
| 5 | Sensitive Data Exposure | CWE-200 / A02:2021 | `/ui/members`, `/ui/db-expose` |
| 6 | CSRF | CWE-352 / A01:2021 | `/ui/profile`, `/ui/csrf-demo`, `/ui/csrf-secure` |
| 7 | Security Misconfiguration | CWE-16 / A05:2021 | `/debug/crash` (vuln: stack trace; secure: clean 500 + CSP/HSTS) |
| 8 | Path Traversal / LFI ★ | CWE-22 / A01:2021 | `/ui/gallery`, `/api/files-vulnerable` vs `/api/files-secure` |
| 9 | Command Injection ★ | CWE-78 / A03:2021 | `/ui/stream-check`, `/api/ping-vulnerable` vs `/api/ping-secure` |

★ Bonus — above the n=3 requirement.

See `PLAN_IMPLEMENTACJI_PODATNOSCI.md` for full per-vulnerability documentation (description, PoC, before/after diff).

## What Changed Recently

Sprint 1-2 (project setup):
1. Stage A — project split into `internal/*` packages
2. Stage B — Templ UI extracted to `.templ` files
3. Stage C — HTMX partial routes for posts/login/register
4. Stage D — Tailwind pipeline + styled views + design handoff from claude.ai/design

Sprint 3 (vulnerability scenarios + UI polish, 2026-05-03 → 2026-05-08):
5. Foundation hardening: bcrypt auth, HTTP-only session cookie, env-driven admin seed, post attachments (multipart upload, 5 MB limit)
6. SQL Injection demo end-to-end: `/api/search` (toggle), `/api/search-vulnerable` (force-vuln), `/ui/search` page with payload hints, integration tests for both modes
7. Stored XSS demo end-to-end: comments table, `/ui/posts/view/:id` page, `@templ.Raw` (vuln) vs `{ }` auto-escape (secure) + server-side HTML strip, integration tests
8. Vuln Demos hub `/ui/vuln-demos` — one-stop nav with CWE/OWASP labels and copy-pastable payloads
9. UI polish: two-column hero on Login/Register, refactored post cards with `Read more →`, cheat-sheet drawer with filter and 14 sections (SQLi, XSS, IDOR, Path, CmdInj, SSRF, CSRF, Auth, SDE, Misconfig, Upload, Burp, Glossary)
10. Tailwind config fix — content globs now scan `.go` files (previously only `.templ`/`*_templ.go`), so utility classes referenced from `layout_helpers.go` no longer get purged

Sprint 4 (Etap E completion, 2026-05-15) — **Stage E done with bonuses**:
11. CSRF (Krok 7) — vuln endpoints `/csrf-vulnerable-form` and `/ui/csrf-demo`; secure endpoint `/ui/csrf-secure` with per-form CSRF token (cookie `bai_csrf_token` + hidden `csrf_token` input validated before email update).
12. Security Misconfiguration (Krok 8) — `SecurityHeadersMiddleware` sets CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy in secure mode. `ErrorSanitizerMiddleware` recovers from panics with a clean JSON 500 (vuln mode keeps the colourful Gin stack trace). New `/debug/crash` endpoint deliberately panics for the side-by-side demo. 4 integration tests.
13. Bonus: Path Traversal / LFI — `/api/files-vulnerable` joins user input via `filepath.Join("./uploads", name)`; `/api/files-secure` rejects `..` and verifies the resolved path stays under `./uploads`.
14. Bonus: Command Injection — `/api/ping-vulnerable` runs `sh -c "ping " + host`; `/api/ping-secure` uses `exec.Command("ping","-c1",host)` + a hostname regex whitelist.
15. Bug fix — `/ui/posts` showed two stacked "log in" banners (template's `if !loggedIn` branch + handler-injected `ResultMessage`). Handler no longer injects the redundant message.
16. Vuln Demos hub now lists 9 ready scenarios (added CSRF, Security Misconfiguration, Path Traversal, Command Injection cards).

## Next Steps (Stage F — finalization)

1. Run one full live-demo rehearsal: attack in vulnerable mode -> repeat in secure mode -> show code difference.
2. Export/attach the screenshots from `docs/screenshots/` and generated reports to the final submission package.
3. Dry-run defense presentation with a stopwatch — confirm the main scenarios run cleanly in the available time.
