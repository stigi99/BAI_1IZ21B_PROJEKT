# Ściąga 1-stronicowa - BAI MikuMiku Fan Hub

## Jednozdaniowy opis projektu

MikuMiku Fan Hub to aplikacja Go/Gin/SQLite pokazująca osiem podatności webowych w normalnych funkcjach strony oraz ich naprawy po przełączeniu `SECURITY_ENABLED=true`.

## Architektura

`Gin router -> handlers -> service -> SQLite -> Templ/JSON`

Tryb vulnerable i secure działa na tej samej aplikacji. Różnica jest po stronie walidacji, autoryzacji, tokenów, parametryzacji SQL, przechowywania haseł i wykonywania komend.

## Mapa podatności

| Funkcja | Vulnerable | Secure |
|---|---|---|
| Library search | SQL konkatenowany z `q` | `LIKE ?` |
| Comments | raw HTML / `templ.Raw` | sanitize + escape |
| Login | hasło ignorowane | bcrypt + rate limit |
| Moderation | usuwanie po samym loginie | autor albo admin |
| Profile | POST bez CSRF | token formularz + cookie |
| Members | plaintext hasła | bcrypt hash |
| Fanart vault | `../` czyta poza uploads | `filepath.Clean` + base check |
| Stream check | `sh -c` z inputem | regex + brak shella |

## Najmocniejszy scenariusz: SQLi

1. Normalnie: `Miku`.
2. Sonda logiczna: `zz' OR EXISTS(...) --`.
3. Kolumny: `zz' ORDER BY 8 --`.
4. Schemat: `UNION SELECT ... FROM sqlite_master`.
5. Użytkownicy: `UNION SELECT ... FROM users`.
6. Drafty: `UNION SELECT ... FROM blog WHERE published=0`.
7. Secure: payload jest tekstem w `LIKE ?`.

## Najważniejsze zdania do powiedzenia

- Dane użytkownika nie mogą zmieniać składni SQL, HTML, ścieżki pliku ani komendy systemowej.
- Samo zalogowanie nie oznacza prawa do każdego zasobu.
- Cookie sesyjne nie oznacza intencji użytkownika, dlatego potrzebny jest token CSRF.
- Ukrycie przycisku w UI nie jest zabezpieczeniem.
- Automatyczny skaner pomaga, ale nie zastępuje testów ręcznych i analizy kodu.

## Testy i narzędzia

```bash
go test ./...
go test -tags=integration -count=1 -v .
sqlmap -u "http://127.0.0.1:8099/api/search?q=Miku" --batch --dbms=SQLite
docker run ... ghcr.io/zaproxy/zaproxy:stable zap-baseline.py ...
```

## Gdy padnie pytanie "co można poprawić dalej?"

- Dodać CSP.
- Dodać audit log.
- Rozdzielić tryb labowy od produkcyjnego.
- Dodać middleware ról.
- Dodać kontrolę typów uploadu.
- Dodać limity odpowiedzi dla podglądu plików.
