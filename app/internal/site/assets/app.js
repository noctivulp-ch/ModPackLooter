/* ModPackLooter: buscador inteligente, filtros de listas y apariencia (modo
   y metal). Sin dependencias; funciona desde file:// porque el índice llega
   como script (window.MPL_INDEX), no con fetch. */
(function () {
  "use strict";

  var root = document.documentElement;
  function store(key, value) {
    try {
      if (value === null) localStorage.removeItem(key); else localStorage.setItem(key, value);
    } catch (e) { /* almacenamiento bloqueado */ }
  }
  function read(key) {
    try { return localStorage.getItem(key); } catch (e) { return null; }
  }

  /* Apariencia ---------------------------------------------------------- */
  var look = document.querySelector(".look");
  if (look) {
    var mode = read("mpl-mode") || "auto";
    var metal = root.dataset.metal || "oro";
    Array.prototype.forEach.call(look.querySelectorAll("input[name=mode]"), function (r) {
      r.checked = r.value === mode;
      r.addEventListener("change", function () {
        if (r.value === "auto") { delete root.dataset.mode; store("mpl-mode", null); }
        else { root.dataset.mode = r.value; store("mpl-mode", r.value); }
      });
    });
    Array.prototype.forEach.call(look.querySelectorAll("input[name=metal]"), function (r) {
      r.checked = r.value === metal;
      r.addEventListener("change", function () { root.dataset.metal = r.value; store("mpl-metal", r.value); });
    });
    document.addEventListener("click", function (ev) { if (look.open && !look.contains(ev.target)) look.open = false; });
    look.addEventListener("keydown", function (ev) {
      if (ev.key === "Escape") { look.open = false; look.querySelector("summary").focus(); }
    });
  }

  /* Motor de búsqueda ----------------------------------------------------
     Une palabras parciales, sin tildes y con erratas (distancia de edición),
     y entiende filtros: @mod (o mod:xyz) y ids con ":" (minecraft:, tacz:ak47). */
  function fold(s) {
    return (s || "").toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "");
  }
  function words(s) { return s.split(/[^a-z0-9ñ]+/).filter(Boolean); }

  // Distancia de edición con tope: devuelve max+1 si la supera.
  function lev(a, b, max) {
    if (Math.abs(a.length - b.length) > max) return max + 1;
    var prev = [], cur = [], i, j;
    for (j = 0; j <= b.length; j++) prev[j] = j;
    for (i = 1; i <= a.length; i++) {
      cur = [i];
      var best = i;
      for (j = 1; j <= b.length; j++) {
        var c = Math.min(prev[j] + 1, cur[j - 1] + 1, prev[j - 1] + (a.charAt(i - 1) === b.charAt(j - 1) ? 0 : 1));
        if (i > 1 && j > 1 && a.charAt(i - 1) === b.charAt(j - 2) && a.charAt(i - 2) === b.charAt(j - 1)) c = Math.min(c, prev[j - 2] + 1);
        cur[j] = c;
        if (c < best) best = c;
      }
      if (best > max) return max + 1;
      prev = cur;
    }
    return prev[b.length];
  }

  function parse(raw) {
    var q = { mods: [], ids: [], words: [], text: "" };
    fold(raw).split(/\s+/).forEach(function (t) {
      if (!t) return;
      if (t.charAt(0) === "@" && t.length > 1) q.mods.push(t.slice(1));
      else if (t.indexOf("mod:") === 0 && t.length > 4) q.mods.push(t.slice(4));
      else if (t.indexOf(":") > 0) q.ids.push(t);
      else q.words = q.words.concat(words(t));
    });
    q.text = q.words.join(" ");
    return q;
  }
  function empty(q) { return !q.mods.length && !q.ids.length && !q.words.length; }

  // prepare guarda las formas plegadas de una entrada {n, k, i, m}.
  function prepare(e) {
    e._n = fold(e.n); e._nw = words(e._n);
    e._k = fold(e.k); e._kw = words(e._k);
    e._i = fold(e.i); e._m = fold(e.m);
    return e;
  }

  function wordScore(w, e) {
    var i, x;
    for (i = 0; i < e._nw.length; i++) {
      x = e._nw[i];
      if (x === w) return 10;
      if (x.indexOf(w) === 0) return 8;
    }
    if (e._n.indexOf(w) >= 0) return 6;
    for (i = 0; i < e._kw.length; i++) {
      x = e._kw[i];
      if (x === w) return 6;
      if (x.indexOf(w) === 0) return 5;
    }
    if (e._k.indexOf(w) >= 0) return 4;
    if (w.length < 4) return 0;
    var max = w.length < 6 ? 1 : 2;
    var lists = [e._nw, e._kw];
    for (var l = 0; l < 2; l++) {
      for (i = 0; i < lists[l].length; i++) {
        x = lists[l][i];
        if (x.length < 3) continue;
        var d = Math.min(lev(w, x, max), x.length > w.length ? lev(w, x.slice(0, w.length), max) : max + 1);
        if (d <= max) return (l === 0 ? 3 : 2) - d * 0.5;
      }
    }
    return 0;
  }

  // score es 0 si la entrada no pasa la consulta.
  function score(e, q) {
    var i;
    for (i = 0; i < q.mods.length; i++) {
      var m = q.mods[i];
      if (e._m.indexOf(m) !== 0 && e._k.indexOf(m) < 0) return 0;
    }
    for (i = 0; i < q.ids.length; i++) {
      var id = q.ids[i];
      if (id.charAt(id.length - 1) === ":" ? e._i.indexOf(id) !== 0 : e._i.indexOf(id) < 0) return 0;
    }
    if (!q.words.length) return 1;
    var total = 0;
    for (i = 0; i < q.words.length; i++) {
      var s = wordScore(q.words[i], e);
      if (!s) return 0;
      total += s;
    }
    if (e._n === q.text) total += 30;
    else if (e._n.indexOf(q.text) === 0) total += 12;
    return total;
  }

  function search(list, raw) {
    var q = parse(raw);
    if (empty(q)) return [];
    var hits = [];
    for (var i = 0; i < list.length; i++) {
      var s = score(list[i], q);
      if (s > 0) hits.push({ e: list[i], s: s });
    }
    hits.sort(function (a, b) {
      return b.s - a.s || a.e.n.length - b.e.n.length || (a.e.n < b.e.n ? -1 : 1);
    });
    return hits;
  }

  // mark resalta en el nombre lo que coincide literalmente.
  function mark(el, name, q) {
    var f = fold(name), spans = [];
    q.words.forEach(function (w) {
      var at = f.indexOf(w);
      if (at >= 0) spans.push([at, at + w.length]);
    });
    spans.sort(function (a, b) { return a[0] - b[0]; });
    var pos = 0;
    spans.forEach(function (s) {
      if (s[0] < pos) return;
      el.appendChild(document.createTextNode(name.slice(pos, s[0])));
      var m = document.createElement("mark");
      m.textContent = name.slice(s[0], s[1]);
      el.appendChild(m);
      pos = s[1];
    });
    el.appendChild(document.createTextNode(name.slice(pos)));
  }

  /* Búsqueda global: panel emergente ------------------------------------ */
  var finder = document.getElementById("finder");
  var input = document.getElementById("q");
  var list = document.getElementById("results");
  var index = (window.MPL_INDEX || []).map(prepare);
  var siteRoot = input ? input.getAttribute("data-root") || "" : "";
  var groups = { o: "Objetos", e: "Estructuras", l: "Lost Cities", c: "Criaturas", b: "Biomas", m: "Mods" };
  var order = ["o", "e", "l", "c", "b", "m"];
  var opener = null;

  var active = -1;
  function options() { return list ? list.querySelectorAll("a[role=option]") : []; }
  function select(n) {
    var as = options();
    if (!as.length) return;
    active = (n + as.length) % as.length;
    for (var i = 0; i < as.length; i++) as[i].setAttribute("aria-selected", i === active ? "true" : "false");
    as[active].scrollIntoView({ block: "nearest" });
    input.setAttribute("aria-activedescendant", as[active].id);
  }

  function li(cls, text) {
    var x = document.createElement("li");
    x.className = cls;
    x.setAttribute("role", "presentation");
    if (text) x.textContent = text;
    return x;
  }

  function render(raw) {
    if (!list) return;
    list.innerHTML = "";
    active = -1;
    input.removeAttribute("aria-activedescendant");
    var typing = !!raw.trim();
    finder.classList.toggle("typing", typing);
    if (!typing) { list.hidden = true; input.setAttribute("aria-expanded", "false"); return; }
    var q = parse(raw);
    var hits = search(index, raw);
    var byType = {};
    hits.forEach(function (h) { (byType[h.e.t] = byType[h.e.t] || []).push(h.e); });
    var shown = 0;
    order.forEach(function (t) {
      var items = (byType[t] || []).slice(0, t === "o" ? 12 : 6);
      if (!items.length) return;
      list.appendChild(li("grp", groups[t] + " · " + byType[t].length));
      items.forEach(function (e) {
        var x = li("");
        var a = document.createElement("a");
        a.href = siteRoot + e.u;
        a.id = "r" + shown;
        a.setAttribute("role", "option");
        var name = document.createElement("span");
        mark(name, e.n, q);
        var sub = document.createElement("span");
        sub.className = "sub";
        sub.textContent = e.s || e.i;
        a.appendChild(name);
        a.appendChild(sub);
        x.appendChild(a);
        list.appendChild(x);
        shown++;
      });
    });
    if ((byType.o || []).length > 12) {
      var all = li("all");
      var a = document.createElement("a");
      a.href = siteRoot + "objetos/index.html?q=" + encodeURIComponent(raw.trim());
      a.textContent = "Ver los " + byType.o.length + " objetos que coinciden";
      a.id = "r" + shown;
      a.setAttribute("role", "option");
      all.appendChild(a);
      list.appendChild(all);
    }
    if (!shown) list.appendChild(li("none", "Sin resultados para «" + raw.trim() + "». Prueba con menos letras, otra palabra o el nombre en inglés."));
    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
  }

  function openFinder(text) {
    if (!finder) return;
    opener = document.activeElement;
    finder.hidden = false;
    document.body.style.overflow = "hidden";
    if (typeof text === "string") input.value = text;
    render(input.value);
    input.focus();
    var end = input.value.length;
    try { input.setSelectionRange(end, end); } catch (e) { /* type=search */ }
  }
  function closeFinder() {
    if (!finder || finder.hidden) return;
    finder.hidden = true;
    document.body.style.overflow = "";
    if (opener && opener.focus) opener.focus();
  }

  if (finder && input && list) {
    document.getElementById("find-open").addEventListener("click", function () { openFinder(); });
    document.getElementById("find-close").addEventListener("click", closeFinder);
    finder.addEventListener("click", function (ev) { if (ev.target === finder) closeFinder(); });
    input.addEventListener("input", function () { render(input.value); });
    input.addEventListener("keydown", function (ev) {
      if (ev.key === "ArrowDown") { ev.preventDefault(); select(active + 1); }
      else if (ev.key === "ArrowUp") { ev.preventDefault(); select(active - 1); }
      else if (ev.key === "Enter") {
        var as = options();
        var target = active >= 0 ? as[active] : as[0];
        if (target) { ev.preventDefault(); closeFinder(); window.location.href = target.href; }
      }
    });
    list.addEventListener("click", function (ev) {
      if (ev.target.closest && ev.target.closest("a")) closeFinder();
    });
    finder.addEventListener("keydown", function (ev) {
      if (ev.key === "Escape") { ev.preventDefault(); closeFinder(); }
      if (ev.key === "Tab") {
        // El foco no sale del panel mientras está abierto.
        var f = finder.querySelectorAll("input, button, a[href]");
        var first = f[0], last = f[f.length - 1];
        if (ev.shiftKey && document.activeElement === first) { ev.preventDefault(); last.focus(); }
        else if (!ev.shiftKey && document.activeElement === last) { ev.preventDefault(); first.focus(); }
      }
    });
  }

  document.addEventListener("keydown", function (ev) {
    var tag = (ev.target.tagName || "").toLowerCase();
    if (ev.key === "/" && tag !== "input" && tag !== "select" && tag !== "textarea") {
      ev.preventDefault();
      var home = document.getElementById("q-home");
      if (home) home.focus(); else openFinder();
    }
  });

  /* La portada tiene su propia barra: al escribir abre el panel con el texto. */
  var proxy = document.getElementById("q-home");
  if (proxy) {
    proxy.addEventListener("input", function () {
      var text = proxy.value;
      proxy.value = "";
      openFinder(text);
    });
    proxy.addEventListener("keydown", function (ev) {
      if (ev.key === "Enter") { ev.preventDefault(); openFinder(proxy.value); }
    });
    Array.prototype.forEach.call(document.querySelectorAll("[data-try]"), function (b) {
      b.addEventListener("click", function () { openFinder(b.getAttribute("data-try")); });
    });
  }

  /* Listas con buscador inteligente y facetas ---------------------------- */
  function plural(n, one, many) { return n + " " + (n === 1 ? one : many); }

  Array.prototype.forEach.call(document.querySelectorAll("input[data-smart]"), function (f) {
    var table = document.getElementById(f.getAttribute("data-smart"));
    var box = f.closest("[data-list]");
    if (!table || !box) return;
    var body = table.tBodies[0];
    var rows = Array.prototype.slice.call(body.rows);
    var entries = rows.map(function (r) {
      var link = r.querySelector("a");
      var cells = r.cells;
      return prepare({ row: r, n: link ? link.textContent : "", k: (r.getAttribute("data-k") || "") + " " + cells[cells.length - 1].textContent,
        i: r.getAttribute("data-k") || "", m: r.getAttribute("data-mod") || "", w: " " + (r.getAttribute("data-w") || "") + " ", ok: r.getAttribute("data-ok") === "1" });
    });
    var facets = box.querySelectorAll("[data-facet]");
    var counter = box.querySelector("[data-count]");
    var emptyMsg = box.querySelector(".filter-empty");
    function apply() {
      var q = parse(f.value);
      var way = "", mod = "", okOnly = false;
      Array.prototype.forEach.call(facets, function (x) {
        var k = x.getAttribute("data-facet");
        if (k === "w") way = x.value;
        else if (k === "mod") mod = x.value;
        else if (k === "ok") okOnly = x.checked;
      });
      var shown = [];
      entries.forEach(function (e) {
        var s = empty(q) ? 1 : score(e, q);
        var pass = s > 0 && (!way || e.w.indexOf(" " + way + " ") >= 0) && (!mod || e.m === mod) && (!okOnly || e.ok);
        e.row.hidden = !pass;
        if (pass) shown.push({ e: e, s: s });
      });
      // Con palabras, lo más parecido primero; sin ellas, el orden original.
      var ordered = q.words.length ? shown.slice().sort(function (a, b) { return b.s - a.s; }) : null;
      var frag = document.createDocumentFragment();
      (ordered ? ordered.map(function (x) { return x.e; }) : entries).forEach(function (e) { frag.appendChild(e.row); });
      if (ordered) entries.forEach(function (e) { if (e.row.hidden) frag.appendChild(e.row); });
      body.appendChild(frag);
      if (counter) counter.textContent = plural(shown.length, "objeto", "objetos");
      if (emptyMsg) emptyMsg.hidden = shown.length > 0;
    }
    f.addEventListener("input", apply);
    Array.prototype.forEach.call(facets, function (x) { x.addEventListener("change", apply); });
    var start = /[?&]q=([^&]*)/.exec(window.location.search);
    if (start) f.value = decodeURIComponent(start[1].replace(/\+/g, " "));
    apply();
  });

  /* Tarjetas (estructuras): se buscan por nombre, mod o lo que contienen. */
  Array.prototype.forEach.call(document.querySelectorAll("input[data-cards]"), function (f) {
    var box = document.getElementById(f.getAttribute("data-cards"));
    if (!box) return;
    var cards = Array.prototype.map.call(box.querySelectorAll(".place"), function (c) {
      var h = c.querySelector("h3");
      return prepare({ el: c, n: h ? h.textContent : "", k: c.getAttribute("data-k") || "", i: c.getAttribute("data-k") || "", m: "" });
    });
    var cats = box.querySelectorAll(".cat");
    var counter = f.parentNode.querySelector("[data-count]");
    var emptyMsg = box.parentNode.querySelector(".filter-empty");
    function apply() {
      var q = parse(f.value), n = 0;
      cards.forEach(function (c) {
        var pass = empty(q) || score(c, q) > 0;
        c.el.hidden = !pass;
        if (pass) n++;
      });
      Array.prototype.forEach.call(cats, function (cat) { cat.hidden = !cat.querySelector(".place:not([hidden])"); });
      if (counter) counter.textContent = plural(n, "estructura", "estructuras");
      if (emptyMsg) emptyMsg.hidden = n > 0;
    }
    f.addEventListener("input", apply);
    apply();
  });

  /* Filtros simples de tablas, con el mismo motor. */
  Array.prototype.forEach.call(document.querySelectorAll("input.filter[data-filter]"), function (f) {
    var table = document.getElementById(f.getAttribute("data-filter"));
    if (!table) return;
    var rows = Array.prototype.map.call(table.querySelectorAll("tbody tr:not(.group)"), function (r) {
      var text = r.textContent;
      return prepare({ row: r, n: text, k: "", i: text, m: "" });
    });
    var holder = f.closest(".pozo") || document;
    var emptyMsg = holder.querySelector(".filter-empty");
    f.addEventListener("input", function () {
      var q = parse(f.value), visible = 0;
      rows.forEach(function (r) {
        var show = empty(q) || score(r, q) > 0;
        r.row.hidden = !show;
        if (show) visible++;
      });
      Array.prototype.forEach.call(table.querySelectorAll("tbody"), function (body) {
        var group = body.querySelector("tr.group");
        if (group) group.hidden = !body.querySelector("tr:not(.group):not([hidden])");
      });
      if (emptyMsg) emptyMsg.hidden = visible > 0;
    });
  });

  /* Un enlace a un detalle plegado lo abre. */
  function openTarget() {
    var id = decodeURIComponent(window.location.hash.slice(1));
    var el = id && document.getElementById(id);
    if (el && el.tagName === "DETAILS") el.open = true;
  }
  window.addEventListener("hashchange", openTarget);
  openTarget();
})();
