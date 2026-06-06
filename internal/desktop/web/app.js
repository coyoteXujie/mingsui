const state = {
  configPath: "",
  config: null,
  status: null,
  systemProxy: null,
  proxyCapabilities: [],
  proxyCheckResults: {},
  logs: [],
};
let autoRefreshTimer = null;
let autoRefreshInFlight = false;

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
    const error = new Error(data.message || `HTTP ${response.status}`);
    error.data = data;
    throw error;
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

function rememberProxyCheckReport(report) {
  if (!report || !Array.isArray(report.results)) return;
  report.results.forEach((result) => {
    if (result && result.name) {
      state.proxyCheckResults[result.name] = result;
    }
  });
}

function proxyCheckStatus(result) {
  if (!result) return null;
  if (!result.tested) {
    return { text: result.skip_reason || "未检测", kind: "warn" };
  }
  if (result.ok) {
    return { text: `正常 · ${result.latency_ms || 0} ms`, kind: "ok" };
  }
  return { text: `失败 · ${result.error || "连接失败"}`, kind: "error" };
}

async function runProxyCheck(body, options = {}) {
  try {
    const result = await api("/api/proxy/check", { method: "POST", body });
    rememberProxyCheckReport(result.proxy_check);
    if (options.refreshAfter) {
      await refresh();
    } else {
      render();
    }
    return result;
  } catch (err) {
    if (err.data && err.data.proxy_check) {
      rememberProxyCheckReport(err.data.proxy_check);
      render();
    }
    throw err;
  }
}

async function refresh() {
  const [data, logData] = await Promise.all([
    api("/api/state"),
    api("/api/logs").catch(() => ({ logs: [] })),
  ]);
  state.configPath = data.config_path;
  state.config = data.config;
  state.status = data.status;
  state.systemProxy = data.system_proxy;
  state.proxyCapabilities = data.proxy_capabilities || [];
  state.logs = logData.logs || [];
  render();
}

async function refreshLogs() {
  const data = await api("/api/logs");
  state.logs = data.logs || [];
  renderLogs(state.logs);
}

async function refreshRuntime() {
  if (!state.config) {
    await refresh();
    return;
  }
  const [data, logData] = await Promise.all([
    api("/api/state"),
    api("/api/logs").catch(() => ({ logs: [] })),
  ]);
  state.status = data.status;
  state.systemProxy = data.system_proxy;
  state.proxyCapabilities = data.proxy_capabilities || state.proxyCapabilities || [];
  state.logs = logData.logs || [];
  renderRuntime();
}

function startAutoRefresh() {
  if (autoRefreshTimer) return;
  autoRefreshTimer = window.setInterval(async () => {
    if (document.hidden || autoRefreshInFlight) return;
    autoRefreshInFlight = true;
    try {
      await refreshRuntime();
    } catch (_err) {
      // Keep transient refresh failures quiet; explicit actions still surface errors.
    } finally {
      autoRefreshInFlight = false;
    }
  }, 3000);
}

function render() {
  renderConfigForm(state.config || {});
  renderRuntime();
  const cfg = state.config || {};
  const profiles = cfg.profiles || [];
  const proxyProfiles = cfg.proxy_profiles || [];
  const selectedProfile = cfg.active_profile || "";
  const activeProxyName = currentActiveProxyName();
  const capabilityMap = new Map((state.proxyCapabilities || []).map((item) => [item.name, item]));
  renderProfiles(profiles, selectedProfile);
  renderProxyProfiles(proxyProfiles, activeProxyName, capabilityMap, state.proxyCheckResults || {});
  renderSubscriptions(cfg.subscriptions || []);
}

function renderRuntime() {
  const cfg = state.config || {};
  const status = state.status || {};
  const systemProxy = state.systemProxy || {};
  const metrics = status.metrics || {};
  const activeProxy = currentActiveProxy();
  const selectedProfile = cfg.active_profile || "";
  const profiles = cfg.profiles || [];
  const proxyProfiles = cfg.proxy_profiles || [];
  const proxyModeWithoutExportable = !activeProxy && !selectedProfile && proxyProfiles.length > 0;
  const nodeLabel = activeProxy ? activeProxy.name : (proxyModeWithoutExportable ? "没有可自动选择的国外节点" : selectedProfile || (profiles.length ? profiles[0].name : ""));
  const relayAddr = activeProxy ? `${activeProxy.protocol || "-"} 机场节点` : (proxyModeWithoutExportable ? "" : status.relay_addr || cfg.relay_addr);
  const localAddr = status.local_addr || cfg.local_addr;
  const httpAddr = status.http_addr || cfg.http_addr;

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
  $("connectionSummary").textContent = proxyModeWithoutExportable ? "当前订阅里没有可自动选择的国外节点" : (nodeLabel ? `${nodeLabel} · ${text(relayAddr)}` : "未选择节点");
  $("connectBtn").textContent = status.running ? "断开" : "连接";
  $("connectBtn").className = status.running ? "primary-action danger-action" : "primary-action";

  renderLogs(state.logs || []);
}

