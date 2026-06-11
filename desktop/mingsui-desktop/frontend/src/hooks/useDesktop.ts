import {createContext, createElement, useCallback, useContext, useEffect, useState} from 'react'
import type {ReactNode} from 'react'

export interface RuntimeStatus {
  running: boolean
  local_addr: string
  http_addr: string
  relay_addr: string
  started_at: string
  last_error: string
  metrics?: {
    active_connections: number
    total_connections: number
    upload_bytes: number
    download_bytes: number
  }
}

export interface SystemProxyStatus {
  supported: boolean
  enabled: boolean
  message: string
}

export interface ReadinessAction {
  id: string
  label: string
  command?: string
  description?: string
  severity: 'info' | 'warning' | 'error'
}

export interface ReadinessStatus {
  ok: boolean
  config_path?: string
  mode: string
  readiness: string
  managed: boolean
  selected_type?: string
  selected_profile?: string
  selected_proxy?: string
  proxy_protocol?: string
  relay_profiles: number
  proxy_profiles: number
  subscriptions: number
  local_addr?: string
  http_addr?: string
  relay_addr?: string
  auth_enabled: boolean
  tls_enabled: boolean
  default_token?: boolean
  local_proxy_exposed?: boolean
  message: string
  warnings?: string[]
  actions?: ReadinessAction[]
}

export interface ProxyCapability {
  name: string
  exportable: boolean
  auto_selectable: boolean
}

export interface ClientConfig {
  local_addr: string
  http_addr: string
  relay_addr: string
  token: string
  dial_timeout_seconds: number
  local_auth: {
    enabled: boolean
    username: string
    password: string
  }
  tls: {
    enabled: boolean
    server_name: string
    ca_file: string
    insecure_skip_verify: boolean
  }
  profiles: RelayProfile[]
  proxy_profiles: ProxyProfile[]
  subscriptions: Subscription[]
  active_profile: string
  active_proxy_profile: string
}

export interface RelayProfile {
  name: string
  relay_addr: string
  token: string
  tls: {
    enabled: boolean
    server_name: string
    ca_file: string
    insecure_skip_verify: boolean
  }
}

export interface ProxyProfile {
  name: string
  protocol: string
  url: string
}

export interface Subscription {
  name: string
  url: string
}

export interface SubscriptionSyncResult {
  name: string
  kind: 'relay' | 'proxy' | string
  imported: number
  relay_profiles: number
  proxy_profiles: number
  exportable_proxy_profiles: number
  auto_selectable_proxy_profiles: number
  imported_exportable_proxy_profiles?: number
  imported_auto_selectable_proxy_profiles?: number
  active_profile?: string
  active_proxy_profile?: string
  selected?: string
  warnings?: string[]
  message: string
}

export interface AppState {
  config_path: string
  config: ClientConfig
  status: RuntimeStatus
  system_proxy: SystemProxyStatus
  proxy_capabilities: ProxyCapability[]
  readiness?: ReadinessStatus
}

export interface RuntimeState {
  status: RuntimeStatus
  system_proxy: SystemProxyStatus
  readiness?: ReadinessStatus
}

const defaultMetrics = {
  active_connections: 0,
  total_connections: 0,
  upload_bytes: 0,
  download_bytes: 0,
}

const defaultLocalAuth = {
  enabled: false,
  username: '',
  password: '',
}

const defaultTLS = {
  enabled: false,
  server_name: '',
  ca_file: '',
  insecure_skip_verify: false,
}

const defaultConfig: ClientConfig = {
  local_addr: '',
  http_addr: '',
  relay_addr: '',
  token: '',
  dial_timeout_seconds: 10,
  local_auth: defaultLocalAuth,
  tls: defaultTLS,
  profiles: [],
  proxy_profiles: [],
  subscriptions: [],
  active_profile: '',
  active_proxy_profile: '',
}

const defaultStatus: RuntimeStatus = {
  running: false,
  local_addr: '',
  http_addr: '',
  relay_addr: '',
  started_at: '',
  last_error: '',
  metrics: defaultMetrics,
}

const defaultSystemProxy: SystemProxyStatus = {
  supported: false,
  enabled: false,
  message: '',
}

