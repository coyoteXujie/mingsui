import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiCheckCircle, FiXCircle} from 'react-icons/fi'
import {useDesktop, RuntimeStatus, ClientConfig} from '../../hooks/useDesktop'

const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const XIcon = FiXCircle as ComponentType<{className?: string}>

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

  if (loading) {
    return <div className="flex items-center justify-center h-64"><div className="text-gray-400">加载中...</div></div>
  }

  return (
    <div className="space-y-6">
      {/* 连接状态卡片 */}
      <div className="bg-[#252525] border border-[#333] rounded-lg p-6">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className={status.running ? 'text-green-500' : 'text-gray-500'}>
              {status.running ? <CheckIcon className="h-10 w-10" /> : <XIcon className="h-10 w-10" />}
            </div>
            <div>
              <h2 className="text-2xl font-bold text-white">{status.running ? '已连接' : '未连接'}</h2>
              <p className="text-gray-400 mt-1">{nodeLabel} · {status.relay_addr || '未选择节点'}</p>
            </div>
          </div>
          <button
            onClick={handleConnect}
            className={`px-6 py-2 rounded-lg font-medium transition-colors ${
              status.running
                ? 'bg-red-600 hover:bg-red-700 text-white'
                : 'bg-[#0b6f65] hover:bg-[#0a5f57] text-white'
            }`}
          >
            {status.running ? '断开' : '连接'}
          </button>
        </div>

        <div className="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-6 gap-4 mt-6 pt-6 border-t border-[#333]">
          <div><p className="text-gray-500 text-sm">当前节点</p><p className="text-white">{nodeLabel}</p></div>
          <div><p className="text-gray-500 text-sm">SOCKS5</p><p className="text-white">{status.local_addr || '-'}</p></div>
          <div><p className="text-gray-500 text-sm">HTTP</p><p className="text-white">{status.http_addr || '-'}</p></div>
          <div><p className="text-gray-500 text-sm">系统代理</p><p className="text-white">{systemProxy.supported ? (systemProxy.enabled ? '已开启' : '未开启') : <span className="text-gray-500">不支持</span>}</p></div>
          <div><p className="text-gray-500 text-sm">活跃连接</p><p className="text-white">{metrics.active_connections}</p></div>
          <div><p className="text-gray-500 text-sm">流量</p><p className="text-white">{formatBytes(metrics.upload_bytes)} / {formatBytes(metrics.download_bytes)}</p></div>
        </div>

        <div className="flex gap-3 mt-6">
          <button
            onClick={systemProxy.enabled ? disableSystemProxy : enableSystemProxy}
            className="px-4 py-2 bg-[#2a2a2a] hover:bg-[#3a3a3a] text-white rounded-lg transition-colors"
          >
            {systemProxy.enabled ? '关闭系统代理' : '开启系统代理'}
          </button>
        </div>
      </div>

      {/* 快速导入和账号 */}
      <div className="grid md:grid-cols-2 gap-6">
        <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">快速导入</h3>
          <textarea
            placeholder="粘贴机场订阅 URL 或节点内容"
            value={importContent}
            onChange={e => setImportContent(e.target.value)}
            className="w-full h-24 bg-[#1a1a1a] border border-[#333] rounded-lg p-3 text-white mb-3 resize-none"
          />
          <div className="flex gap-3 mb-3">
            <input
              placeholder="默认节点名称（可选）"
              value={importSelect}
              onChange={e => setImportSelect(e.target.value)}
              className="flex-1 bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white"
            />
            <label className="flex items-center gap-2 text-gray-400 text-sm">
              <input type="checkbox" checked={importReplace} onChange={e => setImportReplace(e.target.checked)} />
              覆盖同名
            </label>
          </div>
          <button
            onClick={handleImport}
            className="px-4 py-2 bg-[#0b6f65] hover:bg-[#0a5f57] text-white rounded-lg"
          >
            导入并选择
          </button>
        </div>

        <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
          <h3 className="text-lg font-semibold text-white mb-4">明隧账号</h3>
          <input
            placeholder="邮箱"
            className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white mb-3"
          />
          <input
            type="password"
            placeholder="访问令牌"
            className="w-full bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white mb-3"
          />
          <button className="px-4 py-2 bg-[#0b6f65] hover:bg-[#0a5f57] text-white rounded-lg">登录</button>
        </div>
      </div>

      {message && (
        <div className="fixed bottom-4 right-4 bg-[#333] text-white px-4 py-2 rounded-lg shadow-lg">
          {message}
        </div>
      )}
    </div>
  )
}
