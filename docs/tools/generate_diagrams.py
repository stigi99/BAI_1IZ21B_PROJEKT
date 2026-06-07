#!/usr/bin/env python3
"""Generate clean SVG diagrams for the BAI report.

The diagrams are intentionally generated from data so spacing, typography and
arrow anchors stay consistent across all visuals used in the A4 report.
"""

from __future__ import annotations

import html
import math
import textwrap
from dataclasses import dataclass
from pathlib import Path


OUT = Path(__file__).resolve().parents[1] / "diagrams"


@dataclass(frozen=True)
class Box:
    x: int
    y: int
    w: int
    h: int
    title: str
    lines: tuple[str, ...] = ()
    fill: str = "#ffffff"
    stroke: str = "#cbd5e1"
    title_color: str = "#0f172a"


def esc(value: str) -> str:
    return html.escape(value, quote=True)


def wrap(text: str, width: int) -> list[str]:
    return textwrap.wrap(text, width=width, break_long_words=False, replace_whitespace=False)


class SVG:
    def __init__(self, width: int = 1600, height: int = 1000, title: str = "") -> None:
        self.width = width
        self.height = height
        self.items: list[str] = []
        self.title = title

    def add(self, raw: str) -> None:
        self.items.append(raw)

    def defs(self) -> str:
        return """
  <defs>
    <linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#fff7fb"/>
      <stop offset="48%" stop-color="#f8fbff"/>
      <stop offset="100%" stop-color="#ecfdf5"/>
    </linearGradient>
    <filter id="shadow" x="-15%" y="-20%" width="130%" height="150%">
      <feDropShadow dx="0" dy="10" stdDeviation="10" flood-color="#64748b" flood-opacity="0.18"/>
    </filter>
    <marker id="arrow" markerWidth="16" markerHeight="16" refX="12" refY="8" orient="auto" markerUnits="strokeWidth">
      <path d="M2,2 L14,8 L2,14 Z" fill="#334155"/>
    </marker>
    <marker id="arrow-red" markerWidth="16" markerHeight="16" refX="12" refY="8" orient="auto" markerUnits="strokeWidth">
      <path d="M2,2 L14,8 L2,14 Z" fill="#e11d48"/>
    </marker>
    <marker id="arrow-green" markerWidth="16" markerHeight="16" refX="12" refY="8" orient="auto" markerUnits="strokeWidth">
      <path d="M2,2 L14,8 L2,14 Z" fill="#059669"/>
    </marker>
  </defs>"""

    def background(self) -> None:
        self.add(f'<rect width="{self.width}" height="{self.height}" rx="32" fill="url(#bg)"/>')

    def heading(self, title: str, subtitle: str = "") -> None:
        self.add(f'<text x="80" y="90" class="h1">{esc(title)}</text>')
        if subtitle:
            for i, line in enumerate(wrap(subtitle, 92)):
                self.add(f'<text x="80" y="{132 + i * 28}" class="sub">{esc(line)}</text>')

    def box(self, box: Box, wrap_width: int = 26) -> None:
        self.add(
            f'<rect x="{box.x}" y="{box.y}" width="{box.w}" height="{box.h}" rx="20" '
            f'fill="{box.fill}" stroke="{box.stroke}" stroke-width="3" filter="url(#shadow)"/>'
        )
        self.add(
            f'<text x="{box.x + 26}" y="{box.y + 44}" class="box-title" '
            f'fill="{box.title_color}">{esc(box.title)}</text>'
        )
        y = box.y + 82
        for line in box.lines:
            for part in wrap(line, wrap_width):
                self.add(f'<text x="{box.x + 26}" y="{y}" class="small">{esc(part)}</text>')
                y += 28

    def pill(self, x: int, y: int, w: int, text: str, fill: str, color: str = "#ffffff") -> None:
        self.add(f'<rect x="{x}" y="{y}" width="{w}" height="54" rx="27" fill="{fill}" filter="url(#shadow)"/>')
        self.add(f'<text x="{x + w / 2}" y="{y + 35}" text-anchor="middle" class="pill" fill="{color}">{esc(text)}</text>')

    def arrow(self, x1: int, y1: int, x2: int, y2: int, color: str = "#334155", marker: str = "arrow", label: str = "", above: bool = True) -> None:
        self.add(
            f'<line x1="{x1}" y1="{y1}" x2="{x2}" y2="{y2}" stroke="{color}" stroke-width="5" '
            f'stroke-linecap="round" marker-end="url(#{marker})"/>'
        )
        if label:
            mx = (x1 + x2) / 2
            my = (y1 + y2) / 2 + (-18 if above else 34)
            self.add(f'<text x="{mx}" y="{my}" text-anchor="middle" class="label" fill="{color}">{esc(label)}</text>')

    def curved(self, d: str, color: str = "#334155", marker: str = "arrow", label: tuple[int, int, str] | None = None) -> None:
        self.add(f'<path d="{d}" fill="none" stroke="{color}" stroke-width="5" stroke-linecap="round" marker-end="url(#{marker})"/>')
        if label:
            x, y, text = label
            self.add(f'<text x="{x}" y="{y}" text-anchor="middle" class="label" fill="{color}">{esc(text)}</text>')

    def save(self, name: str) -> None:
        css = """
  <style>
    text { font-family: Arial, Helvetica, sans-serif; }
    .h1 { font-size: 46px; font-weight: 800; fill: #0f172a; }
    .sub { font-size: 24px; font-weight: 500; fill: #475569; }
    .box-title { font-size: 26px; font-weight: 800; }
    .small { font-size: 22px; font-weight: 500; fill: #334155; }
    .tiny { font-size: 18px; font-weight: 600; fill: #334155; }
    .label { font-size: 20px; font-weight: 800; paint-order: stroke; stroke: #fff; stroke-width: 5px; }
    .pill { font-size: 22px; font-weight: 800; }
    .axis { font-size: 21px; font-weight: 800; fill: #334155; }
    .tick { font-size: 19px; font-weight: 700; fill: #475569; }
  </style>"""
        payload = "\n".join([
            f'<svg xmlns="http://www.w3.org/2000/svg" width="{self.width}" height="{self.height}" viewBox="0 0 {self.width} {self.height}" role="img" aria-label="{esc(self.title)}">',
            self.defs(),
            css,
            *self.items,
            "</svg>",
        ])
        OUT.mkdir(parents=True, exist_ok=True)
        (OUT / name).write_text(payload, encoding="utf-8")


