# Plan Implementacji Podatnosci (Tydzien po Tygodniu)

## Cel dokumentu
Ten plan sluzy do prowadzenia prac nad aplikacja Vulnerable vs Secure w modelu:
- najpierw implementacja podatnej wersji,
- potem remediacja w trybie Secure,
- na koncu scenariusz atak -> blokada -> roznice w kodzie.

Plan obejmuje komplet podatnosci wymaganych na zajeciach oraz gotowe warianty dla zespolu 2-osobowego i 3-osobowego.

---

## TL;DR — stan na 2026-05-15

### ETAP E ZAKONCZONY ✅ + BONUSY
Finalny zakres dla zespolu **n=2**: **8 podatnosci** do pokazania na obronie.
Minimum dla n=2 to 2 obowiazkowe + 3 dodatkowe; projekt ma 2 obowiazkowe + 6 dodatkowych, wiec jest zapas.

### Co jest zrobione end-to-end (atak -> blokada -> diff kodu + testy)
- **SQL Injection** — `/api/search` (toggle), `/api/search-vulnerable` (force-vuln), `/ui/search` z gotowymi payloadami; testy integracyjne dla obu trybow.
- **Stored XSS** — komentarze pod postami (`/ui/posts/view/:id`); vuln rendering przez `@templ.Raw(c.Body)`, secure przez `{ c.Body }` (templ auto-escape) + strip HTML na zapisie; testy potwierdzaja `<script>` strzela w vuln a jest escape w secure.
- **Broken Authentication** — vuln akceptuje dowolne haslo dla istniejacego usera, secure waliduje przez `bcrypt.CompareHashAndPassword`.
- **Broken Access Control** — w secure trybie tylko autor lub admin moze usunac post (testy autoryzacji dla `/posts/:id` DELETE).
- **Sensitive Data Exposure** — vuln zapisuje hasla plaintext do SQLite, secure zapisuje hash bcrypt (widoczne w `sqlite> SELECT password_hash FROM users;`).
- **CSRF** — vuln: `/csrf-vulnerable-form` i `/ui/csrf-demo` bez tokena; secure: `/ui/csrf-secure` z per-form CSRF tokenem (cookie `bai_csrf_token` + hidden input `csrf_token`) walidowanym przed update emaila.
- **Path Traversal / LFI** (BONUS) — vuln: `/api/files-vulnerable?name=../../etc/passwd` laduje `os.ReadFile(filepath.Join("./uploads", name))`; secure: `/api/files-secure` waliduje, ze sciezka po `filepath.Clean` zostaje wewnatrz `./uploads`.
- **Command Injection** (BONUS) — vuln: `/api/ping-vulnerable?host=8.8.8.8; cat /etc/passwd` laduje `exec.Command("sh","-c","ping "+host)`; secure: `/api/ping-secure` uzywa `exec.Command("ping","-c1",host)` + regex whitelist na hostname.

**8 podatnosci gotowych** (2 obowiazkowe + 6 dodatkowych). **Wymagane minimum dla wariantu n=2 (2 + 3 dodatkowe) jest spelnione z naddatkiem.**

### Co dziala dobrze i nie wymaga poprawek (OK)
- Toggle `SECURITY_ENABLED` jako jedyny przelacznik; zaden scenariusz nie wymaga osobnych aplikacji.
- Cookie sesji `bai_auth_user` (HTTP-only, TTL 8h) plus seedowany admin z env.
- Endpointy "force-vulnerable" (`/api/search-vulnerable`, `/api/comments-vulnerable`) zostaja podatne nawet w trybie secure — to celowe dla side-by-side demo.
- UI handoff z claude.ai/design: sakura, Burp-style Request Inspector, Attack Timeline z eksportem PoC do md, cheat-sheet drawer z 14 sekcjami i filtrem, mascot reagujacy na tryb.
- Hub `/ui/vuln-demos` z 8 kartami (CWE/OWASP) — szybki dostep do kazdego scenariusza w trakcie demo.
- Wszystkie testy integracyjne zielone (`go test -tags=integration -count=1 .`).
- Layout naprawiony: Tailwind config skanuje pliki `.go` (nie tylko `.templ`/`*_templ.go`), wiec klasy z `layout_helpers.go` (`flex-col`, `min-h-screen`, `bg-shell`) trafiaja do CSS.

