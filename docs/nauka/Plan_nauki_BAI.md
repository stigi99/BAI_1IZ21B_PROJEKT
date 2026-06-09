# Plan nauki do obrony projektu BAI

**Projekt:** MikuMiku Fan Hub
**Cel:** przygotowanie do omówienia aplikacji, podatności, kodu i sprawozdania
**Zakres:** Go/Gin/SQLite, tryb `vulnerable/secure`, osiem podatności, testy, narzędzia zewnętrzne

## Jak korzystać z planu

Ucz się aktywnie: po każdym bloku zamknij dokument i odpowiedz z pamięci na pytania z fiszek. Najważniejsze jest nie recytowanie definicji, tylko umiejętność przejścia przez scenariusz: normalne użycie funkcji, podatny input, fragment kodu, skutek, naprawa.

## Dzień 1 - mapa projektu i architektura

**Cel:** umieć opowiedzieć, czym jest aplikacja i dlaczego podatności są w funkcjach strony.

Do opanowania:

- Co to jest MikuMiku Fan Hub.
- Dlaczego projekt nie jest zbiorem osobnych demo.
- Co robi przełącznik `SECURITY_ENABLED`.
- Jak wygląda przepływ: router Gin -> handler -> service -> SQLite -> Templ.
- Jakie są główne trasy: `/ui/library`, `/ui/profile`, `/ui/moderation`, `/ui/members`, `/ui/gallery`, `/ui/stream-check`.

Ćwiczenie:

1. Opowiedz projekt w 2 minuty bez patrzenia w raport.
2. Wymień osiem podatności i przypisz każdą do funkcji aplikacji.
3. Wyjaśnij, dlaczego tryb secure nie usuwa funkcji, tylko zmienia sposób obsługi danych.

## Dzień 2 - SQL Injection jako główny scenariusz

**Cel:** umieć bardzo dobrze wyjaśnić SQL Injection, bo to najmocniejszy fragment projektu.

Do opanowania:

- Normalne wyszukiwanie w `Vocaloid library`.
- Różnica między konkatenacją SQL a parametryzacją.
- Payload logiczny `EXISTS(...)`.
- `ORDER BY` jako rozpoznanie liczby kolumn.
- `UNION SELECT` do odczytu `users`, `sqlite_master` i ukrytych draftów.
- Dlaczego `sqlmap` potwierdza podatność, ale nie zastępuje ręcznego wyjaśnienia.

Ćwiczenie:

1. Narysuj na kartce przepływ payloadu od pola `q` do `db.Query`.
2. Wytłumacz, czemu apostrof w payloadzie jest istotny.
3. Powiedz, co dokładnie zmienia `LIKE ?`.

## Dzień 3 - XSS, CSRF i kontrola intencji użytkownika

**Cel:** rozumieć podatności po stronie przeglądarki i formularzy.

Do opanowania:

- Stored XSS w komentarzach.
- Różnica między tekstem użytkownika a HTML.
- `templ.Raw` jako niebezpieczne renderowanie w trybie vulnerable.
- `html.EscapeString` i usuwanie event handlerów w secure.
- CSRF w `/ui/profile`.
- Dlaczego cookie sesyjne nie oznacza intencji użytkownika.
- Token CSRF w formularzu i cookie.

Ćwiczenie:

1. Wyjaśnij, dlaczego XSS stored jest groźniejszy niż jednorazowy payload w URL.
2. Opowiedz, jak obca strona może zmienić email ofiary.
3. Wyjaśnij, co musi się zgadzać w secure CSRF.

## Dzień 4 - logowanie, hasła i IDOR

**Cel:** umieć pokazać różnicę między uwierzytelnieniem i autoryzacją.

Do opanowania:

- Broken Authentication: sprawdzanie samego istnienia loginu.
- Bcrypt jako bezpieczniejszy sposób przechowywania haseł.
- Rate limit jako warstwa przeciw brute force.
- IDOR w usuwaniu postów.
- Różnica między zalogowaniem a prawem do zasobu.
- `canDeletePost`: autor albo admin.

Ćwiczenie:

1. Powiedz różnicę między authentication i authorization.
2. Wyjaśnij, dlaczego ukrycie przycisku w UI nie naprawia IDOR.
3. Opowiedz łańcuch: słabe logowanie -> usunięcie cudzego posta.

## Dzień 5 - Path Traversal, Command Injection i dane wrażliwe

**Cel:** opanować błędy wynikające z zaufania do ścieżek i komend systemowych.

Do opanowania:

- Path Traversal w `Fanart vault`.
- Dlaczego `uploadsDir + "/" + name` jest błędne.
- `filepath.Clean`, ścieżka absolutna i sprawdzanie katalogu bazowego.
- Command Injection w `Stream relay check`.
- Dlaczego `sh -c` z inputem użytkownika jest niebezpieczne.
- Regex hosta i `exec.CommandContext` bez shella.
- Sensitive Data Exposure przez jawne hasła lub wyciek bazy.

Ćwiczenie:

1. Wytłumacz, jak `../internal/db/db.go` wychodzi poza `uploads`.
2. Wytłumacz różnicę między `exec.Command("sh", "-c", "..."+host)` i `exec.Command("ping", "-c1", host)`.
3. Powiedz, czemu wyciek bazy z plaintext hasłami jest krytyczny.

## Dzień 6 - testy i narzędzia

**Cel:** umieć obronić, że projekt został zweryfikowany.

Do opanowania:

- `go test ./...`.
- `go test -tags=integration -count=1 -v .`.
- Co sprawdzają testy integracyjne.
- `sqlmap` dla SQL Injection.
- `curl` i `HTTPie` jako ręczni klienci HTTP.
- OWASP ZAP baseline w Dockerze jako DAST.
- Dlaczego ZAP nie wykrywa wszystkiego i nie zastępuje analizy kodu.

Ćwiczenie:

1. Powiedz, co oznacza wynik `PASS`.
2. Wyjaśnij różnicę między testem integracyjnym a skanem DAST.
3. Podaj przykład podatności, którą lepiej pokazuje ręczny payload niż skaner.

## Dzień 7 - próba obrony

**Cel:** przećwiczyć odpowiedź ustną.

Scenariusz 10-minutowy:

1. 1 minuta - czym jest aplikacja.
2. 1 minuta - architektura i `SECURITY_ENABLED`.
3. 3 minuty - SQL Injection.
4. 2 minuty - XSS, CSRF, IDOR.
5. 1 minuta - Path Traversal i Command Injection.
6. 1 minuta - testy i narzędzia.
7. 1 minuta - wnioski i rekomendacje.

Minimalny zestaw do zapamiętania:

- Podatności są w użytecznych funkcjach aplikacji.
- SQLi: konkatenacja -> parametryzacja.
- XSS: raw HTML -> escape/sanitize.
- Auth: istnienie loginu -> bcrypt + rate limit.
- IDOR: samo zalogowanie -> autor/admin.
- CSRF: samo cookie -> token z formularza i cookie.
- Traversal: sklejenie ścieżki -> canonical path check.
- Command Injection: `sh -c` -> walidacja + brak shella.
