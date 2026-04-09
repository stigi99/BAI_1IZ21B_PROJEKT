# Security Lab: Vulnerable vs Secure Go App

University project for the course **Network Infrastructure Security**. The application is a small Go-based web service designed to demonstrate the contrast between an insecure implementation and a secure one.

## Tech Stack

- Go
- Gin Gonic
- SQLite

## Overview

The project exposes a minimal HTTP API with a SQLite-backed database. It is intended as a security lab where the same application can be studied in two modes:

- insecure mode, where only basic checks are applied
- secure mode, where authentication and validation logic are expected to be enforced

## Run the App

### Prerequisites

- Go 1.25 or newer
- A working C toolchain, if required by the SQLite driver on your system

### Start the application

```bash
go mod tidy
go run main.go
```

The server starts on `http://localhost:8080` and creates a local SQLite database file named `app.db` automatically if it does not already exist.

### Available endpoints

- `GET /ping` - health check, returns `pong`
- `GET /posts` - returns published blog posts
- `POST /login` - accepts JSON login data

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

## Notes

- The database schema is created automatically on startup.
- Sample blog posts and users are seeded when the tables are empty.
- The current login flow is intentionally minimal so that the difference between insecure and secure handling is easy to observe.
