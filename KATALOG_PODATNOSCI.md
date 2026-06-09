# Katalog podatnosci dla projektu (Go + Gin + SQLite + Templ + HTMX)

## Cel dokumentu
Ten plik opisuje podatnosci, ktore mozna sensownie pokazac w obecnej aplikacji z przelacznikiem `SecurityEnabled`.
Dla kazdej podatnosci masz:
- trudnosc wdrozenia do obecnego kodu,
- trudnosc pokazania na live demo,
- sposob wdrozenia (wariant Vulnerable),
- sposob pokazania ataku,
- kierunek remediacji (wariant Secure).

Skala trudnosci:
- 1/5 = bardzo latwe
- 3/5 = srednie
- 5/5 = trudne

## Szybka mapa podatnosci (pod ten stack)

| Podatnosc | Czy wymagana na kursie | Trudnosc wdrozenia | Trudnosc demo | Dlaczego pasuje do projektu |
|---|---|---:|---:|---|
| SQL Injection | TAK | 2/5 | 2/5 | Macie SQLite i warstwe SQL w service, latwo pokazac roznice query concatenation vs placeholders |
| Stored XSS | TAK | 2/5 | 1/5 | Macie formularze tworzenia/edycji postow i server-rendered widoki (Templ/HTMX) |
| Broken Authentication | DODATKOWA (polecana) | 1/5 | 1/5 | Login juz jest uproszczony, latwo pokazac obejscie hasla i brak ochrony sesji |
| Broken Access Control (IDOR/BOLA) | DODATKOWA (polecana) | 2/5 | 2/5 | Endpointy update/delete postow po ID bez przypisania wlasciciela |
| CSRF | DODATKOWA (polecana) | 3/5 | 3/5 | Macie akcje mutujace przez formularze POST i cookie-based auth |
| Sensitive Data Exposure | DODATKOWA | 2/5 | 2/5 | W bazie mozna celowo trzymac hasla jawnie w vulnerable mode |
| Security Misconfiguration | DODATKOWA | 2/5 | 2/5 | Latwo pokazac nadmiarowe logi, szczegolowe bledy i slabe naglowki |
| Command Injection | DODATKOWA (raczej opcjonalna) | 4/5 | 3/5 | Wymaga dodania funkcji uruchamiania polecen systemowych (bardziej inwazyjne) |
| Path Traversal / LFI | DODATKOWA (opcjonalna) | 3/5 | 2/5 | Wymaga nowego endpointu czytajacego pliki po nazwie |
| Insecure Deserialization | DODATKOWA (opcjonalna) | 4/5 | 3/5 | Wymaga celowego przyjmowania i deserializacji nieufnych obiektow bez walidacji |
| XXE | DODATKOWA (opcjonalna) | 4/5 | 3/5 | Wymaga endpointu XML i niebezpiecznej konfiguracji parsera XML |
| SSRF | DODATKOWA (opcjonalna) | 4/5 | 3/5 | Wymaga endpointu pobierajacego URL podany przez uzytkownika |

---

## 1) SQL Injection (wymagana)

### Trudnosc
- Wdrozenie do obecnego programu: 2/5
- Pokazanie na demo: 2/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint wyszukiwania, np. GET /posts/search?q=...
- W warstwie service zbuduj SQL przez konkatenacje stringa:
  - przyklad idei: ... WHERE title LIKE '%" + q + "%'
- Zwroc wyniki bez dodatkowej walidacji inputu.

Najlepsze miejsce zmian:
- nowa metoda w service (obok GetPublishedPosts/GetAllPosts),
- nowy handler i route w routerze.

### Jak pokazac atak
- Krok 1: normalne zapytanie z bezpieczna fraza i poprawny wynik.
- Krok 2: zapytanie z payloadem SQLi w parametrze q (np. zamkniecie cudzyslowu i OR 1=1).
- Krok 3: pokaz, ze zwrocone sa rekordy, ktore nie powinny byc widoczne.

### Remediacja (Secure)
- Uzyj query parametryzowanych (placeholders `?`) zamiast konkatenacji.
- Dodaj limit wynikow i podstawowa walidacje dlugosci inputu.
- W raporcie pokaz diff: vulnerable string query vs prepared/parameterized query.

---

## 2) Stored XSS (wymagana)

