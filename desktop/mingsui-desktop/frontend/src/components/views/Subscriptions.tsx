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

  if (loading) return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="panel p-4">
          <div className="text-xs text-faint">已保存订阅</div>
          <div className="mt-2 text-2xl font-semibold text-main">{subscriptions.length}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">同步策略</div>
          <div className="mt-2 text-sm font-medium text-main">{replace ? '覆盖同名节点' : '保留已有节点'}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">配置文件</div>
          <div className="mt-2 truncate text-sm text-subtle">{state?.config_path || '未加载'}</div>
        </div>
      </div>

      <div className="panel p-5">
        <div className="mb-5 flex items-center gap-2">
          <span className="icon-tile h-8 w-8"><CloudIcon className="h-4 w-4" /></span>
          <h3 className="text-base font-semibold text-main">节点订阅</h3>
        </div>
        <div className="grid gap-4 md:grid-cols-2">
          <label className="block">
            <span className="mb-1 block text-sm text-subtle">名称</span>
            <input
              placeholder="例如：airport-main"
              value={name}
              onChange={e => setName(e.target.value)}
              className="form-control w-full px-3 py-2"
            />
          </label>
          <label className="block">
            <span className="mb-1 block text-sm text-subtle">订阅 URL</span>
            <div className="relative">
              <LinkIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
              <input
                placeholder="https://example.com/sub"
                value={url}
                onChange={e => setURL(e.target.value)}
                className="form-control w-full py-2 pl-9 pr-3"
              />
            </div>
          </label>
        </div>
        <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
          <label className="flex items-center gap-2 text-sm text-subtle">
            <input type="checkbox" checked={replace} onChange={e => setReplace(e.target.checked)} />
            覆盖同名节点
          </label>
          <div className="flex gap-3">
            <button
              onClick={handleSave}
              disabled={busy !== null}
              className="primary-button px-4 py-2 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400"
            >
              <SaveIcon className="h-4 w-4" />
              {busy === 'save' ? '保存中...' : '保存订阅'}
            </button>
            <button
              onClick={() => handleSync()}
              disabled={busy !== null}
              className="secondary-button px-4 py-2 text-sm disabled:bg-slate-200 disabled:text-slate-400"
            >
              <RefreshIcon className="h-4 w-4" />
              {busy?.startsWith('sync:') ? '同步中...' : '同步当前'}
            </button>
          </div>
        </div>
      </div>

      <div className="panel p-5">
        <div className="mb-5 flex items-center justify-between gap-3">
          <h3 className="text-base font-semibold text-main">已保存订阅</h3>
          <span className="pill px-2.5 py-1 text-xs">{subscriptions.length} 个来源</span>
        </div>
        {subscriptions.length === 0 ? (
          <div className="rounded-lg border border-dashed border-[#ded8f5] p-8 text-center text-subtle">没有节点订阅</div>
        ) : (
          <div className="space-y-3">
            {subscriptions.map(sub => (
              <div key={sub.name} className="row-surface flex items-center justify-between gap-4 p-3">
                <div className="min-w-0">
                  <p className="truncate font-medium text-main">{sub.name}</p>
                  <p className="mt-1 truncate text-sm text-subtle">{sub.url}</p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <button
                    onClick={() => { setName(sub.name); setURL(sub.url) }}
                    className="secondary-button px-3 py-1.5 text-sm"
                  >
                    <EditIcon className="h-4 w-4" />
                    编辑
                  </button>
                  <button
                    onClick={() => handleSync(sub.name)}
                    disabled={busy !== null}
                    className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                  >
                    <RefreshIcon className="h-4 w-4" />
                    {busy === `sync:${sub.name}` ? '同步中...' : '同步'}
                  </button>
                  <button
                    onClick={() => handleDelete(sub.name)}
                    disabled={busy !== null}
                    className="danger-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
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

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
