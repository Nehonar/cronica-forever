// Banco y oficios: lo que tiene cada personaje y qué le falta para subir sus
// profesiones. Los datos los publica Crónica al salir del juego.
(function () {
  "use strict";
  const $ = (id) => document.getElementById(id);
  const DATA = "../oficios/datos/";
  const PROF_EN = { Blacksmithing: "Herrería", Mining: "Minería" };

  let idx = null, rec = null, d = null;
  let icons = {};           // id → nombre del icono del juego (iconos.json)
  const ICON_URL = "https://wow.zamimg.com/images/wow/icons/large/";
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
  // Icono del juego si lo conocemos; si no (o si no carga), un dibujo propio.
  function art(id) {
    const name = icons[id];
    if (!name) return glyph(id);
    const img = el("img", { src: ICON_URL + name + ".jpg", alt: "", loading: "lazy", decoding: "async" });
    img.addEventListener("error", () => img.replaceWith(glyph(id)));
    return img;
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
    return el("button", { class: "cell", type: "button", "aria-label": nameOf(st.id) + ", " + st.n, onclick: () => showItem(st.id, total) },
      el("span", { class: "slot q" + qualityOf(st.id) }, art(st.id), el("span", { class: "n", text: st.n > 1 ? st.n : "" })),
      el("span", { class: "lbl q" + qualityOf(st.id), text: nameOf(st.id) }));
  }
  // Icono pequeño para las listas de materiales.
  function mini(id) { return el("span", { class: "mini q" + qualityOf(id) }, art(id)); }
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
      el("div", { class: "sheet-head" }, el("span", { class: "slot big q" + qualityOf(id) }, art(id)),
        el("h3", { id: "sheet-title", class: "q" + qualityOf(id), text: nameOf(id) })),
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

  // ---------- oficios: lo que puedes hacer y tu lista para farmear ----------
  const profName = (n) => PROF_EN[n] || n;
  const COLOR_NAME = { naranja: "naranja", amarillo: "amarillo", verde: "verde", gris: "gris" };
  let filters = {};          // por profesión: { ver: "sabes"|"nivel"|"todas", q: "" }
  let openProf = null;       // profesión desplegada

  // Color estimado de una receta con tu nivel (si no la has visto en el juego).
  function estColor(r, s) {
    const c = r.colores;
    return s < c[1] ? "naranja" : s < c[2] ? "amarillo" : s < c[3] ? "verde" : "gris";
  }
  function coinText(c) {
    const g = Math.floor(c / 10000), s = Math.floor(c % 10000 / 100), k = c % 100;
    return [g ? g + " o" : "", s ? s + " p" : "", k ? k + " c" : ""].filter(Boolean).join(" ") || "gratis";
  }
  function learnText(r, prof) {
    if (!r) return "";
    if (r.aprender === "instructor") {
      const tr = d.instructor && d.instructor[prof] ? d.instructor[prof].find((t) => t.id === r.id) : null;
      return "Instructor" + (tr && tr.precio ? " (" + coinText(tr.precio) + ")" : "");
    }
    const l = Array.isArray(r.aprender) ? r.aprender[0] : null;
    const where = l && { Vendor: "Plano de vendedor", "Rare Drop": "Plano de botín raro", "Zone Drop": "Plano de botín", "Boss Drop": "Plano de jefe", Quest: "Plano de misión" }[l.fuente];
    return (where || "Plano") + (l && l.precio ? " (" + coinText(l.precio) + ")" : "");
  }

  // Recetas de una profesión: las que sabes (del juego, con su color real) y las de la tabla.
  function recipes(prof) {
    const s = (d.profesiones || {})[prof] || (d.profesiones || {})[Object.keys(PROF_EN).find((k) => PROF_EN[k] === prof)] || { nivel: 0 };
    const game = d.recetas && d.recetas[prof] ? d.recetas[prof].lista || [] : [];
    const table = rec.recetas.filter((r) => r.oficio === prof);
    const byId = new Map();
    for (const r of table) byId.set(r.id, { id: r.id, nivel: r.nivel, mats: r.mats, sabe: false, color: estColor(r, s.nivel), real: false, tabla: r, hace: r.hace || 1 });
    for (const g of game) {
      const key = g.id || "n:" + g.nombre;
      const base = byId.get(key) || { id: g.id, nivel: null, tabla: null, hace: 1 };
      const mats = g.mats && g.mats.length && g.mats.every((m) => m.id) ? g.mats.map((m) => ({ id: m.id, n: m.n })) : base.mats || [];
      byId.set(key, Object.assign({}, base, { nombre: g.nombre, sabe: true, color: g.color || base.color, real: !!g.color, mats }));
    }
    return { list: [...byId.values()], level: s.nivel, hasGame: game.length > 0 };
  }

  // ---- tu lista (se guarda en este móvil) ----
  const listKey = () => "banco.lista." + (d && d.key);
  function loadList() { try { return JSON.parse(store.get(listKey()) || "[]"); } catch (_) { return []; } }
  function saveList(l) { store.set(listKey(), JSON.stringify(l)); }
  function addToList(prof, r, n) {
    const l = loadList();
    const key = r.id || r.nombre;
    const cur = l.find((x) => x.key === key);
    if (cur) cur.n += n; else l.push({ key, id: r.id || null, nombre: r.nombre || null, prof, n, mats: r.mats, hace: r.hace || 1 });
    saveList(l);
    renderOfi();
  }

  function matLine(m, have, times) {
    const need = m.n * (times || 1);
    const enough = have >= need;
    return el("li", { class: "mat" + (enough ? " ok" : " short") }, mini(m.id),
      el("span", { class: "mn", text: need + "× " + nameOf(m.id) }),
      el("span", { class: "mh", text: "tienes " + have }));
  }

  function renderList() {
    const l = loadList();
    const have = stockMap();
    const box = el("div", { class: "card list" }, el("h3", null, "Mi lista para farmear", l.length ? el("button", { class: "link", type: "button", onclick: () => { if (confirm("¿Vaciar la lista?")) { saveList([]); renderOfi(); } } }, "Vaciar") : null));
    if (!l.length) {
      box.append(el("p", { class: "note", text: "Vacía. Despliega una profesión y pulsa «+» en lo que quieras fabricar: aquí se suman los materiales y ves lo que te falta." }));
      return box;
    }
    box.append(el("ul", { class: "picked" }, l.map((x, i) => el("li", null,
      x.id ? mini(x.id) : null,
      el("span", { class: "pn", text: (x.id ? nameOf(x.id) : x.nombre) }),
      el("span", { class: "stepper" },
        el("button", { type: "button", "aria-label": "Una menos", onclick: () => { x.n -= 1; if (x.n <= 0) l.splice(i, 1); saveList(l); renderOfi(); } }, "−"),
        el("b", { text: "×" + x.n }),
        el("button", { type: "button", "aria-label": "Una más", onclick: () => { x.n += 1; saveList(l); renderOfi(); } }, "+"))))));
    // Materiales sumados.
    const need = {};
    for (const x of l) for (const m of x.mats || []) need[m.id] = (need[m.id] || 0) + m.n * x.n;
    const rows = Object.keys(need).map((id) => ({ id: +id, need: need[id], have: have[id] || 0 }))
      .map((m) => Object.assign(m, { miss: Math.max(0, m.need - m.have) }))
      .sort((a, b) => b.miss - a.miss || nameOf(a.id).localeCompare(nameOf(b.id), "es"));
    box.append(el("table", { class: "mats" },
      el("thead", null, el("tr", null, el("th", { text: "Material" }), el("th", { text: "Necesitas" }), el("th", { text: "Tienes" }), el("th", { text: "Falta" }))),
      el("tbody", null, rows.map((m) => el("tr", null,
        el("td", { class: "q" + qualityOf(m.id) }, mini(m.id), nameOf(m.id)), el("td", { text: m.need }), el("td", { text: m.have }),
        el("td", { class: m.miss ? "miss" : "ok", text: m.miss || "✓" }))))));
    // Barras que faltan, en menas.
    const ores = {};
    for (const m of rows) {
      if (!m.miss) continue;
      const sr = rec.recetas.find((r) => r.id === m.id && r.oficio === "Minería");
      if (!sr) continue;
      const times = Math.ceil(m.miss / (sr.hace || 1));
      for (const x of sr.mats) ores[x.id] = (ores[x.id] || 0) + x.n * times;
    }
    const oreRows = Object.keys(ores).map((id) => ({ id: +id, need: ores[id], have: have[id] || 0 }));
    if (oreRows.length) {
      box.append(el("div", { class: "sum", text: "Para fundir las barras que te faltan:" }), el("table", { class: "mats" }, el("tbody", null, oreRows.map((m) => el("tr", null,
        el("td", null, mini(m.id), nameOf(m.id)), el("td", { text: m.need }), el("td", { text: m.have }),
        el("td", { class: m.need > m.have ? "miss" : "ok", text: Math.max(0, m.need - m.have) || "✓" }))))));
    }
    return box;
  }

  function recipeRow(prof, r, have) {
    const name = r.id ? nameOf(r.id) : r.nombre;
    const canMake = r.mats.length ? Math.min(...r.mats.map((m) => Math.floor((have[m.id] || 0) / m.n))) : 0;
    return el("details", { class: "recipe" + (r.sabe ? "" : " unknown") },
      el("summary", null,
        el("span", { class: "dot c-" + (r.color || "gris"), title: "Color " + (r.color || "") + (r.real ? "" : " (estimado)") }),
        r.id ? mini(r.id) : null,
        el("span", { class: "rn q" + (r.id ? qualityOf(r.id) : 1), text: name }),
        el("span", { class: "rl", text: r.nivel ? String(r.nivel) : "" }),
        el("button", { class: "add", type: "button", "aria-label": "Añadir " + name + " a la lista", onclick: (e) => { e.preventDefault(); addToList(prof, r, 1); } }, "+")),
      el("div", { class: "rbody" },
        el("ul", { class: "matlist" }, r.mats.map((m) => matLine(m, have[m.id] || 0, 1))),
        el("p", { class: "note", text: (canMake ? "Con lo que tienes puedes hacer " + canMake + "." : "Ahora no tienes para hacer ninguna.") +
          (r.sabe ? "" : " No la sabes: " + learnText(r.tabla, prof).toLowerCase() + (r.nivel ? ", desde nivel " + r.nivel : "") + ".") })));
  }

  function renderOfi() {
    const box = $("ofi");
    const profs = Object.entries(d.profesiones || {}).map(([n, s]) => [profName(n), s]);
    if (!profs.length) {
      box.replaceChildren(el("p", { class: "msg", text: "Aún no sé qué profesiones tiene. Entra al juego con el addon de Crónica y haz /reload." }));
      return;
    }
    const have = stockMap();
    const kids = [renderList()];
    for (const [prof, s] of profs.sort((a, b) => b[1].nivel - a[1].nivel)) {
      const { list, hasGame } = recipes(prof);
      const f = filters[prof] || (filters[prof] = { ver: hasGame ? "sabes" : "nivel", q: "" });
      const open = openProf === prof;
      const card = el("div", { class: "card" },
        el("button", { class: "prof-head", type: "button", "aria-expanded": String(open), onclick: () => { openProf = open ? null : prof; renderOfi(); } },
          el("span", { class: "pname", text: prof }), el("span", { class: "plevel", text: s.nivel + " / " + s.max }), el("span", { class: "chev", "aria-hidden": "true", text: open ? "▾" : "▸" })),
        el("div", { class: "bar", role: "progressbar", "aria-valuemin": "0", "aria-valuemax": String(s.max), "aria-valuenow": String(s.nivel) },
          el("span", { style: "width:" + Math.min(100, s.nivel / Math.max(1, s.max) * 100).toFixed(1) + "%" })));
      if (open) {
        if (!list.length) {
          card.append(el("p", { class: "note", text: "Abre esta profesión una vez en el juego y haz /reload para ver tus recetas." }));
        } else {
          const chip = (id, label) => el("button", { type: "button", "aria-pressed": String(f.ver === id), onclick: () => { f.ver = id; renderOfi(); } }, label);
          const search = el("input", { type: "search", placeholder: "Buscar receta…", value: f.q, "aria-label": "Buscar receta" });
          search.addEventListener("input", () => { f.q = search.value; draw(); });
          const holder = el("div", { class: "recipes" });
          const draw = () => {
            const q = f.q.trim().toLowerCase();
            const shown = list.filter((r) => (f.ver === "sabes" ? r.sabe : f.ver === "nivel" ? r.sabe || (r.nivel != null && r.nivel <= s.nivel) : true))
              .filter((r) => !q || (r.id ? nameOf(r.id) : r.nombre || "").toLowerCase().includes(q) || r.mats.some((m) => nameOf(m.id).toLowerCase().includes(q)))
              .sort((a, b) => (b.nivel || 0) - (a.nivel || 0));
            holder.replaceChildren(...(shown.length ? shown.map((r) => recipeRow(prof, r, have)) : [el("p", { class: "note", text: f.ver === "sabes" && !hasGame ? "Abre esta profesión en el juego y haz /reload para que sepa qué recetas tienes." : "Nada con ese filtro." })]));
          };
          card.append(el("div", { class: "filters" }, el("div", { class: "chips" }, chip("sabes", "Las que sé"), chip("nivel", "Hasta mi nivel"), chip("todas", "Todas")), search), holder);
          draw();
          if (!hasGame) card.append(el("p", { class: "note", text: "Sin abrir la profesión en el juego, los colores son estimados." }));
        }
      }
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
    try { icons = await getJSON("../oficios/iconos.json"); } catch (_) { icons = {}; }
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
