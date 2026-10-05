// Barra de navegación de Crónica en tu PC. Solo existe en la versión local:
// en la web publicada (GitHub Pages) este archivo no está y no aparece nada.
(function () {
  "use strict";
  const LINKS = [
    ["/", "Mi crónica"],
    ["/misiones/", "Misiones en curso"],
    ["/personaje/", "Personajes"],
    ["/preparar/", "Configuración"],
  ];
  const here = location.pathname.replace(/index\.html$/, "");
  const css = document.createElement("style");
  css.textContent = `
    .appnav { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; padding: 6px 10px; }
    .appnav a { font-family: var(--font-display, Georgia, serif); color: var(--frame-text, #e9e0cc); text-decoration: none;
      letter-spacing: .04em; font-size: .98rem; padding: 8px 14px; border-radius: 4px; min-height: 40px; display: inline-flex; align-items: center; }
    .appnav a:hover { color: var(--frame-title, #ffd24a); background: rgba(255,255,255,.05); }
    .appnav a[aria-current="page"] { color: var(--frame-title, #ffd24a); box-shadow: inset 0 -2px 0 var(--frame-gold, #e8bd4f); }
    .appnav .brand { margin-right: auto; font-family: var(--font-display, Georgia, serif); color: var(--frame-gold, #e8bd4f); letter-spacing: .14em; font-size: .85rem; padding: 0 8px; }
    @media (max-width: 560px) { .appnav .brand { display: none; } .appnav a { padding: 8px 10px; font-size: .9rem; } }
  `;
  document.head.append(css);
  const nav = document.createElement("nav");
  nav.className = "frame appnav";
  nav.setAttribute("aria-label", "Crónica");
  const brand = document.createElement("span");
  brand.className = "brand";
  brand.textContent = "CRÓNICA · EN TU PC";
  nav.append(brand);
  for (const [href, label] of LINKS) {
    const a = document.createElement("a");
    a.href = href;
    a.textContent = label;
    if (here === href) a.setAttribute("aria-current", "page");
    nav.append(a);
  }
  const wrap = document.querySelector(".wrap") || document.body;
  wrap.prepend(nav);

  // El título marca la pestaña como la de Crónica en este PC (el programa la
  // busca para traerla delante al usar el menú del icono).
  const mark = " · Crónica (este PC)";
  const fix = () => { if (!document.title.endsWith(mark)) document.title = document.title.replace(/\s*·?\s*Crónica de Forever$/, "") + mark; };
  fix();
  new MutationObserver(fix).observe(document.querySelector("title") || document.head, { childList: true, subtree: true, characterData: true });

  // El menú del icono de la bandeja manda aquí adónde ir, en vez de abrir otra pestaña.
  if (window.EventSource) {
    const es = new EventSource("/api/eventos");
    es.addEventListener("ir", (e) => {
      try {
        const to = JSON.parse(e.data).ruta;
        if (to && to.startsWith("/")) location.href = to;
      } catch (_) { /* nada */ }
    });
  }
})();