function normalizeConfig(config?: Partial<ClientConfig> | null): ClientConfig {
  const profiles = config?.profiles
  const proxyProfiles = config?.proxy_profiles
  const subscriptions = config?.subscriptions

  return {
    ...defaultConfig,
    ...config,
    local_auth: {
      ...defaultLocalAuth,
      ...(config?.local_auth || {}),
    },
    tls: {
      ...defaultTLS,
      ...(config?.tls || {}),
    },
    profiles: Array.isArray(profiles) ? profiles : [],
    proxy_profiles: Array.isArray(proxyProfiles) ? proxyProfiles : [],
    subscriptions: Array.isArray(subscriptions) ? subscriptions : [],
  }
}

function normalizeStatus(status?: Partial<RuntimeStatus> | null): RuntimeStatus {
  return {
    ...defaultStatus,
    ...status,
    metrics: {
      ...defaultMetrics,
      ...(status?.metrics || {}),
    },
  }
}

function normalizeAppState(state: Partial<AppState>): AppState {
  return {
    config_path: state.config_path || '',
    config: normalizeConfig(state.config),
    status: normalizeStatus(state.status),
    system_proxy: {
      ...defaultSystemProxy,
      ...(state.system_proxy || {}),
    },
    proxy_capabilities: Array.isArray(state.proxy_capabilities) ? state.proxy_capabilities : [],
    readiness: state.readiness,
  }
}

let cachedState: AppState | null = null
let cachedStateSignature = ''
const runtimePollIntervalMS = 5000
const fullSyncIntervalMS = 60000

function stateSignature(state: AppState) {
  return JSON.stringify(state)
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetState: () => Promise<AppState>
          GetRuntimeState: () => Promise<RuntimeState>
          Start: () => Promise<string>
          Stop: () => Promise<string>
          GetConfig: () => Promise<ClientConfig>
          SaveConfig: (cfg: ClientConfig) => Promise<string>
          ImportProfiles: (content: string, replace: boolean, selectName: string) => Promise<SubscriptionSyncResult>
          SelectProxy: (name: string) => Promise<string>
          DeleteProxy: (name: string) => Promise<string>
          CheckProxy: (name: string, timeoutSeconds: number) => Promise<any>
          CheckBestProxy: (timeoutSeconds: number) => Promise<any>
          EnableSystemProxy: () => Promise<string>
          DisableSystemProxy: () => Promise<string>
          GetSystemProxyStatus: () => Promise<SystemProxyStatus>
          SaveRelayProfile: (req: {name: string; relay_addr: string; token: string; tls: any; replace: boolean}) => Promise<string>
          DeleteRelayProfile: (name: string) => Promise<string>
          SelectRelayProfile: (name: string) => Promise<string>
          CheckRelayProfile: (name: string) => Promise<any>
          SaveSubscription: (req: {name: string; url: string; replace: boolean}) => Promise<string>
          DeleteSubscription: (name: string) => Promise<string>
          SyncSubscription: (name: string, replace: boolean) => Promise<SubscriptionSyncResult>
          GetLogs: () => Promise<string[]>
        }
      }
    }
  }
}

function desktopAPI() {
  const app = window.go?.main?.App
  if (!app) {
    throw new Error('桌面后端未连接。请通过 Wails 桌面端启动明隧，不要直接打开 Vite 或浏览器预览页。')
  }
  return app
}

