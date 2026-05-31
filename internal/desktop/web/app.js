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
  state.systemProxy = data.system_proxy;
  state.proxyCapabilities = data.proxy_capabilities || [];
  render();
}

function render() {
  const cfg = state.config || {};
  const status = state.status || {};
  const systemProxy = state.systemProxy || {};
  const metrics = status.metrics || {};
  const profiles = cfg.profiles || [];
  const proxyProfiles = cfg.proxy_profiles || [];
  const capabilityMap = new Map((state.proxyCapabilities || []).map((item) => [item.name, item]));
  const selectedProfile = cfg.active_profile || "";
  const selectedProxy = cfg.active_proxy_profile || "";
  const activeProxy = proxyProfiles.find((profile) => profile.name === selectedProxy)
    || (!selectedProfile && proxyProfiles.length ? proxyProfiles[0] : null);
  const nodeLabel = activeProxy ? activeProxy.name : selectedProfile || (profiles.length ? profiles[0].name : "");
  const relayAddr = activeProxy ? `${activeProxy.protocol || "-"} 机场节点` : status.relay_addr || cfg.relay_addr;
  const localAddr = status.local_addr || cfg.local_addr;
  const httpAddr = status.http_addr || cfg.http_addr;

  renderConfigForm(cfg);
  $("configPath").textContent = text(state.configPath, "配置未加载");
  $("relayAddr").textContent = text(relayAddr);
  $("localAddr").textContent = text(localAddr);
  $("httpAddr").textContent = text(httpAddr);
  $("systemProxy").textContent = systemProxy.supported
    ? (systemProxy.enabled ? "已开启" : "未开启")
    : text(systemProxy.message, "不支持");
  $("activeProfile").textContent = text(nodeLabel, "未选择");
  $("authState").textContent = cfg.local_auth && cfg.local_auth.enabled ? "已启用" : "未启用";
  $("tlsState").textContent = cfg.tls && cfg.tls.enabled ? "已启用" : "未启用";
  $("activeConnections").textContent = text(metrics.active_connections, "0");
  $("totalConnections").textContent = text(metrics.total_connections, "0");
  $("traffic").textContent = `${bytes(metrics.upload_bytes)} / ${bytes(metrics.download_bytes)}`;

  const badge = $("statusBadge");
  badge.textContent = status.running ? "已连接" : "未连接";
  badge.className = status.running ? "badge running" : "badge";
  $("connectionTitle").textContent = status.running ? "已连接" : "未连接";
  $("connectionSummary").textContent = nodeLabel ? `${nodeLabel} · ${text(relayAddr)}` : "未选择节点";
  $("connectBtn").textContent = status.running ? "断开" : "连接";
  $("connectBtn").className = status.running ? "primary-action danger-action" : "primary-action";

  renderProfiles(profiles, selectedProfile);
  renderProxyProfiles(proxyProfiles, activeProxy ? activeProxy.name : "", capabilityMap);
  renderSubscriptions(cfg.subscriptions || []);
}

function renderConfigForm(cfg) {
  const auth = cfg.local_auth || {};
  const tls = cfg.tls || {};
  $("configLocal").value = cfg.local_addr || "";
  $("configHTTP").value = cfg.http_addr || "";
  $("configRelay").value = cfg.relay_addr || "";
  $("configToken").value = cfg.token || "";
  $("configTimeout").value = cfg.dial_timeout_seconds || 10;
  $("configAuthEnabled").checked = Boolean(auth.enabled);
  $("configAuthUser").value = auth.username || "";
  $("configAuthPass").value = auth.password || "";
  $("configTLSEnabled").checked = Boolean(tls.enabled);
  $("configTLSServerName").value = tls.server_name || "";
  $("configTLSCAFile").value = tls.ca_file || "";
  $("configTLSInsecure").checked = Boolean(tls.insecure_skip_verify);
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
    const edit = button("编辑", "secondary", async () => {
      fillProfileForm(profile);
    });
    const select = button("选择", "secondary", async () => {
      await api("/api/profile/select", { method: "POST", body: { name: profile.name } });
      setMessage(`已选择 ${profile.name}`, "ok");
      await refresh();
    });
    const check = button("检测", "", async () => {
      const result = await api("/api/profile/check", { method: "POST", body: { name: profile.name } });
      setMessage(result.message || `${profile.name} 可连接`, "ok");
    });
    actions.append(edit, select, check);
    item.append(info, actions);
    root.append(item);
  });
}

