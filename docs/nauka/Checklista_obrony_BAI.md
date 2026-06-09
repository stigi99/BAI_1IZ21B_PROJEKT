# Checklista obrony BAI

## Start odpowiedzi

- [ ] Powiedzieć, że aplikacja to MikuMiku Fan Hub, portal fanowski z funkcjami laboratoryjnymi.
- [ ] Podkreślić, że podatności są w funkcjach strony, a nie w oderwanych demo.
- [ ] Wskazać tryb `SECURITY_ENABLED=false/true`.
- [ ] Wymienić technologie: Go, Gin, SQLite, Templ, HTMX, Tailwind.

## Osiem podatności - mapa funkcji

| Funkcja | Podatność | Naprawa |
|---|---|---|
| `Vocaloid library` | SQL Injection | `LIKE ?`, parametry SQL |
| Komentarze | Stored XSS | czyszczenie + escape HTML |
| Login | Broken Authentication | bcrypt + rate limit |
| Moderation queue | IDOR | autor albo admin |
| Profile settings | CSRF | token formularza + cookie |
| Member directory | Sensitive Data Exposure | bcrypt zamiast plaintext |
| Fanart vault | Path Traversal/LFI | canonical path check |
| Stream relay check | Command Injection | walidacja + brak `sh -c` |

## Najważniejsze rzeczy do pokazania na żywo

1. Normalne wyszukiwanie `Miku`.
2. SQLi: sonda logiczna albo `UNION SELECT`.
3. Ten sam payload w secure.
4. Komentarz XSS jako raw HTML.
5. CSRF: POST bez tokena vs 403.
6. IDOR: zwykły użytkownik próbuje usunąć cudzy post.
7. Path Traversal: `../internal/db/db.go`.
8. Command Injection: `127.0.0.1 ; whoami ; uname -a`.

## Pytania, które może zadać prowadzący

**Dlaczego SQL Injection jest najważniejszy?**
Bo publiczne pole wyszukiwania pozwala przejść od zwykłego inputu do odczytu tabel, użytkowników i ukrytych draftów.

**Czy ukrycie przycisku naprawia IDOR?**
Nie. Serwer musi sprawdzić uprawnienia do konkretnego zasobu.

**Czy ZAP wystarczy jako test?**
Nie. ZAP jest DAST i wsparciem, ale projekt wymaga ręcznych scenariuszy, testów integracyjnych i analizy kodu.

**Czym różni się XSS od CSRF?**
XSS wykonuje kod w przeglądarce ofiary. CSRF wykorzystuje sesję ofiary do wysłania niechcianego żądania.

**Dlaczego `exec.Command("ping", "-c1", host)` jest lepsze niż `sh -c`?**
Bo nie używa powłoki, więc metaznaki nie są interpretowane jako kolejne komendy.

**Czy aplikacja jest produkcyjna?**
Nie. To środowisko laboratoryjne z celowo podatnym trybem i trybem secure do porównania.

## Szybka odpowiedź 2-minutowa

Projekt to aplikacja MikuMiku Fan Hub napisana w Go/Gin z bazą SQLite. Jej celem jest pokazanie ośmiu podatności webowych w realnych funkcjach strony, takich jak wyszukiwarka, komentarze, logowanie, profil, moderacja, katalog członków, podgląd plików i narzędzie diagnostyczne. Tryb `SECURITY_ENABLED=false` pokazuje wariant podatny, a `SECURITY_ENABLED=true` pokazuje naprawę tej samej funkcji.

Najważniejszy scenariusz to SQL Injection w wyszukiwarce. W trybie podatnym input jest doklejany do SQL, więc payload może użyć `UNION SELECT`, odczytać `sqlite_master`, tabelę `users` i ukryte drafty. W trybie secure ten sam input trafia do placeholdera `?`, więc jest traktowany jako tekst. Pozostałe podatności pokazują analogiczną zasadę: XSS wynika z renderowania raw HTML, CSRF z braku tokena, IDOR z braku autoryzacji zasobu, Path Traversal z zaufania do ścieżki, Command Injection z użycia `sh -c`, a Sensitive Data Exposure z zapisu haseł jawnych.

Projekt został zweryfikowany testami Go, testami integracyjnymi, ręcznymi żądaniami HTTP, `sqlmap`, HTTPie i baseline scanem OWASP ZAP.

## Ostatnia kontrola przed oddaniem

- [ ] Umiesz opowiedzieć SQLi bez czytania payloadu z kartki.
- [ ] Umiesz wskazać różnicę vulnerable vs secure w kodzie.
- [ ] Umiesz powiedzieć, które endpointy odpowiadają za każdą podatność.
- [ ] Umiesz wyjaśnić, co sprawdzają testy.
- [ ] Umiesz powiedzieć, dlaczego testy są lokalne i etyczne.
- [ ] Umiesz podać przynajmniej dwie dalsze rekomendacje.
