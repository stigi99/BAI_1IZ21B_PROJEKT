# Fiszki BAI - MikuMiku Fan Hub

Format: **Pytanie -> Odpowiedź**. Najlepiej powtarzać w trybie aktywnym: najpierw zasłoń odpowiedź, powiedz ją własnymi słowami, potem sprawdź.

## Projekt i architektura

1. **Czym jest MikuMiku Fan Hub?**
   Aplikacją laboratoryjną Go/Gin/SQLite pokazującą podatności webowe w normalnych funkcjach portalu fanowskiego.

2. **Co jest głównym założeniem projektu?**
   Porównanie tej samej funkcjonalności w trybie `vulnerable` i `secure`.

3. **Dlaczego podatności nie są osobnymi demami?**
   Bo zostały osadzone w realnych funkcjach: wyszukiwarce, komentarzach, logowaniu, profilu, moderacji, plikach i narzędziu ping.

4. **Co robi `SECURITY_ENABLED`?**
   Przełącza zachowanie aplikacji między podatnym i zabezpieczonym wariantem.

5. **Jaki jest ogólny przepływ żądania?**
   Gin router -> handler -> service -> SQLite -> widok Templ albo JSON.

6. **Jakie technologie wykorzystano?**
   Go, Gin, SQLite, Templ, HTMX, Tailwind CSS.

7. **Dlaczego tryb secure nie usuwa funkcji?**
   Bo celem jest zachowanie użyteczności i naprawienie obsługi danych, nie ukrycie problemu.

8. **Jakie konto jest ważne w scenariuszach?**
   `admin` i `user1`, bo pozwalają pokazać logowanie, IDOR, role i wycieki.

## SQL Injection

9. **Gdzie występuje SQL Injection?**
   W wyszukiwarce `Vocaloid library`, parametr `q`.

10. **Co jest normalnym użyciem wyszukiwarki?**
    Wpisanie frazy, np. `Miku`, i otrzymanie publicznych postów.

11. **Dlaczego wariant vulnerable jest podatny?**
    Bo dokleja input użytkownika do zapytania SQL jako tekst.

12. **Jaki fragment kodu jest błędny w SQLi?**
    `"... LIKE '%" + query + "%' ..."` oraz późniejsze `db.Query(sqlQuery)`.

13. **Jak działa payload SQLi?**
    Apostrof kończy literał SQL, a dalsza część wejścia staje się składnią SQL.

14. **Po co użyto `ORDER BY 8 --`?**
    Do rozpoznania liczby kolumn przez błąd SQL.

15. **Po co użyto `sqlite_master`?**
    Do enumeracji tabel i instrukcji `CREATE TABLE`.

16. **Co daje `UNION SELECT FROM users`?**
    Pozwala pokazać rekordy użytkowników jako wyniki wyszukiwarki.

17. **Co pokazuje payload na `published=0`?**
    Że podatność może ujawnić ukryte drafty, których normalna wyszukiwarka nie powinna zwracać.

18. **Jak naprawiono SQL Injection?**
    Przez zapytania parametryzowane z placeholderami `?`.

19. **Dlaczego `LIKE ?` jest bezpieczniejsze?**
    Bo payload jest wartością parametru, a nie częścią składni SQL.

20. **Co potwierdził `sqlmap`?**
    Że parametr `q` w trybie vulnerable jest wykrywalnie podatny na SQL Injection.

21. **Czy `sqlmap` zastępuje analizę ręczną?**
    Nie. Potwierdza podatność, ale nie tłumaczy tak dobrze mechanizmu błędu.

## Stored XSS

22. **Gdzie występuje Stored XSS?**
    W komentarzach pod fan postami.

23. **Co jest normalnym użyciem komentarzy?**
    Dodanie zwykłej treści tekstowej pod wpisem.

24. **Dlaczego XSS jest stored?**
    Bo payload jest zapisany w bazie i wykonuje się przy późniejszym odczycie strony.