function useDesktopStore() {
  const [state, setState] = useState<AppState | null>(cachedState)
  const [loading, setLoading] = useState(!cachedState)
  const [error, setError] = useState<string | null>(null)

  const applyState = useCallback((nextState: AppState) => {
    const nextSignature = stateSignature(nextState)
    cachedState = nextState
    if (nextSignature !== cachedStateSignature) {
      cachedStateSignature = nextSignature
      setState(nextState)
    }
  }, [])

  const refresh = useCallback(async (showLoading = false) => {
    try {
      if (showLoading && !cachedState) {
        setLoading(true)
      }
      const data = await desktopAPI().GetState()
      applyState(normalizeAppState(data))
      setError(null)
    } catch (err: any) {
      setError(err.message || 'Failed to fetch state')
    } finally {
      setLoading(false)
    }
  }, [applyState])

  const refreshRuntime = useCallback(async () => {
    try {
      if (!cachedState) {
        await refresh(false)
        return
      }
      const data = await desktopAPI().GetRuntimeState()
      applyState(normalizeAppState({
        ...cachedState,
        status: data.status,
        system_proxy: data.system_proxy,
        readiness: data.readiness,
      }))
      setError(null)
    } catch (err: any) {
      setError(err.message || 'Failed to fetch runtime state')
    }
  }, [applyState, refresh])

  useEffect(() => {
    refresh(true)
    const runtimeInterval = setInterval(() => refreshRuntime(), runtimePollIntervalMS)
    const fullSyncInterval = setInterval(() => refresh(false), fullSyncIntervalMS)
    return () => {
      clearInterval(runtimeInterval)
      clearInterval(fullSyncInterval)
    }
  }, [refresh, refreshRuntime])

  const start = useCallback(async () => {
    await desktopAPI().Start()
    await refresh()
  }, [refresh])

  const stop = useCallback(async () => {
    await desktopAPI().Stop()
    await refresh()
  }, [refresh])

  const selectProxy = useCallback(async (name: string) => {
    await desktopAPI().SelectProxy(name)
    await refresh()
  }, [refresh])

  const deleteProxy = useCallback(async (name: string) => {
    await desktopAPI().DeleteProxy(name)
    await refresh()
  }, [refresh])

  const checkProxy = useCallback(async (name: string, timeoutSeconds: number = 10) => {
    const result = await desktopAPI().CheckProxy(name, timeoutSeconds)
    await refresh()
    return result
  }, [refresh])

  const checkBestProxy = useCallback(async (timeoutSeconds: number = 10) => {
    const result = await desktopAPI().CheckBestProxy(timeoutSeconds)
    await refresh()
    return result
  }, [refresh])

  const importProfiles = useCallback(async (content: string, replace: boolean = true, selectName: string = '') => {
    const result = await desktopAPI().ImportProfiles(content, replace, selectName)
    await refresh()
    return result
  }, [refresh])

  const enableSystemProxy = useCallback(async () => {
    await desktopAPI().EnableSystemProxy()
    await refresh()
  }, [refresh])

  const disableSystemProxy = useCallback(async () => {
    await desktopAPI().DisableSystemProxy()
    await refresh()
  }, [refresh])

  const saveConfig = useCallback(async (cfg: ClientConfig) => {
    await desktopAPI().SaveConfig(cfg)
    await refresh()
  }, [refresh])

  const saveRelayProfile = useCallback(async (req: {name: string; relay_addr: string; token: string; tls: any; replace: boolean}) => {
    await desktopAPI().SaveRelayProfile(req)
    await refresh()
  }, [refresh])

  const deleteRelayProfile = useCallback(async (name: string) => {
    await desktopAPI().DeleteRelayProfile(name)
    await refresh()
  }, [refresh])

  const selectRelayProfile = useCallback(async (name: string) => {
    await desktopAPI().SelectRelayProfile(name)
    await refresh()
  }, [refresh])

  const checkRelayProfile = useCallback(async (name: string) => {
    const result = await desktopAPI().CheckRelayProfile(name)
    await refresh()
    return result
  }, [refresh])

  const saveSubscription = useCallback(async (req: {name: string; url: string; replace: boolean}) => {
    await desktopAPI().SaveSubscription(req)
    await refresh()
  }, [refresh])

  const deleteSubscription = useCallback(async (name: string) => {
    await desktopAPI().DeleteSubscription(name)
    await refresh()
  }, [refresh])

  const syncSubscription = useCallback(async (name: string, replace: boolean = true) => {
    const result = await desktopAPI().SyncSubscription(name, replace)
    await refresh()
    return result
  }, [refresh])

  const getLogs = useCallback(async () => {
    return await desktopAPI().GetLogs()
  }, [])

  return {
    state,
    loading,
    error,
    refresh,
    start,
    stop,
    selectProxy,
    deleteProxy,
    checkProxy,
    checkBestProxy,
    importProfiles,
    enableSystemProxy,
    disableSystemProxy,
    saveConfig,
    saveRelayProfile,
    deleteRelayProfile,
    selectRelayProfile,
    checkRelayProfile,
    saveSubscription,
    deleteSubscription,
    syncSubscription,
    getLogs,
    refreshRuntime,
  }
}

type DesktopContextValue = ReturnType<typeof useDesktopStore>

const DesktopContext = createContext<DesktopContextValue | null>(null)

export function DesktopProvider({children}: {children: ReactNode}) {
  const value = useDesktopStore()
  return createElement(DesktopContext.Provider, {value}, children)
}

export function useDesktop() {
  const value = useContext(DesktopContext)
  if (!value) {
    throw new Error('useDesktop must be used inside DesktopProvider')
  }
  return value
}
