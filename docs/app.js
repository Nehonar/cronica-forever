// Crónica de Forever: lee docs/data/*.json (los genera el programa «cronica») y los muestra.
(function () {
  "use strict";

  const SVGNS = "http://www.w3.org/2000/svg";
  const QNAME = { 3: "Raro", 4: "Épico", 5: "Legendario" };
  const QCLASS = { 3: "q-rare", 4: "q-epic", 5: "q-legendary" };
  const KIND = {
    cadena: { mark: "⚜", label: "Cadena" },
    sueltas: { mark: "✎", label: "Encargos" },
    equipo: { mark: "⚔", label: "Hito" },
  };
  const GLYPH = {
    weapon: "M18.5 2.5 21.5 5.5 11 16 8 13zM7 14l3 3-1.6 1.6-1-1-2.6 2.6L3.4 19l2.6-2.6-1-1z",
    shield: "M12 2.5 19.5 5v6.2c0 4.6-3.2 8.4-7.5 10.3-4.3-1.9-7.5-5.7-7.5-10.3V5zM12 5v14.3c3-1.6 5-4.6 5-8V6.7z",
    trinket: "M12 4a6 6 0 1 1 0 12 6 6 0 0 1 0-12zm0 2.5a3.5 3.5 0 1 0 0 7 3.5 3.5 0 0 0 0-7zM11 16h2v5h-2z",
    armor: "M8 3 12 5 16 3 20.5 6.5 18 10.5V21H6V10.5L3.5 6.5zM9.5 9v9h5V9z",
  };
  const STAT_NAMES = [
    ["STRENGTH", "Fuerza"], ["STAMINA", "Aguante"], ["AGILITY", "Agilidad"], ["INTELLECT", "Intelecto"],
    ["SPIRIT", "Espíritu"], ["DEFENSE", "Defensa"], ["ATTACK_POWER", "Poder de ataque"], ["SPELL_POWER", "Poder con hechizos"],
    ["CRIT", "Golpe crítico"], ["HIT", "Golpe"], ["BLOCK", "Bloqueo"], ["DODGE", "Esquiva"], ["PARRY", "Parada"],
  ];
  function statLines(it) {
    const out = [];
    if (it.armor) out.push(el("div", { text: it.armor + " de armadura" }));
    for (const [k, v] of Object.entries(it.stats || {})) {
      const n = STAT_NAMES.find(([key]) => k.toUpperCase().includes(key));
      out.push(el("div", { text: (v > 0 ? "+" : "") + v + " " + (n ? n[1] : k.replace(/^ITEM_MOD_|_SHORT$/g, "").toLowerCase()) }));
    }
    return out;
  }
  function glyphFor(item) {
    const s = String(item.slot || "").toUpperCase();
    if (/HAND|RANGED|WEAPON/.test(s)) return "weapon";
    if (/SHIELD|OFFHAND/.test(s)) return "shield";
    if (/FINGER|TRINKET|NECK/.test(s)) return "trinket";
    return "armor";
  }

  const state = { index: [], docs: {}, key: null, view: null, tab: "cronica" };
  const $ = (id) => document.getElementById(id);

  function el(tag, attrs, ...kids) {
    const n = document.createElement(tag);
    if (attrs) for (const [k, v] of Object.entries(attrs)) {
      if (v == null || v === false) continue;
      if (k === "class") n.className = v;
      else if (k === "text") n.textContent = v;
      else if (k.startsWith("on")) n.addEventListener(k.slice(2), v);
      else n.setAttribute(k, v === true ? "" : v);
    }
    for (const k of kids.flat()) if (k != null) n.append(k.nodeType ? k : document.createTextNode(String(k)));
    return n;
  }
  function svg(tag, attrs, text) {
    const n = document.createElementNS(SVGNS, tag);
    for (const [k, v] of Object.entries(attrs || {})) n.setAttribute(k, v);
    if (text != null) n.textContent = text;
    return n;
  }
  function banner(msg) { const b = $("banner"); b.textContent = msg || ""; b.hidden = !msg; }

  function fmtDate(t) {
    return t ? new Date(t * 1000).toLocaleDateString("es-ES", { day: "numeric", month: "long", year: "numeric" }) : "";
  }
  function fmtDur(sec) {
    sec = Math.max(0, Math.round(sec || 0));
    const h = Math.floor(sec / 3600), m = Math.floor((sec % 3600) / 60);
    if (h >= 24) return Math.floor(h / 24) + " d " + (h % 24) + " h";
    return h ? h + " h " + m + " min" : m + " min";
  }
  function initials(name) {
    return String(name || "·").split(/\s+/).filter(Boolean).slice(0, 2).map((w) => w[0].toUpperCase()).join("");
  }

  function flourish() {
    const s = svg("svg", { class: "flourish", viewBox: "0 0 180 14", "aria-hidden": "true" });
    s.append(svg("path", { d: "M2 7 H70 M110 7 H178", stroke: "currentColor", "stroke-width": "1", fill: "none" }));
    s.append(svg("path", { d: "M90 1 L96 7 L90 13 L84 7 Z", fill: "currentColor" }));
    s.append(svg("circle", { cx: "76", cy: "7", r: "2", fill: "currentColor" }));
    s.append(svg("circle", { cx: "104", cy: "7", r: "2", fill: "currentColor" }));
    return s;
  }
  function seal(text) {
    const s = svg("svg", { viewBox: "0 0 80 80", "aria-hidden": "true" });
    s.append(svg("path", { d: "M40 4c6 0 8 4 13 5s9-1 12 4 0 9 2 13 6 7 5 13-6 7-7 12 2 9-3 13-9 2-13 4-6 6-12 6-8-4-13-5-9 1-12-4 0-9-2-13-6-7-5-13 6-7 7-12-2-9 3-13 9-2 13-4 6-6 11-6z", fill: "#8e1b17" }));
    s.append(svg("circle", { cx: "40", cy: "40", r: "24", fill: "none", stroke: "#5c0f0c", "stroke-width": "2" }));
    s.append(svg("circle", { cx: "40", cy: "40", r: "20", fill: "none", stroke: "#b8392f", "stroke-width": "1" }));
    s.append(svg("text", { x: "40", y: "47", "text-anchor": "middle", "font-family": "Marcellus SC, Georgia, serif", "font-size": "19", fill: "#4d0c09" }, text));
    return el("div", { class: "seal" }, s);
  }
  function prose(text) {
    const box = el("div", { class: "prose" });
    let first = true;
    for (const raw of String(text || "").replace(/\r/g, "").split(/\n\s*\n/)) {
      const t = raw.trim();
      if (!t) continue;
      const p = el("p", { text: t.replace(/\n/g, " ") });
      if (first) p.classList.add("drop");
      first = false;
      box.append(p);
    }
    return box;
  }

  /* ---------- Datos ---------- */
  async function getJSON(url) {
    const r = await fetch(url, { cache: "no-store" });
    if (!r.ok) throw new Error(r.status + " " + url);
    return r.json();
  }
  async function loadDoc(key) {
    if (!state.docs[key]) state.docs[key] = await getJSON("data/" + encodeURIComponent(key) + ".json");
    return state.docs[key];
  }

  /* ---------- Navegación por hash: #clave[/relato|/diario] ---------- */
  function parseHash() {
    const [key, view] = decodeURIComponent(location.hash.slice(1)).split("/");
    return { key: key || null, view: view || null };
  }
  function go(key, view) {
    const h = "#" + encodeURIComponent(key) + (view ? "/" + encodeURIComponent(view) : "");
    if (location.hash !== h) location.hash = h; else route();
  }

  async function route() {
    const h = parseHash();
    const key = h.key && state.index.some((c) => c.key === h.key) ? h.key : (state.index[0] && state.index[0].key);
    if (!key) {
      renderEmpty();
      return;
    }
    let doc;
    try { doc = await loadDoc(key); } catch (e) { banner("No he podido cargar la crónica de este personaje."); return; }
    state.key = key;
    state.tab = h.view === "diario" ? "diario" : "cronica";
    const stories = doc.stories || [];
    state.view = h.view && h.view !== "diario" ? h.view : (stories.length ? stories[stories.length - 1].id : "trasfondo");
    render(doc);
  }

  /* ---------- Pintado ---------- */
  function render(doc) {
    renderHero(doc);
    renderRoute(doc);
    renderTabs();
    renderIndex(doc);
    if (state.tab === "diario") renderLedger(doc);
    else if (state.view === "trasfondo") renderBackstory(doc);
    else {
      const s = (doc.stories || []).find((x) => x.id === state.view);
      if (s) renderStory(doc, s); else renderBackstory(doc);
    }
  }

  function renderHero(doc) {
    const ch = doc.character || {};
    $("h-name").textContent = ch.name || doc.key;
    $("h-epithet").textContent = ch.epithet || (ch.backstory ? "" : "Sin trasfondo todavía");
    $("h-initials").textContent = initials(ch.name || doc.key);
    const meta = $("h-meta");
    meta.replaceChildren(el("b", { text: ch.class || "" }), " " + String(ch.race || "").toLowerCase());
    const lvl = doc.level || (doc.stats && doc.stats.level) || 0;
    $("h-lvl").textContent = lvl || "–";
    $("h-xp").style.width = lvl ? Math.min(100, (lvl / 60) * 100) + "%" : "0%";
    document.title = (ch.name || doc.key) + " · Crónica de Forever";

    const side = $("h-side"), pick = $("pick");
    side.hidden = state.index.length < 2;
    pick.replaceChildren(...state.index.map((c) => el("option", { value: c.key, selected: c.key === doc.key },
      c.name + " · " + (c.class || "") + " " + (c.level || "") + (c.new ? " (nuevo)" : ""))));
  }

  function renderRoute(doc) {
    const stops = [], seen = new Set();
    for (const s of doc.stories || []) {
      const places = s.subzones && s.subzones.length ? s.subzones : [s.zone];
      for (const p of places) {
        const k = String(p || "").trim();
        if (!k || seen.has(k)) continue;
        seen.add(k);
        stops.push({ place: k, zone: s.zone });
      }
    }
    const box = $("route");
    const step = 160, padX = 96, H = 112;
    if (!stops.length) stops.push({ place: "El camino empieza aquí", zone: "" });
    const W = Math.max(padX * 2 + (stops.length - 1) * step, box.clientWidth || 600);
    const s = svg("svg", { width: W, viewBox: "0 0 " + W + " " + H, role: "img", "aria-label": "Lugares recorridos: " + stops.map((x) => x.place).join(", ") });
    const pts = stops.map((x, i) => ({ x: padX + i * step, y: i % 2 ? 66 : 46 }));
    let d = "M" + pts[0].x + " " + pts[0].y;
    if (pts.length === 1) d += " H" + (W - padX);
    for (let i = 1; i < pts.length; i++) {
      const a = pts[i - 1], b = pts[i], mx = (a.x + b.x) / 2;
      d += " C" + mx + " " + a.y + " " + mx + " " + b.y + " " + b.x + " " + b.y;
    }
    s.append(svg("path", { class: "path", d }));
    let lastZone = null;
    stops.forEach((st, i) => {
      const p = pts[i], now = i === stops.length - 1, below = i % 2 === 0;
      if (now) s.append(svg("circle", { class: "ring", cx: p.x, cy: p.y, r: 8 }));
      s.append(svg("circle", { class: "node" + (now ? " now" : ""), cx: p.x, cy: p.y, r: now ? 7 : 5 }));
      s.append(svg("text", { x: p.x, y: below ? p.y + 30 : p.y - 18, "text-anchor": "middle", class: now ? "now" : "" }, st.place));
      if (st.zone && st.zone !== lastZone) {
        s.append(svg("text", { x: p.x, y: below ? p.y - 16 : p.y + 28, "text-anchor": "middle", class: "ch" }, st.zone.toUpperCase()));
        lastZone = st.zone;
      }
    });
    box.replaceChildren(s);
    $("route-hint").textContent = (doc.stories || []).length ? seen.size + " lugares en la crónica" : "";
    requestAnimationFrame(() => { box.scrollLeft = box.scrollWidth; });
  }

  function renderTabs() {
    $("tab-cronica").setAttribute("aria-selected", String(state.tab === "cronica"));
    $("tab-diario").setAttribute("aria-selected", String(state.tab === "diario"));
  }

  function renderIndex(doc) {
    const nav = $("index");
    nav.replaceChildren();
    const item = (id, mark, title, sub) => el("button", {
      class: "log-item", type: "button",
      "aria-current": state.tab === "cronica" && state.view === id ? "true" : "false",
      onclick: () => go(doc.key, id),
    }, el("span", { class: "log-mark", "aria-hidden": "true", text: mark }), el("span", { class: "log-title", text: title }),
       sub ? el("span", { class: "log-sub", text: sub }) : null);

    nav.append(item("trasfondo", "✦", "Trasfondo", doc.character && doc.character.backstory ? "Quién es" : "Por escribir"));
    let zone = null;
    for (const s of doc.stories || []) {
      if (s.zone !== zone) { zone = s.zone; nav.append(el("div", { class: "arc", text: zone || "Sin zona" })); }
      const k = KIND[s.kind] || { mark: "•", label: s.kind };
      const sub = k.label + (s.quests && s.quests.length ? " · " + s.quests.length + " misiones" : "");
      nav.append(item(s.id, k.mark, s.title, sub));
    }

    const p = doc.pending || {};
    const parts = [];
    if (p.loose && p.loose.length) parts.push(p.loose.length + (p.loose.length === 1 ? " encargo esperando compañía" : " encargos esperando compañía"));
    if (p.openChains && p.openChains.length) parts.push(p.openChains.length + (p.openChains.length === 1 ? " cadena en curso" : " cadenas en curso"));
    const pend = $("pending");
    pend.textContent = parts.length ? "Aún por contar: " + parts.join(" y ") + "." : "";
    pend.hidden = !parts.length;
  }

  function renderBackstory(doc) {
    const ch = doc.character || {};
    const kids = [el("div", { class: "rubric", text: "TRASFONDO" }), el("h2", { text: ch.name || doc.key }), flourish()];
    if (ch.backstory) kids.push(prose(ch.backstory));
    else kids.push(el("p", { class: "page-note", text: "Este personaje aún no tiene trasfondo. Escríbelo en personajes/" + doc.key + ".json y la crónica empezará a contarse con su voz." }));
    if (ch.threads && ch.threads.length) {
      kids.push(el("div", { class: "rubric", style: "margin-top:24px", text: "HILOS ABIERTOS" }));
      kids.push(el("ul", { class: "threads" }, ch.threads.map((t) => el("li", { text: t }))));
    }
    if (ch.motto) kids.push(seal(initials(ch.name)));
    kids.push(pager(doc, -1));
    $("reader").replaceChildren(...kids.filter(Boolean));
  }

  function renderStory(doc, s) {
    const k = KIND[s.kind] || { label: s.kind };
    const facts = el("div", { class: "facts" });
    if (s.subzones && s.subzones.length) facts.append(el("span", { text: s.subzones.join(" · ") }));
    else if (s.zone) facts.append(el("span", { text: s.zone }));
    if (s.levelTo && s.levelFrom && s.levelTo > s.levelFrom) facts.append(el("span", { class: "lvlup", text: "Nivel " + s.levelFrom + " → " + s.levelTo }));
    if (s.end) facts.append(el("span", { text: fmtDate(s.end) }));

    const kids = [el("div", { class: "rubric", text: k.label.toUpperCase() }), el("h2", { text: s.title }), flourish(), facts];
    if (s.quests && s.quests.length) kids.push(el("ul", { class: "quests", "aria-label": "Misiones" }, s.quests.map((q) => el("li", { text: q.title }))));
    if (s.items && s.items.length) {
      kids.push(el("ul", { class: "loot", "aria-label": "Equipo" }, s.items.map((it) => {
        const g = svg("svg", { viewBox: "0 0 24 24", "aria-hidden": "true" });
        g.append(svg("path", { d: GLYPH[glyphFor(it)] }));
        return el("li", { class: QCLASS[it.quality] || "q-rare", tabindex: "0" },
          el("span", { class: "slot" }, g), el("span", { class: "nm", text: it.name }),
          el("span", { class: "tip", role: "tooltip" },
            el("div", { class: "t-name", text: it.name }),
            el("div", { class: "t-q", text: [QNAME[it.quality], it.subType].filter(Boolean).join(" · ") }),
            statLines(it),
            s.zone ? el("div", { text: "Se lo puso en " + s.zone }) : null,
            el("div", { class: "t-src", text: s.title })));
      })));
    }
    kids.push(prose(s.text));
    kids.push(seal(initials((doc.character || {}).name || doc.key)));
    kids.push(pager(doc, (doc.stories || []).findIndex((x) => x.id === s.id)));
    $("reader").replaceChildren(...kids.filter(Boolean));
  }

  function pager(doc, i) {
    const list = doc.stories || [];
    const prev = i > 0 ? list[i - 1] : (i === 0 ? { id: "trasfondo", title: "Trasfondo" } : null);
    const next = i + 1 < list.length ? list[i + 1] : null;
    if (!prev && !next) return null;
    return el("nav", { class: "pager", "aria-label": "Relatos vecinos" },
      prev ? el("a", { href: "#" + encodeURIComponent(doc.key) + "/" + encodeURIComponent(prev.id), text: "← " + prev.title }) : null,
      next ? el("a", { class: "next", href: "#" + encodeURIComponent(doc.key) + "/" + encodeURIComponent(next.id), text: next.title + " →" }) : null);
  }

  function renderLedger(doc) {
    const st = doc.stats || {};
    const tile = (v, k) => el("div", { class: "tile" }, el("div", { class: "v", text: v }), el("div", { class: "k", text: k }));
    const kids = [el("div", { class: "rubric", text: "DIARIO DE BATALLA" }), el("h2", { text: "Lo que dicen los números" }), flourish(),
      el("div", { class: "ledger" },
        el("div", { class: "tiles" },
          tile(st.level || doc.level || "–", "Nivel"),
          tile(st.played ? fmtDur(st.played) : "–", "Tiempo jugado"),
          tile(st.questsDone || 0, "Misiones entregadas"),
          tile(st.deaths || 0, "Caídas"),
          tile((doc.stories || []).length, "Relatos"),
          tile((st.zones || []).length, "Zonas")),
        levelTable(st),
        st.zones && st.zones.length ? [el("h3", { text: "Zonas pisadas" }), el("ul", { class: "zones" }, st.zones.map((z) => el("li", { text: z })))] : null,
        el("p", { class: "page-note", text: "Más adelante, con el registro de combate: enemigos derrotados, daño, DPS y tu rotación." }))];
    $("reader").replaceChildren(...kids.flat().filter(Boolean));
  }

  function levelTable(st) {
    const ups = st.levelUps || [];
    if (!ups.length) return null;
    const rows = [];
    let prevT = st.firstSeen || ups[0].t;
    for (const u of ups) {
      rows.push(el("tr", null, el("td", { text: "Nivel " + u.level }), el("td", { text: fmtDate(u.t) }), el("td", { class: "num", text: fmtDur(u.t - prevT) })));
      prevT = u.t;
    }
    return [el("h3", { text: "Subidas de nivel" }),
      el("table", { class: "levels" },
        el("thead", null, el("tr", null, el("th", { text: "NIVEL" }), el("th", { text: "FECHA" }), el("th", { class: "num", text: "DESDE EL ANTERIOR" }))),
        el("tbody", null, rows))];
  }

  function renderEmpty() {
    $("h-name").textContent = "Crónica de Forever";
    $("reader").replaceChildren(el("div", { class: "rubric", text: "AÚN SIN PÁGINAS" }), el("h2", { text: "La crónica está en blanco" }), flourish(),
      el("p", { class: "page-note", text: "Cuando el programa «cronica» procese tu primera sesión, aquí aparecerá la historia de tus personajes." }));
    $("index").replaceChildren();
  }

  /* ---------- Arranque ---------- */
  $("tab-cronica").addEventListener("click", () => go(state.key, state.view && state.view !== "diario" ? state.view : null));
  $("tab-diario").addEventListener("click", () => go(state.key, "diario"));
  $("pick").addEventListener("change", (e) => go(e.target.value, null));
  window.addEventListener("hashchange", route);
  let rT; window.addEventListener("resize", () => { clearTimeout(rT); rT = setTimeout(() => { const d = state.docs[state.key]; if (d) renderRoute(d); }, 150); });

  let lastUpdated = null;
  getJSON("data/index.json")
    .then((idx) => { state.index = idx.characters || []; lastUpdated = idx.updated; route(); })
    .catch(() => { renderEmpty(); });

  // Refrescarse sola cuando el cronista escribe algo nuevo (en tu PC cada 20 s;
  // en la web publicada cada 2 min, que es lo que tarda GitHub en publicar).
  const local = location.hostname === "127.0.0.1" || location.hostname === "localhost";
  setInterval(async () => {
    if (document.hidden) return;
    try {
      const idx = await getJSON("data/index.json?t=" + Date.now());
      if (!idx.updated || idx.updated === lastUpdated) return;
      lastUpdated = idx.updated;
      state.index = idx.characters || [];
      state.docs = {};
      route();
    } catch (_) { /* sin conexión: lo intenta más tarde */ }
  }, local ? 20000 : 120000);
})();
