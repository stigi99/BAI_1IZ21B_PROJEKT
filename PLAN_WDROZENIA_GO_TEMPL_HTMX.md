# Plan Dzialania: Go + Templ + HTMX + Tailwind + SQLite

## 1) Gdzie jestesmy teraz (stan aktualny)

- [x] Dzialajacy backend w Go (Gin) w pliku `main.go`
- [x] Security toggle: `SecurityEnabled` (vulnerable/secure)
- [x] Endpointy: `GET /ping`, `GET /posts`, `POST /login`
- [x] Testy integracyjne: `main_integration_test.go`
- [x] Baza lokalna SQLite (`app.db`)
- [x] Warstwa widokow Templ (wydzielona do plikow `.templ`)
- [x] Trasy UI: `GET /ui/posts`, `GET /ui/login`, `POST /ui/login`
- [x] Testy integracyjne dla tras UI
- [x] HTMX (partiale dla posts/login)
- [x] Tailwind pipeline i stylowanie widokow
- [x] SQLite jako docelowa baza dla tego projektu
- [x] Zakres projektu zamkniety na obecnym stacku

Wniosek: mamy dobry MVP backend i testy. Rozwijamy warstwe aplikacji (struktura + UI + scenariusze bezpieczenstwa) i finalizujemy projekt na SQLite.

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

### Etap F - Finalizacja scenariuszy bezpieczenstwa i obrony

1. Domknij wymagane podatnosci (SQL Injection i XSS) w trybach vulnerable/secure.
2. Domknij dodatkowe podatnosci zgodnie z liczba osob w zespole.
3. Przygotuj stabilne dane testowe i checklisty do live demo.
4. Przejdz pelny dry-run obrony (atak -> blokada -> code review).

Definition of done:
- wszystkie scenariusze da sie powtorzyc na tej samej instancji aplikacji
- brak regresji endpointow i tras UI
- material demo jest gotowy do prezentacji

## 4) Co robimy teraz i co dalej (konkretnie)

### Aktualny sprint: Sprint 1 (w trakcie)

- [x] MVP backend i testy integracyjne
- [x] Refaktor struktury (`internal/*`)
- [x] Start warstwy widokow (Templ + podstawowe strony)
- [x] Etap C (HTMX) dla flow posts i login
- [x] Etap D (Tailwind) - pipeline + stylowanie widokow

### Nastepny krok (najblizsze 2-3 dni)

1. Rozwinac HTMX o dodatkowe partiale i obsluge bledow UX (komunikaty i loading states).
2. Wejsc w Etap E i zaczas implementowac pierwsze podatnosci vulnerable vs secure.
3. Wejsc w Etap F i przygotowac komplet scenariuszy oraz materialow pod obrone.

## 4a) Co sie zmienilo od ostatniego czasu

1. Wdrozone Etap A (struktura pakietow `internal/*`).
2. Logika DB przeniesiona do `internal/db`.
3. Routing oparty o handler/service.
4. Widoki przeniesione z recznego HTML w Go do pliku `internal/views/pages.templ`.
5. Dodane i dzialajace trasy UI (`/ui/posts`, `/ui/login`).
6. Dodane i dzialajace trasy partial HTMX (`/ui/partials/posts`, `/ui/partials/login`).
7. Dodane testy integracyjne dla tras UI i tras partial HTMX.
8. Dodany pipeline Tailwind (`package.json`, `tailwind.config.js`, `assets/css/input.css`, `static/css/app.css`).
9. Widoki ostylowane klasami Tailwind i podlaczone przez `/static/css/app.css`.
10. Dodatkowe utwardzenie kodu:
   - sprawdzanie `rows.Err()` po iteracji
   - spojna obsluga bledow renderowania widokow

## 5) Minimalna checklista techniczna

- [ ] Konfiguracja env (`PORT`, `DB_PATH`, `SECURITY_ENABLED`)
- [ ] Migracje uruchamiane automatycznie przy starcie lub komenda `make migrate`
- [ ] Logowanie bledow bez wycieku wrazliwych danych
- [ ] Testy integracyjne odpalane w CI
- [ ] README z instrukcja local run (Go + npm)

## 6) Proponowany podzial prac dla zespolu 2-3 osoby

1. Osoba A: DB + migracje + dane seed
2. Osoba B: Templ + HTMX + Tailwind
3. Osoba C (lub rotacyjnie): podatnosci vulnerable/secure + PoC + dokumentacja

## 7) Kryterium gotowosci do obrony

- Dla kazdej podatnosci macie:
  - dzialajacy atak (vulnerable)
  - zablokowany atak (secure)
  - roznice w kodzie i wyjasnienie
- Calosc uruchamiana lokalnie (`go run main.go` + zbudowane CSS)
- Raport techniczny uzupelniony zgodnie z wymaganiami z `info.md`