25. **Co jest błędne w wariancie vulnerable XSS?**
    Renderowanie treści użytkownika jako raw HTML.

26. **Jaki przykład payloadu XSS pokazano?**
    HTML/JS dodający fałszywą nakładkę na stronę.

27. **Jak naprawiono Stored XSS?**
    Przez oczyszczenie treści i `html.EscapeString`.

28. **Dlaczego samo usunięcie `<script>` nie wystarcza zawsze?**
    Bo atak może używać event handlerów, `javascript:` albo HTML bez tagu `script`.

## Broken Authentication i hasła

29. **Gdzie występuje Broken Authentication?**
    W logowaniu `/ui/login` i `/login`.

30. **Jaki jest błąd w trybie vulnerable?**
    Serwer sprawdza tylko, czy użytkownik istnieje, a ignoruje hasło.

31. **Jaki przykład ataku pokazano?**
    `username=admin`, `password=anything`.

32. **Jak naprawiono logowanie?**
    Przez walidację hasła względem bcrypt hash i rate limit.

33. **Po co jest bcrypt?**
    Do przechowywania hasła jako wolnego, solonego hasha zamiast tekstu jawnego.

34. **Dlaczego plaintext hasła są groźne?**
    Wyciek bazy natychmiast ujawnia sekret logowania.

35. **Co daje rate limit?**
    Utrudnia brute force przez ograniczenie liczby błędnych prób.

## IDOR / Broken Access Control

36. **Gdzie występuje IDOR?**
    W kolejce moderacyjnej i usuwaniu postów po ID.

37. **Jaka jest różnica między authentication i authorization?**
    Authentication potwierdza tożsamość, authorization sprawdza prawo do konkretnego zasobu.

38. **Co jest błędne w vulnerable IDOR?**
    Serwer sprawdza tylko, czy użytkownik jest zalogowany.

39. **Jaki jest atak IDOR?**
    `user1` usuwa post admina, zmieniając ID w żądaniu.

40. **Jak naprawiono IDOR?**
    Sprawdzeniem, czy użytkownik jest autorem posta albo adminem.

41. **Dlaczego ukrycie przycisku w UI nie wystarcza?**
    Bo atakujący może wysłać HTTP POST ręcznie.

## CSRF

42. **Gdzie występuje CSRF?**
    W zmianie emaila profilu przez `/ui/profile`.

43. **Co jest normalnym użyciem funkcji profilu?**
    Zalogowany użytkownik zmienia email powiadomień.

44. **Dlaczego cookie sesyjne nie wystarcza?**
    Bo przeglądarka dołączy je także do żądania wysłanego przez obcą stronę.

45. **Jaki jest payload CSRF?**
    Ukryty formularz POST z `new_email`, automatycznie wysłany przez JavaScript.

46. **Jak naprawiono CSRF?**
    Tokenem w formularzu i cookie, które muszą się zgadzać.

47. **Co oznacza błąd 403 w secure CSRF?**
    Żądanie nie miało poprawnego tokena, więc serwer nie uznał intencji użytkownika.

## Sensitive Data Exposure

48. **Gdzie pokazano Sensitive Data Exposure?**
    W `Member directory` i przy możliwym wycieku tabeli `users`.

49. **Co jest problemem w trybie vulnerable?**
    Kolumna `password_hash` może zawierać hasła jawne.

50. **Jak SQLi łączy się z Sensitive Data Exposure?**
    SQLi może odczytać `username`, `email`, `role` i `password_hash`.

51. **Jak Path Traversal łączy się z Sensitive Data Exposure?**
    Może próbować pobrać `app.db`, czyli bazę z danymi kont.

52. **Co zmienia bcrypt w secure?**
    Wyciek bazy nie ujawnia bezpośrednio haseł.

## Path Traversal / LFI

53. **Gdzie występuje Path Traversal?**
    W `Fanart vault`, parametr `name`.

