const state = {
  configPath: "",
  config: null,
  status: null,
};

const $ = (id) => document.getElementById(id);

async function api(path, options = {}) {
  const init = {
    method: options.method || "GET",
    headers: { "Content-Type": "application/json" },
  };
  if (options.body !== undefined) {
    init.body = JSON.stringify(options.body);
  }
  const response = await fetch(path, init);
  const data = await response.json();
  if (!response.ok || data.ok === false) {
    throw new Error(data.message || `HTTP ${response.status}`);
  }
  return data;
}

function setMessage(message, kind = "") {
  const box = $("message");
  box.textContent = message || "";
  box.className = `message ${kind}`.trim();
}

function text(value, fallback = "-") {
  if (value === undefined || value === null || value === "") return fallback;
  return String(value);
}

function bytes(value) {
  const n = Number(value || 0);
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

async function refresh() {
  const data = await api("/api/state");
  state.configPath = data.config_path;
  state.config = data.config;
  state.status = data.status;
  render();
}

function render() {
  const cfg = state.config || {};
  const status = state.status || {};
  const metrics = status.metrics || {};

  $("configPath").textContent = text(state.configPath, "配置未加载");
  $("relayAddr").textContent = text(cfg.relay_addr);
  $("localAddr").textContent = text(cfg.local_addr);
  $("httpAddr").textContent = text(cfg.http_addr);
  $("activeProfile").textContent = text(cfg.active_profile);
  $("authState").textContent = cfg.local_auth && cfg.local_auth.enabled ? "已启用" : "未启用";
  $("tlsState").textContent = cfg.tls && cfg.tls.enabled ? "已启用" : "未启用";
  $("activeConnections").textContent = text(metrics.active_connections, "0");
  $("totalConnections").textContent = text(metrics.total_connections, "0");
  $("traffic").textContent = `${bytes(metrics.upload_bytes)} / ${bytes(metrics.download_bytes)}`;

  const badge = $("statusBadge");
  badge.textContent = status.running ? "运行中" : "已停止";
  badge.className = status.running ? "badge running" : "badge";

  renderProfiles(cfg.profiles || [], cfg.active_profile || "");
  renderSubscriptions(cfg.subscriptions || []);
}

function renderProfiles(profiles, active) {
  const root = $("profiles");
  root.innerHTML = "";
  if (!profiles.length) {
    root.append(emptyItem("没有 relay profile"));
    return;
  }
  profiles.forEach((profile) => {
    const item = document.createElement("div");
    item.className = "item";
    const info = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = profile.name === active ? `${profile.name} · 当前` : profile.name;
    const addr = document.createElement("span");
    addr.textContent = `${profile.relay_addr || "-"} · TLS ${profile.tls && profile.tls.enabled ? "开" : "关"}`;
    info.append(title, addr);

    const actions = document.createElement("div");
    actions.className = "item-actions";
    const select = button("选择", "secondary", async () => {
      await api("/api/profile/select", { method: "POST", body: { name: profile.name } });
      setMessage(`已选择 ${profile.name}`, "ok");
      await refresh();
    });
    const check = button("检测", "", async () => {
      const result = await api("/api/profile/check", { method: "POST", body: { name: profile.name } });
      setMessage(result.message || `${profile.name} 可连接`, "ok");
    });
    actions.append(select, check);
    item.append(info, actions);
    root.append(item);
  });
}

function renderSubscriptions(subscriptions) {
  const root = $("subscriptions");
  root.innerHTML = "";
  if (!subscriptions.length) {
    root.append(emptyItem("没有 relay 订阅"));
    return;
  }
  subscriptions.forEach((sub) => {
    const item = document.createElement("button");
    item.type = "button";
    item.className = "item secondary";
    item.addEventListener("click", () => {
      $("subName").value = sub.name || "";
      $("subURL").value = sub.url === "******" ? "" : sub.url || "";
    });
    const info = document.createElement("span");
    info.innerHTML = `<strong>${escapeHTML(sub.name || "-")}</strong><span>${escapeHTML(sub.url || "-")}</span>`;
    item.append(info);
    root.append(item);
  });
}

function emptyItem(label) {
  const item = document.createElement("div");
  item.className = "item";
  const textNode = document.createElement("span");
  textNode.textContent = label;
  item.append(textNode);
  return item;
}

function button(label, className, handler) {
  const btn = document.createElement("button");
  btn.type = "button";
  btn.textContent = label;
  btn.className = className || "";
  btn.addEventListener("click", () => runAction(handler));
  return btn;
}

async function runAction(fn) {
  try {
    setMessage("");
    await fn();
  } catch (err) {
    setMessage(err.message, "error");
  }
}

function escapeHTML(value) {
  return String(value).replace(/[&<>"']/g, (ch) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#39;",
  })[ch]);
}

function bind() {
  $("startBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/start", { method: "POST", body: {} });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("stopBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/stop", { method: "POST", body: {} });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("checkBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/check", { method: "POST", body: {} });
    setMessage(result.message, "ok");
  }));
  $("importBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/profiles/import", {
      method: "POST",
      body: {
        content: $("importContent").value,
        replace: $("importReplace").checked,
        select: $("importSelect").value,
      },
    });
    setMessage(`${result.message}：${result.count}`, "ok");
    await refresh();
  }));
  $("subSaveBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/subscription", {
      method: "POST",
      body: {
        name: $("subName").value,
        url: $("subURL").value,
        replace: true,
      },
    });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("subSyncBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/subscription/sync", {
      method: "POST",
      body: {
        name: $("subName").value,
        replace: $("subReplace").checked,
      },
    });
    setMessage(`${result.message}：${result.count}`, "ok");
    await refresh();
  }));
  $("subDeleteBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/subscription/delete", {
      method: "POST",
      body: { name: $("subName").value },
    });
    setMessage(result.message, "ok");
    await refresh();
  }));
}

bind();
refresh().catch((err) => setMessage(err.message, "error"));