### Co wymaga poprawy / dopracowania (warto zrobic przed obrona)
- **Walidacja inputow** — niewielka po stronie serwera (długosci, format email). W secure mode warto dolozyc twardy minimum dla raportu.
- **Sample data** — alice (autor postu Stored XSS) ma haslo `alicepass` w seedzie; warto wymienic na coś realniejszego dla demo.
- **CHEAT-SHEET drawer** — zostaly tytuly z poprzedniej iteracji, czesc bez sekcji "What/Detect/Defense"; warto zunifikowac nazewnictwo (anglojezyczne vs polskojezyczne wpisy w sekcji "PAYLOADS").

### Co zostalo do zrobienia (Etap F — finalizacja)
1. **Dopisac sekcje raportu** dla wybranych 8 podatnosci — opis + PoC + diff przed/po dla SQLi, XSS, Broken Auth, BAC, CSRF, SDE, Path Traversal, Command Injection. (~S kazda)
2. **Probna obrona** — przejscie po wszystkich scenariuszach z zegarkiem, stoperowanie i polish skryptu prezentacji. (~M)
3. **Slajdy / prezentacja** — opcjonalnie 3-5 slajdow na obrone (architektura + checklista wszystkich podatnosci).

Opcjonalnie (jezeli sie zmiesci czasowo, jako ekstra plus):
- **Security Misconfiguration** — middleware naglowkow security + sanitizacja bledow. Nie jest wymagane w finalnym zakresie n=2, bo Path Traversal i Command Injection sa juz wybrane jako bonusy.

### Stan testow
```
$ go test -tags=integration -count=1 .
ok    BAI_1IZ21B_PROJEKT    4.094s
```
Wszystkie 13+ testow integracyjnych zielone (auth, posts CRUD, search SQLi vuln/secure, stored XSS vuln/secure, autoryzacja delete).

---

## Zasady pracy
- Zachowajcie stale API i trasy UI.
- Dla kazdej podatnosci zrobcie branch logiczny przez `SECURITY_ENABLED`.
- Kazdy scenariusz musi miec:
  - PoC ataku,
  - dowod blokady w secure,
  - krotki diff kodu przed/po.
- Po kazdym tygodniu aktualizujcie raport postepu.

## Skala estymacji
- S (Small): 0.5-1.5 dnia
- M (Medium): 1.5-3 dni
- L (Large): 3-5 dni

## Priorytety
- P1: krytyczne dla zaliczenia i najszybszy efekt
- P2: mocna wartosc merytoryczna
- P3: zaawansowane, opcjonalne przy braku czasu

---

## Backlog podatnosci (priorytet + estymacja)

### Obowiazkowe
1. SQL Injection
- Priorytet: P1
- Estymacja: M
- Zaleznosci: endpoint wyszukiwania lub filtrowania

2. Stored/Reflected XSS
- Priorytet: P1
- Estymacja: M
- Zaleznosci: render danych uzytkownika w widokach

### Dodatkowe (najlatwiejsze do wdrozenia)
3. Broken Authentication
- Priorytet: P1
- Estymacja: S
- Zaleznosci: login/session cookie

4. Broken Access Control (IDOR/BOLA)
- Priorytet: P1
- Estymacja: M
- Zaleznosci: owner_id i kontrola uprawnien przy modyfikacji

5. CSRF
- Priorytet: P2
- Estymacja: M
- Zaleznosci: formularze POST + sesja cookie

6. Sensitive Data Exposure
- Priorytet: P2
- Estymacja: S
- Zaleznosci: rejestracja/logowanie + przechowywanie hasel

7. Security Misconfiguration
- Priorytet: P2
- Estymacja: S
- Zaleznosci: middleware naglowkow, obsluga bledow

### Dodatkowe (bardziej zaawansowane)
8. Path Traversal / LFI
- Priorytet: P2
- Estymacja: M
- Zaleznosci: endpoint odczytu plikow

9. Command Injection
- Priorytet: P3
- Estymacja: L
- Zaleznosci: endpoint operacji systemowej

10. Insecure Deserialization
- Priorytet: P3
- Estymacja: L
- Zaleznosci: endpoint importu/restore obiektow

11. XXE
- Priorytet: P3
- Estymacja: L
- Zaleznosci: endpoint XML + parser

12. SSRF
- Priorytet: P3
- Estymacja: L
- Zaleznosci: endpoint fetch URL

---

## Harmonogram 6-tygodniowy (realny i bezpieczny czasowo)

## Tydzien 1 - Fundament pod demo i raport
Zakres:
1. Domkniecie scenariusza Broken Authentication (vulnerable + secure).
2. Domkniecie scenariusza Stored XSS (vulnerable + secure).
3. Szkic sekcji raportu: opis + PoC + remediacja dla 2 podatnosci.