### Trudnosc
- Wdrozenie do obecnego programu: 2/5
- Pokazanie na demo: 1/5

### Jak wdrozyc (Vulnerable)
- Upewnij sie, ze tresc posta moze zawierac HTML/JS i jest renderowana jako raw HTML.
- W widoku listy/szczegolu postow dodaj renderowanie bez escapingu (np. trusted/raw HTML).

Najlepsze miejsce zmian:
- wejscie danych: create/edit posta,
- wyswietlenie danych: komponenty Templ renderujace post_content.

### Jak pokazac atak
- Krok 1: utworz post z payloadem JS (np. prosty alert lub odczyt document.cookie w wersji demo).
- Krok 2: otworz liste postow i pokaz automatyczne wykonanie skryptu.
- Krok 3: odswiez fragment HTMX i pokaz, ze payload wykonuje sie rowniez po partial refresh.

### Remediacja (Secure)
- Nigdy nie renderuj nieufnych danych jako raw HTML.
- Escape output + opcjonalnie whitelista tagow (sanityzacja po stronie serwera).
- Dodaj CSP (Content-Security-Policy), zeby ograniczyc skutki ewentualnego XSS.

---

## 3) Broken Authentication (bardzo polecana)

### Trudnosc
- Wdrozenie do obecnego programu: 1/5
- Pokazanie na demo: 1/5

### Jak wdrozyc (Vulnerable)
- Pozostaw logowanie oparte tylko o istnienie username (bez weryfikacji hasla).
- Brak limitu prob logowania i brak opoznien miedzy probami.
- Utrzymuj proste cookie sesyjne bez dodatkowych kontroli.

Najlepsze miejsce zmian:
- login JSON i login UI,
- helpery auth cookie.

### Jak pokazac atak
- Krok 1: zaloguj sie na istniejace konto podajac dowolne haslo.
- Krok 2: pokaz, ze serwer zwraca sukces i pozwala wykonywac operacje wymagajace logowania.

### Remediacja (Secure)
- Hashowanie hasel (bcrypt/argon2) i pelna weryfikacja hashy.
- Ograniczanie prob (rate limit/lockout), komunikaty bez ujawniania czy user istnieje.
- Cookie: HttpOnly, Secure, SameSite + rotacja sesji po loginie.

---

## 4) Broken Access Control (IDOR/BOLA) (polecana)

### Trudnosc
- Wdrozenie do obecnego programu: 2/5
- Pokazanie na demo: 2/5

### Jak wdrozyc (Vulnerable)
- Dodaj relacje post -> owner (user).
- Celowo nie sprawdzaj, czy zalogowany user jest wlascicielem przy edit/delete.
- Pozwol modyfikowac post po samym ID.

Najlepsze miejsce zmian:
- migracja DB (owner_id),
- create/update/delete w service,
- kontrola uprawnien w handlerach.

### Jak pokazac atak
- Krok 1: user A tworzy post.
- Krok 2: user B loguje sie i wywoluje edit/delete na ID posta usera A.
- Krok 3: pokaz, ze operacja przechodzi mimo braku uprawnien.

### Remediacja (Secure)
- W kazdej operacji mutujacej sprawdz owner_id vs aktualny user.
- Dla braku uprawnien zwracaj 403.
- Rozwaz policy helper (jedno miejsce dla reguly autoryzacji).

---

## 5) CSRF (polecana)

### Trudnosc
- Wdrozenie do obecnego programu: 3/5
- Pokazanie na demo: 3/5

### Jak wdrozyc (Vulnerable)
- Utrzymuj operacje POST oparte tylko o cookie sesyjne.
- Nie dodawaj tokenow CSRF do formularzy i endpointow UI.

Najlepsze miejsce zmian:
- formularze create/edit/delete postow,
- endpointy UI POST.

### Jak pokazac atak
- Krok 1: user loguje sie do aplikacji.
- Krok 2: odwiedza zlosliwa strone, ktora automatycznie wysyla ukryty formularz POST do Waszej aplikacji.
- Krok 3: pokaz, ze akcja (np. usuniecie/edycja) wykonuje sie bez wiedzy usera.

### Remediacja (Secure)
- Synchronizer token (unikalny CSRF token per sesja/formularz).
- Weryfikacja Origin/Referer dla operacji krytycznych.
- Cookie SameSite=Lax/Strict jako warstwa dodatkowa (nie jedyna).

