/* ModPackLooter: búsqueda global, filtros de tablas y preferencias (modo y
   metal). Sin dependencias; funciona desde file:// porque el índice llega
   como script (window.MPL_INDEX), no con fetch. */
(function () {
  "use strict";

  var root = document.documentElement;
  function store(key, value) {
    try { localStorage.setItem(key, value); } catch (e) { /* almacenamiento bloqueado */ }
  }

  /* Preferencias -------------------------------------------------------- */
  var modeBtn = document.getElementById("mode");
  var metalSel = document.getElementById("metal");
  function currentMode() {
    if (root.dataset.mode) return root.dataset.mode;
    return window.matchMedia && matchMedia("(prefers-color-scheme: light)").matches ? "claro" : "oscuro";
  }
  function paintModeButton() {
    if (!modeBtn) return;
    var next = currentMode() === "oscuro" ? "claro" : "oscuro";
    modeBtn.textContent = next === "claro" ? "Modo claro" : "Modo oscuro";
    modeBtn.setAttribute("aria-label", "Cambiar a modo " + next);
  }
  if (modeBtn) {
    paintModeButton();
    modeBtn.addEventListener("click", function () {
      var next = currentMode() === "oscuro" ? "claro" : "oscuro";
      root.dataset.mode = next;
      store("mpl-mode", next);
      paintModeButton();
    });
  }
  if (metalSel) {
    metalSel.value = root.dataset.metal || "oro";
    metalSel.addEventListener("change", function () {
      root.dataset.metal = metalSel.value;
      store("mpl-metal", metalSel.value);
    });
  }

  /* Búsqueda global ----------------------------------------------------- */
  var input = document.getElementById("q");
  var list = document.getElementById("results");
  var index = window.MPL_INDEX || [];
  var siteRoot = input ? input.getAttribute("data-root") || "" : "";
  var groups = { o: "Objetos", e: "Estructuras", b: "Biomas", m: "Mods" };
  var order = ["o", "e", "b", "m"];

  function fold(s) {
    return (s || "").toLowerCase().normalize("NFD").replace(/[̀-ͯ]/g, "");
  }
  for (var i = 0; i < index.length; i++) {
    index[i].fn = fold(index[i].n);
    index[i].fi = fold(index[i].i);
  }

  function score(entry, q, words) {
    var name = entry.fn;
    if (name === q) return 100;
    if (name.indexOf(q) === 0) return 80;
    var all = true, wordStart = true;
    for (var w = 0; w < words.length; w++) {
      var at = name.indexOf(words[w]);
      if (at < 0) { all = false; break; }
      if (at > 0 && name.charAt(at - 1) !== " ") wordStart = false;
    }
    if (all) return wordStart ? 60 : 40;
    if (entry.fi.indexOf(q) >= 0) return 20;
    return 0;
  }

  function search(raw) {
    var q = fold(raw).trim();
    if (!q) return [];
    var words = q.split(/\s+/);
    var hits = [];
    for (var i = 0; i < index.length; i++) {
      var s = score(index[i], q, words);
      if (s > 0) hits.push({ e: index[i], s: s });
    }
    hits.sort(function (a, b) {
      return b.s - a.s || a.e.n.length - b.e.n.length || (a.e.n < b.e.n ? -1 : 1);
    });
    return hits;
  }

  var active = -1;
  function links() { return list ? list.querySelectorAll("a") : []; }
  function select(n) {
    var as = links();
    if (!as.length) return;
    active = (n + as.length) % as.length;
    for (var i = 0; i < as.length; i++) as[i].setAttribute("aria-selected", i === active ? "true" : "false");
    as[active].scrollIntoView({ block: "nearest" });
  }

  function render(raw) {
    if (!list) return;
    list.innerHTML = "";
    active = -1;
    if (!raw.trim()) { close(); return; }
    var hits = search(raw);
    var byType = {};
    for (var i = 0; i < hits.length; i++) {
      var t = hits[i].e.t;
      (byType[t] = byType[t] || []).push(hits[i].e);
    }
    var shown = 0;
    order.forEach(function (t) {
      var items = (byType[t] || []).slice(0, t === "o" ? 12 : 6);
      if (!items.length) return;
      var head = document.createElement("li");
      head.className = "grp";
      head.setAttribute("role", "presentation");
      head.textContent = groups[t] + " (" + byType[t].length + ")";
      list.appendChild(head);
      items.forEach(function (e) {
        var li = document.createElement("li");
        li.setAttribute("role", "presentation");
        var a = document.createElement("a");
        a.href = siteRoot + e.u;
        a.setAttribute("role", "option");
        var name = document.createElement("span");
        name.textContent = e.n;
        var sub = document.createElement("span");
        sub.className = "sub";
        sub.textContent = e.s || e.i;
        a.appendChild(name);
        a.appendChild(sub);
        li.appendChild(a);
        list.appendChild(li);
        shown++;
      });
    });
    if (!shown) {
      var none = document.createElement("li");
      none.className = "none";
      none.textContent = "Sin resultados para «" + raw.trim() + "». Prueba con otra palabra o con el ID (p. ej. minecraft:diamond).";
      list.appendChild(none);
    }
    list.hidden = false;
    input.setAttribute("aria-expanded", "true");
  }

  function close() {
    if (!list) return;
    list.hidden = true;
    input.setAttribute("aria-expanded", "false");
  }

  if (input && list) {
    input.addEventListener("input", function () { render(input.value); });
    input.addEventListener("focus", function () { if (input.value.trim()) render(input.value); });
    input.addEventListener("keydown", function (ev) {
      if (ev.key === "ArrowDown") { ev.preventDefault(); select(active + 1); }
      else if (ev.key === "ArrowUp") { ev.preventDefault(); select(active - 1); }
      else if (ev.key === "Enter") {
        var as = links();
        var target = active >= 0 ? as[active] : as[0];
        if (target) { ev.preventDefault(); window.location.href = target.href; }
      } else if (ev.key === "Escape") { close(); }
    });
    document.addEventListener("click", function (ev) {
      if (!ev.target.closest || !ev.target.closest(".search")) close();
    });
  }

  document.addEventListener("keydown", function (ev) {
    var tag = (ev.target.tagName || "").toLowerCase();
    if (ev.key === "/" && tag !== "input" && tag !== "select" && tag !== "textarea") {
      ev.preventDefault();
      var target = document.getElementById("q-home") || input;
      if (target) target.focus();
    }
  });

  /* El buscador grande de la portada escribe en el global. */
  var proxy = document.getElementById("q-home");
  if (proxy && input) {
    proxy.addEventListener("input", function () {
      input.value = proxy.value;
      render(proxy.value);
      window.scrollTo(0, 0);
    });
    proxy.addEventListener("keydown", function (ev) {
      if (ev.key === "Enter" || ev.key === "ArrowDown") {
        input.focus();
        input.dispatchEvent(new KeyboardEvent("keydown", { key: ev.key }));
        ev.preventDefault();
      }
    });
  }

  /* Filtros de tablas ----------------------------------------------------- */
  var filters = document.querySelectorAll("input.filter[data-filter]");
  Array.prototype.forEach.call(filters, function (f) {
    var table = document.getElementById(f.getAttribute("data-filter"));
    if (!table) return;
    var rows = table.querySelectorAll("tbody tr:not(.group)");
    var texts = Array.prototype.map.call(rows, function (r) { return fold(r.textContent); });
    var empty = f.closest(".pozo").querySelector(".filter-empty");
    f.addEventListener("input", function () {
      var q = fold(f.value).trim();
      var visible = 0;
      for (var i = 0; i < rows.length; i++) {
        var show = !q || texts[i].indexOf(q) >= 0;
        rows[i].hidden = !show;
        if (show) visible++;
      }
      Array.prototype.forEach.call(table.querySelectorAll("tbody"), function (body) {
        var group = body.querySelector("tr.group");
        if (group) group.hidden = !body.querySelector("tr:not(.group):not([hidden])");
      });
      if (empty) empty.hidden = visible > 0;
    });
  });
})();