54. **Co jest normalnym użyciem vaulta?**
    Podgląd pliku z katalogu `uploads/`.

55. **Dlaczego `uploadsDir + "/" + name` jest błędne?**
    Bo `name` może zawierać `../` i wyjść poza katalog uploadów.

56. **Jaki payload pokazuje Path Traversal?**
    `../internal/db/db.go` albo `../go.sum`.

57. **Jak naprawiono Path Traversal?**
    Przez odrzucenie ścieżek absolutnych, `filepath.Clean` i sprawdzenie katalogu bazowego.

58. **Co oznacza LFI?**
    Local File Inclusion, czyli odczyt lokalnych plików przez aplikację.

## Command Injection

59. **Gdzie występuje Command Injection?**
    W `Stream relay check`, parametr `host`.

60. **Co jest normalnym użyciem stream check?**
    Sprawdzenie hosta komendą `ping`.

61. **Dlaczego `sh -c` jest niebezpieczne?**
    Bo powłoka interpretuje metaznaki takie jak `;`, `&&`, `|`.

62. **Jaki payload pokazuje Command Injection?**
    `127.0.0.1 ; whoami ; uname -a`.

63. **Jak naprawiono Command Injection?**
    Walidacją hosta i `exec.CommandContext("ping", "-c1", host)` bez shella.

64. **Dlaczego sama walidacja to za mało?**
    Bo bezpieczniej jest też nie używać interpretera powłoki.

## Narzędzia i testy

65. **Do czego użyto `curl`?**
    Do ręcznego wysyłania żądań HTTP.

66. **Do czego użyto HTTPie?**
    Do czytelnego pokazania żądań i odpowiedzi HTTP.

67. **Do czego użyto OWASP ZAP?**
    Do pasywnego skanu DAST lokalnej aplikacji.

68. **Czy ZAP wykrywa wszystkie podatności?**
    Nie. Jest wsparciem, ale nie zastępuje testów ręcznych i analizy kodu.

69. **Co sprawdzają testy integracyjne?**
    Zachowanie endpointów vulnerable/secure, np. SQLi, XSS, CSRF, IDOR, Traversal, Command Injection.

70. **Jaka komenda uruchamia testy integracyjne?**
    `go test -tags=integration -count=1 -v .`

71. **Co oznacza wynik `PASS`?**
    Że testowane scenariusze zachowują się zgodnie z oczekiwaniem.

## Pytania przekrojowe

72. **Która podatność jest najważniejsza w projekcie?**
    SQL Injection, bo pokazuje pełny łańcuch rozpoznania i wycieku danych.

73. **Jaki jest najważniejszy wniosek projektu?**
    Zabezpieczenie musi być po stronie serwera i dotyczyć interpretacji danych wejściowych.

74. **Dlaczego projekt ma charakter etyczny?**
    Testy wykonano lokalnie, na własnej aplikacji laboratoryjnej.

75. **Co łączy większość podatności?**
    Zaufanie do danych użytkownika bez walidacji, parametryzacji albo autoryzacji.

76. **Jak najlepiej tłumaczyć tryb secure?**
    Funkcja dalej działa, ale input jest traktowany bezpiecznie: jako dane, nie jako kod/ścieżka/komenda.

77. **Co można dodać jako dalszą rekomendację?**
    CSP, role middleware, audit log, limity uploadów, separację trybu labowego od produkcyjnego.

78. **Co powiedzieć, gdy prowadzący zapyta, czy aplikacja jest produkcyjna?**
    Nie. To świadome środowisko laboratoryjne z przełącznikiem vulnerable/secure.

79. **Jak pokazać, że podatności są w funkcjonalności?**
    Wskazać zwykłe ekrany: library, comments, login, profile, moderation, members, gallery, stream check.

80. **Jak jednym zdaniem opisać projekt?**
    To aplikacja Go/Gin pokazująca osiem podatności webowych w normalnych funkcjach strony i ich naprawy w trybie secure.