---

## 6) Sensitive Data Exposure (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 2/5
- Pokazanie na demo: 2/5

### Jak wdrozyc (Vulnerable)
- Przechowuj hasla jawnie (plain text) albo pseudo-hash bez soli.
- Zwracaj zbyt wiele informacji o userze w odpowiedziach lub logach.

### Jak pokazac atak
- Krok 1: pokaz wpisy w DB (np. users.password_hash z faktycznym haslem).
- Krok 2: pokaz, ze wyciek danych natychmiast kompromituje konta.

### Remediacja (Secure)
- Tylko bezpieczne hashowanie (bcrypt/argon2 + odpowiedni cost).
- Minimalizacja danych w logach i odpowiedziach API.
- Ograniczenie dostepu do artefaktow backup/log.

---

## 7) Security Misconfiguration (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 2/5
- Pokazanie na demo: 2/5

### Jak wdrozyc (Vulnerable)
- Zostaw debug mode i szczegolowe komunikaty bledow SQL na produkcji demo.
- Brak naglowkow bezpieczenstwa (CSP, X-Content-Type-Options, itp.).

### Jak pokazac atak
- Krok 1: wymus blad i pokaz, ze aplikacja ujawnia szczegoly wewnetrzne.
- Krok 2: pokaz brak kluczowych naglowkow bezpieczenstwa w odpowiedzi HTTP.

### Remediacja (Secure)
- Generyczne komunikaty bledow dla klienta, szczegoly tylko w logach serwera.
- Middleware ustawiajacy minimalny zestaw bezpiecznych naglowkow.

---

## 8) Path Traversal / LFI (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 3/5
- Pokazanie na demo: 2/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint typu GET /files?name=..., ktory czyta plik z dysku na podstawie parametru.
- W vulnerable nie waliduj sciezki i lacz katalog bazowy z parametrem bez oczyszczania.

Najlepsze miejsce zmian:
- nowy handler (np. /files),
- nowa metoda serwisowa do odczytu pliku.

### Jak pokazac atak
- Krok 1: pokaz legalny odczyt pliku z katalogu dozwolonego.
- Krok 2: wyslij parametr z sekwencja przejscia po katalogach.
- Krok 3: pokaz odczyt pliku spoza katalogu aplikacji.

### Remediacja (Secure)
- Canonicalize path (np. filepath.Clean) i twardo sprawdz, czy wynik zostaje w dozwolonym katalogu.
- Wprowadz allowliste nazw/rozszerzen plikow.
- Dla naruszenia zasad zwracaj 403.

---

## 9) Command Injection (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 4/5
- Pokazanie na demo: 3/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint administracyjny uruchamiajacy polecenie systemowe na podstawie inputu (np. ping/check).
- W vulnerable przekaz parametr bezposrednio do shella.

Najlepsze miejsce zmian:
- osobny handler admin,
- warstwa serwisowa odpalajaca polecenia.

### Jak pokazac atak
- Krok 1: pokaz poprawne dzialanie dla legalnej wartosci.
- Krok 2: dodaj payload z separatorem polecen i dodatkowa komenda.
- Krok 3: pokaz, ze serwer wykonal nieautoryzowane polecenie.

### Remediacja (Secure)
- Nie uruchamiaj shella na danych uzytkownika.
- Jesli musisz, stosuj scisla allowliste argumentow i wywolania bez shell interpolation.
- Ogranicz uprawnienia procesu aplikacji.

---

## 10) Insecure Deserialization (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 4/5
- Pokazanie na demo: 3/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint przyjmujacy zserializowany obiekt (np. JSON z polem typu/akcji) i wykonujacy logike na podstawie danych bez walidacji.
- W vulnerable traktuj przychodzacy obiekt jako zaufany i pozwol sterowac krytycznymi polami.

Najlepsze miejsce zmian:
- nowy endpoint importu/restore,
- dedykowany model payloadu.

### Jak pokazac atak
- Krok 1: wyslij poprawny obiekt i pokaz normalne dzialanie.
- Krok 2: wyslij zlosliwy obiekt z nadpisanymi polami uprzywilejowanymi.
- Krok 3: pokaz nieuprawniony efekt (np. zmiana roli/flagi).

