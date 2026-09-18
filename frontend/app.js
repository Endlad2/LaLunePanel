// LaLune Panel frontend — vanilla SPA, no build step.

const state = {
  protocols: [],
  selectedProtocol: null,
  step: 1,
  params: {},
};

// --- helpers ---------------------------------------------------------------

const $ = (sel) => document.querySelector(sel);
const $$ = (sel) => document.querySelectorAll(sel);

async function api(path, opts = {}) {
  const res = await fetch("/api" + path, {
    headers: { "Content-Type": "application/json" },
    ...opts,
  });
  const text = await res.text();
  let data;
  try { data = text ? JSON.parse(text) : null; } catch { data = text; }
  if (!res.ok) {
    throw new Error((data && data.error) || `HTTP ${res.status}`);
  }
  return data;
}

function fmtBytes(kb) {
  if (!kb && kb !== 0) return "—";
  const units = ["KB", "MB", "GB", "TB"];
  let v = kb, i = 0;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++; }
  return v.toFixed(1) + " " + units[i];
}

function fmtUptime(s) {
  if (!s && s !== 0) return "—";
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d) return `${d}d ${h}h`;
  if (h) return `${h}h ${m}m`;
  return `${m}m`;
}

function fmtDate(iso) {
  if (!iso) return "—";
  return new Date(iso).toLocaleDateString();
}

function daysLeft(iso) {
  if (!iso) return 0;
  const diff = new Date(iso) - new Date();
  return Math.ceil(diff / 86400000);
}

// --- routing ---------------------------------------------------------------

