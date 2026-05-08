# Plan Dzialania: Go + Templ + HTMX + Tailwind + SQLite

> Aktualizacja: 2026-05-08. Etapy A-D zakonczone, Etap E w trakcie (5 z 7 podatnosci wymaganych dla wariantu n=3 zaimplementowane), Etap F (finalizacja) — czeka na CSRF + Security Misconfiguration + sekcje raportu.

## 1) Gdzie jestesmy teraz (stan aktualny)

### Backend i routing
- [x] Backend w Go (Gin) — `main.go` + `internal/{config,db,handlers,service,views}`
- [x] Security toggle: `SecurityEnabled` (vulnerable/secure) sterowany env `SECURITY_ENABLED`
- [x] Endpointy JSON: `GET /ping`, `GET /posts`, `POST /posts`, `PUT/DELETE /posts/:id`, `POST /login`, `POST /register`, `POST /logout`
- [x] Endpointy SQLi demo: `GET /api/search` (toggle), `GET /api/search-vulnerable` (force-vuln)
- [x] Endpointy Stored XSS demo: `GET /ui/posts/view/:id`, `POST /ui/posts/view/:id/comments`, `POST /api/comments-vulnerable` (force-vuln)
- [x] Endpoint CSRF demo: `GET/POST /csrf-vulnerable-form` (vuln only — secure brak)
- [x] Cookie sesji `bai_auth_user` (HTTP-only, TTL 8h)
- [x] Hardcoded admin seedowany z env (`ADMIN_USERNAME`/`ADMIN_PASSWORD`/`ADMIN_EMAIL`)

### Warstwa widokow
- [x] Templ — `internal/views/pages.templ` z layoutem, header, footer, navem
- [x] Strony UI: `/ui/posts`, `/ui/login`, `/ui/register`, `/ui/search`, `/ui/vuln-demos`, `/ui/posts/view/:id`, `/ui/posts/edit/:id`
- [x] HTMX partials: `/ui/partials/posts`, `/ui/partials/posts/create`, `/ui/partials/login`, `/ui/partials/register`, `/ui/partials/search`, `/ui/partials/posts/view/:id/comments`
- [x] HX-Redirect dla loginu/rejestracji (full reload zeby pokazac zielony badge w navbarze)
- [x] HX-Trigger `post-created` po utworzeniu posta dla efektow JS

### Stylowanie i UI extras
- [x] Tailwind pipeline — `assets/css/input.css` -> `static/css/app.css` (`npm run build:css`)
- [x] Tailwind config skanuje `internal/views/**/*.{templ,go}` + `internal/handlers/**/*.go` + safelist body utility classes (poprawka 2026-05-08)
- [x] Design handoff z claude.ai/design: sakura petals, Burp-style Request Inspector, Attack Timeline z eksportem PoC do md, cheat-sheet drawer z 14 sekcjami i filtrem
- [x] Mascot reagujacy na tryb (`sec-vuln`/`sec-secure` body classes)
- [x] Strony Login/Register z dwukolumnowym hero + formularzem (od 2026-05-08)
- [x] Hub `/ui/vuln-demos` z 6 kartami CWE/OWASP (od 2026-05-08)
- [x] Auth pages na desktopie maja info-panel obok formularza, na mobile fall-back do single column

### Baza danych
- [x] SQLite (`app.db`)
- [x] Migracje idempotentne (`MigrateDB`) — tworzenie + `ALTER TABLE ADD COLUMN`
- [x] Tabele: `users`, `blog`, `comments`
- [x] Seed: 3 posty demo (Welcome / SQL Injection 101 / Stored XSS demo) + 3 userzy (admin / user1 / alice)
- [x] Zalaczniki do postow (multipart upload, limit 5 MB, sanitizacja path)

### Testy
- [x] `main_integration_test.go` z tagiem `integration`
- [x] 13+ testow: ping, posts CRUD, login JSON, login UI, partial routes, register, delete authorization, SQLi vuln/secure, Stored XSS vuln/secure, force-vuln endpointy
- [x] Wszystkie zielone (`go test -tags=integration -count=1 . -> ok`)

Wniosek: backend i UI maja stabilny MVP. Etapy A-D + 5 z 7 podatnosci dla n=3 sa gotowe. Brakuje CSRF (P1), Security Misconfiguration (P2), oraz uzupelnienia raportu i probnej obrony.

## 2) Cel docelowy (target stack)

- Backend: Go + Gin
- Renderowanie: Templ
- Interaktywnosc UI: HTMX
- Stylowanie: Tailwind CSS
- Baza danych: SQLite
- Uruchamianie: lokalnie (`go run main.go`)

