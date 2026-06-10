import {useState, useEffect, useCallback} from 'react'

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

export interface AppState {
  config_path: string
  config: ClientConfig
  status: RuntimeStatus
  system_proxy: SystemProxyStatus
  proxy_capabilities: ProxyCapability[]
  readiness?: ReadinessStatus
}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetState: () => Promise<AppState>
          Start: () => Promise<string>
          Stop: () => Promise<string>
          GetConfig: () => Promise<ClientConfig>
          SaveConfig: (cfg: ClientConfig) => Promise<string>
          ImportProfiles: (content: string, replace: boolean, selectName: string) => Promise<[number, string]>
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
          SyncSubscription: (name: string, replace: boolean) => Promise<[number, string]>
          GetLogs: () => Promise<string[]>
        }
      }
    }
  }
}

export function useDesktop() {
  const [state, setState] = useState<AppState | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    try {
      setLoading(true)
      const data = await window.go.main.App.GetState()
      setState(data)
      setError(null)
    } catch (err: any) {
      setError(err.message || 'Failed to fetch state')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    refresh()
    const interval = setInterval(refresh, 3000)
    return () => clearInterval(interval)
  }, [refresh])

  const start = useCallback(async () => {
    await window.go.main.App.Start()
    await refresh()
  }, [refresh])

  const stop = useCallback(async () => {
    await window.go.main.App.Stop()
    await refresh()
  }, [refresh])

  const selectProxy = useCallback(async (name: string) => {
    await window.go.main.App.SelectProxy(name)
    await refresh()
  }, [refresh])

  const deleteProxy = useCallback(async (name: string) => {
    await window.go.main.App.DeleteProxy(name)
    await refresh()
  }, [refresh])

  const checkProxy = useCallback(async (name: string, timeoutSeconds: number = 10) => {
    const result = await window.go.main.App.CheckProxy(name, timeoutSeconds)
    await refresh()
    return result
  }, [refresh])

  const checkBestProxy = useCallback(async (timeoutSeconds: number = 10) => {
    const result = await window.go.main.App.CheckBestProxy(timeoutSeconds)
    await refresh()
    return result
  }, [refresh])

  const importProfiles = useCallback(async (content: string, replace: boolean = true, selectName: string = '') => {
    const [count] = await window.go.main.App.ImportProfiles(content, replace, selectName)
    await refresh()
    return count
  }, [refresh])

  const enableSystemProxy = useCallback(async () => {
    await window.go.main.App.EnableSystemProxy()
    await refresh()
  }, [refresh])

  const disableSystemProxy = useCallback(async () => {
    await window.go.main.App.DisableSystemProxy()
    await refresh()
  }, [refresh])

  const saveConfig = useCallback(async (cfg: ClientConfig) => {
    await window.go.main.App.SaveConfig(cfg)
    await refresh()
  }, [refresh])

  const saveRelayProfile = useCallback(async (req: {name: string; relay_addr: string; token: string; tls: any; replace: boolean}) => {
    await window.go.main.App.SaveRelayProfile(req)
    await refresh()
  }, [refresh])

  const deleteRelayProfile = useCallback(async (name: string) => {
    await window.go.main.App.DeleteRelayProfile(name)
    await refresh()
  }, [refresh])

  const selectRelayProfile = useCallback(async (name: string) => {
    await window.go.main.App.SelectRelayProfile(name)
    await refresh()
  }, [refresh])

  const checkRelayProfile = useCallback(async (name: string) => {
    const result = await window.go.main.App.CheckRelayProfile(name)
    await refresh()
    return result
  }, [refresh])

  const saveSubscription = useCallback(async (req: {name: string; url: string; replace: boolean}) => {
    await window.go.main.App.SaveSubscription(req)
    await refresh()
  }, [refresh])

  const deleteSubscription = useCallback(async (name: string) => {
    await window.go.main.App.DeleteSubscription(name)
    await refresh()
  }, [refresh])

  const syncSubscription = useCallback(async (name: string, replace: boolean = true) => {
    const [count] = await window.go.main.App.SyncSubscription(name, replace)
    await refresh()
    return count
  }, [refresh])

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
  }
}