def architecture_overview() -> None:
    s = SVG(title="BAI Security Lab architecture")
    s.background()
    s.heading("Architektura aplikacji BAI Security Lab", "Warstwy aplikacji i miejsca, w których tryb secure zamienia podatne ścieżki na kontrolowane zachowanie.")
    boxes = [
        Box(80, 210, 250, 170, "Klient", ("Przeglądarka", "curl / Burp / Postman", "HTML, formularze, payloady"), "#fff1f2", "#fb7185"),
        Box(440, 210, 260, 170, "Gin Router", ("Routing endpointów", "API + UI + HTMX", "main.go"), "#eff6ff", "#60a5fa"),
        Box(810, 210, 280, 170, "Handlers", ("HTTP, sesja, walidacja", "tryb vulnerable / secure", "internal/handlers"), "#faf5ff", "#c084fc"),
        Box(1210, 210, 250, 170, "Views", ("Templ", "HTML + HTMX", "pages.templ"), "#ecfdf5", "#34d399"),
        Box(440, 570, 260, 170, "Static / upload", ("CSS, JS, grafiki", "uploady użytkownika", "static / assets"), "#ecfeff", "#22d3ee"),
        Box(810, 570, 280, 170, "Service", ("Logika domenowa", "SQL, hasła, komentarze", "internal/service"), "#fff7ed", "#fb923c"),
        Box(1210, 570, 250, 170, "SQLite", ("blog, users, comments", "migracje + seed", "app.db"), "#f8fafc", "#cbd5e1"),
    ]
    for b in boxes:
        s.box(b)
    s.arrow(330, 295, 440, 295)
    s.arrow(700, 295, 810, 295)
    s.arrow(1090, 295, 1210, 295)
    s.curved("M950 380 C920 470 830 510 700 590")
    s.arrow(1090, 655, 1210, 655)
    s.arrow(700, 655, 810, 655)
    s.pill(130, 850, 600, "VULNERABLE: SQL concat, raw HTML, plaintext", "#fb5a42")
    s.pill(875, 850, 600, "SECURE: SQL params, walidacja, escaping, bcrypt", "#0ea5a8")
    s.save("architecture-overview.svg")


