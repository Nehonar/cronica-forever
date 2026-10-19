// Banco y oficios: lo que tiene cada personaje y qué le falta para subir sus
// profesiones. Los datos los publica Crónica al salir del juego.
(function () {
  "use strict";
  const $ = (id) => document.getElementById(id);
  const DATA = "../oficios/datos/";
  const PROF_EN = { Blacksmithing: "Herrería", Mining: "Minería" };
  const TIERS = [75, 150, 225, 300];

  let idx = null, rec = null, d = null;
  let goals = {};            // objetivo elegido por profesión
  let tab = "inv";

  function el(tag, attrs, ...kids) {
    const n = document.createElement(tag);
    for (const [k, v] of Object.entries(attrs || {})) {
      if (v == null || v === false) continue;
      if (k === "class") n.className = v;
      else if (k === "text") n.textContent = v;
      else if (k.startsWith("on")) n.addEventListener(k.slice(2), v);
      else n.setAttribute(k, v === true ? "" : v);
    }
    for (const k of kids.flat()) if (k != null) n.append(k.nodeType ? k : document.createTextNode(String(k)));
    return n;
  }
  const store = {
    get(k) { try { return localStorage.getItem(k); } catch (_) { return null; } },
    set(k, v) { try { localStorage.setItem(k, v); } catch (_) { /* sin almacenamiento */ } },
  };
  async function getJSON(url) {
    const r = await fetch(url, { cache: "no-cache" });
    if (!r.ok) throw new Error(r.status);
    return r.json();
  }
  function msg(t) { const m = $("msg"); m.textContent = t || ""; m.hidden = !t; }

  // ---------- objetos ----------
  const obj = (id) => (rec && rec.objetos[id]) || {};
  const nameOf = (id) => (idx && idx.nombres && idx.nombres[id]) || obj(id).en || "Objeto " + id;
  const qualityOf = (id) => { const q = idx && idx.calidad && idx.calidad[id]; return q != null ? q : (obj(id).q != null ? obj(id).q : 1); };
  function stockMap() {
    const m = {};
    for (const s of d.bolsas || []) m[s.id] = (m[s.id] || 0) + s.n;
    for (const s of d.banco || []) m[s.id] = (m[s.id] || 0) + s.n;
    return m;
  }
  function money(c) {
    c = c || 0;
    const g = Math.floor(c / 10000), s = Math.floor(c % 10000 / 100), k = c % 100;
    const out = [];
    if (g) out.push(el("span", { class: "coin g", text: g }));
    if (g || s) out.push(el("span", { class: "coin s", text: s }));
    out.push(el("span", { class: "coin c", text: k }));
    return out;
  }

  // Dibujos sencillos por tipo de objeto (propios, no los iconos del juego).
  const GLYPH = {
    bar: "M3 15 L8 8 H21 L16 15 Z M3 15 V18 H16 V15 M16 18 L21 11 V8",
    ore: "M5 17 L3 11 L8 5 L15 4 L21 9 L19 17 L11 20 Z M8 9 L11 12 M14 8 L16 12",
    stone: "M4 16 C4 9 9 6 14 6 C19 6 21 10 20 14 C19 18 14 19 10 19 C6 19 4 18 4 16 Z",
    armor: "M6 3 L10 5 H14 L18 3 L22 7 L19 10 V21 H5 V10 L2 7 Z",
    weapon: "M19 2 L22 5 L10 17 L12 19 L10 21 L8 19 L5 22 L2 19 L5 16 L3 14 L5 12 L7 14 Z",
    hide: "M6 3 C9 5 15 5 18 3 L21 8 C19 10 19 14 21 16 L18 21 C15 19 9 19 6 21 L3 16 C5 14 5 10 3 8 Z",
    potion: "M9 2 H15 V7 C19 9 20 12 20 15 C20 19 17 22 12 22 C7 22 4 19 4 15 C4 12 5 9 9 7 Z",
    pouch: "M8 4 H16 L14 8 C19 10 21 14 20 18 C19 21 15 22 12 22 C9 22 5 21 4 18 C3 14 5 10 10 8 Z",
  };
  function glyphFor(id) {
    const o = obj(id), en = (o.en || "").toLowerCase(), t = o.tipo || "";
    if (/ ore$/.test(en)) return "ore";
    if (/ bar$/.test(en)) return "bar";
    if (/stone|flux|gem|pearl/.test(en) || t === "Metal & Stone") return "stone";
    if (["Mail", "Plate", "Shield", "Leather", "Cloth"].includes(t)) return "armor";
    if (/sword|axe|mace|dagger|hammer|polearm|staff/i.test(t)) return "weapon";
    if (/leather|hide|scale/.test(en)) return "hide";
    if (/potion|elixir|oil|flask/.test(en)) return "potion";
    return "pouch";
  }
  function glyph(id) {
    const s = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    s.setAttribute("viewBox", "0 0 24 24"); s.setAttribute("aria-hidden", "true");
    const p = document.createElementNS("http://www.w3.org/2000/svg", "path");
    p.setAttribute("d", GLYPH[glyphFor(id)]);
    s.append(p);
    return s;
  }

  // ---------- inventario ----------
  function slot(st, total) {
    return el("button", { class: "slot q" + qualityOf(st.id), type: "button", "aria-label": nameOf(st.id) + ", " + st.n,
      onclick: () => showItem(st.id, total) },
      glyph(st.id), el("span", { class: "lbl", text: nameOf(st.id) }), el("span", { class: "n", text: st.n > 1 ? st.n : "" }));
  }
  function sortItems(list) {
    return list.slice().sort((a, b) => qualityOf(b.id) - qualityOf(a.id) || nameOf(a.id).localeCompare(nameOf(b.id), "es"));
  }
  function renderInv() {
    const q = $("q").value.trim().toLowerCase();
    const match = (s) => !q || nameOf(s.id).toLowerCase().includes(q);
    const total = stockMap();
    const box = $("inv");
    const bags = sortItems((d.bolsas || []).filter(match));
    const bank = sortItems((d.banco || []).filter(match));
    box.replaceChildren(
      el("h2", null, "Bolsas", el("small", { text: (d.bolsas || []).length + " objetos distintos" })),
      bags.length ? el("div", { class: "grid" }, bags.map((s) => slot(s, total))) : el("p", { class: "note", text: q ? "Nada con ese nombre." : "Las bolsas están vacías." }),
      el("h2", null, "Banco", el("small", { text: d.bancoVisto ? (d.banco || []).length + " objetos distintos" : "" })),
      !d.bancoVisto ? el("p", { class: "note", text: "Aún no lo conozco: abre el banco una vez en el juego y sal (o haz /reload)." })
        : bank.length ? el("div", { class: "grid" }, bank.map((s) => slot(s, total)))
        : el("p", { class: "note", text: q ? "Nada con ese nombre." : "El banco está vacío." }));
  }
  function showItem(id, total) {
    const bags = (d.bolsas || []).find((s) => s.id === id), bank = (d.banco || []).find((s) => s.id === id);
    const uses = rec ? rec.recetas.filter((r) => r.mats.some((m) => m.id === id)) : [];
    const made = rec ? rec.recetas.find((r) => r.id === id) : null;
    $("sheet-body").replaceChildren(
      el("h3", { id: "sheet-title", class: "q" + qualityOf(id), text: nameOf(id) }),
      el("dl", { class: "kv" },
        el("dt", { text: "En bolsas" }), el("dd", { text: bags ? bags.n : 0 }),
        el("dt", { text: "En el banco" }), el("dd", { text: d.bancoVisto ? (bank ? bank.n : 0) : "sin ver" }),
        el("dt", { text: "Total" }), el("dd", { text: total[id] || 0 }),
        made ? el("dt", { text: "Se hace con" }) : null,
        made ? el("dd", { text: made.oficio + " " + made.nivel + ": " + made.mats.map((m) => m.n + " " + nameOf(m.id)).join(", ") }) : null,
        uses.length ? el("dt", { text: "Se usa en" }) : null,
        uses.length ? el("dd", { text: uses.slice(0, 6).map((r) => nameOf(r.id) + " (" + r.oficio + " " + r.nivel + ")").join(", ") + (uses.length > 6 ? "… y " + (uses.length - 6) + " más" : "") }) : null));
    $("sheet").hidden = false;
    $("sheet-close").focus();
  }
  $("sheet-close").addEventListener("click", () => { $("sheet").hidden = true; });
  $("sheet").addEventListener("click", (e) => { if (e.target === $("sheet")) $("sheet").hidden = true; });
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") $("sheet").hidden = true; });
  $("q").addEventListener("input", renderInv);

  // ---------- oficios: rutas para subir ----------
  const profName = (n) => PROF_EN[n] || n;

  // Umbrales de color de una receta, corregidos con el color que tiene ahora en el juego.
  function colors(r, seen, at) {
    let [o, y, g, gr] = r.colores;
    if (seen && at) {
      if (seen === "naranja" && at >= y) { y = at + 1; g = y + 20; gr = g + 20; }
      if (seen === "amarillo") { if (at < y) y = at; if (at >= g) { g = at + 1; gr = Math.max(gr, g + 20); } }
      if (seen === "verde") { if (at < g) { g = at; y = Math.min(y, g); } if (at >= gr) gr = at + 1; }
      if (seen === "gris" && at < gr) { gr = at; g = Math.min(g, gr); y = Math.min(y, g); }
    }
    return [o, y, g, gr];
  }
  // Probabilidad de subir un punto al fabricarla con la habilidad s.
  function chance(c, s) {
    if (s < c[0] || s >= c[3]) return 0;
    if (s < c[1]) return 1;
    return Math.max(0, (c[3] - s) / (c[3] - c[1]));
  }
  function colorName(c, s) { return s < c[1] ? "naranja" : s < c[2] ? "amarillo" : s < c[3] ? "verde" : "gris"; }

  function plan(prof, from, to, mode) {
    const all = rec.recetas.filter((r) => r.oficio === prof);
    const knownList = d.recetas && d.recetas[prof] ? d.recetas[prof].lista || [] : null;
    const at = d.recetas && d.recetas[prof] ? d.recetas[prof].nivel : null;
    const seen = {};
    for (const k of knownList || []) if (k.id) seen[k.id] = k.color;
    const known = (r) => knownList ? r.id in seen : false;
    const pool = all.filter((r) => mode === "sabes" ? known(r) : known(r) || r.aprender === "instructor");
    const col = new Map(pool.map((r) => [r, colors(r, seen[r.id], at)]));
    const stock = Object.assign({}, stockMap());
    const crafts = new Map();
    const steps = [];
    let stuck = null;
    for (let s = from; s < to; s++) {
      let best = null, bestScore = Infinity, bestP = 0;
      for (const r of pool) {
        const p = chance(col.get(r), s);
        if (p <= 0) continue;
        let cost = 0;
        for (const m of r.mats) {
          // Un material raro (que se vende caro) cuenta más que uno corriente.
          const rare = 1 + Math.min(4, (obj(m.id).venta || 0) / 50);
          const w = mode === "banco" && (stock[m.id] || 0) >= m.n ? 0.15 : rare;
          cost += m.n * w;
        }
        const score = cost / p;
        if (score < bestScore - 1e-9 || (Math.abs(score - bestScore) < 1e-9 && p > bestP)) { best = r; bestScore = score; bestP = p; }
      }
      if (!best) { stuck = s; break; }
      const n = 1 / bestP;
      crafts.set(best, (crafts.get(best) || 0) + n);
      for (const m of best.mats) stock[m.id] = (stock[m.id] || 0) - m.n * n;
      const last = steps[steps.length - 1];
      if (last && last.r === best) { last.to = s + 1; last.n += n; } else steps.push({ r: best, from: s, to: s + 1, n, color: colorName(col.get(best), s) });
    }
    // Materiales: lo que hace falta en total, lo que tienes y lo que falta.
    const have = stockMap();
    const need = {};
    let totalCrafts = 0;
    for (const [r, n] of crafts) {
      const times = Math.ceil(n - 1e-9);
      totalCrafts += times;
      for (const m of r.mats) need[m.id] = (need[m.id] || 0) + m.n * times;
    }
    const mats = Object.keys(need).map((id) => ({ id: +id, need: need[id], have: have[id] || 0 }))
      .map((m) => Object.assign(m, { miss: Math.max(0, m.need - m.have) }))
      .sort((a, b) => b.miss - a.miss || b.need - a.need);
    // Barras que faltan, en menas (con las recetas de fundir).
    const smelt = [];
    for (const m of mats) {
      if (!m.miss) continue;
      const sr = rec.recetas.find((r) => r.id === m.id && r.oficio === "Minería");
      if (!sr) continue;
      const times = Math.ceil(m.miss / (sr.hace || 1));
      for (const x of sr.mats) smelt.push({ id: x.id, need: x.n * times, have: have[x.id] || 0, for: m.id });
    }
    for (const x of smelt) x.miss = Math.max(0, x.need - x.have);
    // Recetas por aprender.
    const learn = [];
    for (const [r] of crafts) {
      if (known(r)) continue;
      const tr = d.instructor && d.instructor[prof] ? d.instructor[prof].find((t) => t.id === r.id) : null;
      learn.push({ r, price: tr && tr.precio, how: r.aprender === "instructor" ? "instructor" : "plano" });
    }
    const missing = mats.reduce((a, m) => a + m.miss, 0);
    return { mode, steps, mats, smelt, learn, totalCrafts, missing, stuck, knownCount: knownList ? knownList.length : null };
  }

  const MODES = [
    { id: "menos", title: "Menos materiales", sub: "Recetas del instructor y las que ya sabes; gasta lo mínimo." },
    { id: "banco", title: "Aprovechando tu banco", sub: "Prefiere lo que ya tienes en bolsas y banco." },
    { id: "sabes", title: "Solo lo que ya sabes", sub: "Sin aprender nada nuevo." },
  ];

  function routeCard(p, best) {
    const m = MODES.find((x) => x.id === p.mode);
    const kids = [el("h4", null, m.title, best ? el("small", { text: "★ la que menos te falta" }) : null),
      el("div", { class: "sum", text: m.sub })];
    if (p.mode === "sabes" && p.knownCount == null) {
      kids.push(el("p", { class: "note", text: "Abre una vez la ventana de esta profesión en el juego para que sepa qué recetas tienes." }));
      return el("div", { class: "route" }, kids);
    }
    if (!p.steps.length) {
      kids.push(el("p", { class: "note", text: p.stuck != null ? "Con estas recetas no se puede subir desde " + p.stuck + "." : "Nada que subir." }));
      return el("div", { class: "route" }, kids);
    }
    kids.push(el("div", { class: "sum", text: "≈ " + p.totalCrafts + " fabricaciones · te faltan " + p.missing + " materiales" + (p.stuck != null ? " · llega hasta " + p.stuck : "") }));
    kids.push(el("ul", { class: "steps" }, p.steps.map((s) => el("li", null,
      el("span", { class: "rng", text: s.from + "→" + s.to }),
      el("span", null, el("span", { class: "dot c-" + s.color, "aria-hidden": "true" }), nameOf(s.r.id)),
      el("span", { class: "times", text: "×" + Math.ceil(s.n - 1e-9) })))));
    kids.push(el("table", { class: "mats" },
      el("thead", null, el("tr", null, el("th", { text: "Material" }), el("th", { text: "Hace falta" }), el("th", { text: "Tienes" }), el("th", { text: "Falta" }))),
      el("tbody", null, p.mats.map((m) => el("tr", null,
        el("td", { class: "q" + qualityOf(m.id), text: nameOf(m.id) }), el("td", { text: m.need }), el("td", { text: m.have }),
        el("td", { class: m.miss ? "miss" : "ok", text: m.miss || "✓" }))))));
    if (p.smelt.length) {
      kids.push(el("details", { class: "more", open: true }, el("summary", { text: "Las barras que faltan, en menas" }),
        el("table", { class: "mats" }, el("tbody", null, p.smelt.map((x) => el("tr", null,
          el("td", { text: nameOf(x.id) + " (para " + nameOf(x.for) + ")" }), el("td", { text: x.need }), el("td", { text: x.have }),
          el("td", { class: x.miss ? "miss" : "ok", text: x.miss || "✓" })))))));
    }
    if (p.learn.length) {
      kids.push(el("div", { class: "sum", text: "Por aprender:" }), el("ul", { class: "buy" }, p.learn.map((l) => el("li", { text:
        nameOf(l.r.id) + " — " + (l.how === "instructor" ? "instructor, desde " + l.r.nivel + (l.price ? " (" + coinText(l.price) + ")" : "") : planSource(l.r)) }))));
    }
    return el("div", { class: "route" + (best ? " best" : "") }, kids);
  }
  function coinText(c) {
    const g = Math.floor(c / 10000), s = Math.floor(c % 10000 / 100), k = c % 100;
    return [g ? g + " o" : "", s ? s + " p" : "", k ? k + " c" : ""].filter(Boolean).join(" ") || "gratis";
  }
  function planSource(r) {
    const l = Array.isArray(r.aprender) ? r.aprender[0] : null;
    if (!l) return "plano";
    const where = { Vendor: "plano de vendedor", "Rare Drop": "plano de botín raro", "Zone Drop": "plano de botín", "Boss Drop": "plano de jefe", Quest: "plano de misión" }[l.fuente] || "plano";
    return where + (l.precio ? " (" + coinText(l.precio) + ")" : "");
  }

  function renderOfi() {
    const box = $("ofi");
    const profs = Object.entries(d.profesiones || {}).map(([n, s]) => [profName(n), s]);
    if (!profs.length) {
      box.replaceChildren(el("p", { class: "msg", text: "Aún no sé qué profesiones tiene. Entra al juego con el addon de Crónica y sal (o haz /reload)." }));
      return;
    }
    const kids = [];
    for (const [prof, s] of profs.sort((a, b) => b[1].nivel - a[1].nivel)) {
      const hasTable = rec.recetas.some((r) => r.oficio === prof);
      const card = el("div", { class: "card" },
        el("h3", null, prof, el("span", { text: s.nivel + " / " + s.max })),
        el("div", { class: "bar", role: "progressbar", "aria-valuemin": "0", "aria-valuemax": String(s.max), "aria-valuenow": String(s.nivel) },
          el("span", { style: "width:" + Math.min(100, s.nivel / Math.max(1, s.max) * 100).toFixed(1) + "%" })));
      if (!hasTable) {
        card.append(el("p", { class: "note", text: "Por ahora las cuentas solo están para Herrería y Minería." }));
        kids.push(card);
        continue;
      }
      const cap = TIERS.find((t) => t > s.nivel) || 300;
      let goal = goals[prof] || Math.min(Math.max(s.max, s.nivel + 1), cap);
      if (goal <= s.nivel) goal = Math.min(300, s.nivel + 10);
      const input = el("input", { type: "number", min: String(s.nivel + 1), max: "300", value: String(goal), inputmode: "numeric", "aria-label": "Nivel al que quieres llegar" });
      input.addEventListener("change", () => { const v = Math.max(s.nivel + 1, Math.min(300, parseInt(input.value, 10) || goal)); goals[prof] = v; renderOfi(); });
      const chip = (label, v) => el("button", { type: "button", "aria-pressed": String(goal === v), onclick: () => { goals[prof] = v; renderOfi(); } }, label);
      card.append(el("div", { class: "goal" }, el("label", null, "Subir hasta ", input),
        el("div", { class: "chips" }, chip("+10", Math.min(300, s.nivel + 10)), chip("+25", Math.min(300, s.nivel + 25)), chip("Máximo " + cap, cap))));
      if (prof === "Minería") card.append(el("p", { class: "note", text: "La minería también sube picando vetas; aquí solo se cuenta fundir." }));
      if (s.nivel >= s.max && goal > s.max) card.append(el("p", { class: "note", text: "Estás en el tope (" + s.max + "): primero aprende el siguiente rango con el instructor." }));
      const plans = MODES.map((m) => plan(prof, s.nivel, goal, m.id));
      const valid = plans.filter((p) => p.steps.length && !(p.mode === "sabes" && p.knownCount == null));
      const best = valid.length ? valid.reduce((a, b) => (b.missing < a.missing || (b.missing === a.missing && b.totalCrafts < a.totalCrafts) ? b : a)) : null;
      card.append(el("div", { class: "routes" }, plans.map((p) => routeCard(p, p === best))));
      kids.push(card);
    }
    box.replaceChildren(...kids);
  }

  // ---------- pestañas y carga ----------
  function setTab(t) {
    tab = t;
    $("t-inv").setAttribute("aria-selected", String(t === "inv"));
    $("t-ofi").setAttribute("aria-selected", String(t === "ofi"));
    $("p-inv").hidden = t !== "inv";
    $("p-ofi").hidden = t !== "ofi";
    store.set("banco.tab", t);
  }
  $("t-inv").addEventListener("click", () => setTab("inv"));
  $("t-ofi").addEventListener("click", () => setTab("ofi"));

  async function loadChar(key) {
    try {
      d = await getJSON(DATA + encodeURIComponent(key) + ".json");
    } catch (_) { msg("No he podido cargar los datos de este personaje."); return; }
    msg("");
    store.set("banco.pj", key);
    $("money").replaceChildren(...money(d.dinero));
    renderInv();
    renderOfi();
  }

  async function start() {
    setTab(store.get("banco.tab") === "ofi" ? "ofi" : "inv");
    try {
      [idx, rec] = await Promise.all([getJSON(DATA + "indice.json"), getJSON("../oficios/recetas.json")]);
    } catch (_) {
      msg("Aún no hay datos. Instala la última versión de Crónica en tu PC, entra al juego y sal: se publicarán solos.");
      return;
    }
    const list = idx.personajes || [];
    if (!list.length) { msg("Aún no hay personajes con datos."); return; }
    const sel = $("pj");
    const saved = store.get("banco.pj");
    sel.replaceChildren(...list.map((c) => el("option", { value: c.key, selected: c.key === saved ? true : null }, c.nombre + (c.nivel ? " · " + c.nivel : ""))));
    sel.addEventListener("change", () => loadChar(sel.value));
    loadChar(list.some((c) => c.key === saved) ? saved : list[0].key);
  }

  if ("serviceWorker" in navigator) navigator.serviceWorker.register("sw.js").catch(() => {});
  start();
})();
