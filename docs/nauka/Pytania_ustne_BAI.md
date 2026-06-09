# Pytania ustne do przećwiczenia

## Poziom podstawowy

1. Co było celem projektu?
2. Jak działa przełącznik `SECURITY_ENABLED`?
3. Jakie technologie wykorzystano?
4. Jakie podatności zaimplementowano?
5. Dlaczego podatności są w normalnych funkcjach aplikacji?
6. Jakie są główne endpointy UI?
7. Jak uruchomić aplikację w trybie vulnerable?
8. Jak uruchomić aplikację w trybie secure?

## Poziom kodu

1. Gdzie dokładnie powstaje SQL Injection?
2. Czym różni się `SearchPostsVulnerable` od `SearchPostsSecure`?
3. Dlaczego `LIKE ?` naprawia SQLi?
4. Gdzie w kodzie pojawia się `templ.Raw`?
5. Dlaczego `html.EscapeString` neutralizuje HTML?
6. Jak działa `ValidateUserCredentials`?
7. Dlaczego bcrypt jest lepszy od plaintext?
8. Jak działa `canDeletePost`?
9. Jak działa `safeUploadPath`?
10. Dlaczego `exec.CommandContext` bez shella jest bezpieczniejsze?

## Poziom ataku

1. Jak przeprowadzić SQL Injection krok po kroku?
2. Po co w SQLi używa się `ORDER BY`?
3. Po co w SQLi używa się `sqlite_master`?
4. Jak wyciągnąć użytkowników przez `UNION SELECT`?
5. Jak pokazać XSS bez samego `alert`?
6. Jak wygląda CSRF PoC?
7. Jak zwykły użytkownik może wykonać IDOR?
8. Jak Path Traversal ujawnia kod źródłowy?
9. Jak Command Injection wykonuje `whoami`?
10. Jakie łańcuchy ataków pokazuje projekt?

## Poziom obrony i wniosków

1. Jakie są najważniejsze różnice między trybem vulnerable i secure?
2. Dlaczego zabezpieczenia muszą być po stronie serwera?
3. Dlaczego walidacja wejścia nie zawsze wystarczy?
4. Dlaczego automatyczny skaner nie zastępuje ręcznego testu?
5. Co w projekcie można dalej poprawić?
6. Jak oddzielić tryb laboratoryjny od produkcyjnego?
7. Jakie nagłówki bezpieczeństwa warto dodać?
8. Dlaczego CSP byłoby przydatne przy XSS?
9. Jak audit log pomógłby przy IDOR?
10. Jakie są ograniczenia projektu?

## Krótkie odpowiedzi wzorcowe

**Co jest najważniejszym błędem w SQLi?**
Traktowanie danych użytkownika jako fragmentu języka SQL.

**Co jest najważniejszym błędem w XSS?**
Traktowanie danych użytkownika jako zaufanego HTML.

**Co jest najważniejszym błędem w CSRF?**
Założenie, że obecność cookie oznacza intencję użytkownika.

**Co jest najważniejszym błędem w IDOR?**
Brak sprawdzenia, czy użytkownik ma prawo do konkretnego zasobu.

**Co jest najważniejszym błędem w Path Traversal?**
Zaufanie do ścieżki podanej przez użytkownika.

**Co jest najważniejszym błędem w Command Injection?**
Przekazanie inputu użytkownika do interpretera powłoki.