### Remediacja (Secure)
- Scisla walidacja schematu i whitelistowanie pol.
- Ignorowanie pol, ktorych klient nie powinien kontrolowac.
- Podpisywanie danych lub mapowanie DTO -> model domenowy po stronie serwera.

---

## 11 XXE (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 4/5
- Pokazanie na demo: 3/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint XML (np. import konfiguracji).
- Skonfiguruj parser tak, by przetwarzal encje zewnetrzne.

Najlepsze miejsce zmian:
- nowy endpoint XML,
- osobna funkcja parsowania XML.

### Jak pokazac atak
- Krok 1: wyslij poprawny XML i pokaz normalny wynik.
- Krok 2: wyslij payload XML z deklaracja encji zewnetrznej.
- Krok 3: pokaz odczyt lokalnego zasobu lub probe pobrania zasobu zewnetrznego.

### Remediacja (Secure)
- Wylacz przetwarzanie DTD i encji zewnetrznych.
- Ogranicz parser do minimalnego podzbioru XML.
- Rozwaz przejscie na JSON dla wejscia od uzytkownika.

---

## 12) SSRF (dodatkowa)

### Trudnosc
- Wdrozenie do obecnego programu: 4/5
- Pokazanie na demo: 3/5

### Jak wdrozyc (Vulnerable)
- Dodaj endpoint typu /fetch?url=..., ktory pobiera zawartosc URL podanego przez uzytkownika.
- W vulnerable nie filtruj hosta/IP/protokolu i zwracaj odpowiedz dalej.

Najlepsze miejsce zmian:
- nowy handler fetch,
- logika HTTP client w service.

### Jak pokazac atak
- Krok 1: pokaz poprawne pobranie publicznej strony.
- Krok 2: wyslij URL do zasobu wewnetrznego lub lokalnego.
- Krok 3: pokaz, ze aplikacja staje sie proxy do niedostepnych zasobow.

### Remediacja (Secure)
- Allowlista docelowych hostow/protokolow.
- Blokada adresow lokalnych, loopback, link-local i prywatnych zakresow IP.
- Limit timeoutow, rozmiaru odpowiedzi i przekierowan.

---

## Rekomendowany zestaw na obrone (2-osobowy zespol)
Wymagane minimum dla n=2 to:
- 2 podatnosci obowiazkowe: SQL Injection, XSS
- 3 dodatkowe: Broken Authentication, Broken Access Control (IDOR/BOLA), CSRF

Finalny zestaw projektu pokazuje minimum oraz sensowne bonusy pasujace do aplikacji:
- 2 obowiazkowe: SQL Injection, Stored XSS
- 6 dodatkowych: Broken Authentication, Broken Access Control, CSRF, Sensitive Data Exposure, Path Traversal / LFI, Command Injection

To jest dobry zakres na obrone: minimum dla n=2 jest spelnione, a Path Traversal i Command Injection sa naturalnie osadzone w blogu przez uploady/pliki oraz endpoint diagnostyczny ping.

## Rekomendowany zestaw na obrone (3-osobowy zespol)
Wymagane minimum dla n=3 to:
- 2 podatnosci obowiazkowe: SQL Injection, XSS
- 5 dodatkowych: Broken Authentication, Broken Access Control, CSRF, Sensitive Data Exposure, Security Misconfiguration

## Pelna lista dodatkowych podatnosci do wyboru (2n - 1)
1. Broken Access Control
2. CSRF
3. Insecure Deserialization
4. Security Misconfiguration
5. Broken Authentication
6. Path Traversal / LFI
7. Command Injection
8. Sensitive Data Exposure
9. XXE
10. SSRF

## Gotowe pakiety wyboru, jesli jeszcze nie podjeliscie decyzji

## Zestawy dla zespolu 2-osobowego (n=2, wybieracie 3 dodatkowe)

Uwaga: to sa zestawy przykladowe. Nie zawieraja naraz wszystkich podatnosci dodatkowych,
bo dla n=2 wybieracie tylko 3 pozycje. Dlatego ponizej sa tez warianty obejmujace
Insecure Deserialization, XXE i SSRF.

Skala ryzyka obrony:
- 1/5 = bardzo niskie ryzyko
- 3/5 = srednie ryzyko
- 5/5 = wysokie ryzyko