Definition of done:
- 2 podatnosci gotowe end-to-end do pokazania na zywo.
- Materialy PoC zapisane i powtarzalne.

Szacowany naklad:
- 2-4 dni robocze lacznie.

## Tydzien 2 - SQL Injection i stabilizacja flow
Zakres:
1. Dodanie scenariusza SQL Injection (vulnerable).
2. Naprawa SQLi (secure, parametryzacja i walidacja inputu).
3. Ujednolicenie logow i odpowiedzi bledow dla demo.

Definition of done:
- 3 podatnosci (w tym 2 wymagane) gotowe do obrony.

Szacowany naklad:
- 2-3 dni robocze.

## Tydzien 3 - Dodatkowe P1
Zakres:
1. Broken Access Control (IDOR/BOLA) z owner_id i kontrola autoryzacji.
2. CSRF (token + walidacja + zmiany formularzy).

Definition of done:
- Wariant n=2 praktycznie gotowy (2 obowiazkowe + 3 dodatkowe).

Szacowany naklad:
- 3-4 dni robocze.

## Tydzien 4 - Dodatkowe P2
Zakres:
1. Sensitive Data Exposure (jawne hasla vs hashowanie).
2. Security Misconfiguration (naglowki, bezpieczne bledy).
3. Porzadki w dokumentacji i finalizacja tabel przed/po.

Definition of done:
- Wariant n=3 gotowy na poziomie 2+5.

Szacowany naklad:
- 2-3 dni robocze.

## Tydzien 5 - Opcje zaawansowane (jesli jest zapas czasu)
Zakres (wybieracie 1-2):
1. Path Traversal / LFI.
2. Command Injection.
3. SSRF.
4. Insecure Deserialization.
5. XXE.

Definition of done:
- Minimum 1 dodatkowa podatnosc zaawansowana zrobiona end-to-end.

Szacowany naklad:
- 3-5 dni roboczych (zalezy od wyboru).

## Tydzien 6 - Probna obrona i domkniecie
Zakres:
1. Pelny dry-run: Vulnerable -> Secure -> Code review.
2. Uporzadkowanie slajdow i raportu.
3. Rezerwa na poprawki po probnej obronie.

Definition of done:
- Czas prezentacji miesci sie w limicie.
- Kazdy czlonek zespolu umie poprowadzic swoja czesc bez improwizacji.

Szacowany naklad:
- 2-3 dni robocze.

---

## Warianty planu wg liczby osob

## Wariant A (n=2, minimum wymagane)
Cel:
- 2 obowiazkowe + 3 dodatkowe

Rekomendowany zestaw:
1. SQL Injection
2. XSS
3. Broken Authentication
4. IDOR
5. CSRF

Minimalny harmonogram:
- Tydzien 1-3: implementacja i testy
- Tydzien 4: raport i probna obrona

## Wariant B (n=3, minimum wymagane)
Cel:
- 2 obowiazkowe + 5 dodatkowych

Rekomendowany zestaw:
1. SQL Injection
2. XSS
3. Broken Authentication
4. IDOR
5. CSRF
6. Sensitive Data Exposure
7. Security Misconfiguration

Minimalny harmonogram:
- Tydzien 1-4: implementacja i testy
- Tydzien 5-6: zaawansowana opcja lub dopracowanie obrony

---

## Podzial pracy w zespole

### Zespol 2-osobowy
1. Osoba A
- backend security logic (SQLi, IDOR, auth)
- testy integracyjne

2. Osoba B
- UI/Templ/HTMX (XSS, CSRF)
- dokumentacja PoC i raport

### Zespol 3-osobowy
1. Osoba A
- warstwa DB + service + autoryzacja

2. Osoba B
- widoki i flow formularzy (XSS/CSRF)

3. Osoba C
- testy, scenariusz demo, materialy raportowe i dry-run

---

## Ryzyka i plan awaryjny

Ryzyko 1: za malo czasu na podatnosci zaawansowane (XXE/SSRF/Deserialization)
- Plan awaryjny: utrzymac mocny zestaw P1/P2 i nie rozszerzac zakresu.

Ryzyko 2: niestabilny scenariusz demo
- Plan awaryjny: osobna baza demonstracyjna i checklisty krokow przed wejsciem na obrone.

Ryzyko 3: regresje po zmianach secure
- Plan awaryjny: po kazdej podatnosci odpalac testy integracyjne i krotki smoke-test UI.

---

