import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiCloud, FiEdit3, FiLink, FiRefreshCw, FiSave, FiTrash2} from 'react-icons/fi'
import {useDesktop} from '../../hooks/useDesktop'

const CloudIcon = FiCloud as ComponentType<{className?: string}>
const EditIcon = FiEdit3 as ComponentType<{className?: string}>
const LinkIcon = FiLink as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>

export function Subscriptions() {
  const {state, loading, saveSubscription, syncSubscription, deleteSubscription} = useDesktop()
  const [name, setName] = useState('')
  const [url, setURL] = useState('')
  const [replace, setReplace] = useState(true)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState<string | null>(null)

  const subscriptions = state?.config?.subscriptions || []

  const handleSave = async () => {
    if (!name.trim()) {
      setMessage('订阅名称不能为空')
      return
    }
    if (!url.trim()) {
      setMessage('订阅 URL 不能为空')
      return
    }
    try {
      setBusy('save')
      await saveSubscription({name, url, replace})
      setMessage('订阅已保存')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  const handleSync = async (subName = name) => {
    if (!subName.trim()) {
      setMessage('请选择要同步的订阅')
      return
    }
    try {
      setBusy(`sync:${subName}`)
      const count = await syncSubscription(subName, replace)
      setMessage(`订阅已同步：${count} 个节点`)
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  const handleDelete = async (subName: string) => {
    if (!confirm(`删除订阅 ${subName}？`)) return
    try {
      setBusy(`delete:${subName}`)
      await deleteSubscription(subName)
      if (name === subName) {
        setName('')
        setURL('')
      }
      setMessage('订阅已删除')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-[#8b949e]">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-4">
          <div className="text-xs text-[#6e7681]">已保存订阅</div>
          <div className="mt-2 text-2xl font-semibold text-white">{subscriptions.length}</div>
        </div>
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-4">
          <div className="text-xs text-[#6e7681]">同步策略</div>
          <div className="mt-2 text-sm font-medium text-white">{replace ? '覆盖同名节点' : '保留已有节点'}</div>
        </div>
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-4">
          <div className="text-xs text-[#6e7681]">配置文件</div>
          <div className="mt-2 truncate text-sm text-[#c9d1d9]">{state?.config_path || '未加载'}</div>
        </div>
      </div>

      <div className="rounded-lg border border-white/10 bg-[#17191c] p-5">
        <div className="mb-5 flex items-center gap-2">
          <CloudIcon className="h-4 w-4 text-[#0b6f65]" />
          <h3 className="text-base font-semibold text-white">节点订阅</h3>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">名称</span>
            <input
              placeholder="例如：airport-main"
              value={name}
              onChange={e => setName(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 px-3 py-2 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-[#8b949e]">订阅 URL</span>
            <div className="relative">
              <LinkIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#6e7681]" />
              <input
                placeholder="https://example.com/sub"
                value={url}
                onChange={e => setURL(e.target.value)}
                className="w-full rounded-lg border border-white/10 bg-black/20 py-2 pl-9 pr-3 text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
              />
            </div>
          </label>
        </div>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-[#c9d1d9]">
            <input type="checkbox" checked={replace} onChange={e => setReplace(e.target.checked)} />
            覆盖同名节点
          </label>
          <div className="flex gap-3">
            <button
              onClick={handleSave}
              disabled={busy !== null}
              className="inline-flex items-center gap-2 rounded-lg bg-[#0b6f65] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[#0a5f57] disabled:bg-white/10 disabled:text-[#6e7681]"
            >
              <SaveIcon className="h-4 w-4" />
              {busy === 'save' ? '保存中...' : '保存订阅'}
            </button>
            <button
              onClick={() => handleSync()}
              disabled={busy !== null}
              className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm text-white transition-colors hover:bg-white/10 disabled:bg-white/10 disabled:text-[#6e7681]"
            >
              <RefreshIcon className="h-4 w-4" />
              {busy?.startsWith('sync:') ? '同步中...' : '同步当前'}
            </button>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-white/10 bg-[#17191c] p-5">
        <div className="mb-5 flex items-center justify-between gap-3">
          <h3 className="text-base font-semibold text-white">已保存订阅</h3>
          <span className="rounded-full border border-white/10 bg-white/5 px-2.5 py-1 text-xs text-[#8b949e]">{subscriptions.length} 个来源</span>
        </div>
        {subscriptions.length === 0 ? (
          <div className="rounded-lg border border-dashed border-white/10 p-8 text-center text-[#8b949e]">没有节点订阅</div>
        ) : (
          <div className="space-y-3">
            {subscriptions.map(sub => (
              <div key={sub.name} className="flex items-center justify-between gap-4 rounded-lg border border-white/10 bg-black/20 p-3">
                <div className="min-w-0">
                  <p className="truncate font-medium text-white">{sub.name}</p>
                  <p className="mt-1 truncate text-sm text-[#8b949e]">{sub.url}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <button
                    onClick={() => { setName(sub.name); setURL(sub.url) }}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm text-white transition-colors hover:bg-white/10"
                  >
                    <EditIcon className="h-4 w-4" />
                    编辑
                  </button>
                  <button
                    onClick={() => handleSync(sub.name)}
                    disabled={busy !== null}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm text-white transition-colors hover:bg-white/10 disabled:bg-white/10 disabled:text-[#6e7681]"
                  >
                    <RefreshIcon className="h-4 w-4" />
                    {busy === `sync:${sub.name}` ? '同步中...' : '同步'}
                  </button>
                  <button
                    onClick={() => handleDelete(sub.name)}
                    disabled={busy !== null}
                    className="inline-flex items-center gap-1.5 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-1.5 text-sm text-red-200 transition-colors hover:bg-red-500/20 disabled:bg-white/10 disabled:text-[#6e7681]"
                  >
                    <TrashIcon className="h-4 w-4" />
                    {busy === `delete:${sub.name}` ? '删除中...' : '删除'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {message && <div className="fixed bottom-4 right-4 rounded-lg border border-white/10 bg-[#17191c] px-4 py-2 text-white shadow-2xl shadow-black/30">{message}</div>}
    </div>
  )
}
