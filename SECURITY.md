# Security Policy

## Charakter Projektu

To repozytorium jest projektem edukacyjnym z celowo zaimplementowanymi podatnościami.
Aplikacja służy do lokalnego porównania trybu `vulnerable` i `secure` w ramach
laboratorium bezpieczeństwa aplikacji internetowych.

Nie należy uruchamiać jej publicznie, wystawiać do Internetu ani używać jako
wzorca produkcyjnego bez usunięcia ścieżek podatnych.

## Zakres Celowych Podatności

Projekt zawiera między innymi:

- SQL Injection,
- Stored XSS,
- Broken Authentication,
- Broken Access Control / IDOR,
- Sensitive Data Exposure,
- CSRF,
- Security Misconfiguration,
- Path Traversal,
- Command Injection.

Te zachowania są kontrolowaną częścią projektu. Nie zgłaszaj ich jako błędów,
jeżeli występują w opisanych scenariuszach laboratoryjnych.

## Co Warto Zgłaszać

Zgłaszaj problemy, które nie są zamierzonym elementem laboratorium, na przykład:

- podatność aktywną w trybie secure,
- błąd pozwalający ominąć przełącznik bezpieczeństwa,
- wyciek sekretów spoza danych testowych,
- niestabilny test lub błąd uruchomienia,
- niezgodność dokumentacji z faktycznym kodem.

## Bezpieczne Uruchomienie

Uruchamiaj aplikację wyłącznie lokalnie:

```bash
SECURITY_ENABLED=false go run .
SECURITY_ENABLED=true go run .
```

Nie ustawiaj publicznego reverse proxy, tunelu ani hostingu dla trybu vulnerable.
