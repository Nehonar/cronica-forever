// El cronista: entrevista con Claude para crear el trasfondo de un personaje.
(function () {
  "use strict";
  const $ = (id) => document.getElementById(id);
  const state = { chars: [], key: null, history: [], busy: false, sheet: null };

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

  // Markdown mínimo y seguro: párrafos, listas con «- », *cursiva* y **negrita**.
  function inline(text) {
    const out = [];
    const re = /(\*\*[^*]+\*\*|\*[^*]+\*)/g;
    let last = 0, m;
    while ((m = re.exec(text))) {
      if (m.index > last) out.push(text.slice(last, m.index));
      const t = m[0];
      out.push(t.startsWith("**") ? el("strong", { text: t.slice(2, -2) }) : el("em", { text: t.slice(1, -1) }));
      last = m.index + t.length;
    }
    if (last < text.length) out.push(text.slice(last));
    return out;
  }
  function rich(text) {
    const box = el("div", { class: "body" });
    for (const block of String(text || "").replace(/\r/g, "").split(/\n\s*\n/)) {
      const lines = block.split("\n").map((l) => l.trim()).filter(Boolean);
      if (!lines.length) continue;
      if (lines.every((l) => /^[-•*]\s+/.test(l))) {
        box.append(el("ul", null, lines.map((l) => el("li", null, inline(l.replace(/^[-•*]\s+/, ""))))));
      } else {
        let para = [];
        const flush = () => { if (para.length) { box.append(el("p", null, inline(para.join(" ")))); para = []; } };
        for (const l of lines) {
          if (/^[-•*]\s+/.test(l)) { flush(); box.append(el("ul", null, el("li", null, inline(l.replace(/^[-•*]\s+/, ""))))); }
          else para.push(l);
        }
        flush();
      }
    }
    return box;
  }

  async function api(path, body) {
    const opt = body ? { method: "POST", headers: { "Content-Type": "application/json", "X-Cronica": "1" }, body: JSON.stringify(body) } : {};
    const r = await fetch(path, opt);
    const data = await r.json().catch(() => ({}));
    if (!r.ok) throw new Error(data.error || "Error " + r.status);
    return data;
  }

  /* ---------- Guardado de la conversación en este navegador ---------- */
  const storeKey = (k) => "cronica-entrevista-" + k;
  function saveLocal() {
    try { localStorage.setItem(storeKey(state.key), JSON.stringify({ history: state.history, sheet: state.sheet })); } catch (_) {}
  }
  function loadLocal(k) {
    try { return JSON.parse(localStorage.getItem(storeKey(k)) || "null"); } catch (_) { return null; }
  }
  function clearLocal(k) { try { localStorage.removeItem(storeKey(k)); } catch (_) {} }

  /* ---------- Lista de personajes ---------- */
  function renderChars() {
    const box = $("chars");
    box.replaceChildren();
    if (!state.chars.length) {
      box.append(el("div", { class: "log-empty", text: "Aún no hay personajes: juega un rato con el addon instalado." }));
      return;
    }
    for (const c of state.chars) {
      box.append(el("button", {
        class: "log-item", type: "button", "aria-current": c.key === state.key ? "true" : "false",
        onclick: () => select(c.key),
      },
        el("span", { class: "log-mark", "aria-hidden": "true", text: c.new ? "✦" : "✓" }),
        el("span", { class: "log-title" }, c.title || c.name, c.new ? el("span", { class: "badge", text: "nuevo" }) : null),
        el("span", { class: "log-sub", text: [c.race, c.class, c.level ? "nivel " + c.level : ""].filter(Boolean).join(" · ") + (c.waiting ? " · " + c.waiting + " relato(s) esperando" : "") })));
    }
  }

  /* ---------- Conversación ---------- */
  function addMsg(role, text) {
    const who = role === "jugador" ? "TÚ" : "EL CRONISTA";
    const m = el("div", { class: "msg " + (role === "jugador" ? "you" : "") }, el("div", { class: "who", text: who }), rich(text));
    $("thread").append(m);
    m.scrollIntoView({ block: "end", behavior: "smooth" });
  }
  function addError(text) {
    $("thread").append(el("div", { class: "msg err" }, el("div", { class: "who", text: "AVISO" }), rich(text)));
  }
  function setBusy(b) {
    state.busy = b;
    $("send").disabled = b;
    $("msg").disabled = b;
    const t = document.querySelector(".typing");
    if (b && !t) $("thread").append(el("div", { class: "typing", text: "El cronista escribe" }));
    if (!b && t) t.remove();
  }

  function renderSheet(sh) {
    const old = document.querySelector(".sheet");
    if (old) old.remove();
    const paras = String(sh.backstory || "").split(/\n\s*\n/).filter(Boolean);
    const card = el("section", { class: "sheet", "aria-label": "Ficha del personaje" },
      el("div", { class: "rubric", text: "SU HISTORIA" }),
      el("h3", { text: sh.name || "" }),
      sh.epithet ? el("div", { class: "ep", text: "«" + sh.epithet + "»" }) : null,
      el("div", { class: "prose" }, paras.map((p, i) => el("p", { class: i === 0 ? "drop" : null, text: p.replace(/\n/g, " ") }))),
      sh.threads && sh.threads.length ? [el("div", { class: "rubric", style: "margin-top:14px", text: "HILOS ABIERTOS" }), el("ul", { class: "threads" }, sh.threads.map((t) => el("li", { text: t })))] : null,
      sh.voice ? el("p", { class: "voice", text: "Cómo se narrarán sus relatos: " + sh.voice }) : null,
      sh.motto ? el("div", { class: "motto", text: "Lema: " + sh.motto }) : null,
      el("div", { class: "sheet-actions" },
        el("button", { class: "btn", type: "button", onclick: () => save(sh) }, "Guardar esta historia"),
        el("button", { class: "link-btn", type: "button", onclick: () => { $("msg").focus(); $("msg").placeholder = "Dime qué quieres cambiar…"; } }, "Pedir cambios")));
    $("thread").append(card);
    card.scrollIntoView({ block: "start", behavior: "smooth" });
  }

  async function ask(text) {
    if (state.busy) return;
    if (text) { state.history.push({ role: "jugador", content: text }); addMsg("jugador", text); saveLocal(); }
    setBusy(true);
    try {
      const r = await api("/api/entrevista", { key: state.key, history: state.history });
      setBusy(false);
      if (r.text) { state.history.push({ role: "cronista", content: r.text }); addMsg("cronista", r.text); }
      if (r.sheet) {
        state.sheet = r.sheet;
        // El modelo necesita ver la ficha anterior si se piden cambios.
        state.history.push({ role: "cronista", content: "FICHA:\n```json\n" + JSON.stringify(r.sheet) + "\n```" });
        renderSheet(r.sheet);
      }
      saveLocal();
    } catch (e) {
      setBusy(false);
      if (text) { state.history.pop(); saveLocal(); $("msg").value = text; }
      addError(e.message);
    }
    $("msg").focus();
  }

  async function save(sh) {
    try {
      await api("/api/guardar", { key: state.key, sheet: sh });
      clearLocal(state.key);
      $("composer").hidden = true;
      $("thread").append(el("p", { class: "done" },
        "Guardado. El cronista ya está escribiendo los relatos que esperaban a esta historia. ",
        el("a", { href: "/#" + encodeURIComponent(state.key) }, "Ver su crónica")));
      const c = state.chars.find((x) => x.key === state.key);
      if (c) { c.new = false; c.title = sh.name; c.waiting = 0; }
      renderChars();
    } catch (e) {
      addError(e.message);
    }
  }

  function select(key) {
    const c = state.chars.find((x) => x.key === key);
    if (!c) return;
    state.key = key;
    history.replaceState(null, "", "?p=" + encodeURIComponent(key));
    renderChars();
    $("c-rubric").textContent = c.new ? "PERSONAJE NUEVO" : "REESCRIBIR SU HISTORIA";
    $("c-title").textContent = c.title || c.name;
    $("c-facts").replaceChildren(...[c.race, c.class, c.level ? "Nivel " + c.level : null, c.waiting ? c.waiting + " relato(s) esperando a esta historia" : null]
      .filter(Boolean).map((t) => el("span", { text: t })));
    $("thread").replaceChildren();
    $("composer").hidden = false;
    const saved = loadLocal(key);
    state.history = saved && saved.history ? saved.history : [];
    state.sheet = saved ? saved.sheet : null;
    if (state.history.length) {
      for (const m of state.history) if (!m.content.startsWith("FICHA:")) addMsg(m.role, m.content);
      if (state.sheet) renderSheet(state.sheet);
    } else {
      ask(null);
    }
  }

  $("composer").addEventListener("submit", (e) => {
    e.preventDefault();
    const t = $("msg").value.trim();
    if (!t) return;
    $("msg").value = "";
    ask(t);
  });
  $("msg").addEventListener("keydown", (e) => {
    if (e.key === "Enter" && !e.shiftKey) { e.preventDefault(); $("composer").requestSubmit(); }
  });
  $("restart").addEventListener("click", () => {
    if (!state.key) return;
    clearLocal(state.key);
    select(state.key);
  });

  api("/api/personajes").then((d) => {
    state.chars = d.characters || [];
    renderChars();
    const want = new URLSearchParams(location.search).get("p");
    const first = state.chars.find((c) => c.key === want) || state.chars.find((c) => c.new);
    if (first) select(first.key);
    else $("thread").replaceChildren(el("p", { class: "empty-note", text: state.chars.length ? "Todos tus personajes tienen ya su historia. Elige uno si quieres reescribirla." : "Aún no hay personajes registrados." }));
  }).catch((e) => addError(e.message));
})();