## Checklista gotowosci do obrony
- [ ] Dla kazdej pokazanej podatnosci dziala atak w vulnerable.
- [ ] Ten sam atak nie dziala w secure.
- [ ] Mamy przygotowane payloady i komendy testowe.
- [ ] Mamy fragmenty kodu przed/po do szybkiego pokazania.
- [ ] Raport zawiera opis, PoC i remediacje dla kazdej wybranej podatnosci.
- [ ] Zrobiona minimum jedna proba generalna calej prezentacji.

## Status board (uzupelniacie na biezaco)
- [x] SQL Injection (vulnerable + secure side-by-side; `/api/search` toggle, `/api/search-vulnerable` force-vulnerable, `/ui/search` z payloadami)
- [x] Stored XSS (komentarze pod postem `/ui/posts/view/:id`; vuln: `@templ.Raw(c.Body)` -> `<script>` strzela; secure: `{ c.Body }` auto-escape + strip HTML server-side; force-vuln `/api/comments-vulnerable`)
- [x] Broken Authentication (insecure: dowolne haslo dla istniejacego usera; secure: bcrypt)
- [x] Broken Access Control (admin/owner check w `PostDelete` w trybie secure)
- [x] CSRF (vuln: `/csrf-vulnerable-form` + `/ui/csrf-demo`; secure: `/ui/csrf-secure` z per-form tokenem cookie+hidden input)
- [ ] Insecure Deserialization (opcjonalne)
- [ ] Security Misconfiguration (opcjonalne poza finalnym zakresem n=2)
- [x] Path Traversal / LFI (BONUS, vuln: `/api/files-vulnerable?name=../../etc/passwd`; secure: `/api/files-secure` z `filepath.Clean` + prefix check)
- [x] Command Injection (BONUS, vuln: `/api/ping-vulnerable?host=...; cat /etc/passwd`; secure: `/api/ping-secure` z `exec.Command("ping",...)` + regex whitelist)
- [x] Sensitive Data Exposure (insecure: hasla plain w DB; secure: bcrypt hash)
- [ ] XXE (opcjonalne)
- [ ] SSRF (opcjonalne)

---

## Krok 0 — Fundament aplikacji (zrealizowany 2026-05-03)

Przed wlasciwa implementacja podatnosci ukonczono prace fundamentalne potrzebne do prowadzenia kazdego scenariusza atak/obrona oraz zaimplementowano design handoff z claude.ai/design.

### Auth + sesja
- bcrypt hash hasel w secure mode (`internal/service/service.go` + `internal/db/db.go`)
- HTTP-only cookie `bai_auth_user`, TTL 8h
- Hardcoded admin (env `ADMIN_USERNAME`/`ADMIN_PASSWORD`) seedowany do bazy
- Insecure login celowo akceptuje dowolne haslo dla istniejacego usera (Broken Auth demo)
- Secure login waliduje przez `bcrypt.CompareHashAndPassword`

### Zalaczniki do postow
- Kolumny `attachment_path`, `attachment_name` w tabeli `blog`
- `multipart/form-data` w formularzach Create/Edit (limit 5 MB)
- Zapis pod `./uploads/{nanos}_{filename}` z sanitizacja base path
- Statyczne serwowanie `/uploads/*`
- HTMX partial `/ui/partials/posts/create` — dynamiczny refresh listy

### HTMX flow
- Login partial `/ui/partials/login` zwraca `HX-Redirect` przy sukcesie (full reload pokazuje zielone pole z nickiem)
- Register partial `/ui/partials/register` z `HX-Redirect` na login
- Create post partial z `HX-Trigger: post-created`

### Design (handoff z claude.ai/design)
- Sakura petals (CSS animation co 700ms) + body classes `sec-vuln`/`sec-secure`
- Burp-style Request Inspector (lewy dolny rog) hookowany do `htmx:beforeRequest/afterRequest`
- Attack Timeline (prawy dolny rog) z eksportem PoC do markdown (`bai-lab-poc.md`)
- Cheat-Sheet drawer "PAYLOADS" — 14 sekcji (SQLi/XSS/IDOR/Path/CmdInj/SSRF/CSRF/Auth/SDE/Misconfig/Upload/Burp/Glossary), klik=copy
- Maskotka reaguje na tryb: vuln -> czerwona pulsujaca aura + dymki "uwazaj na te apostrofy!"; secure -> niebieska poswiata + "bcrypt > plaintext"
- Skeleton shimmer dla refreshu listy postow
- Achievement toasts (`SQLi pwned!` / `XSS blocked`)
- Pliki: `static/js/bai-lab-extras.js` (630 linii), `static/css/app.css` (Tailwind + design CSS), `assets/css/input.css` (zrodla)