function renderProxyProfiles(profiles, active, capabilityMap) {
  const root = $("proxyProfiles");
  root.innerHTML = "";
  if (!profiles.length) {
    root.append(emptyItem("没有机场节点"));
    return;
  }
  profiles.forEach((profile) => {
    const capability = capabilityMap.get(profile.name) || {};
    const exportable = capability.exportable !== false;
    const item = document.createElement("div");
    item.className = "item";
    const info = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = profile.name === active ? `${profile.name} · 当前` : profile.name;
    const addr = document.createElement("span");
    addr.textContent = `${(profile.protocol || "-").toUpperCase()} · ${exportable ? "可连接" : "暂不支持直接连接"}`;
    addr.className = exportable ? "" : "warn-text";
    info.append(title, addr);

    const actions = document.createElement("div");
    actions.className = "item-actions";
    const select = button("选择", "secondary", async () => {
      await api("/api/proxy/select", { method: "POST", body: { name: profile.name } });
      setMessage(`已选择 ${profile.name}`, "ok");
      await refresh();
    });
    actions.append(select);
    item.append(info, actions);
    root.append(item);
  });
}

function fillProfileForm(profile) {
  const tls = profile.tls || {};
  $("profileName").value = profile.name || "";
  $("profileRelay").value = profile.relay_addr || "";
  $("profileToken").value = profile.token || "";
  $("profileTLSEnabled").checked = Boolean(tls.enabled);
  $("profileTLSServerName").value = tls.server_name || "";
  $("profileTLSCAFile").value = tls.ca_file || "";
  $("profileTLSInsecure").checked = Boolean(tls.insecure_skip_verify);
}

function clearProfileForm() {
  fillProfileForm({ tls: {} });
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
  $("advancedToggleBtn").addEventListener("click", () => {
    const panel = $("advancedPanel");
    const opening = panel.hasAttribute("hidden");
    if (opening) {
      panel.removeAttribute("hidden");
    } else {
      panel.setAttribute("hidden", "");
    }
    $("advancedToggleBtn").textContent = opening ? "收起高级" : "高级设置";
  });
  $("configSaveBtn").addEventListener("click", () => runAction(async () => {
    const cfg = buildConfigFromForm();
    const result = await api("/api/config", { method: "POST", body: cfg });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("profileSaveBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/profile", {
      method: "POST",
      body: {
        name: $("profileName").value,
        relay_addr: $("profileRelay").value,
        token: $("profileToken").value,
        replace: true,
        tls: {
          enabled: $("profileTLSEnabled").checked,
          server_name: $("profileTLSServerName").value,
          ca_file: $("profileTLSCAFile").value,
          insecure_skip_verify: $("profileTLSInsecure").checked,
        },
      },
    });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("profileDeleteBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/profile/delete", {
      method: "POST",
      body: { name: $("profileName").value },
    });
    clearProfileForm();
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("profileClearBtn").addEventListener("click", () => {
    clearProfileForm();
    setMessage("");
  });
  $("connectBtn").addEventListener("click", () => runAction(async () => {
    const status = state.status || {};
    const path = status.running ? "/api/stop" : "/api/start";
    const result = await api(path, { method: "POST", body: {} });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("checkBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/check", { method: "POST", body: {} });
    setMessage(result.message, "ok");
  }));
  $("systemProxyOnBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/system-proxy/enable", { method: "POST", body: {} });
    setMessage(result.message, "ok");
    await refresh();
  }));
  $("systemProxyOffBtn").addEventListener("click", () => runAction(async () => {
    const result = await api("/api/system-proxy/disable", { method: "POST", body: {} });
    setMessage(result.message, "ok");
    await refresh();
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
  $("loginBtn").addEventListener("click", () => {
    setMessage("账号服务尚未接入，当前可以先导入机场节点连接。");
  });
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

function buildConfigFromForm() {
  const cfg = JSON.parse(JSON.stringify(state.config || {}));
  cfg.local_addr = $("configLocal").value;
  cfg.http_addr = $("configHTTP").value;
  cfg.relay_addr = $("configRelay").value;
  cfg.token = $("configToken").value;
  cfg.dial_timeout_seconds = Number.parseInt($("configTimeout").value || "0", 10);
  cfg.local_auth = {
    enabled: $("configAuthEnabled").checked,
    username: $("configAuthUser").value,
    password: $("configAuthPass").value,
  };
  cfg.tls = {
    enabled: $("configTLSEnabled").checked,
    server_name: $("configTLSServerName").value,
    ca_file: $("configTLSCAFile").value,
    insecure_skip_verify: $("configTLSInsecure").checked,
  };
  cfg.profiles = cfg.profiles || [];
  cfg.proxy_profiles = cfg.proxy_profiles || [];
  cfg.subscriptions = cfg.subscriptions || [];
  return cfg;
}

bind();
refresh().catch((err) => setMessage(err.message, "error"));