def security_toggle_flow() -> None:
    s = SVG(title="Security toggle flow")
    s.background()
    s.heading("Jeden payload, dwa wyniki", "Ten sam input użytkownika trafia do tej samej funkcji aplikacji, ale ModeStore wybiera podatną albo bezpieczną gałąź kodu.")
    s.box(
        Box(
            530,
            210,
            540,
            170,
            "Payload użytkownika",
            (
                "SQL: ORDER BY probe -> UNION sqlite_master -> draft pivot",
                "XSS: onerror overlay, LFI: ../internal/db/db.go, CMD: ; whoami",
            ),
            "#ffffff",
            "#cbd5e1",
        ),
        48,
    )
    s.box(Box(115, 560, 560, 245, "Vulnerable mode", ("Input staje się składnią", "SQL: konkatenacja stringów", "XSS: render przez templ.Raw", "LFI: dokładanie ścieżki", "CMD: sh -c interpretuje metaznaki"), "#fff1f2", "#fb7185"), 46)
    s.box(Box(925, 560, 560, 245, "Secure mode", ("Input pozostaje danymi", "SQL: placeholdery ?", "XSS: escaping + sanitizacja", "LFI: canonical path check", "CMD: argument bez shella + regex"), "#ecfdf5", "#34d399"), 46)
    s.curved("M800 355 C740 430 560 480 390 560", "#e11d48", "arrow-red", (530, 455, "SecurityEnabled = false"))
    s.curved("M800 355 C860 430 1040 480 1210 560", "#059669", "arrow-green", (1070, 455, "SecurityEnabled = true"))
    s.pill(180, 880, 1240, "Wniosek: dane użytkownika nie mogą stać się kodem, zapytaniem SQL, ścieżką ani poleceniem systemowym.", "#0f172a")
    s.save("security-toggle-flow.svg")


def attack_chain_map() -> None:
    s = SVG(height=1080, title="Attack chain map")
    s.background()
    s.heading("Mapa łańcuchów ataku", "Pojedyncze podatności są w projekcie izolowane, ale w realnym incydencie mogą wzmacniać się nawzajem.")
    nodes = [
        Box(80, 210, 300, 135, "SQL Injection", ("schema, users, hidden drafts",), "#fff7ed", "#fb7185"),
        Box(80, 470, 300, 135, "Path Traversal", ("source code / app.db",), "#fff7ed", "#fb923c"),
        Box(515, 330, 350, 155, "Sensitive Data", ("plaintext password_hash", "dane kont i sesji"), "#fff1f2", "#fb7185"),
        Box(1010, 230, 335, 145, "Broken Auth", ("dowolne hasło", "znany login"), "#eff6ff", "#60a5fa"),
        Box(1180, 525, 300, 145, "IDOR", ("usuwanie cudzych postów",), "#faf5ff", "#c084fc"),
        Box(260, 760, 300, 145, "Stored XSS", ("fake login overlay",), "#ecfdf5", "#34d399"),
        Box(720, 760, 300, 145, "CSRF", ("mutacja bez intencji",), "#ecfeff", "#22d3ee"),
        Box(1120, 820, 300, 145, "Command Injection", ("wykonanie komendy",), "#0f172a", "#0f172a", "#ffffff"),
    ]
    for n in nodes:
        s.box(n, 28)
    s.curved("M380 278 C445 278 465 360 515 392", "#e11d48", "arrow-red", (450, 300, "wyciek users"))
    s.curved("M380 538 C450 538 460 450 515 430", "#e11d48", "arrow-red", (450, 520, "wyciek DB"))
    s.curved("M865 390 C940 350 955 300 1010 300", "#e11d48", "arrow-red", (942, 338, "hasła jawne"))
    s.curved("M1178 375 C1220 430 1250 485 1280 525", "#e11d48", "arrow-red", (1265, 445, "sesja"))
    s.curved("M560 832 C625 832 660 832 720 832", "#0ea5e9", "arrow", (640, 810, "żądanie w kontekście ofiary"))
    s.curved("M1020 845 C1070 845 1090 875 1120 895", "#e11d48", "arrow-red", (1095, 850, "eskalacja"))
    s.pill(90, 980, 610, "Czerwone strzałki: możliwy łańcuch ataku", "#fff", "#e11d48")
    s.pill(900, 980, 520, "Niebieska strzałka: akcja przeglądarki ofiary", "#fff", "#0369a1")
    s.save("attack-chain-map.svg")