### Zestaw 2A - Najszybszy do dowiezienia
- Broken Authentication
- Broken Access Control
- CSRF
- Szacowany czas realizacji: 4-6 dni roboczych
- Ryzyko na obronie: 2/5
- Rozpiska pod technologie:
  - Gin/HTTP: kontrola dostepu i sesji w [internal/handlers/handlers.go](internal/handlers/handlers.go), bez zmiany kontraktu endpointow.
  - SQLite/service: owner check i operacje mutujace w [internal/service/service.go](internal/service/service.go).
  - Templ/HTMX: token CSRF i ukryte pola formularzy w [internal/views/pages.templ](internal/views/pages.templ), walidacja na POST.
  - DB schema: owner_id i powiazania tylko przez migracje w [internal/db/db.go](internal/db/db.go).
  - Toggle secure/vulnerable: rozgalezienia przez SecurityEnabled z [main.go](main.go).
- Jak pokazac na demo (krotko):
  - Broken Auth: login z blednym haslem przechodzi w vulnerable, blokowany w secure.
  - IDOR: user B edytuje post usera A po ID w vulnerable, dostaje 403 w secure.
  - CSRF: ukryty formularz wykonuje akcje w vulnerable, token blokuje w secure.

### Zestaw 2B - Latwy i bardzo efektowny na demo
- Broken Authentication
- Broken Access Control
- Sensitive Data Exposure
- Szacowany czas realizacji: 3-5 dni roboczych
- Ryzyko na obronie: 1/5
- Rozpiska pod technologie:
  - Gin/handlers: logowanie i cookie flow w [internal/handlers/handlers.go](internal/handlers/handlers.go).
  - SQLite/users: sposob zapisu i odczytu hasel w [internal/service/service.go](internal/service/service.go) oraz seed/migracje w [internal/db/db.go](internal/db/db.go).
  - API/UI spojne: ten sam blad auth widoczny na /login i /ui/login.
  - Toggle secure/vulnerable: ten sam endpoint, rozne zachowanie po SecurityEnabled.
- Jak pokazac na demo (krotko):
  - Broken Auth: dowolne haslo dla istniejacego usera.
  - IDOR: modyfikacja cudzego zasobu po ID.
  - Sensitive Data: pokaz roznice w przechowywaniu hasel i skutki wycieku DB.

### Zestaw 2C - Backend + konfiguracja (stabilny)
- Broken Authentication
- Security Misconfiguration
- Sensitive Data Exposure
- Szacowany czas realizacji: 3-4 dni robocze
- Ryzyko na obronie: 1/5
- Rozpiska pod technologie:
  - Gin middleware: naglowki i obsluga bledow centralnie w [main.go](main.go) i handlerach.
  - SQLite: model danych usera i seed pod wariant vulnerable/secure w [internal/db/db.go](internal/db/db.go).
  - Service layer: bezpieczne porownanie hasel i minimalizacja wyciekow w [internal/service/service.go](internal/service/service.go).
  - Templ: komunikaty dla UI bez ujawniania detali systemowych w [internal/views/pages.templ](internal/views/pages.templ).
- Jak pokazac na demo (krotko):
  - Broken Auth: obejscie logowania vs poprawna walidacja.
  - Misconfiguration: szczegolowy blad i brak naglowkow vs generyczny blad i naglowki ochronne.
  - Sensitive Data: porownanie danych hasel przed/po remediacji.

### Zestaw 2D - Rownowaga miedzy prostota i technicznoscia
- Broken Access Control
- CSRF
- Security Misconfiguration
- Szacowany czas realizacji: 4-6 dni roboczych
- Ryzyko na obronie: 2/5
- Rozpiska pod technologie:
  - Gin routes: kontrola uprawnien na PUT/DELETE/POST w [internal/handlers/handlers.go](internal/handlers/handlers.go).
  - Templ/HTMX forms: token CSRF i obsluga bledow walidacji w [internal/views/pages.templ](internal/views/pages.templ).
  - SQLite/service: owner-based authorization i checki przed mutacja w [internal/service/service.go](internal/service/service.go).
  - Bezpieczna konfiguracja: naglowki i odpowiedzi bledow bez szczegolow wewnetrznych.