## 3) Kolejnosc prac (krok po kroku)

### Etap A - Uporzadkowanie struktury projektu

1. Wydziel pakiety:
   - `internal/db`
   - `internal/handlers`
   - `internal/service`
   - `internal/views`
   - `internal/config`
2. Przenies logike DB (`InitDB`, `MigrateDB`, `SeedDB`) do `internal/db`.
3. Zostaw routing w `main.go`, ale oparty o handler/service.

Definition of done:
- `go test -tags=integration -v .` przechodzi
- brak zmiany kontraktu endpointow

Status: [x] ZAKONCZONY

### Etap B - Widoki Templ

1. Dodaj layout bazowy (header, footer, nav).
2. Dodaj strony:
   - lista postow
   - formularz logowania
   - panel statusu security mode
3. Renderuj HTML przez Templ z backendu.

Definition of done:
- mozna korzystac z UI bez klienta SPA
- endpointy API nadal dzialaja

Status: [x] ZAKONCZONY

### Etap C - HTMX

1. Dodaj czesciowe endpointy HTML (partiale):
   - fragment listy postow
   - fragment odpowiedzi loginu
2. Podlacz `hx-get`, `hx-post`, `hx-target`, `hx-swap` w widokach.
3. Dodaj obsluge bledow 4xx/5xx w fragmentach.

Definition of done:
- interakcje dzialaja bez reloadu calej strony
- fallback bez JS dalej mozliwy

Status: [x] ZAKONCZONY

### Etap D - Tailwind CSS

1. Dodaj pipeline Tailwind (CLI) i plik wejsciowy CSS.
2. Zbuduj prosty design system (kolory, spacing, typografia).
3. Ostyluj kluczowe widoki: login, posts, komunikaty bledow.

Definition of done:
- UI czytelne na desktop i mobile
- style budowane automatycznie do katalogu statycznego

Status: [x] ZAKONCZONY

### Etap E - Bezpieczenstwo Vulnerable vs Secure

1. Dla kazdej podatnosci przygotuj dwa warianty:
   - vulnerable branch w kodzie
   - secure branch w kodzie
2. Steruj zachowaniem przez `SecurityEnabled`.
3. Udokumentuj dla kazdej podatnosci:
   - opis
   - PoC
   - remediacja (przed/po)

Definition of done:
- ten sam atak dziala w vulnerable
- ten sam atak jest blokowany w secure

Status: [~] W TRAKCIE
- [x] SQL Injection (Krok 1, 2026-05-03)
- [x] Stored XSS (Krok 2, 2026-05-04)
- [x] Broken Authentication (Krok 3, 2026-05-03)
- [x] Broken Access Control (Krok 4, 2026-05-03)
- [x] Sensitive Data Exposure (Krok 5, 2026-05-03)
- [ ] CSRF — vuln endpoint istnieje, brak secure (token + walidacja)
- [ ] Security Misconfiguration — brak middleware naglowkow + release mode
- [ ] (opcjonalnie) Path Traversal / LFI, Command Injection

### Etap F - Finalizacja scenariuszy bezpieczenstwa i obrony

1. Domknij wymagane podatnosci (SQL Injection i XSS) w trybach vulnerable/secure.
2. Domknij dodatkowe podatnosci zgodnie z liczba osob w zespole.
3. Przygotuj stabilne dane testowe i checklisty do live demo.
4. Przejdz pelny dry-run obrony (atak -> blokada -> code review).

Definition of done:
- wszystkie scenariusze da sie powtorzyc na tej samej instancji aplikacji
- brak regresji endpointow i tras UI
- material demo jest gotowy do prezentacji

Status: [ ] DO ZROBIENIA
- [x] SQLi i XSS gotowe end-to-end
- [x] Hub `/ui/vuln-demos` jako one-stop shop dla demo
- [x] Cheat-sheet drawer z payloadami pod reka
- [ ] Sekcje raportu dla Broken Auth / BAC / SDE (kod jest, brak rozdzialu w PLAN_IMPLEMENTACJI_PODATNOSCI.md)
- [ ] Probna obrona z zegarkiem
- [ ] Checklista atakow (gotowe payloady + kolejnosc krokow demo)

## 4) Co robimy teraz i co dalej (konkretnie)

### Aktualny sprint: Sprint 3 (Etap E w trakcie, Etap F do zaczecia)

Ukonczone w Sprint 1-2:
- [x] MVP backend i testy integracyjne
- [x] Refaktor struktury (`internal/*`)
- [x] Etap B (Templ + podstawowe strony)
- [x] Etap C (HTMX dla flow posts/login/register/comments/search)
- [x] Etap D (Tailwind pipeline + stylowanie + design handoff)

