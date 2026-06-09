# Contributing

Ten projekt jest aplikacją laboratoryjną, dlatego najważniejsza zasada brzmi:
nie usuwamy kontrastu vulnerable vs secure bez świadomej decyzji projektowej.

## Lokalny Setup

```bash
go mod tidy
npm install
npm run build:css
go run .
```

Alternatywnie:

```bash
make deps
make run-vulnerable
```

## Standard Pracy

- Zachowuj istniejące endpointy i nazwy pól JSON, chyba że zmiana jest celowa.
- Dla podatności utrzymuj dwa zachowania: podatne i zabezpieczone.
- Secure path powinien używać prostych, czytelnych mechanizmów: prepared statements,
  escaping, walidacji wejścia, kontroli roli, tokenu CSRF albo bezpiecznej obsługi błędu.
- Nie edytuj ręcznie `internal/views/pages_templ.go`; generuj go z `pages.templ`.
- Nie commituj lokalnej bazy `app.db`, binarki `app`, `node_modules` ani `.DS_Store`.

## Testy Przed Committem

```bash
make css
make templ
make test
make test-integration
```

Jeżeli zmieniasz endpoint lub zachowanie trybu security, dodaj albo zaktualizuj
test integracyjny w `main_integration_test.go`.

## Generowanie Widoków

```bash
go run github.com/a-h/templ/cmd/templ@v0.3.1001 generate ./internal/views
```

## Build CSS

```bash
npm run build:css
```

## Dokumentacja

Po większej zmianie funkcjonalnej zaktualizuj:

- `README.md`,
- `VULNERABLE_ENDPOINTS.md`, jeżeli zmienił się endpoint podatny,
- sprawozdanie w `docs/`, jeżeli zmienił się opis scenariusza ataku albo obrony.