### Testy
- Wszystkie integration testy zielone (`go test -tags=integration -count=1 .`)
- Tryb insecure i secure zweryfikowane curlem dla loginu i uploadu

---

## Krok 1 — SQL Injection (zrealizowany 2026-05-03)

### Implementacja
- `internal/service/service.go`:
  - `SearchPostsVulnerable(query)` — `"... LIKE '%" + query + "%' ..."` (string concat)
  - `SearchPostsSecure(query)` — `"... LIKE ?"` z `?` placeholderem; dodatkowo `WHERE published = 1` (defense in depth)
  - `SearchPosts(query)` — toggle wg `securityEnabled`
- `internal/handlers/handlers.go`:
  - `GET /api/search` — JSON, respektuje `SECURITY_ENABLED`
  - `GET /api/search-vulnerable` — zawsze konkatenacja (force vulnerable, do side-by-side demo)
  - `GET /ui/search` + `POST /ui/partials/search` — UI z formularzem HTMX i sekcjami payloads/defense
- Layout: link nawigacyjny "🔎 Search"

### PoC
- Vulnerable mode (`SECURITY_ENABLED=false`):
  - `' OR 1=1 --` → wycieka rowy (drafty tez)
  - `' UNION SELECT id, username, password_hash, 1, '', '', '' FROM users --` → leakuje `admin` i `user1` jako tytuly
- Secure mode (`SECURITY_ENABLED=true`):
  - Te same payloady → 0 wynikow
  - Drafty nigdy nie wychodza
  - `/api/search-vulnerable` nadal podatny (do porownania na zywo)

### Testy integracyjne
- `TestIntegration_SearchSQLi_VulnerableMode` (3 podtesty): brak match dla losowego termu, OR 1=1 wycieka drafty, UNION exfiltruje `users`
- `TestIntegration_SearchSQLi_SecureMode` (4 podtesty): OR 1=1 → 0, UNION → 0, drafty nigdy nie widoczne, force-vulnerable wciaz dziala

### Diff kodu (przed/po)
```go
// VULNERABLE — SearchPostsVulnerable
sqlQuery := "... LIKE '%" + query + "%' OR ..."
rows, _ := s.db.Query(sqlQuery)

// SECURE — SearchPostsSecure
pattern := "%" + query + "%"
rows, _ := s.db.Query(
    "... WHERE published = 1 AND (title LIKE ? OR post_content LIKE ?)",
    pattern, pattern,
)
```

---

## Krok 2 — Stored XSS (zrealizowany 2026-05-04)

### Implementacja
- Tabela `comments` (id, post_id, author, body, created_at) w `internal/db/db.go`
- `internal/service/service.go`:
  - `Comment` struct + `CreateComment(postID, author, body)` honoruje `securityEnabled`
  - secure: `sanitizeCommentBody()` strip wszystkich `<...>` przez regex (defense in depth)
  - vulnerable: zapis verbatim
  - `CreateCommentVulnerable(...)` — zawsze verbatim (force-vuln demo)
  - `GetCommentsForPost(postID)` — listing
- `internal/views/pages.templ`:
  - `templ CommentsList(comments, securityEnabled)` — kluczowy split:
    - vuln: `@templ.Raw(c.Body)` → `<script>` strzela
    - secure: `{ c.Body }` → templ auto-escape, HTML jako tekst
  - `templ PostDetailPage(...)` — strona /ui/posts/view/:id z formularzem komentarzy + listą
- `internal/handlers/handlers.go`:
  - `PagePostDetail()` GET `/ui/posts/view/:id`
  - `PagePostCommentSubmit()` POST `/ui/posts/view/:id/comments` (oraz partial `/ui/partials/posts/view/:id/comments`)
  - `CommentsVulnerable()` POST `/api/comments-vulnerable` przepisany — teraz faktycznie zapisuje do `comments` przez `CreateCommentVulnerable`
- Posts list: każdy card ma teraz link `Read more →` do `/ui/posts/view/:id`
- Layout: nawigacja zaktualizowana (Search → Vuln Demos), Cheat-Sheet drawer z filtrem + licznikiem `cheat-counter`
- Nowa strona `/ui/vuln-demos` (handler `PageVulnDemos`) — hub wszystkich scenariuszy z CWE/OWASP

### PoC
- Vulnerable mode (`SECURITY_ENABLED=false`):
  - GET `/ui/posts/view/3` → formularz "Add a comment"
  - POST `body=<script>alert('XSS-' + document.cookie)</script>` → render zwraca surowy `<script>`, JS się wykonuje
  - POST `body=<img src=x onerror="alert(1)">` → atrybut `onerror` zostaje