- Jak pokazac na demo (krotko):
  - IDOR: modyfikacja cudzego posta po ID.
  - CSRF: wymuszone POST z obcej strony.
  - Misconfiguration: porownanie odpowiedzi HTTP i naglowkow.

### Zestaw 2E - Pod API i pliki
- Broken Access Control
- Path Traversal / LFI
- Security Misconfiguration
- Szacowany czas realizacji: 5-7 dni roboczych
- Ryzyko na obronie: 3/5
- Rozpiska pod technologie:
  - Nowy endpoint plikowy w Gin (np. /files) w [internal/handlers/handlers.go](internal/handlers/handlers.go).
  - Odczyt plikow i walidacja sciezek po stronie service w [internal/service/service.go](internal/service/service.go).
  - Kontrola uprawnien owner-based dla operacji na postach pozostaje w service/handlers.
  - Templ: prosty UI do testu odczytu pliku i komunikatow bledu.
- Jak pokazac na demo (krotko):
  - IDOR: dostep do cudzego rekordu.
  - LFI: odczyt pliku spoza katalogu dozwolonego w vulnerable.
  - Misconfiguration: zbyt szczegolne bledy ulatwiajace atak.

### Zestaw 2F - Ambitny, ale nadal realny
- CSRF
- Path Traversal / LFI
- Command Injection
- Szacowany czas realizacji: 6-9 dni roboczych
- Ryzyko na obronie: 4/5
- Rozpiska pod technologie:
  - Gin: nowy endpoint do akcji systemowej (tylko labowo) i endpoint plikowy.
  - Service: separacja logiki plikow i wywolan systemowych w [internal/service/service.go](internal/service/service.go).
  - Templ/HTMX: formularze testowe dla CSRF oraz wejscia do endpointow plik/command w [internal/views/pages.templ](internal/views/pages.templ).
  - Konieczna ostroznosc: to najbardziej inwazyjny wariant dla aktualnej architektury.
- Jak pokazac na demo (krotko):
  - CSRF: wymuszona akcja przez obcy formularz.
  - LFI: traversal po sciezce.
  - Command Injection: dodatkowe polecenie przez payload w vulnerable.

### Zestaw 2G - Zaawansowany pod JSON/API
- Broken Authentication
- Insecure Deserialization
- SSRF
- Szacowany czas realizacji: 7-10 dni roboczych
- Ryzyko na obronie: 4/5
- Rozpiska pod technologie:
  - Gin/API: endpoint importu obiektu i endpoint fetch URL w [internal/handlers/handlers.go](internal/handlers/handlers.go).
  - Service: walidacja schematu danych i kontrola docelowych hostow URL w [internal/service/service.go](internal/service/service.go).
  - SQLite: ewentualny zapis importowanych danych i logika domenowa bez zaufania do pol klienta.
  - Toggle: ten sam request pokazany w vulnerable i secure.
- Jak pokazac na demo (krotko):
  - Insecure Deserialization: zlosliwy obiekt nadpisuje pola krytyczne w vulnerable.
  - SSRF: aplikacja pobiera URL lokalny/wewnetrzny w vulnerable.
  - Broken Auth: latwy punkt startowy i szybki kontrast secure.

### Zestaw 2H - Zaawansowany pod XML
- Broken Authentication
- XXE
- SSRF
- Szacowany czas realizacji: 7-10 dni roboczych
- Ryzyko na obronie: 4/5
- Rozpiska pod technologie:
  - Gin: endpoint XML (import) i endpoint fetch URL.
  - Parser XML: celowo niebezpieczna konfiguracja w vulnerable, blokada DTD/encji w secure.
  - Service: filtrowanie hostow/IP i timeouty dla SSRF.
  - UI opcjonalne: wystarczy API demo przez curl/Burp.
- Jak pokazac na demo (krotko):
  - XXE: payload XML z encja zewnetrzna dziala w vulnerable, nie dziala w secure.
  - SSRF: pobranie zasobu lokalnego/wewnetrznego blokowane w secure.
  - Broken Auth: szybka i czytelna podatnosc uzupelniajaca zestaw.

