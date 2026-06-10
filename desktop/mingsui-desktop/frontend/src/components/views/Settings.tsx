import {useEffect, useState} from 'react'
import {useDesktop} from '../../hooks/useDesktop'

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

  if (loading) return <div className="flex items-center justify-center h-64"><div className="text-gray-400">加载中...</div></div>

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold text-white">设置</h2>

      <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
        <h3 className="text-lg font-semibold text-white mb-4">客户端配置</h3>
        <div className="grid md:grid-cols-3 gap-4 mb-4">
          <div>
            <label className="block text-gray-400 text-sm mb-1">SOCKS5 监听</label>
            <input placeholder="127.0.0.1:1080" value={localAddr} onChange={e => update(setLocalAddr, e.target.value)} className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white" />
          </div>
          <div>
            <label className="block text-gray-400 text-sm mb-1">HTTP 监听</label>
            <input placeholder="127.0.0.1:8080" value={httpAddr} onChange={e => update(setHttpAddr, e.target.value)} className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white" />
          </div>
          <div>
            <label className="block text-gray-400 text-sm mb-1">默认 relay</label>
            <input placeholder="relay.example.com:443" value={relayAddr} onChange={e => update(setRelayAddr, e.target.value)} className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white" />
          </div>
          <div>
            <label className="block text-gray-400 text-sm mb-1">Token</label>
            <input type="password" value={token} onChange={e => update(setToken, e.target.value)} className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white" />
          </div>
          <div>
            <label className="block text-gray-400 text-sm mb-1">超时秒数</label>
            <input type="number" value={timeoutSeconds} onChange={e => update(setTimeoutSeconds, parseInt(e.target.value) || 10)} className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white" />
          </div>
        </div>

        <div className="mb-4 p-3 bg-[#1a1a1a] rounded-lg">
          <label className="flex items-center gap-2 text-white mb-2">
            <input type="checkbox" checked={authEnabled} onChange={e => update(setAuthEnabled, e.target.checked)} />
            启用本地代理认证
          </label>
          {authEnabled && (
            <div className="grid md:grid-cols-2 gap-4 mt-3">
              <input placeholder="认证用户名" value={authUser} onChange={e => update(setAuthUser, e.target.value)} className="bg-[#252525] border border-[#333] rounded-lg px-3 py-2 text-white" />
              <input type="password" placeholder="认证密码" value={authPass} onChange={e => update(setAuthPass, e.target.value)} className="bg-[#252525] border border-[#333] rounded-lg px-3 py-2 text-white" />
            </div>
          )}
        </div>

        <div className="mb-4 p-3 bg-[#1a1a1a] rounded-lg">
          <label className="flex items-center gap-2 text-white mb-2">
            <input type="checkbox" checked={tlsEnabled} onChange={e => update(setTlsEnabled, e.target.checked)} />
            启用 relay TLS
          </label>
          {tlsEnabled && (
            <div className="mt-3 space-y-3">
              <input placeholder="TLS ServerName" value={tlsServerName} onChange={e => update(setTlsServerName, e.target.value)} className="w-full bg-[#252525] border border-[#333] rounded-lg px-3 py-2 text-white" />
              <input placeholder="TLS CA 文件" value={tlsCAFile} onChange={e => update(setTlsCAFile, e.target.value)} className="w-full bg-[#252525] border border-[#333] rounded-lg px-3 py-2 text-white" />
              <label className="flex items-center gap-2 text-gray-400">
                <input type="checkbox" checked={tlsInsecure} onChange={e => update(setTlsInsecure, e.target.checked)} />
                跳过证书校验
              </label>
            </div>
          )}
        </div>

        <button
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 bg-[#0b6f65] hover:bg-[#0a5f57] disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg"
        >
          {saving ? '保存中...' : '保存配置'}
        </button>
      </div>

      <div className="grid md:grid-cols-2 gap-4">
        <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-3">本地代理</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-gray-400">认证</span><span className="text-white">{authEnabled ? '已启用' : '未启用'}</span></div>
          </div>
        </div>
        <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-3">当前 relay</h3>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between"><span className="text-gray-400">TLS</span><span className="text-white">{tlsEnabled ? '已启用' : '未启用'}</span></div>
          </div>
        </div>
      </div>

      {message && <div className="fixed bottom-4 right-4 bg-[#333] text-white px-4 py-2 rounded-lg">{message}</div>}
    </div>
  )
}