- Secure mode (`SECURITY_ENABLED=true`):
  - Te same payloady → tagi strip-owane na zapisie + auto-escape na render
  - W odpowiedzi brak `<script>alert(` ani `onerror=` jako surowy HTML
  - `/api/comments-vulnerable` nadal zapisuje verbatim (do porównania na zywo)

### Testy integracyjne
- `TestIntegration_StoredXSS_VulnerableMode` (2 podtesty): script payload + img onerror — oba renderują się raw w partialu i pełnej stronie
- `TestIntegration_StoredXSS_SecureMode` (3 podtesty): script escape, onerror strip, force-vuln endpoint nadal verbatim

### Diff kodu (przed/po — render komentarzy)
```go
// VULNERABLE — CommentsList template
<div class="text-sm">
    @templ.Raw(c.Body)   // surowy HTML z DB
</div>

// SECURE — CommentsList template
<div class="text-sm">
    { c.Body }           // templ auto-escape: < → &lt;
</div>

// SECURE — service.go (defense in depth na wejściu)
func sanitizeCommentBody(body string) string {
    return strings.TrimSpace(stripHTMLRegex.ReplaceAllString(body, ""))
}
```

---

## Krok 3 — Broken Authentication (zrealizowany 2026-05-03)

### Implementacja
- `internal/service/service.go::ValidateUserCredentials`:
  - vuln: po znalezieniu wpisu w `users` zwraca `true` bez sprawdzenia hasla (Broken Auth — kazde haslo dziala dla istniejacego usera)
  - secure: `bcrypt.CompareHashAndPassword(stored, password)`
- `internal/service/service.go::preparePassword` + `internal/db/db.go::encodePasswordForSeed`:
  - vuln: zapisuje plaintext (zwiazane z Sensitive Data Exposure)
  - secure: bcrypt hash przy `CreateUser` i przy seedzie admina
- Cookie `bai_auth_user` (HTTP-only, TTL 8h) ustawiany w `setAuthCookie`

### PoC
- Vuln: `POST /login {"username":"admin","password":"anything"}` -> 200 + `bai_auth_user=admin`
- Secure: ten sam request -> 401 ("Invalid username or password"); poprawne haslo -> 200

### Testy
- `TestIntegration_Login` (vuln) — admin loguje sie z dowolnym haslem
- `TestIntegration_DeleteAuthorization_SecurityEnabled` — secure odrzuca wrong password przed authorization checkiem

### Diff
```go
// VULNERABLE
if !s.securityEnabled {
    return true, nil // <- ignoruje haslo
}
// SECURE
if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err != nil {
    return false, nil
}
```

---

## Krok 4 — Broken Access Control (zrealizowany 2026-05-03)

### Implementacja
- `internal/handlers/handlers.go::canDeletePost(username, postID)`:
  - admin -> dozwolone
  - inny user -> dozwolone tylko gdy `GetPostAuthor(postID) == username`
- Hook w `PostDelete` (`DELETE /posts/:id`) i `PagePostDelete` (`POST /ui/posts/delete/:id`):
  - vuln: bez sprawdzenia, kazdy zalogowany usuwa kazdy post
  - secure: `canDeletePost` musi zwrocic `true`, inaczej `403 Forbidden`
- `internal/db/db.go` seeduje rola `admin` dla seedowanego admina i `user` dla pozostalych

### PoC
- Vuln: `user1` loguje sie -> `DELETE /posts/1` (post admina) -> 200 (post zniknal)
- Secure: ten sam request -> 403 `{"error":"You can only delete your own posts"}`

### Testy
- `TestIntegration_DeleteAuthorization_SecurityEnabled` (4 sub-testy):
  - non-admin nie usuwa cudzego posta -> 403
  - admin usuwa cudzy -> 200
  - autor usuwa wlasny -> 200
  - bez sesji -> 401

### Diff
```go
// VULNERABLE — brak gating w PostDelete
return func(c *gin.Context) {
    if !h.requireLoginJSON(c) { return }
    h.svc.DeletePost(id)
}

// SECURE
if h.securityEnabled {
    allowed, _ := h.canDeletePost(username, id)
    if !allowed {
        c.JSON(403, gin.H{"error": "You can only delete your own posts"})
        return
    }
}
```

---

## Krok 5 — Sensitive Data Exposure (zrealizowany 2026-05-03)