### Zestaw 2I - Najbardziej merytoryczny (trudny)
- Insecure Deserialization
- XXE
- SSRF
- Szacowany czas realizacji: 9-12 dni roboczych
- Ryzyko na obronie: 5/5
- Rozpiska pod technologie:
  - Wymaga dodania trzech nowych wektorow wejscia: JSON obiektowy, XML parser, fetch URL.
  - Najwiecej zmian w [internal/handlers/handlers.go](internal/handlers/handlers.go) i [internal/service/service.go](internal/service/service.go).
  - Wysokie ryzyko czasowe, ale bardzo mocny pokaz techniczny na obronie.
- Jak pokazac na demo (krotko):
  - Insecure Deserialization: manipulacja obiektem i skutkiem biznesowym.
  - XXE: odczyt zasobu przez encje zewnetrzna w vulnerable.
  - SSRF: aplikacja jako proxy do zasobow niedostepnych z zewnatrz.

## Zestawy dla zespolu 3-osobowego (n=3, wybieracie 5 dodatkowych)

### Zestaw 3A - Najlatwiejszy do wdrozenia i obrony
- Broken Authentication
- Broken Access Control
- CSRF
- Sensitive Data Exposure
- Security Misconfiguration

### Zestaw 3B - Srednia trudnosc i dobra roznorodnosc
- Broken Authentication
- Broken Access Control
- CSRF
- Path Traversal / LFI
- Command Injection

### Zestaw 3C - Zaawansowany (duzo pracy, mocny efekt na obronie)
- Broken Authentication
- Broken Access Control
- Insecure Deserialization
- XXE
- SSRF

## Proponowana kolejnosc wdrozen (najmniejsze ryzyko czasowe)
1. Broken Authentication (najszybszy win na demo)
2. Stored XSS
3. SQL Injection
4. Broken Access Control
5. CSRF
6. Sensitive Data Exposure
7. Security Misconfiguration
8. Path Traversal / LFI
9. Command Injection
10. Insecure Deserialization
11. XXE
12. SSRF

## Format prezentacji na live demo (uniwersalny szablon)
Dla kazdej podatnosci trzy kroki:
1. Vulnerable mode: pokaz atak krok po kroku i efekt.
2. Secure mode: powtorz ten sam atak i pokaz, ze nie dziala.
3. Code review: pokaz konkretny fragment przed/po i nazwij mechanizm obronny.

Dzieki temu obrona jest czytelna, powtarzalna i zgodna z wymaganiami z info.md.

## 13) Jakie funkcje dodac do kawaii bloga, zeby podatnosci mialy naturalne miejsce

To jest praktyczny opis funkcjonalnosci aplikacji, ktore warto miec w takim blogu, aby kazda podatnosc wygladala wiarygodnie, pasowala do fabuly aplikacji i mogla byc pokazana bez sztucznego dopisywania osobnego labu.

### Rdzen aplikacji, ktory powinien zostac

1. Konta uzytkownikow i sesja.
- Rejestracja, logowanie, wylogowanie.
- Profil z nazwa uzytkownika, avatarem i krotkim bio.
- Zapamietywanie zalogowania przez cookie sesyjne.

2. Posty blogowe.
- Tworzenie posta, edycja posta, usuwanie posta.
- Status postu: draft / published / featured.
- Tytul, tresc, tagi, miniatura i kategoria.

3. Interakcje pod postami.
- Komentarze.
- Odpowiedzi na komentarze.
- Reakcje typu like / heart / star.
- Zglaszanie komentarza do moderacji.

4. Wyszukiwanie i filtrowanie.
- Wyszukiwarka po tytule i tresci.
- Filtry po tagach, autorze, dacie i statusie.
- Stronicowanie wynikow.

5. Elementy kawaii / social.
- Ozdobne awatary.
- Miejsce na emoji, naklejki, badge i pastelowe motywy.
- Panel ustawien wygladu profilu i motywu.

6. Upload plikow.
- Avatar uzytkownika.
- Zdjecia do postow.
- Zalaczniki, np. pliki z inspiracjami, szkicami lub moodboardami.

7. Funkcje administracyjne.
- Panel admina do moderacji postow i komentarzy.
- Lista aktywnosci, logow i bledow.
- Narzedzia pomocnicze, np. podglad statusu, restart integracji, eksport danych.

8. Integracje zewnetrzne.
- Podglad linkow z innych stron.
- Pobieranie miniatur z podanego URL.
- Import posta z zewnetrznego feeda.

