import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiCheckCircle, FiCpu, FiShield, FiTerminal, FiUploadCloud, FiXCircle} from 'react-icons/fi'
import {useDesktop, RuntimeStatus, ClientConfig} from '../../hooks/useDesktop'

const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const XIcon = FiXCircle as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const ImportIcon = FiUploadCloud as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>
const CpuIcon = FiCpu as ComponentType<{className?: string}>

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

export function Overview() {
  const {state, loading, start, stop, importProfiles, enableSystemProxy, disableSystemProxy} = useDesktop()
  const [importContent, setImportContent] = useState('')
  const [importSelect, setImportSelect] = useState('')
  const [importReplace, setImportReplace] = useState(true)
  const [message, setMessage] = useState('')

  const status: RuntimeStatus = state?.status || {running: false, local_addr: '', http_addr: '', relay_addr: '', started_at: '', last_error: ''}
  const config: ClientConfig = state?.config || {
    local_addr: '', http_addr: '', relay_addr: '', token: '', dial_timeout_seconds: 10,
    local_auth: {enabled: false, username: '', password: ''},
    tls: {enabled: false, server_name: '', ca_file: '', insecure_skip_verify: false},
    profiles: [], proxy_profiles: [], subscriptions: [], active_profile: '', active_proxy_profile: ''
  }
  const systemProxy = state?.system_proxy || {supported: false, enabled: false, message: ''}

  const handleConnect = async () => {
    try {
      if (status.running) {
        await stop()
        setMessage('已断开')
      } else {
        await start()
        setMessage('已连接')
      }
    } catch (err: any) {
      setMessage(err.message)
    }
  }

  const handleImport = async () => {
    if (!importContent.trim()) {
      setMessage('请输入订阅内容')
      return
    }
    try {
      const count = await importProfiles(importContent, importReplace, importSelect)
      setMessage(`已导入 ${count} 个节点`)
      setImportContent('')
    } catch (err: any) {
      setMessage(err.message)
    }
  }

  const activeProxy = config.proxy_profiles.find(p => p.name === config.active_proxy_profile)
  const nodeLabel = activeProxy ? activeProxy.name : config.active_profile || '未选择'
  const metrics = status.metrics || {active_connections: 0, total_connections: 0, upload_bytes: 0, download_bytes: 0}
  const httpProxy = `http://${status.http_addr || config.http_addr || '-'}`
  const socksProxy = `socks5://${status.local_addr || config.local_addr || '-'}`

  if (loading) {
    return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div className="panel p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className={`grid h-14 w-14 place-items-center rounded-lg border ${
              status.running
                ? 'border-emerald-500/30 bg-emerald-50 text-emerald-700'
                : 'border-slate-200 bg-white/70 text-faint'
            }`}>
              {status.running ? <CheckIcon className="h-10 w-10" /> : <XIcon className="h-10 w-10" />}
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h2 className="text-3xl font-semibold text-main">{status.running ? '已连接' : '未连接'}</h2>
                <span className="pill px-2.5 py-1 text-xs">
                  {activeProxy ? activeProxy.protocol.toUpperCase() : config.active_profile ? 'RELAY' : 'IDLE'}
                </span>
              </div>
              <p className="mt-2 text-sm text-subtle">{nodeLabel} · {status.relay_addr || '未选择节点'}</p>
            </div>
          </div>
          <button
            onClick={handleConnect}
            className={`min-w-28 rounded-lg px-6 py-2.5 font-medium transition-colors ${
              status.running
                ? 'danger-button'
                : 'primary-button'
            }`}
          >
            {status.running ? '断开' : '连接'}
          </button>
        </div>

        <div className="mt-6 grid grid-cols-2 gap-3 border-t border-[#ded8f5] pt-6 md:grid-cols-3 xl:grid-cols-6">
          {[
            ['当前节点', nodeLabel],
            ['SOCKS5', status.local_addr || '-'],
            ['HTTP', status.http_addr || '-'],
            ['系统代理', systemProxy.supported ? (systemProxy.enabled ? '已开启' : '未开启') : '不支持'],
            ['活跃连接', String(metrics.active_connections)],
            ['流量', `${formatBytes(metrics.upload_bytes)} / ${formatBytes(metrics.download_bytes)}`],
          ].map(([label, value]) => (
            <div key={label} className="panel-soft p-3">
              <p className="text-xs text-faint">{label}</p>
              <p className="mt-1 truncate text-sm font-medium text-main">{value}</p>
            </div>
          ))}
        </div>

        <div className="flex gap-3 mt-6">
          <button
            onClick={systemProxy.enabled ? disableSystemProxy : enableSystemProxy}
            className="secondary-button px-4 py-2"
          >
            <ShieldIcon className="h-4 w-4" />
            {systemProxy.enabled ? '关闭系统代理' : '开启系统代理'}
          </button>
        </div>
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <div className="panel p-5">
          <div className="mb-4 flex items-center gap-2">
            <span className="icon-tile h-8 w-8"><ImportIcon className="h-4 w-4" /></span>
            <h3 className="text-lg font-semibold text-main">快速导入</h3>
          </div>
          <textarea
            placeholder="粘贴机场订阅 URL 或节点内容"
            value={importContent}
            onChange={e => setImportContent(e.target.value)}
            className="form-control mb-3 h-28 w-full resize-none p-3"
          />
          <div className="flex gap-3 mb-3">
            <input
              placeholder="默认节点名称（可选）"
              value={importSelect}
              onChange={e => setImportSelect(e.target.value)}
              className="form-control min-w-0 flex-1 px-3 py-2"
            />
            <label className="flex items-center gap-2 text-sm text-subtle">
              <input type="checkbox" checked={importReplace} onChange={e => setImportReplace(e.target.checked)} />
              覆盖同名
            </label>
          </div>
          <button
            onClick={handleImport}
            className="primary-button px-4 py-2"
          >
            导入并选择
          </button>
        </div>

        <div className="panel p-5">
          <div className="mb-4 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="icon-tile h-8 w-8 text-emerald-700"><TerminalIcon className="h-4 w-4" /></span>
              <h3 className="text-lg font-semibold text-main">终端 / AI Agent</h3>
            </div>
            <CpuIcon className="h-4 w-4 text-faint" />
          </div>
          <div className="divide-y divide-[#ded8f5]">
            <div className="py-3">
              <div className="text-xs text-faint">HTTP_PROXY / HTTPS_PROXY</div>
              <div className="mt-1 break-all font-mono text-sm text-subtle">{httpProxy}</div>
            </div>
            <div className="py-3">
              <div className="text-xs text-faint">ALL_PROXY</div>
              <div className="mt-1 break-all font-mono text-sm text-subtle">{socksProxy}</div>
            </div>
            <div className="py-3">
              <div className="text-xs text-faint">配置共享</div>
              <div className="mt-1 break-all text-sm text-subtle">{state?.config_path || '未加载'}</div>
            </div>
          </div>
        </div>
      </div>

      {message && (
        <div className="toast fixed bottom-4 right-4 px-4 py-2">
          {message}
        </div>
      )}
    </div>
  )
}
