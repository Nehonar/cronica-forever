// Preparar Crónica: carpeta del juego, Claude, GitHub y arranque, sin consolas.
(function () {
  "use strict";
  const $ = (id) => document.getElementById(id);
  let st = null;          // último estado del servidor
  let lastJSON = "";
  let busy = "";          // paso que espera respuesta («juego», «github»…)
  let errors = {};        // error por paso
  let polling = null;
  const drafts = { ruta: "", url: "", codigo: "" };

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
  const btn = (label, onclick, cls) => el("button", { class: "btn" + (cls ? " " + cls : ""), type: "button", onclick, disabled: !!busy }, label);

  async function api(path, body) {
    const opt = body === undefined ? {} : {
      method: "POST", headers: { "Content-Type": "application/json", "X-Cronica": "1" }, body: JSON.stringify(body),
    };
    const r = await fetch(path, opt);
    let data = null;
    try { data = await r.json(); } catch (_) { /* vacío */ }
    if (!r.ok) throw new Error((data && data.error) || "No he podido hablar con Crónica (¿sigue abierta?)");
    return data;
  }

  async function load() {
    try { apply(await api("/api/preparar/estado")); }
    catch (e) { errors.general = e.message; render(); }
  }

  function apply(data) {
    st = data;
    const j = JSON.stringify(data);
    if (j !== lastJSON) { lastJSON = j; render(); } else renderTask();
    if (st.tarea && st.tarea.activa) startPolling();
  }

  async function act(step, path, body) {
    busy = step; errors[step] = ""; render();
    try { apply(await api(path, body)); }
    catch (e) { errors[step] = e.message; }
    busy = ""; lastJSON = ""; render();
  }

  // ---------- trabajo en marcha (instalar, iniciar sesión, GitHub) ----------

  function startPolling() {
    if (polling) return;
    polling = setInterval(async () => {
      try {
        const { tarea } = await api("/api/preparar/tarea");
        st.tarea = tarea;
        renderTask();
        if (!tarea || !tarea.activa) { clearInterval(polling); polling = null; lastJSON = ""; load(); }
      } catch (_) { /* reintenta */ }
    }, 1500);
  }

  function taskStep(t) { return /github/i.test(t.nombre) ? "github" : "claude"; }

  function taskBox(t) {
    const box = el("div", { class: "body", id: "task" });
    box.append(t.activa ? el("p", { class: "working", text: t.nombre + "…" })
      : t.ok ? el("p", { class: "state", text: "✓ " + t.nombre + ": hecho." })
      : el("p", { class: "err", text: t.error || "No ha terminado bien." }));
    if (t.activa && t.url) {
      box.append(el("div", { class: "callout" },
        el("div", { class: "rubric", text: "INICIA SESIÓN EN EL NAVEGADOR" }),
        el("p", null, "Se ha abierto la página de Anthropic en tu navegador. Entra con tu cuenta de Claude y acepta. Si no se ha abierto, ",
          el("a", { href: t.url, target: "_blank", rel: "noopener", text: "ábrela aquí" }), ".")));
    }
    if (t.codigo) {
      const input = el("input", { class: "field", id: "codigo", autocomplete: "off", placeholder: "Pega aquí el código",
        "aria-label": "Código de inicio de sesión", value: drafts.codigo, oninput: (e) => { drafts.codigo = e.target.value; } });
      box.append(el("p", { class: "hint", text: "Si la página te enseña un código al terminar, cópialo y pégalo aquí:" }),
        el("div", { class: "row" }, input, el("button", { class: "btn", type: "button", onclick: sendCode }, "Enviar código")));
    }
    if (t.salida && t.salida.trim()) {
      const d = el("details", { open: t.activa ? null : (!t.ok || null) },
        el("summary", { class: "hint", text: "Lo que va diciendo el instalador" }),
        el("pre", { class: "scroll", text: t.salida.trim() }));
      box.append(d);
    }
    return box;
  }

  async function sendCode() {
    const code = drafts.codigo.trim();
    if (!code) return;
    try { await api("/api/preparar/codigo", { texto: code }); drafts.codigo = ""; }
    catch (e) { alert(e.message); }
  }

  function renderTask() {
    const old = $("task");
    if (!st || !st.tarea) { if (old) old.remove(); return; }
    const t = st.tarea;
    const host = $("s-" + taskStep(t)).querySelector(".content");
    if (!host) return;
    const openDetails = old && old.querySelector("details") ? old.querySelector("details").open : null;
    const pre = old && old.querySelector("pre");
    const atBottom = !pre || pre.scrollTop + pre.clientHeight >= pre.scrollHeight - 8;
    const focused = document.activeElement && document.activeElement.id === "codigo";
    const box = taskBox(t);
    if (old && old.parentNode === host) old.replaceWith(box); else { if (old) old.remove(); host.append(box); }
    const d = box.querySelector("details");
    if (d && openDetails !== null) d.open = openDetails;
    const npre = box.querySelector("pre");
    if (npre && atBottom) npre.scrollTop = npre.scrollHeight;
    if (focused && $("codigo")) $("codigo").focus();
  }

  // ---------- pasos ----------

  // append que ignora null (para los avisos opcionales).
  function add(box, ...kids) { for (const k of kids.flat()) if (k != null) box.append(k); return box; }

  function step(id, n, title, ok, state, bad, content) {
    const li = $(id);
    li.className = "step" + (ok ? " ok" : "") + (id === "s-github" ? " optional" : "");
    li.replaceChildren(
      el("div", { class: "num", "aria-hidden": "true", text: ok ? "✓" : String(n) }),
      el("h3", { text: title }),
      el("p", { class: "state" + (bad ? " bad" : ""), text: state }),
      el("div", { class: "content" }, content));
  }

  function errLine(key) { return errors[key] ? el("p", { class: "err", role: "alert", text: errors[key] }) : null; }

  function stepGame() {
    const g = st.juego;
    const ok = !!g.ruta && g.addon;
    const body = el("div", { class: "body" });
    if (ok) {
      body.append(el("p", { class: "hint", text: g.datos
        ? "El addon ya ha guardado datos: Crónica los está leyendo."
        : "Entra al juego, activa «Crónica» en la lista de AddOns y haz /reload." }));
      body.append(el("div", { class: "row" }, btn("Cambiar de carpeta…", () => act("juego", "/api/preparar/juego", { elegir: true }), "dark")));
    } else {
      if (g.opciones && g.opciones.length) {
        body.append(el("p", { text: "Dentro de esa carpeta hay varias versiones del juego. ¿Cuál es Forever?" }),
          el("div", { class: "choice" }, g.opciones.map((f) =>
            btn(el("span", { class: "path", text: f.ruta }), () => act("juego", "/api/preparar/juego", { ruta: f.ruta }), "dark"))));
      } else if (g.detectadas && g.detectadas.length) {
        body.append(el("p", { text: g.detectadas.length === 1 ? "He encontrado el juego aquí. ¿Es WoW Forever?" : "He encontrado estas instalaciones. Elige la de WoW Forever:" }),
          el("div", { class: "choice" }, g.detectadas.map((f) =>
            btn(el("span", { class: "path", text: f.ruta }), () => act("juego", "/api/preparar/juego", { ruta: f.ruta }), "dark"))));
      } else {
        body.append(el("p", { text: "No encuentro el juego en las carpetas habituales. Elige la carpeta de WoW Forever: la que tiene dentro «Interface» y «WTF» (por ejemplo «_forever_»)." }));
      }
      body.append(el("div", { class: "row" },
        btn("Elegir carpeta…", () => act("juego", "/api/preparar/juego", { elegir: true }))));
      const input = el("input", { class: "field", placeholder: "C:\\Program Files (x86)\\World of Warcraft\\_forever_", "aria-label": "Ruta de la carpeta del juego",
        value: drafts.ruta, oninput: (e) => { drafts.ruta = e.target.value; } });
      body.append(el("details", null, el("summary", { class: "hint", text: "O escribe la ruta" }),
        el("div", { class: "row", style: "margin-top:8px" }, input,
          btn("Usar esta", () => drafts.ruta.trim() && act("juego", "/api/preparar/juego", { ruta: drafts.ruta.trim() }), "dark"))));
      if (busy === "juego") body.append(el("p", { class: "working", text: "Mira si se ha abierto la ventana para elegir la carpeta" }));
    }
    add(body, errLine("juego"), g.error ? el("p", { class: "err", text: g.error }) : null);
    const state = ok ? "Addon instalado en " + g.ruta : g.ruta ? "Falta el addon en " + g.ruta : "Aún no sé dónde está el juego.";
    step("s-juego", 1, "El juego", ok, state, false, body);
  }

  function stepClaude() {
    const c = st.claude;
    const t = st.tarea && taskStep(st.tarea) === "claude" && st.tarea.activa;
    const ok = c.instalado && c.sesion;
    const body = el("div", { class: "body" });
    if (!ok && !t) {
      if (!c.instalado) {
        body.append(el("p", { text: "El cronista escribe con tu suscripción de Claude a través de Claude Code, el programa oficial de Anthropic. Aún no está en este equipo." }),
          el("p", { class: "hint", text: "Lo instalo con el instalador oficial y después se abrirá tu navegador para que entres en tu cuenta. Crónica nunca ve tu contraseña." }),
          el("div", { class: "row" }, btn("Instalar Claude Code", () => act("claude", "/api/preparar/claude", { accion: "instalar" }))));
      } else {
        body.append(el("p", { text: "Claude Code está instalado pero sin la sesión iniciada." }),
          el("p", { class: "hint", text: "Se abrirá tu navegador en la página de Anthropic para que entres con tu cuenta de Claude. Crónica nunca ve tu contraseña." }),
          el("div", { class: "row" }, btn("Iniciar sesión", () => act("claude", "/api/preparar/claude", { accion: "sesion" }))));
      }
    }
    if (ok) body.append(el("p", { class: "hint", text: "El cronista puede escribir. No hace falta hacer nada más." }));
    add(body, errLine("claude"));
    const state = ok ? "Instalado y con la sesión iniciada." : c.instalado ? "Instalado, falta iniciar sesión." : "No está instalado.";
    step("s-claude", 2, "Claude", ok, c.error && !ok ? c.error : state, !!c.error && !ok, body);
  }

  function stepGitHub() {
    const g = st.github;
    const t = st.tarea && taskStep(st.tarea) === "github" && st.tarea.activa;
    const ok = g.publicar;
    const body = el("div", { class: "body" });
    if (ok) {
      const ago = (t) => {
        const m = Math.round((Date.now() / 1000 - t) / 60);
        return m < 1 ? "hace un momento" : m < 60 ? "hace " + m + " min" : "a las " + new Date(t * 1000).toLocaleTimeString("es-ES", { hour: "2-digit", minute: "2-digit" });
      };
      if (g.error) {
        body.append(el("div", { class: "callout" },
          el("div", { class: "rubric", text: "NO SE HA PODIDO PUBLICAR" }),
          el("p", { class: "err", text: g.error })));
      } else if (g.ultima) {
        body.append(el("p", { class: "hint", text: "✓ Última publicación " + ago(g.ultima) + "." }));
      } else {
        body.append(el("p", { class: "hint", text: "Aún no se ha publicado nada en esta sesión: se publica solo cada vez que el cronista escribe o guardas una historia." }));
      }
      body.append(el("div", { class: "row" }, btn("Publicar ahora", () => act("github", "/api/preparar/publicar", {}))));
      body.append(el("p", { class: "hint", text: "Recuerda activar GitHub Pages en el repositorio: Settings → Pages → rama principal, carpeta /docs." }),
        el("div", { class: "row" },
          btn("Cambiar de repositorio", () => { st.github.publicar = false; lastJSON = ""; render(); }, "dark"),
          el("button", { class: "link-btn", type: "button", disabled: !!busy, onclick: () => act("github", "/api/preparar/github", { quitar: true }) }, "Dejar de publicar")));
    } else if (!g.git) {
      body.append(el("p", { text: "Tu crónica siempre se ve en este PC. Si además quieres una web pública, necesitas una cuenta de GitHub y Git." }),
        el("p", null, "Instala ", el("a", { href: "https://git-scm.com/download/win", target: "_blank", rel: "noopener", text: "Git para Windows" }),
          " (todo con las opciones por defecto) y vuelve a esta página."),
        el("div", { class: "row" }, btn("Ya lo he instalado", load, "dark")));
    } else if (!t) {
      const input = el("input", { class: "field", type: "url", placeholder: "https://github.com/TuUsuario/mi-cronica", "aria-label": "Dirección de tu repositorio",
        value: drafts.url, oninput: (e) => { drafts.url = e.target.value; } });
      body.append(el("p", { text: "Tu crónica siempre se ve en este PC. Si además quieres una web pública, pega la dirección de un repositorio tuyo de GitHub." }),
        el("div", { class: "row" }, input, btn("Conectar", () => act("github", "/api/preparar/github", { url: drafts.url }), "dark")),
        el("p", { class: "hint", text: "La primera vez se abrirá una ventana de GitHub para que entres en tu cuenta. Si no quieres web pública, sáltate este paso." }));
    }
    add(body, errLine("github"));
    step("s-github", 3, "Web pública en GitHub (opcional)", ok && !g.error, ok ? "Publicando en " + (g.url || "GitHub") : "Solo en este PC.", !!g.error, body);
  }

  function stepAutostart() {
    const on = st.arranque;
    const body = el("div", { class: "body" });
    if (st.puede_arrancar) {
      body.append(el("button", { class: "toggle", type: "button", role: "switch", "aria-checked": String(on), disabled: !!busy,
        onclick: () => act("arranque", "/api/preparar/arranque", { activo: !on }) },
        el("span", { class: "track", "aria-hidden": "true" }), "Abrir Crónica al encender el PC"));
      body.append(el("p", { class: "hint", text: on
        ? "Crónica arrancará sola, con su icono en la bandeja (junto al reloj; puede estar dentro de la flechita ^)."
        : "Si no, ábrela tú con el programa antes de jugar." }));
    } else {
      body.append(el("p", { class: "hint", text: "En este sistema, abre Crónica tú antes de jugar." }));
    }
    add(body, errLine("arranque"));
    step("s-arranque", 4, "Arranque", on, on ? "Se abre sola al encender el PC." : "Se abre a mano.", false, body);
  }

  function finale() {
    const box = $("finale");
    const ready = st.juego.ruta && st.juego.addon && st.claude.instalado && st.claude.sesion;
    box.hidden = !ready;
    if (!ready) return;
    box.replaceChildren(
      el("div", { class: "rubric", text: "TODO LISTO" }),
      el("h2", { text: "Que empiece la historia" }),
      el("ol", null,
        el("li", null, "Entra al juego y, en la pantalla de personajes, pulsa ", el("b", { text: "AddOns" }), " y activa «Crónica»."),
        el("li", null, "Ya dentro, escribe ", el("kbd", { text: "/cronica" }), " para ver que funciona. En Opciones → Atajos → Crónica puedes asignar teclas."),
        el("li", null, "Haz ", el("kbd", { text: "/reload" }), ". El icono de Crónica te propondrá crear la historia de tu personaje con el cronista.")),
      el("div", { class: "row" },
        el("a", { class: "btn", href: "/personaje/" }, "Personajes e historias"),
        el("a", { class: "btn dark", href: "/misiones/" }, "Misiones en curso"),
        el("a", { class: "btn dark", href: "/" }, "Mi crónica")));
  }

  function render() {
    if (!st) {
      if (errors.general) $("s-juego").replaceChildren(el("p", { class: "err", text: errors.general }));
      return;
    }
    $("ver").textContent = st.version ? "v" + st.version : "";
    stepGame(); stepClaude(); stepGitHub(); stepAutostart(); finale();
    renderTask();
  }

  // Al volver del navegador (inicio de sesión, GitHub), comprobar de nuevo.
  window.addEventListener("focus", () => { if (!busy && !polling) { lastJSON = ""; load(); } });
  load();
})();