### Jakie funkcje daja miejsce dla konkretnej podatnosci

1. SQL Injection.
- Dodaj wyszukiwarke postow, komentarzy albo uzytkownikow.
- Dobry punkt to filtr po tagu, autorze, tytule albo frazie w tresci.
- Wystarczy zwykle pole tekstowe z parametrem `q` albo `author`.

2. Stored XSS.
- Pozwolenie na HTML w tresci posta, opisie profilu albo komentarzach.
- Sekcja „bio”, „status”, „about me” lub „wyraz siebie” jest naturalnym miejscem na payload.
- XSS dobrze wyglada tez w podgladzie komentarzy i w partialu HTMX.

3. Broken Authentication.
- Logowanie na username + password, reset hasla, remember me, zmiana emaila.
- Strona „Moje konto” z sesja cookie daje naturalne miejsce na obejscie logowania.
- W vulnerable wariancie wystarczy sprawdzanie samego username albo slaby flow sesji.

4. Broken Access Control / IDOR.
- Edycja i usuwanie wlasnych postow po ID.
- Prywatne szkice, zapisane drafty, prywatne komentarze albo ulubione posty.
- To naturalnie prowadzi do prob modyfikacji zasobow innego uzytkownika po zmianie identyfikatora.

5. CSRF.
- Formularze zmiany hasla, emaila, bio, avatara i publikacji posta.
- Akcje „follow / unfollow”, „like”, „publish”, „delete draft” albo „change avatar” sa bardzo dobre do demonstracji.
- Im wiecej formularzy POST i ustawien konta, tym latwiej pokazac skutki braku tokena CSRF.

6. Sensitive Data Exposure.
- Profil uzytkownika z emailem, tokenem API, data urodzenia, preferencjami i lista prywatnych wpisow.
- Panel admina lub export konta, gdzie zbyt duzo danych wraca w odpowiedzi.
- W blogu sensownie wyglada tez trzymanie zbyt wielu danych o autorze w odpowiedzi JSON.

7. Security Misconfiguration.
- Strony bledu z debugiem, stack trace, nazwa bazy i pelnym SQL.
- Publiczny endpoint healthcheck, status integracji, debug panel lub pagina dla admina.
- Brak naglowkow bezpieczenstwa i zbyt gadatliwe komunikaty sa naturalne w „szybko postawionym” blogu.

8. Path Traversal / LFI.
- Odczyt avatarow, motywow, szkicow lub zalacznikow po nazwie pliku.
- Podglad pliku „theme”, „banner”, „cover image” albo „draft export”.
- W blogu z plikami i motywami traversal wyglada wiarygodnie.

9. Command Injection.
- Narzedzia admina typu „optimize image”, „ping remote mirror”, „generate sitemap”, „export thumbnails”.
- Panel konserwacyjny przy uploadach lub generowaniu miniaturek daje naturalny pretekst do wywolania polecen systemowych.

10. Insecure Deserialization.
- Import ustawien profilu, import motywu, import listy ulubionych albo restore konta z pliku.
- JSON jest tutaj mniej pasujacy do klasycznej deserializacji, wiec lepiej brzmi import zarchiwizowanego pakietu danych.

11. XXE.
- Import konfiguracji profilu, feed XML albo starszy format eksportu postow.
- Kiedy blog przyjmuje XML dla importu ustawien, latwo pokazac blad parsera.

12. SSRF.
- Podglad linkow, pobieranie miniatur, unfurling URL, import z zewnetrznego feeda albo webhook preview.
- To bardzo naturalne w blogu, bo aplikacja „sprawdza link”, „pobiera miniaturke” albo „synchronizuje post z innym serwisem”.

### Najbardziej naturalny zestaw funkcjonalnosci dla Waszego kawaii bloga

Jesli chcesz, zeby blog wygladal spójnie i nie byl przeładowany, warto oprzec go o ten zestaw:
- konta i profil,
- posty i szkice,
- komentarze i reakcje,
- wyszukiwarka,
- upload avatara i obrazkow,
- panel admina,
- import / export ustawien,
- podglad linkow zewnetrznych.

To wystarczy, zeby wszystkie wymagane i dodatkowe podatnosci mialy wiarygodne miejsce w fabule jednej aplikacji blogowej, zamiast wygladac jak przypadkowo doklejone laby.
