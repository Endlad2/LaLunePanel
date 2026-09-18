// LaLune Panel frontend — vanilla SPA, no build step.

const state = {
  protocols: [],
  selectedProtocol: null,
  selectedRole: "srv",
  step: 1,
};

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
  if (!res.ok) throw new Error((data && data.error) || `HTTP ${res.status}`);
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
  return Math.ceil((new Date(iso) - new Date()) / 86400000);
}

function escapeHtml(s) {
  return String(s ?? "").replace(/[&<>"']/g, (m) => ({
    "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
  }[m]));
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
    $("#stat-mem").textContent = fmtBytes(s.host?.mem_used_kb || 0) + " / " + fmtBytes(s.host?.mem_total_kb || 0);
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
      const role = c.role === "cnc" ? "cnc" : "srv";
      const roleBadge = role === "cnc"
        ? `<span class="badge role-cnc">cnc</span>`
        : `<span class="badge role-srv">srv</span>`;
      const tr = document.createElement("tr");
      tr.innerHTML = `
        <td>${escapeHtml(c.name)}</td>
        <td>${roleBadge}</td>
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
    tbody.innerHTML = `<tr><td colspan="6" class="muted">${escapeHtml(e.message)}</td></tr>`;
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
    showResult(c);
  } catch (e) {
    alert("Load failed: " + e.message);
  }
}

// --- add-client modal ------------------------------------------------------

function openModal() {
  state.step = 1;
  state.selectedProtocol = null;
  state.selectedRole = "srv";
  $("#f-name").value = "";
  $("#f-days").value = "30";
  document.querySelector('input[name="role"][value="srv"]').checked = true;
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
  // Hide the role radio for protocols without roles (OpenFlux).
  const roleField = $("#role-field");
  if (state.selectedProtocol?.has_roles) {
    roleField.classList.remove("hidden");
  } else {
    roleField.classList.add("hidden");
  }
}

function getSelectedRole() {
  if (state.selectedProtocol && state.selectedProtocol.has_roles === false) {
    return "srv";
  }
  const el = document.querySelector('input[name="role"]:checked');
  return el ? el.value : "srv";
}

function fieldsFor(proto, role) {
  const common = proto.common_fields || [];
  if (!proto.has_roles) return common;
  const specific = role === "cnc" ? (proto.fields_client || []) : (proto.fields_server || []);
  return [...common, ...specific];
}

// showIf evaluation mirrors the backend sentinel "__nonempty__".
function showIfSatisfied(cond, params) {
  if (!cond) return true;
  const v = params[cond.key] || "";
  if (cond.value === "__nonempty__") return v.trim() !== "";
  return v === cond.value;
}

function renderParams() {
  const p = state.selectedProtocol;
  const role = getSelectedRole();
  const box = $("#params-container");
  box.innerHTML = "";

  for (const f of fieldsFor(p, role)) {
    const label = document.createElement("label");
    label.className = "field";
    label.dataset.fieldKey = f.key;
    if (f.show_if) {
      label.dataset.showIfKey = f.show_if.key;
      label.dataset.showIfValue = f.show_if.value;
    }

    let input;
    if (f.type === "select" || f.type === "bool") {
      input = document.createElement("select");
      const opts = f.type === "bool" ? ["false", "true"] : (f.options || []);
      for (const o of opts) {
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
    input.addEventListener("input", refreshConditionalFields);
    input.addEventListener("change", refreshConditionalFields);

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

  refreshConditionalFields();
}

function currentParams() {
  const params = {};
  $("#params-container").querySelectorAll("[data-key]").forEach((el) => {
    params[el.dataset.key] = el.value;
  });
  return params;
}

function refreshConditionalFields() {
  const params = currentParams();
  $("#params-container").querySelectorAll("[data-field-key]").forEach((label) => {
    const key = label.dataset.showIfKey;
    if (!key) return;
    const cond = { key, value: label.dataset.showIfValue };
    if (showIfSatisfied(cond, params)) {
      label.classList.remove("hidden");
    } else {
      label.classList.add("hidden");
    }
  });
}

function collectParams() {
  const params = {};
  $("#params-container").querySelectorAll("[data-key]").forEach((el) => {
    // Skip values of hidden conditional fields to avoid stale input.
    const label = el.closest("[data-field-key]");
    if (label && label.classList.contains("hidden")) return;
    params[el.dataset.key] = el.value;
  });
  return params;
}

async function submitClient() {
  const name = $("#f-name").value.trim();
  const days = parseInt($("#f-days").value, 10) || 30;
  const protocol = $("#f-protocol").value;
  const role = getSelectedRole();

  if (!name) return showModalError("Name is required");

  const params = collectParams();
  const err = $("#modal-error");
  err.classList.add("hidden");
  $("#btn-next").disabled = true;
  $("#btn-next").textContent = "Creating…";

  try {
    const c = await api("/clients", {
      method: "POST",
      body: JSON.stringify({ name, protocol, role, days, params }),
    });
    closeModal();
    showResult(c);
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

function showResult(c) {
  const isClient = c.role === "cnc";
  $("#result-title").textContent = isClient ? "Client (cnc) created" : "Server (srv) created";
  $("#result-hint").textContent = isClient
    ? "Run this config on the client machine. Point your browser at the SOCKS5 address below."
    : "Give this URI to the client side (or scan the QR from the olcbox app).";
  $("#result-config").textContent = c.config || "";

  if (c.uri) {
    $("#result-uri-wrap").classList.remove("hidden");
    $("#result-uri").value = c.uri;
  } else {
    $("#result-uri-wrap").classList.add("hidden");
  }

  if (c.socks_addr) {
    $("#result-socks-wrap").classList.remove("hidden");
    $("#result-socks").value = c.socks_addr;
  } else {
    $("#result-socks-wrap").classList.add("hidden");
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

  $$('input[name="role"]').forEach((el) =>
    el.addEventListener("change", () => { state.selectedRole = getSelectedRole(); }));

  $("#btn-next").addEventListener("click", () => {
    if (state.step === 1) {
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
