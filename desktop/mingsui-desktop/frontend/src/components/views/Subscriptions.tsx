import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiCheckCircle, FiCloud, FiEdit3, FiLink, FiPlus, FiRefreshCw, FiSave, FiServer, FiTrash2, FiZap} from 'react-icons/fi'
import {useDesktop} from '../../hooks/useDesktop'

const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const CloudIcon = FiCloud as ComponentType<{className?: string}>
const EditIcon = FiEdit3 as ComponentType<{className?: string}>
const LinkIcon = FiLink as ComponentType<{className?: string}>
const PlusIcon = FiPlus as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const ServerIcon = FiServer as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>
const ZapIcon = FiZap as ComponentType<{className?: string}>

function sourceHost(rawURL: string) {
  try {
    return new URL(rawURL).host || '本地来源'
  } catch {
    return rawURL.startsWith('file:') ? '本地文件' : '自定义来源'
  }
}

export function Subscriptions() {
  const {state, loading, saveSubscription, syncSubscription, checkBestProxy, deleteSubscription} = useDesktop()
  const [name, setName] = useState('')
  const [url, setURL] = useState('')
  const [replace, setReplace] = useState(true)
  const [syncCheck, setSyncCheck] = useState(true)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState<string | null>(null)

  const subscriptions = state?.config?.subscriptions || []
  const proxyCount = state?.config?.proxy_profiles?.length || 0
  const activeProxy = state?.config?.active_proxy_profile || ''
  const selectedSubscription = subscriptions.find(sub => sub.name === name)
  const canSyncAll = subscriptions.length > 0 && busy === null

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
      if (!syncCheck) {
        setMessage(`订阅已同步：${count} 个节点`)
        return
      }
      try {
        const result = await checkBestProxy(10)
        setMessage(`订阅已同步：${count} 个节点；${result?.message || '测速选优完成'}`)
      } catch (checkErr: any) {
        setMessage(`订阅已同步：${count} 个节点；测速选优失败：${checkErr.message}`)
      }
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  const handleSyncAll = async () => {
    if (subscriptions.length === 0) {
      setMessage('没有可同步的订阅')
      return
    }
    try {
      setBusy('sync:__all__')
      let total = 0
      for (const sub of subscriptions) {
        total += await syncSubscription(sub.name, replace)
      }
      if (!syncCheck) {
        setMessage(`已同步 ${subscriptions.length} 个订阅，共 ${total} 个节点`)
        return
      }
      try {
        const result = await checkBestProxy(10)
        setMessage(`已同步 ${subscriptions.length} 个订阅，共 ${total} 个节点；${result?.message || '测速选优完成'}`)
      } catch (checkErr: any) {
        setMessage(`已同步 ${subscriptions.length} 个订阅，共 ${total} 个节点；测速选优失败：${checkErr.message}`)
      }
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

  const clearEditor = () => {
    setName('')
    setURL('')
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-4">
        <div className="panel p-4">
          <div className="text-xs text-faint">已保存订阅</div>
          <div className="mt-2 text-2xl font-semibold text-main">{subscriptions.length}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">节点库存</div>
          <div className="mt-2 text-2xl font-semibold text-main">{proxyCount}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">同步策略</div>
          <div className="mt-2 text-sm font-medium text-main">{replace ? '覆盖同名节点' : '保留已有节点'} · {syncCheck ? '测速选优' : '不测速'}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">配置文件</div>
          <div className="mt-2 truncate text-sm text-subtle">{state?.config_path || '未加载'}</div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
        <div className="panel p-5">
          <div className="mb-5 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="icon-tile h-8 w-8"><CloudIcon className="h-4 w-4" /></span>
              <h3 className="text-base font-semibold text-main">订阅编辑器</h3>
            </div>
            <div className="flex items-center gap-2">
              <span className="pill px-2.5 py-1 text-xs">{selectedSubscription ? '编辑已保存来源' : '新建来源'}</span>
              <button
                onClick={clearEditor}
                disabled={busy !== null}
                className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
              >
                <PlusIcon className="h-4 w-4" />
                新建
              </button>
            </div>
          </div>
          <div className="grid gap-4 md:grid-cols-[0.8fr_1.2fr]">
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
          <div className="mt-4 grid gap-3 md:grid-cols-2">
            <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
              <span>覆盖同名节点</span>
              <input type="checkbox" checked={replace} onChange={e => setReplace(e.target.checked)} />
            </label>
            <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
              <span>同步后测速选优</span>
              <input type="checkbox" checked={syncCheck} onChange={e => setSyncCheck(e.target.checked)} />
            </label>
          </div>
          <div className="mt-5 flex flex-wrap justify-end gap-3">
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

        <div className="panel p-5">
          <div className="mb-5 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="icon-tile h-8 w-8 text-emerald-700"><ServerIcon className="h-4 w-4" /></span>
              <h3 className="text-base font-semibold text-main">同步控制台</h3>
            </div>
            <span className="pill px-2.5 py-1 text-xs">{busy === 'sync:__all__' ? '同步中' : '就绪'}</span>
          </div>
          <div className="space-y-3">
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div>
                <div className="text-sm font-medium text-main">同步全部来源</div>
                <div className="mt-1 text-xs text-subtle">依次拉取全部订阅并按策略更新节点</div>
              </div>
              <button
                onClick={handleSyncAll}
                disabled={!canSyncAll}
                className="primary-button shrink-0 px-3 py-1.5 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400"
              >
                <RefreshIcon className="h-4 w-4" />
                {busy === 'sync:__all__' ? '同步中' : '同步全部'}
              </button>
            </div>
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div>
                <div className="text-sm font-medium text-main">测速选优</div>
                <div className="mt-1 text-xs text-subtle">同步后自动选择最快可用节点</div>
              </div>
              <span className={syncCheck ? 'text-sm text-emerald-700' : 'text-sm text-subtle'}>{syncCheck ? '已开启' : '已关闭'}</span>
            </div>
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div>
                <div className="text-sm font-medium text-main">当前节点</div>
                <div className="mt-1 max-w-52 truncate text-xs text-subtle">{activeProxy || '未选择'}</div>
              </div>
              <ZapIcon className="h-4 w-4 text-faint" />
            </div>
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
                <div className="flex min-w-0 items-center gap-3">
                  <div className="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-[#0b8a7e]/20 bg-emerald-50 text-emerald-700">
                    <CheckIcon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="truncate font-medium text-main">{sub.name}</p>
                      <span className="max-w-52 truncate rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs text-subtle">{sourceHost(sub.url)}</span>
                    </div>
                    <p className="mt-1 truncate text-sm text-subtle">{sub.url}</p>
                  </div>
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