def risk_matrix() -> None:
    s = SVG(height=1040, title="Risk matrix")
    s.background()
    s.heading("Macierz ryzyka podatności", "Ocena orientacyjna dla aplikacji laboratoryjnej: prawdopodobieństwo kontra skutek.")
    x0, y0, cell = 430, 230, 165
    colors = [
        ["#dcfce7", "#bef264", "#fde68a", "#fdba74"],
        ["#bbf7d0", "#fde68a", "#fdba74", "#fb7185"],
        ["#fde68a", "#fdba74", "#fb7185", "#e11d48"],
        ["#fdba74", "#fb7185", "#e11d48", "#9f1239"],
    ]
    for r in range(4):
        for c in range(4):
            s.add(f'<rect x="{x0 + c * cell}" y="{y0 + (3-r) * cell}" width="{cell}" height="{cell}" fill="{colors[r][c]}" stroke="#ffffff" stroke-width="4"/>')
    s.add(f'<rect x="{x0}" y="{y0}" width="{4*cell}" height="{4*cell}" rx="22" fill="none" stroke="#cbd5e1" stroke-width="4"/>')
    for i, label in enumerate(["niskie", "średnie", "wysokie", "krytyczne"]):
        s.add(f'<text x="{x0 + i*cell + cell/2}" y="{y0 + 4*cell + 58}" text-anchor="middle" class="tick">{esc(label)}</text>')
        s.add(f'<text x="{x0 - 42}" y="{y0 + (3-i)*cell + cell/2 + 8}" text-anchor="end" class="tick">{esc(label)}</text>')
    s.add(f'<text x="{x0 + 2*cell}" y="{y0 + 4*cell + 110}" text-anchor="middle" class="axis">Skutek</text>')
    s.add(f'<text x="{x0 - 200}" y="{y0 + 2*cell}" transform="rotate(-90 {x0 - 200} {y0 + 2*cell})" text-anchor="middle" class="axis">Prawdopodobieństwo</text>')
    labels = [
        (3, 3, "Broken Auth"), (3, 3, "SQLi"), (3, 3, "Cmd Inj."),
        (2, 3, "IDOR"), (2, 2, "LFI"), (2, 1, "XSS"),
        (1, 2, "CSRF"), (1, 1, "Data Exposure"),
    ]
    offsets: dict[tuple[int, int], int] = {}
    for prob, impact, text in labels:
        key = (prob, impact)
        n = offsets.get(key, 0)
        offsets[key] = n + 1
        cx = x0 + impact * cell + cell / 2
        cy = y0 + (3 - prob) * cell + 40 + n * 54
        pill_w = 132
        s.add(
            f'<rect x="{cx - pill_w/2}" y="{cy - 22}" width="{pill_w}" height="42" rx="21" '
            f'fill="#ffffff" filter="url(#shadow)"/>'
        )
        s.add(f'<text x="{cx}" y="{cy + 7}" text-anchor="middle" class="tick">{esc(text)}</text>')
    s.box(Box(1190, 290, 285, 380, "Legenda", ("zielony: niskie", "żółty: średnie", "pomarańczowy: wysokie", "czerwony: krytyczne", "Ocena nie jest CVSS.", "Służy do priorytetyzacji."), "#ffffff", "#cbd5e1"), 26)
    s.save("risk-matrix.svg")


def request_lifecycle() -> None:
    s = SVG(title="HTTP request lifecycle")
    s.background()
    s.heading("Cykl życia żądania HTTP", "Miejsca, w których dane użytkownika muszą zostać potraktowane jako niezaufane.")
    steps = [
        ("1", "Input", "formularz, URL, JSON, upload"),
        ("2", "Router", "wybór endpointu main.go"),
        ("3", "Handler", "sesja, token, tryb handlers.go"),
        ("4", "Service", "SQL, auth, XSS service.go"),
        ("5", "DB / OS", "SQLite, pliki, ping"),
    ]
    x = 80
    for i, (num, title, desc) in enumerate(steps):
        color = ["#ec4899", "#3b82f6", "#8b5cf6", "#f97316", "#10b981"][i]
        s.add(
            f'<rect x="{x}" y="320" width="245" height="210" rx="20" fill="#ffffff" '
            f'stroke="{color}" stroke-width="3" filter="url(#shadow)"/>'
        )
        s.add(f'<circle cx="{x+42}" cy="362" r="28" fill="{color}"/>')
        s.add(f'<text x="{x+42}" y="371" text-anchor="middle" class="pill" fill="#fff">{num}</text>')
        s.add(f'<text x="{x+86}" y="370" class="box-title" fill="#0f172a">{esc(title)}</text>')
        for j, part in enumerate(wrap(desc, 18)):
            s.add(f'<text x="{x+30}" y="{410+j*28}" class="small">{esc(part)}</text>')
        if i < len(steps) - 1:
            s.arrow(x + 245, 425, x + 315, 425)
        x += 315
    s.pill(160, 760, 1280, "Kontrole: sesja, CSRF, autoryzacja, parametryzowany SQL, bcrypt, escaping, whitelisty ścieżek i brak shella.", "#0f172a")
    s.save("request-lifecycle.svg")