### Implementacja
- `internal/db/db.go::encodePasswordForSeed(password, securityEnabled)`:
  - vuln: zwraca plaintext, idzie wprost do `password_hash` w `users` (mylacy nazewnictwo kolumny — celowe dla demo)
  - secure: zwraca `bcrypt.GenerateFromPassword(...)`
- `internal/service/service.go::preparePassword` zachowuje sie tak samo dla nowych rejestracji

### PoC
- Vuln (przed migracja): `sqlite3 app.db "SELECT username, password_hash FROM users;"`
  ```
  admin|admin
  user1|user1pass
  alice|alicepass
  ```
- Secure: ta sama komenda zwraca bcrypt hashes (`$2a$10$...`)

### Testy
- Sprawdzane posrednio przez `TestIntegration_DeleteAuthorization_SecurityEnabled` (logowanie wymaga bcrypt match, wiec jezeli secure seed by zapisal plaintext, test by polegl)

### Diff
```go
// VULNERABLE
func encodePasswordForSeed(password string, securityEnabled bool) string {
    if !securityEnabled { return password } // plaintext
    ...
}

// SECURE
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
if err != nil { return password } // graceful fallback
return string(hash)
```

---

## Krok 6 — UI / design refactor (zrealizowany 2026-05-08)

Nieformalnie — to nie podatnosc, ale ulatwia zywe demo:
- Nawigacja: `📝 Posts`, `✏️ Vuln Demos`, `🔑 Login`, `✍️ Register`, badge `🛡️ Secure`/`⚠️ Vulnerable`
- Hub `/ui/vuln-demos` z 6 kartami (CWE/OWASP/status pill/payload preview)
- Strona detalu posta `/ui/posts/view/:id` (Stored XSS demo)
- Strony Login/Register dwukolumnowe: hero "BAI Security Lab" po lewej, formularz po prawej
- Cheat-sheet drawer z filtrem `🔍 Filtruj (sql, xss, csrf, jwt...)` i licznikiem `X/Y sekcji`
- Tailwind config skanuje pliki `.go` (poprzednio tylko `.templ`/`*_templ.go`) — naprawiony bug, gdzie klasy uzywane w helperach Go byly purgowane


---

## Krok 7 — CSRF (zrealizowany 2026-05-15)

### Implementacja
- Nowy plik `internal/handlers/csrf.go`:
  - `EnsureCSRFCookieMiddleware()` — na kazdym requeście zapewnia ze przegladarka ma cookie `bai_csrf` (32 bajty z `crypto/rand`, base64url). Cookie NIE jest HTTP-only zeby JS mogl go odczytac dla double-submit.
  - `ValidateCSRFMiddleware(securityEnabled)` — w secure mode dla POST/PUT/PATCH/DELETE wymaga `_csrf_token` (form field) LUB `X-CSRF-Token` (naglowek) zgodnego z cookie.
  - `isCSRFExempt(path)` — zwalnia z walidacji JSON API (`/posts`, `/login`, `/register`, `/logout`, `/api/*`) oraz `/csrf-vulnerable-form` (force-vuln demo).
  - `secureCompare` — porownanie constant-time.
- Wpiete w `main.go::buildRouter` zaraz za `Static` mountami (router.Use).
- Klient JS (`static/js/bai-lab-extras.js`):
  - `wireFormCsrfInjection()` — przed kazdym submit form-em (capture phase) dodaje hidden input `_csrf_token` z wartoscia cookie.
  - `wireHtmxCsrfHeader()` — hook na `htmx:configRequest` dodaje naglowek `X-CSRF-Token` dla HTMX POST/PUT/DELETE.

### PoC
- Vuln (`SECURITY_ENABLED=false`):
  - `curl -X POST http://localhost:8080/ui/posts/view/1/comments -d "body=hello"` -> 200 (brak walidacji)
  - Atak CSRF z trzeciej strony: `<form action="/ui/posts/delete/1" method="POST"></form><script>document.forms[0].submit()</script>` dziala
- Secure (`SECURITY_ENABLED=true`):
  - Bootstrap: `curl -i http://localhost:8080/ui/posts` -> `Set-Cookie: bai_csrf=...`
  - POST bez tokena -> 403 `{"error":"CSRF validation failed","detail":"missing CSRF token in form or header"}`
  - POST ze zlym tokenem -> 403 `{"detail":"CSRF token mismatch"}`
  - POST z zgodnym tokenem (cookie + form field) -> 200
  - `/csrf-vulnerable-form` nadal akceptuje POST bez tokena (force-vuln endpoint)