function currentActiveProxy() {
  const cfg = state.config || {};
  const proxyProfiles = cfg.proxy_profiles || [];
  const selectedProfile = cfg.active_profile || "";
  const selectedProxy = cfg.active_proxy_profile || "";
  const capabilityMap = new Map((state.proxyCapabilities || []).map((item) => [item.name, item]));
  const firstAutoSelectableProxy = proxyProfiles.find((profile) => {
    const capability = capabilityMap.get(profile.name) || {};
    return capability.exportable !== false && capability.auto_selectable !== false;
  });
  return proxyProfiles.find((profile) => profile.name === selectedProxy)
    || (!selectedProfile ? firstAutoSelectableProxy || null : null);
}

function currentActiveProxyName() {
  const proxy = currentActiveProxy();
  return proxy ? proxy.name : "";
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

function renderProxyProfiles(profiles, active, capabilityMap, checkResults) {
  const root = $("proxyProfiles");
  root.innerHTML = "";
  if (!profiles.length) {
    root.append(emptyItem("没有机场节点"));
    return;
  }
  profiles.forEach((profile) => {
    const capability = capabilityMap.get(profile.name) || {};
    const exportable = capability.exportable !== false;
    const autoSelectable = capability.auto_selectable !== false;
    const item = document.createElement("div");
    item.className = "item";
    const info = document.createElement("div");
    const title = document.createElement("strong");
    title.textContent = profile.name === active ? `${profile.name} · 当前` : profile.name;
    const addr = document.createElement("span");
    let compatibility = "可连接";
    if (!exportable) {
      compatibility = "暂不支持直接连接";
    } else if (!autoSelectable) {
      compatibility = "可连接，国内节点不自动选择";
    }
    addr.textContent = `${(profile.protocol || "-").toUpperCase()} · ${compatibility}`;
    addr.className = exportable && autoSelectable ? "" : "warn-text";
    info.append(title, addr);
    const status = proxyCheckStatus(checkResults[profile.name]);
    if (status) {
      const statusLine = document.createElement("span");
      statusLine.textContent = status.text;
      statusLine.className = `result-status ${status.kind}`;
      info.append(statusLine);
    }

    const actions = document.createElement("div");
    actions.className = "item-actions";
    const select = button(exportable ? "选择" : "不可连接", "secondary", async () => {
      await api("/api/proxy/select", { method: "POST", body: { name: profile.name } });
      setMessage(`已选择 ${profile.name}`, "ok");
      await refresh();
    });
    select.disabled = !exportable;
    const check = button("检测", "", async () => {
      const result = await runProxyCheck({ name: profile.name, timeout_seconds: 10 });
      setMessage(result.message, "ok");
    });
    check.disabled = !exportable;
    const remove = button("删除", "danger", async () => {
      if (!window.confirm(`删除机场节点 ${profile.name}？`)) return;
      const result = await api("/api/proxy/delete", { method: "POST", body: { name: profile.name } });
      delete state.proxyCheckResults[profile.name];
      setMessage(result.message, "ok");
      await refresh();
    });
    actions.append(select, check, remove);
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
    root.append(emptyItem("没有节点订阅"));
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

function renderLogs(logs) {
  const root = $("logs");
  if (!root) return;
  root.textContent = logs.length ? logs.join("\n") : "暂无日志";
  root.scrollTop = root.scrollHeight;
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
    await refreshLogs().catch(() => {});
  } catch (err) {
    setMessage(err.message, "error");
    await refreshLogs().catch(() => {});
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

function setActiveView(viewID) {
  document.querySelectorAll(".view").forEach((view) => {
    view.classList.toggle("active", view.id === viewID);
  });
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.classList.toggle("active", item.dataset.view === viewID);
  });
}

function bind() {
  document.querySelectorAll(".nav-item").forEach((item) => {
    item.addEventListener("click", () => {
      setActiveView(item.dataset.view);
    });
  });
  $("refreshLogsBtn").addEventListener("click", () => runAction(async () => {
    await refreshLogs();
  }));
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
  $("bestProxyBtn").addEventListener("click", () => runAction(async () => {
    const result = await runProxyCheck({ select_best: true, timeout_seconds: 10 }, { refreshAfter: true });
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
    state.proxyCheckResults = {};
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
    state.proxyCheckResults = {};
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
refresh()
  .then(startAutoRefresh)
  .catch((err) => setMessage(err.message, "error"));