def defense_controls() -> None:
    s = SVG(height=1080, title="Defense controls")
    s.background()
    s.heading("Mapa podatność -> kontrola bezpieczeństwa", "Każda podatność ma konkretny mechanizm naprawczy widoczny w kodzie i testach.")
    rows = [
        ("1", "SQL Injection", "placeholders ?", "SearchPostsSecure"),
        ("2", "Stored XSS", "sanitize + escape", "safe comments"),
        ("3", "Broken Auth", "bcrypt + limiter", "ValidateUserCredentials"),
        ("4", "IDOR", "owner/admin check", "canDeletePost"),
        ("5", "CSRF", "token + SameSite", "csrf_token cookie"),
        ("6", "Sensitive Data", "bcrypt seed + register", "password_hash"),
        ("7", "Path Traversal", "canonical path", "safeFilePath"),
        ("8", "Command Injection", "no shell + regex", "exec.Command"),
    ]
    for i, row in enumerate(rows):
        col = i % 2
        r = i // 2
        x = 90 + col * 740
        y = 220 + r * 185
        color = ["#f43f5e", "#ec4899", "#8b5cf6", "#6366f1", "#06b6d4", "#14b8a6", "#f97316", "#ef4444"][i]
        s.add(
            f'<rect x="{x}" y="{y}" width="620" height="135" rx="20" fill="#ffffff" '
            f'stroke="{color}" stroke-width="3" filter="url(#shadow)"/>'
        )
        s.add(f'<circle cx="{x+52}" cy="{y+52}" r="28" fill="{color}"/>')
        s.add(f'<text x="{x+52}" y="{y+61}" text-anchor="middle" class="pill" fill="#fff">{row[0]}</text>')
        s.add(f'<text x="{x+105}" y="{y+50}" class="box-title" fill="#0f172a">{esc(row[1])}</text>')
        s.add(f'<text x="{x+105}" y="{y+85}" class="small">{esc("Kontrola: " + row[2])}</text>')
        s.add(f'<text x="{x+105}" y="{y+115}" class="small">{esc("Kod/test: " + row[3])}</text>')
    s.pill(180, 970, 1240, "Zasada wspólna: walidacja i autoryzacja są po stronie serwera, nie w UI.", "#0f172a")
    s.save("defense-controls-infographic.svg")


def feature_vulnerability_map() -> None:
    s = SVG(height=1080, title="Feature vulnerability map")
    s.background()
    s.heading("Funkcje aplikacji jako nośniki podatności", "Scenariusze nie są oderwanymi sztuczkami: każdy payload wchodzi przez naturalną funkcję bloga, konta, plików albo narzędzia ping.")
    pairs = [
        ("Wyszukiwarka", "/ui/library", "SQL Injection", "schema + draft pivot"),
        ("Komentarze", "/ui/posts/view/{id}", "Stored XSS", "raw HTML / DOM"),
        ("Logowanie", "/ui/login", "Broken Auth", "hasło ignorowane"),
        ("Usuwanie posta", "POST /delete/{id}", "IDOR", "brak ownership check"),
        ("Konto użytkownika", "/ui/profile", "CSRF", "brak tokena"),
        ("Katalog członków", "/ui/members", "Data Exposure", "jawne hasła"),
        ("Przeglądarka plików", "../internal/db/db.go", "Path Traversal", "wyjście z uploads"),
        ("Narzędzie ping", "host=127.0.0.1; id", "Command Injection", "sh -c user input"),
    ]
    for i, (feature, route, vuln, detail) in enumerate(pairs):
        col = i // 4
        r = i % 4
        y = 220 + r * 180
        fx = 90 + col * 760
        vx = fx + 390
        s.box(Box(fx, y, 315, 115, feature, (route,), "#f8fafc", "#7dd3fc"), 26)
        s.box(Box(vx, y, 350, 115, vuln, (detail,), "#fff1f2", "#fb7185"), 28)
        s.arrow(fx + 315, y + 58, vx, y + 58)
    s.pill(210, 970, 1180, "Ten sam workflow: funkcja -> payload -> wynik vulnerable -> przełączenie trybu -> wynik secure.", "#a855f7")
    s.save("feature-vulnerability-map.svg")


