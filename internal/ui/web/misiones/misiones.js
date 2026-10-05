// Misiones en curso: se refresca sola y muestra cada misión nada más narrarla.
(function () {
  "use strict";
  const $ = (id) => document.getElementById(id);
  let key = new URLSearchParams(location.search).get("p") || "";
  let lastJSON = "";
  let seen = null; // ids ya mostrados, para resaltar los nuevos

  function el(tag, attrs, ...kids) {
    const n = document.createElement(tag);
    if (attrs) for (const [k, v] of Object.entries(attrs)) {
      if (v == null || v === false) continue;
      if (k === "class") n.className = v;
      else if (k === "text") n.textContent = v;
      else n.setAttribute(k, v === true ? "" : v);
    }
    for (const k of kids.flat()) if (k != null) n.append(k.nodeType ? k : document.createTextNode(String(k)));
    return n;
  }
  function flourish() {
    const ns = "http://www.w3.org/2000/svg";
    const s = document.createElementNS(ns, "svg");
    s.setAttribute("class", "flourish"); s.setAttribute("viewBox", "0 0 180 14"); s.setAttribute("aria-hidden", "true");
    s.innerHTML = '<path d="M2 7 H70 M110 7 H178" stroke="currentColor" stroke-width="1" fill="none"/><path d="M90 1 L96 7 L90 13 L84 7 Z" fill="currentColor"/>';
    return s;
  }
  function prose(text) {
    const box = el("div", { class: "prose" });
    String(text || "").split(/\n\s*\n/).map((t) => t.trim()).filter(Boolean)
      .forEach((t, i) => box.append(el("p", { class: i === 0 ? "drop" : null, text: t.replace(/\n/g, " ") })));
    return box;
  }

  function render(d) {
    const list = $("list");
    const chars = d.characters || [];
    const pick = $("pick");
    pick.hidden = chars.length < 2;
    pick.replaceChildren(...chars.map((c) => el("option", { value: c.key, selected: c.key === d.key ? true : null }, c.title || c.name)));
    if (!d.key) {
      list.replaceChildren(el("p", { class: "empty-note", text: "Aún no hay personajes: juega un rato con el addon instalado." }));
      return;
    }
    key = d.key;
    const ch = chars.find((c) => c.key === d.key) || {};
    $("sub").textContent = (ch.title || d.name) + " · lo que te han encargado, contado por el cronista";
    const all = d.encargos || [];
    const active = all.filter((e) => e.state === "activa");
    const done = all.filter((e) => e.state === "entregada").slice(0, 10);
    const fresh = new Set();
    if (seen) active.forEach((e) => { if (e.text && !seen.has(e.id + ":" + e.narrated)) fresh.add(e.id); });
    seen = new Set(active.map((e) => e.id + ":" + e.narrated));

    const kids = [];
    if (!active.length) {
      kids.push(el("div", { class: "rubric", text: "SIN ENCARGOS" }), el("h2", { text: "Nada pendiente" }), flourish(),
        el("p", { class: "empty-note", text: "Tus misiones aparecen aquí contadas por el cronista cuando el juego guarda: al cerrar sesión o si haces /reload en una pausa." }));
    }
    for (const e of active) {
      const place = [e.subzone, e.zone].filter(Boolean).join(", ");
      const card = el("article", { class: "q" + (fresh.has(e.id) ? " fresh" : ""), id: "q" + e.id },
        el("div", { class: "rubric", text: [e.giver, place].filter(Boolean).join(" · ").toUpperCase() }),
        el("h2", { text: e.heading || e.title }),
        e.heading && e.heading !== e.title ? el("div", { class: "orig", text: "«" + e.title + "»" }) : null,
        e.text ? prose(e.text) : (ch.new
          ? el("p", { class: "empty-note" }, "Para que el cronista narre sus misiones, crea antes la historia de este personaje. ", el("a", { href: "/personaje/?p=" + encodeURIComponent(d.key) }, "Crear su historia"))
          : el("p", { class: "writing", text: "El cronista está escribiendo esta misión" })),
        e.objectives ? el("div", { class: "todo" }, el("div", { class: "rubric", text: "QUÉ HAY QUE HACER" }), el("p", { text: e.objectives })) : null);
      kids.push(card);
    }
    if (done.length) {
      kids.push(el("details", { class: "notes" }, el("summary", { text: "Entregadas hace poco" }),
        el("ul", { class: "done-list" }, done.map((e) => el("li", { text: e.heading && e.heading !== e.title ? e.heading + " — «" + e.title + "»" : e.title })))));
    }
    list.replaceChildren(...kids.filter(Boolean));
    if (fresh.size) {
      window.scrollTo({ top: 0, behavior: "smooth" });
      document.title = "● Nueva misión · Crónica";
      setTimeout(() => { document.title = "Misiones · Crónica de Forever"; }, 8000);
    }
  }

  async function poll() {
    try {
      const r = await fetch("/api/misiones" + (key ? "?p=" + encodeURIComponent(key) : ""), { cache: "no-store" });
      const txt = await r.text();
      if (txt !== lastJSON) {
        lastJSON = txt;
        render(JSON.parse(txt));
      }
    } catch (_) { /* el programa puede estar reiniciándose */ }
    setTimeout(poll, 3000);
  }

  $("pick").addEventListener("change", (e) => {
    key = e.target.value;
    history.replaceState(null, "", "?p=" + encodeURIComponent(key));
    lastJSON = ""; seen = null;
  });
  poll();
})();