function route() {
  const hash = location.hash.replace(/^#\/?/, "") || "dashboard";
  $$(".view").forEach((v) => v.classList.add("hidden"));
  $$(".nav-item").forEach((n) => n.classList.remove("active"));

  const view = document.getElementById("view-" + hash) || document.getElementById("view-dashboard");
  view.classList.remove("hidden");
  const nav = document.querySelector(`.nav-item[data-route="${hash}"]`);
  if (nav) nav.classList.add("active");

  if (hash === "dashboard") loadStats();
  if (hash === "clients") loadClients();
}

window.addEventListener("hashchange", route);

// --- dashboard -------------------------------------------------------------

async function loadStats() {
  try {
    const s = await api("/stats");
    $("#stat-clients").textContent = s.clients ?? "—";
    $("#stat-active").textContent = s.active ?? "—";
    $("#stat-load").textContent = (s.host?.load1 ?? 0).toFixed(2);
    $("#stat-mem").textContent = fmtBytes((s.host?.mem_used_kb) || 0) + " / " + fmtBytes(s.host?.mem_total_kb || 0);
    $("#stat-cores").textContent = s.host?.cpu_cores ?? "—";
    $("#stat-uptime").textContent = fmtUptime(s.uptime_s);
    $("#hostname").textContent = s.host?.hostname || "—";
  } catch (e) {
    console.error("stats:", e);
  }
}

// --- clients ---------------------------------------------------------------

async function loadClients() {
  const tbody = $("#clients-body");
  const empty = $("#clients-empty");
  tbody.innerHTML = "";
  try {
    const clients = await api("/clients");
    if (!clients.length) {
      empty.classList.remove("hidden");
      return;
    }
    empty.classList.add("hidden");
    for (const c of clients) {
      const left = daysLeft(c.expires_at);
      const expired = left <= 0;
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td>${escapeHtml(c.name)}</td>
        <td>${escapeHtml(c.protocol)}</td>
        <td>${fmtDate(c.expires_at)} ${expired ? "" : `<span class="muted">(${left}d)</span>`}</td>
        <td><span class="badge ${expired ? "exp" : "ok"}">${expired ? "expired" : "active"}</span></td>
        <td class="actions">
          <button class="btn btn-ghost" data-view="${c.id}">View</button>
          <button class="btn btn-danger" data-del="${c.id}">Delete</button>
        </td>`;
      tbody.appendChild(tr);
    }

    tbody.querySelectorAll("[data-del]").forEach((btn) =>
      btn.addEventListener("click", () => deleteClient(btn.dataset.del)));
    tbody.querySelectorAll("[data-view]").forEach((btn) =>
      btn.addEventListener("click", () => viewClient(btn.dataset.view)));

  } catch (e) {
    tbody.innerHTML = `<tr><td colspan="5" class="muted">${escapeHtml(e.message)}</td></tr>`;
  }
}

async function deleteClient(id) {
  if (!confirm("Delete this client?")) return;
  try {
    await api("/clients/" + id, { method: "DELETE" });
    loadClients();
  } catch (e) {
    alert("Delete failed: " + e.message);
  }
}

async function viewClient(id) {
  try {
    const c = await api("/clients/" + id);
    showResult(c.config, c.uri);
  } catch (e) {
    alert("Load failed: " + e.message);
  }
}

function escapeHtml(s) {
  return String(s ?? "").replace(/[&<>"']/g, (m) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[m]));
}

// --- add-client modal ------------------------------------------------------

function openModal() {
  state.step = 1;
  state.selectedProtocol = null;
  state.params = {};
  $("#f-name").value = "";
  $("#f-days").value = "30";
  $("#modal-error").classList.add("hidden");
  $("#step-2").classList.add("hidden");
  $("#step-1").classList.remove("hidden");
  $("#btn-next").textContent = "Next";
  $("#btn-back").style.visibility = "hidden";
  renderProtocolSelect();
  $("#modal").classList.remove("hidden");
}

function closeModal() {
  $("#modal").classList.add("hidden");
}

function renderProtocolSelect() {
  const sel = $("#f-protocol");
  sel.innerHTML = "";
  for (const p of state.protocols) {
    const opt = document.createElement("option");
    opt.value = p.id;
    opt.textContent = p.name;
    sel.appendChild(opt);
  }
  onProtocolChange();
  sel.onchange = onProtocolChange;
}

function onProtocolChange() {
  const id = $("#f-protocol").value;
  state.selectedProtocol = state.protocols.find((p) => p.id === id) || null;
  $("#proto-desc").textContent = state.selectedProtocol?.description || "";
}

function renderParams() {
  const p = state.selectedProtocol;
  const box = $("#params-container");
  box.innerHTML = "";

  for (const f of p.fields) {
    const label = document.createElement("label");
    label.className = "field";

    let input;
    if (f.type === "select") {
      input = document.createElement("select");
      for (const o of f.options || []) {
        const opt = document.createElement("option");
        opt.value = o;
        opt.textContent = o;
        input.appendChild(opt);
      }
      if (f.default) input.value = f.default;
    } else {
      input = document.createElement("input");
      input.type = f.type === "number" ? "number" : (f.type === "password" ? "password" : "text");
      if (f.default) input.value = f.default;
      if (f.placeholder) input.placeholder = f.placeholder;
    }
    input.dataset.key = f.key;

    const span = document.createElement("span");
    span.textContent = f.label + (f.required ? " *" : "");
    label.appendChild(span);
    label.appendChild(input);

    if (f.help) {
      const h = document.createElement("p");
      h.className = "hint";
      h.textContent = f.help;
      label.appendChild(h);
    }
    box.appendChild(label);
  }
}

function collectParams() {
  const params = {};
  $("#params-container").querySelectorAll("[data-key]").forEach((el) => {
    params[el.dataset.key] = el.value;
  });
  return params;
}

async function submitClient() {
  const name = $("#f-name").value.trim();
  const days = parseInt($("#f-days").value, 10) || 30;
  const protocol = $("#f-protocol").value;

  if (!name) return showModalError("Name is required");

  const params = collectParams();
  const err = $("#modal-error");
  err.classList.add("hidden");
  $("#btn-next").disabled = true;
  $("#btn-next").textContent = "Creating…";

  try {
    const c = await api("/clients", {
      method: "POST",
      body: JSON.stringify({ name, protocol, days, params }),
    });
    closeModal();
    showResult(c.config, c.uri);
    if (location.hash.includes("clients")) loadClients();
  } catch (e) {
    showModalError(e.message);
  } finally {
    $("#btn-next").disabled = false;
    $("#btn-next").textContent = "Create";
  }
}

function showModalError(msg) {
  const err = $("#modal-error");
  err.textContent = msg;
  err.classList.remove("hidden");
}

// --- result modal ----------------------------------------------------------

function showResult(config, uri) {
  $("#result-config").textContent = config || "";
  if (uri) {
    $("#result-uri-wrap").classList.remove("hidden");
    $("#result-uri").value = uri;
  } else {
    $("#result-uri-wrap").classList.add("hidden");
  }
  $("#result-modal").classList.remove("hidden");
}

// --- wiring ----------------------------------------------------------------

async function init() {
  try {
    state.protocols = await api("/protocols");
  } catch (e) {
    console.error("protocols:", e);
  }

  $("#btn-add").addEventListener("click", openModal);
  $("#modal-close").addEventListener("click", closeModal);
  $("#result-close").addEventListener("click", () => $("#result-modal").classList.add("hidden"));
  $("#result-done").addEventListener("click", () => $("#result-modal").classList.add("hidden"));

  $("#btn-next").addEventListener("click", () => {
    if (state.step === 1) {
      // move to params step
      state.step = 2;
      $("#step-1").classList.add("hidden");
      $("#step-2").classList.remove("hidden");
      renderParams();
      $("#btn-next").textContent = "Create";
      $("#btn-back").style.visibility = "visible";
    } else {
      submitClient();
    }
  });

  $("#btn-back").addEventListener("click", () => {
    state.step = 1;
    $("#step-2").classList.add("hidden");
    $("#step-1").classList.remove("hidden");
    $("#btn-next").textContent = "Next";
    $("#btn-back").style.visibility = "hidden";
  });

  route();
  setInterval(() => {
    if (location.hash.includes("dashboard") || !location.hash) loadStats();
  }, 5000);
}

document.addEventListener("DOMContentLoaded", init);
