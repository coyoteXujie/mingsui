import {useEffect, useState} from 'react'
import type {ComponentType} from 'react'
import {FiClock, FiLock, FiSave, FiServer, FiShield, FiSliders} from 'react-icons/fi'
import {useDesktop} from '../../hooks/useDesktop'

const ClockIcon = FiClock as ComponentType<{className?: string}>
const LockIcon = FiLock as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const ServerIcon = FiServer as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const SlidersIcon = FiSliders as ComponentType<{className?: string}>

export function Settings() {
  const {state, loading, saveConfig} = useDesktop()
  const [message, setMessage] = useState('')
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  const config = state?.config
  const [localAddr, setLocalAddr] = useState(config?.local_addr || '')
  const [httpAddr, setHttpAddr] = useState(config?.http_addr || '')
  const [relayAddr, setRelayAddr] = useState(config?.relay_addr || '')
  const [token, setToken] = useState(config?.token || '')
  const [timeoutSeconds, setTimeoutSeconds] = useState(config?.dial_timeout_seconds || 10)
  const [authEnabled, setAuthEnabled] = useState(config?.local_auth?.enabled || false)
  const [authUser, setAuthUser] = useState(config?.local_auth?.username || '')
  const [authPass, setAuthPass] = useState(config?.local_auth?.password || '')
  const [tlsEnabled, setTlsEnabled] = useState(config?.tls?.enabled || false)
  const [tlsServerName, setTlsServerName] = useState(config?.tls?.server_name || '')
  const [tlsCAFile, setTlsCAFile] = useState(config?.tls?.ca_file || '')
  const [tlsInsecure, setTlsInsecure] = useState(config?.tls?.insecure_skip_verify || false)

  useEffect(() => {
    if (!config || dirty) return
    setLocalAddr(config.local_addr || '')
    setHttpAddr(config.http_addr || '')
    setRelayAddr(config.relay_addr || '')
    setToken(config.token || '')
    setTimeoutSeconds(config.dial_timeout_seconds || 10)
    setAuthEnabled(Boolean(config.local_auth?.enabled))
    setAuthUser(config.local_auth?.username || '')
    setAuthPass(config.local_auth?.password || '')
    setTlsEnabled(Boolean(config.tls?.enabled))
    setTlsServerName(config.tls?.server_name || '')
    setTlsCAFile(config.tls?.ca_file || '')
    setTlsInsecure(Boolean(config.tls?.insecure_skip_verify))
  }, [config, dirty])

  const update = <T,>(setter: (value: T) => void, value: T) => {
    setDirty(true)
    setter(value)
  }

  const handleSave = async () => {
    if (!config) {
      setMessage('配置尚未加载')
      return
    }
    try {
      setSaving(true)
      await saveConfig({
        ...config,
        local_addr: localAddr,
        http_addr: httpAddr,
        relay_addr: relayAddr,
        token,
        dial_timeout_seconds: timeoutSeconds,
        local_auth: {
          enabled: authEnabled,
          username: authUser,
          password: authPass,
        },
        tls: {
          enabled: tlsEnabled,
          server_name: tlsServerName,
          ca_file: tlsCAFile,
          insecure_skip_verify: tlsInsecure,
        },
        profiles: config.profiles || [],
        proxy_profiles: config.proxy_profiles || [],
        subscriptions: config.subscriptions || [],
      })
      setDirty(false)
      setMessage('配置已保存')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-[#8b949e]">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-4">
        {[
          ['SOCKS5', localAddr || '-'],
          ['HTTP', httpAddr || '-'],
          ['认证', authEnabled ? '已启用' : '未启用'],
          ['TLS', tlsEnabled ? '已启用' : '未启用'],
        ].map(([label, value]) => (
          <div key={label} className="rounded-lg border border-white/10 bg-[#17191c] p-4">
            <div className="text-xs text-[#6e7681]">{label}</div>
            <div className="mt-2 truncate text-sm font-medium text-white">{value}</div>
          </div>
        ))}
      </div>

      <div className="rounded-lg border border-white/10 bg-[#17191c] p-5">
        <div className="mb-5 flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            <SlidersIcon className="h-4 w-4 text-[#0b6f65]" />
            <h3 className="text-base font-semibold text-white">客户端配置</h3>
          </div>
          {dirty && <span className="rounded-full border border-amber-500/20 bg-amber-500/10 px-2.5 py-1 text-xs text-amber-200">未保存</span>}
        </div>

        <div className="grid gap-4 md:grid-cols-3">
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">SOCKS5 监听</span>
            <input
              placeholder="127.0.0.1:1080"
              value={localAddr}
              onChange={e => update(setLocalAddr, e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">HTTP 监听</span>
            <input
              placeholder="127.0.0.1:8080"
              value={httpAddr}
              onChange={e => update(setHttpAddr, e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">默认 relay</span>
            <input
              placeholder="relay.example.com:443"
              value={relayAddr}
              onChange={e => update(setRelayAddr, e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">Token</span>
            <div className="relative">
              <LockIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#6e7681]" />
              <input
                type="password"
                value={token}
                onChange={e => update(setToken, e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-black/20 py-2 pl-9 pr-3 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
            </div>
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">超时秒数</span>
            <div className="relative">
              <ClockIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#6e7681]" />
              <input
                type="number"
                value={timeoutSeconds}
                onChange={e => update(setTimeoutSeconds, parseInt(e.target.value) || 10)}
                className="w-full rounded-lg border border-white/10 bg-black/20 py-2 pl-9 pr-3 text-white focus:border-[#0b6f65] focus:outline-none"
              />
            </div>
          </label>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-5">
          <div className="mb-4 flex items-center gap-2">
            <ShieldIcon className="h-4 w-4 text-[#3fb950]" />
            <h3 className="text-base font-semibold text-white">本地代理认证</h3>
          </div>
          <label className="mb-4 flex items-center gap-2 text-sm text-[#c9d1d9]">
            <input type="checkbox" checked={authEnabled} onChange={e => update(setAuthEnabled, e.target.checked)} />
            启用本地代理认证
          </label>
          {authEnabled ? (
            <div className="grid gap-4 md:grid-cols-2">
              <input
                placeholder="认证用户名"
                value={authUser}
                onChange={e => update(setAuthUser, e.target.value)}
                className="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
              <input
                type="password"
                placeholder="认证密码"
                value={authPass}
                onChange={e => update(setAuthPass, e.target.value)}
                className="rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
            </div>
          ) : (
            <div className="rounded-lg border border-dashed border-white/10 p-4 text-sm text-[#8b949e]">本机 HTTP/SOCKS5 代理当前无需用户名密码。</div>
          )}
        </div>

        <div className="rounded-lg border border-white/10 bg-[#17191c] p-5">
          <div className="mb-4 flex items-center gap-2">
            <ServerIcon className="h-4 w-4 text-[#3fb950]" />
            <h3 className="text-base font-semibold text-white">Relay TLS</h3>
          </div>
          <label className="mb-4 flex items-center gap-2 text-sm text-[#c9d1d9]">
            <input type="checkbox" checked={tlsEnabled} onChange={e => update(setTlsEnabled, e.target.checked)} />
            启用 relay TLS
          </label>
          {tlsEnabled ? (
            <div className="space-y-3">
              <input
                placeholder="TLS ServerName"
                value={tlsServerName}
                onChange={e => update(setTlsServerName, e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
              <input
                placeholder="TLS CA 文件"
                value={tlsCAFile}
                onChange={e => update(setTlsCAFile, e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
              <label className="flex items-center gap-2 text-sm text-[#c9d1d9]">
                <input type="checkbox" checked={tlsInsecure} onChange={e => update(setTlsInsecure, e.target.checked)} />
                跳过证书校验
              </label>
            </div>
          ) : (
            <div className="rounded-lg border border-dashed border-white/10 p-4 text-sm text-[#8b949e]">默认使用明文 relay 连接；需要公网 TLS 时再启用。</div>
          )}
        </div>
      </div>

      <div className="flex justify-end">
        <button
          onClick={handleSave}
          disabled={saving}
          className="inline-flex items-center gap-2 rounded-lg bg-[#0b6f65] px-5 py-2.5 text-sm font-medium text-white transition-colors hover:bg-[#0a5f57] disabled:bg-white/10 disabled:text-[#6e7681]"
        >
          <SaveIcon className="h-4 w-4" />
          {saving ? '保存中...' : '保存配置'}
        </button>
      </div>

      {message && <div className="fixed bottom-4 right-4 rounded-lg border border-white/10 bg-[#17191c] px-4 py-2 text-white shadow-2xl shadow-black/30">{message}</div>}
    </div>
  )
}