def method_result_overview() -> None:
    s = SVG(height=1200, title="Method result overview")
    s.background()
    s.heading("Podatność, metoda wywołania i wynik", "Tabela graficzna do szybkiego porównania payloadu, efektu ataku i reakcji trybu secure.")
    headers = ["Podatność", "Metoda wywołania", "Wynik vulnerable", "Wynik secure"]
    widths = [270, 450, 390, 330]
    x0, y0, row_h = 80, 210, 92
    x = x0
    for h, w in zip(headers, widths):
        s.add(f'<rect x="{x}" y="{y0}" width="{w}" height="66" rx="16" fill="#e0f2fe" stroke="#38bdf8" stroke-width="2"/>')
        s.add(f'<text x="{x+w/2}" y="{y0+42}" text-anchor="middle" class="box-title" fill="#075985">{esc(h)}</text>')
        x += w
    rows = [
        ("SQLi", "ORDER BY + UNION schema/drafts", "wyciek draftów / users", "0 wyników, brak injection"),
        ("Stored XSS", "img onerror fake overlay", "DOM zmieniony u ofiary", "payload jako tekst"),
        ("Broken Auth", "admin / anything", "logowanie udane", "401 + rate limit"),
        ("IDOR", "POST /delete/1 jako user1", "usunięcie cudzego posta", "403 / redirect z błędem"),
        ("CSRF", "auto POST new_email", "email zmieniony", "403 token mismatch"),
        ("Data leak", "SELECT password_hash", "jawne hasła", "bcrypt hash"),
        ("Path LFI", "name=../internal/db/db.go", "odczyt spoza uploads", "blokada canonical path"),
        ("CMD inj", "127.0.0.1 ; whoami", "wykonanie komendy", "walidacja hosta"),
    ]
    colors = ["#fb7185", "#f97316", "#a855f7", "#ef4444", "#06b6d4", "#64748b", "#22c55e", "#0f172a"]
    for i, row in enumerate(rows):
        y = y0 + 86 + i * row_h
        x = x0
        s.add(f'<rect x="{x0}" y="{y}" width="{sum(widths)}" height="{row_h-14}" rx="18" fill="#ffffff" filter="url(#shadow)"/>')
        for j, (txt, w) in enumerate(zip(row, widths)):
            if j == 0:
                s.pill(x + 28, y + 20, 180, txt, colors[i])
            else:
                for k, part in enumerate(wrap(txt, 28 if j == 1 else 25)):
                    s.add(f'<text x="{x+26}" y="{y+34+k*25}" class="small">{esc(part)}</text>')
            x += w
    s.save("method-result-overview.svg")


def architecture_deep_dive() -> None:
    s = SVG(height=1100, title="Architecture deep dive")
    s.background()
    s.heading("Architektura aplikacji i punkty kontroli", "Ścieżka żądania, warstwy danych oraz miejsca, w których zapada decyzja vulnerable/secure.")
    top = [
        Box(90, 210, 235, 130, "Browser", ("formularze, query string", "cookies, uploady"), "#f8fafc", "#cbd5e1"),
        Box(435, 210, 235, 130, "Gin Router", ("routing UI / API", "main.go"), "#eff6ff", "#93c5fd"),
        Box(780, 210, 235, 130, "Handlers", ("sesja, CSRF", "currentUsername"), "#faf5ff", "#c084fc"),
        Box(1125, 210, 235, 130, "Service", ("SQL, hasła", "SavePost"), "#ecfdf5", "#34d399"),
    ]
    for b in top:
        s.box(b, 23)
    for a, b in zip(top, top[1:]):
        s.arrow(a.x + a.w, a.y + 65, b.x, b.y + 65)
    bottom = [
        Box(90, 540, 270, 135, "SQLite", ("posts, users, comments", "SQLi + data exposure"), "#fff7ed", "#fb923c"),
        Box(505, 540, 270, 135, "Filesystem", ("uploads + attachments", "path traversal"), "#ecfeff", "#22d3ee"),
        Box(920, 540, 270, 135, "OS Process", ("ping command", "command injection boundary"), "#f8fafc", "#94a3b8"),
    ]
    for b in bottom:
        s.box(b, 26)
    s.arrow(552, 340, 225, 540)
    s.arrow(552, 340, 640, 540)
    s.arrow(898, 340, 1055, 540)
    s.pill(470, 750, 660, "ModeStore decyduje o ścieżce w jednym spójnym miejscu", "#0f172a")
    s.pill(120, 860, 600, "VULNERABLE: SQL concat, raw HTML, no CSRF", "#fb5a42")
    s.pill(870, 860, 600, "SECURE: placeholders, escaping, bcrypt, token", "#0ea5a8")
    s.save("app-architecture-deep-dive.svg")


