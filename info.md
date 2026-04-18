Projekt
Projekty wykonywane są w grupach 2-3 osobowych. Każdy zespół wybiera Team Leadera, który odpowiada za koordynację prac, repozytorium oraz kontakt z Prowadzącym.

Metodyka pracy:

Projekty wykonywane są w grupach 2-3 osobowych

Zajęcia na Uczelni mają charakter przeglądu postępów – zespół prezentuje postępy w implementacji.

Po każdych zajęciach, jedna osoba z zespołu wysyła mailem krótkie podsumowanie z opisem uzyskanych postępów w projekcie oraz z planem dalszego rozwoju
Wymagania dotyczące projektu:

Głównym zadaniem jest stworzenie aplikacji w dwóch wariantach: Vulnerable(podatna) oraz Secure (bezpieczna). Dopuszczalne formy to: dwie osobne - identyczne aplikacje, rozłączne endpointy lub dedykowany „przełącznik” (security toggle) wewnątrz aplikacji.

Obowiązkowy zakres podatności:

SQL Injection (np. Error-based, Union-based lub Blind).

Cross-Site Scripting (XSS) (np. Stored lub Reflected).

(2n - 1) dodatkowe podatności,  wybrane z poniższej listy (n - oznacza liczbę członków zespołu):

Broken Access Control (np. IDOR – dostęp do danych innego użytkownika przez zmianę ID w URL).

CSRF (Cross-Site Request Forgery) – np. nieautoryzowana zmiana hasła.

Insecure Deserialization – wykonanie kodu poprzez złośliwy obiekt.

Security Misconfiguration – np. listowanie plików w katalogach lub jawne błędy bazy danych.

Broken Authentication – np. podatność na brute-force lub słabe zarządzanie sesją.

Path Traversal / LFI – odczyt plików systemowych serwera.

Command Injection – wykonanie poleceń systemowych (OS) przez formularz.

Sensitive Data Exposure – np. przechowywanie haseł otwartym tekstem.

XXE (XML External Entity) – ataki poprzez parser XML.

SSRF (Server-Side Request Forgery).

Przebieg zaliczenia (Obrona):

Live Demo (Atak): Skuteczne przeprowadzenie ataku na wersję podatną przy użyciu narzędzi (np. Burp Suite, lub inne).

Live Demo (Obrona): Próba powtórzenia tego samego ataku na wersji zabezpieczonej i pokazanie, że został zablokowany.

Code Review: Wyjaśnienie różnic w kodzie źródłowym (co dokładnie powodowało błąd i jak zostało naprawione).

Dokumentacja: Złożenie raportu zawierającego opisy podatności.


Zakres sprawozdania (Dokumentacja techniczna):

Opis architektury i użytych technologii.

Katalog podatności: Dla każdej z podatności:

Opis teoretyczny (co to za błąd).

Proof of Concept (PoC): Krok po kroku jak wywołać błąd (zrzuty ekranu, payloady).

Remediacja: Fragment kodu "przed" i "po" poprawce wraz z opisem mechanizmu obronnego.

Checklista bezpieczeństwa: Krótkie podsumowanie zastosowanych dobrych praktyk (np. użycie Prepared Statements, Content Security Policy, bezpieczne flagi ciasteczek).