Ukonczone w Sprint 3 (do 2026-05-08):
- [x] Etap E czesciowo: 5 z 7 wymaganych podatnosci dla wariantu n=3 (SQLi, XSS, Broken Auth, BAC, SDE)
- [x] Hub `/ui/vuln-demos` z 6 kartami CWE/OWASP
- [x] Strona detalu posta `/ui/posts/view/:id` z formularzem komentarzy (Stored XSS demo)
- [x] Force-vuln endpointy `/api/search-vulnerable` i `/api/comments-vulnerable` dla side-by-side
- [x] Naprawiony bug Tailwind purge (klasy z helperow Go)
- [x] Polish UI Login/Register (dwukolumnowe z hero)

### Nastepny krok (najblizsze 2-3 dni)

1. **CSRF secure mode** (P1, ~M) — middleware token + walidacja w POST/PUT/DELETE; hidden input w formularzach UI; test integracyjny z brakujacym tokenem -> 403.
2. **Security Misconfiguration** (P2, ~S) — middleware `SecurityHeaders` (CSP, HSTS, X-Frame-Options, X-Content-Type-Options, Referrer-Policy), `gin.SetMode("release")` w secure trybie, ukryty stack trace.
3. **Sekcje raportu** (~S kazda) — uzupelnic PLAN_IMPLEMENTACJI_PODATNOSCI.md o Krok 3 (Broken Auth), Krok 4 (BAC), Krok 5 (SDE) — opisy, PoC, diff.
4. **Probna obrona** (~M) — przejscie po wszystkich scenariuszach z zegarkiem.

Opcjonalnie:
- Path Traversal / LFI (P2)
- Command Injection (P3)

## 4a) Co sie zmienilo od ostatniego czasu

Ostatnie 6 dni (2026-05-03 -> 2026-05-08):
1. Krok 0: Fundament (auth bcrypt, cookie sesji, hardcoded admin z env, zalaczniki do postow, design handoff z claude.ai/design)
2. Krok 1: SQL Injection (vuln + secure side-by-side, force-vuln endpoint, UI z payloadami, testy)
3. Krok 2: Stored XSS (komentarze, `@templ.Raw` vs `{ }`, HTML strip server-side, force-vuln endpoint, testy)
4. Krok 3-5: Broken Auth + BAC + SDE (zaimplementowane, sa w status board, brak osobnego rozdzialu w PLAN_IMPLEMENTACJI_PODATNOSCI.md)
5. Krok 6 (UI/design refactor): hub `/ui/vuln-demos`, Login/Register dwukolumnowe, naprawa Tailwind purge

## 5) Minimalna checklista techniczna

- [x] Konfiguracja env (`PORT`, `DB_PATH`, `SECURITY_ENABLED`, `ADMIN_USERNAME`, `ADMIN_PASSWORD`, `ADMIN_EMAIL`)
- [x] Migracje uruchamiane automatycznie przy starcie (`MigrateDB` w `main.go::main`)
- [x] Testy integracyjne (`go test -tags=integration -count=1 .`) — wszystkie zielone
- [x] README z instrukcja local run (Go + npm)
- [ ] Logowanie bledow bez wycieku stack trace w `release` mode (czeka na Security Misconfiguration secure)
- [ ] CI z testami (opcjonalnie, nie wymagane przez sylabus)

## 6) Proponowany podzial prac dla zespolu 2-3 osoby

1. Osoba A: DB + migracje + dane seed (zrealizowane). Teraz: CSRF secure (token middleware + walidacja).
2. Osoba B: Templ + HTMX + Tailwind (zrealizowane). Teraz: hidden CSRF input w formularzach + sekcje raportu Krok 3-5.
3. Osoba C (jezeli jest): Security Misconfiguration (middleware naglowkow) + opcjonalnie Path Traversal / LFI.

## 7) Kryterium gotowosci do obrony

Dla kazdej podatnosci:
- [x] dzialajacy atak (vulnerable)
- [x] zablokowany atak (secure)
- [x] roznice w kodzie i wyjasnienie
- [~] sekcja w raporcie (kompletna dla SQLi i XSS, brakuje dla Broken Auth/BAC/SDE)

Calosc:
- [x] uruchamianie lokalne (`PORT=:8080 SECURITY_ENABLED=false go run .`, `npm run build:css`)
- [x] testy integracyjne zielone
- [x] hub `/ui/vuln-demos` ulatwia nawigacje miedzy scenariuszami w trakcie demo
- [ ] dry-run prezentacji
- [ ] CSRF secure (zeby zespol n=3 mial wymagane minimum 5 dodatkowych)