def security_score_chart() -> None:
    s = SVG(height=1000, title="Security score comparison")
    s.background()
    s.heading("Porównanie efektu zabezpieczeń", "Wykres pokazuje, jak tryb secure zmienia wynik scenariuszy: atak ma pozostać danymi albo zostać odrzucony.")
    labels = ["SQLi", "XSS", "Auth", "IDOR", "CSRF", "Data", "LFI", "CMD"]
    vulnerable = [95, 90, 85, 80, 75, 88, 82, 90]
    secure = [8, 12, 18, 10, 8, 20, 10, 7]
    x0, y0, w, max_h = 180, 780, 120, 470
    for i, label in enumerate(labels):
        x = x0 + i * 165
        vh = vulnerable[i] / 100 * max_h
        sh = secure[i] / 100 * max_h
        s.add(f'<rect x="{x}" y="{y0-vh}" width="54" height="{vh}" rx="12" fill="#fb7185"/>')
        s.add(f'<rect x="{x+62}" y="{y0-sh}" width="54" height="{sh}" rx="12" fill="#10b981"/>')
        s.add(f'<text x="{x+58}" y="{y0+46}" text-anchor="middle" class="tick">{esc(label)}</text>')
    for pct in [0, 25, 50, 75, 100]:
        y = y0 - pct / 100 * max_h
        s.add(f'<line x1="120" y1="{y}" x2="1490" y2="{y}" stroke="#e2e8f0" stroke-width="2"/>')
        s.add(f'<text x="95" y="{y+7}" text-anchor="end" class="tick">{pct}%</text>')
    s.add('<text x="120" y="185" class="axis">Skuteczność ataku / ekspozycja ryzyka</text>')
    s.pill(1010, 165, 180, "vulnerable", "#fb7185")
    s.pill(1210, 165, 145, "secure", "#10b981")
    s.save("security-score-comparison.svg")


def test_coverage_chart() -> None:
    s = SVG(height=900, title="Test coverage chart")
    s.background()
    s.heading("Pokrycie scenariuszy testami", "Każda główna klasa podatności ma w raporcie metodę wywołania, wynik vulnerable, wynik secure i walidację testową.")
    labels = ["SQLi", "XSS", "Auth", "IDOR", "CSRF", "Data", "LFI", "CMD"]
    values = [4, 3, 4, 3, 3, 2, 2, 2]
    x0, y0 = 330, 250
    max_w = 860
    for i, (label, value) in enumerate(zip(labels, values)):
        y = y0 + i * 70
        width = value / 4 * max_w
        s.add(f'<text x="175" y="{y+33}" text-anchor="end" class="axis">{esc(label)}</text>')
        s.add(f'<rect x="{x0}" y="{y}" width="{max_w}" height="44" rx="22" fill="#e2e8f0"/>')
        s.add(f'<rect x="{x0}" y="{y}" width="{width}" height="44" rx="22" fill="#38bdf8"/>')
        s.add(f'<text x="{x0 + width - 24}" y="{y+30}" text-anchor="end" class="pill" fill="#ffffff">{value}/4</text>')
    s.box(Box(1230, 280, 270, 240, "Skala", ("1: tylko opis", "2: payload + wynik", "3: test secure", "4: pełna para vul/secure"), "#ffffff", "#cbd5e1"), 24)
    s.save("test-coverage-chart.svg")


def main() -> None:
    architecture_overview()
    security_toggle_flow()
    attack_chain_map()
    risk_matrix()
    request_lifecycle()
    defense_controls()
    feature_vulnerability_map()
    method_result_overview()
    architecture_deep_dive()
    security_score_chart()
    test_coverage_chart()


if __name__ == "__main__":
    main()