### Testy
- `TestIntegration_CSRF_VulnerableMode` — tokenless POST przechodzi w vuln
- `TestIntegration_CSRF_SecureMode` (6 sub-tests):
  - blokada bez tokena
  - blokada na mismatch
  - sukces z form field
  - sukces z `X-CSRF-Token` headerem
  - `/csrf-vulnerable-form` nadal podatny (force-vuln)
  - JSON `/login` zwolniony z CSRF

### Diff
```go
// VULNERABLE — brak validatora w stacku middleware
router.Use(handlers.EnsureCSRFCookieMiddleware()) // tylko ustawia cookie

// SECURE — dodatkowo validator
router.Use(handlers.EnsureCSRFCookieMiddleware())
router.Use(handlers.ValidateCSRFMiddleware(SecurityEnabled))
```

```go
// ValidateCSRFMiddleware - kluczowy fragment
cookie, _ := c.Cookie(csrfCookieName)
submitted := c.GetHeader(csrfHeader)
if submitted == "" { submitted = c.PostForm(csrfFormField) }
if !secureCompare(submitted, cookie) {
    c.AbortWithStatusJSON(403, gin.H{"error": "CSRF validation failed"})
    return
}
```

---

## Krok 8 — Security Misconfiguration (zrealizowany 2026-05-15)

### Implementacja
- Nowy plik `internal/handlers/security_headers.go`:
  - `SecurityHeadersMiddleware(securityEnabled)` — w secure mode ustawia 6 naglowkow:
    - `Content-Security-Policy: default-src 'self'; script-src 'self' https://unpkg.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'`
    - `Strict-Transport-Security: max-age=31536000; includeSubDomains`
    - `X-Frame-Options: DENY` (clickjacking)
    - `X-Content-Type-Options: nosniff` (MIME sniffing)
    - `Referrer-Policy: strict-origin-when-cross-origin`
    - `Permissions-Policy: geolocation=(), microphone=(), camera=()`
  - `ErrorSanitizerMiddleware(securityEnabled)` — w secure mode `recover()` zwraca clean `{"error":"Internal server error"}`; w vuln zostaje domyslny Gin Recovery z pelnym stack trace.
- Nowy endpoint `GET /debug/crash` w handlers.go — `_ = posts[42]` zeby celowo wywolac panic.
- Wpiete w `main.go::buildRouter` po CSRF middleware.

### PoC
- Vuln (`SECURITY_ENABLED=false`):
  - `curl -i http://localhost:8080/ui/posts` -> brak `Content-Security-Policy`, brak `Strict-Transport-Security` itd.
  - `curl -i http://localhost:8080/debug/crash` -> 500 + na stderr pelny stack trace (`goroutine 23 [running]:`, `runtime error: index out of range`, sciezki plikow w `/internal/handlers/`)
- Secure (`SECURITY_ENABLED=true`):
  - `curl -i http://localhost:8080/ui/posts` -> wszystkie 6 naglowkow obecnych
  - `curl -i http://localhost:8080/debug/crash` -> `500 {"error":"Internal server error"}`, brak stack trace w body

### Testy
- `TestIntegration_SecurityHeaders_VulnerableMode` — zaden z naglowkow nie powinien byc ustawiony
- `TestIntegration_SecurityHeaders_SecureMode` — wszystkie 4 kluczowe naglowki ustawione na oczekiwane wartosci + CSP zawiera kluczowe dyrektywy
- `TestIntegration_DebugCrash_LeaksInVulnerableMode` — body NIE jest sanitized JSON
- `TestIntegration_DebugCrash_SanitizedInSecureMode` — body to `{"error":"Internal server error"}` + brak slow `goroutine`/`panic:`/`runtime error`/sciezek

### Diff
```go
// VULNERABLE
// Brak SecurityHeadersMiddleware
// Brak ErrorSanitizerMiddleware - Gin domyslny Recovery rzuca stack trace

// SECURE
router.Use(handlers.SecurityHeadersMiddleware(SecurityEnabled))
router.Use(handlers.ErrorSanitizerMiddleware(SecurityEnabled))
```

```go
// SecurityHeadersMiddleware - kluczowy fragment
h := c.Writer.Header()
h.Set("Content-Security-Policy", "default-src 'self'; ...")
h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
h.Set("X-Frame-Options", "DENY")
h.Set("X-Content-Type-Options", "nosniff")
h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

// ErrorSanitizerMiddleware
defer func() {
    if r := recover(); r != nil {
        c.AbortWithStatusJSON(500, gin.H{"error": "Internal server error"})
    }
}()
```
