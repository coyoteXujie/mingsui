import {useMemo, useState} from 'react'
import type {ComponentType, ReactNode} from 'react'
import {
  FiCheckCircle,
  FiCloud,
  FiCopy,
  FiEdit3,
  FiFileText,
  FiLink,
  FiPlus,
  FiRefreshCw,
  FiSave,
  FiServer,
  FiShield,
  FiTrash2,
  FiWifi,
  FiZap,
} from 'react-icons/fi'
import {ClipboardSetText} from '../../../wailsjs/runtime/runtime'
import {useDesktop} from '../../hooks/useDesktop'
import type {Subscription, SubscriptionSyncResult} from '../../hooks/useDesktop'

const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const CloudIcon = FiCloud as ComponentType<{className?: string}>
const CopyIcon = FiCopy as ComponentType<{className?: string}>
const EditIcon = FiEdit3 as ComponentType<{className?: string}>
const FileIcon = FiFileText as ComponentType<{className?: string}>
const LinkIcon = FiLink as ComponentType<{className?: string}>
const PlusIcon = FiPlus as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const ServerIcon = FiServer as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>
const WifiIcon = FiWifi as ComponentType<{className?: string}>
const ZapIcon = FiZap as ComponentType<{className?: string}>

type Tone = 'success' | 'warning' | 'danger' | 'neutral'

interface StatCard {
  label: string
  value: string
  detail: string
  tone: Tone
  icon: ReactNode
}

interface SyncResult {
  title: string
  detail: string
  tone: Tone
}

const toneClasses: Record<Tone, string> = {
  success: 'border-emerald-500/20 bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200',
  warning: 'border-amber-500/25 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  danger: 'border-red-500/25 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
  neutral: 'border-slate-200 bg-white/60 text-subtle dark:border-white/10 dark:bg-white/5',
}

function sourceHost(rawURL: string) {
  try {
    return new URL(rawURL).host || '本地来源'
  } catch {
    return rawURL.startsWith('file:') ? '本地文件' : '自定义来源'
  }
}

function sourceType(rawURL: string) {
  if (!rawURL.trim()) return '未设置'
  if (rawURL.startsWith('file:')) return '本地文件'
  if (rawURL.startsWith('http://') || rawURL.startsWith('https://')) return '远程订阅'
  return '自定义来源'
}

function shellQuote(value: string) {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

function syncResultTitle(result: SubscriptionSyncResult) {
  if (result.kind === 'proxy') return `${result.imported} 个节点`
  if (result.kind === 'relay') return `${result.imported} 个 profile`
  return `${result.imported} 个项目`
}

function syncResultDetail(result: SubscriptionSyncResult) {
  const parts: string[] = []
  if (result.kind === 'proxy') {
    const exportable = result.imported_exportable_proxy_profiles ?? result.exportable_proxy_profiles
    const autoSelectable = result.imported_auto_selectable_proxy_profiles ?? result.auto_selectable_proxy_profiles
    parts.push(`${exportable} 可连接`)
    parts.push(`${autoSelectable} 自动候选`)
  } else if (result.kind === 'relay') {
    parts.push(`${result.relay_profiles} 个 relay profile`)
  }
  if (result.selected) {
    parts.push(`当前 ${result.selected}`)
  }
  if (result.warnings?.length) {
    parts.push(result.warnings.join('；'))
  }
  return parts.join(' · ') || result.message
}

function syncResultTone(result: SubscriptionSyncResult): Tone {
  if (result.warnings?.length) return 'warning'
  if (result.imported === 0) return 'warning'
  return 'success'
}

function shouldCheckAfterSync(result: SubscriptionSyncResult, syncCheck: boolean) {
  const autoSelectable = result.imported_auto_selectable_proxy_profiles ?? result.auto_selectable_proxy_profiles
  return syncCheck && result.kind === 'proxy' && autoSelectable > 0
}

function StatTile({item}: {item: StatCard}) {
  return (
    <div className="panel p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs text-faint">{item.label}</div>
          <div className="mt-2 truncate text-xl font-semibold text-main">{item.value}</div>
          <div className="mt-1 truncate text-xs text-subtle">{item.detail}</div>
        </div>
        <span className={`grid h-9 w-9 shrink-0 place-items-center rounded-lg border ${toneClasses[item.tone]}`}>
          {item.icon}
        </span>
      </div>
    </div>
  )
}

function SectionHeader({icon, title, detail, action}: {icon: ReactNode; title: string; detail: string; action?: ReactNode}) {
  return (
    <div className="mb-4 flex items-start justify-between gap-3">
      <div className="flex items-start gap-3">
        <span className="icon-tile h-9 w-9 shrink-0">{icon}</span>
        <div>
          <h3 className="text-base font-semibold text-main">{title}</h3>
          <p className="mt-1 text-xs text-subtle">{detail}</p>
        </div>
      </div>
      {action}
    </div>
  )
}

export function Subscriptions() {
  const {state, loading, saveSubscription, syncSubscription, checkBestProxy, deleteSubscription} = useDesktop()
  const [name, setName] = useState('')
  const [url, setURL] = useState('')
  const [replace, setReplace] = useState(true)
  const [syncCheck, setSyncCheck] = useState(true)
  const [message, setMessage] = useState('')
  const [busy, setBusy] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<SyncResult | null>(null)

  const subscriptions = state?.config?.subscriptions || []
  const proxyCount = state?.config?.proxy_profiles?.length || 0
  const activeProxy = state?.config?.active_proxy_profile || ''
  const selectedSubscription = subscriptions.find(sub => sub.name === name)
  const canSave = Boolean(name.trim() && url.trim() && busy === null)
  const canSyncCurrent = Boolean(selectedSubscription && busy === null)
  const canSyncAll = subscriptions.length > 0 && busy === null
  const activeSource = selectedSubscription || (name || url ? {name: name || '未命名', url} : null)
  const syncCommand = selectedSubscription
    ? `mingsui config subscription sync ${shellQuote(selectedSubscription.name)}${syncCheck ? ' -check' : ''}`
    : '保存订阅后可复制同步命令'
  const statCards: StatCard[] = [
    {
      label: '已保存订阅',
      value: String(subscriptions.length),
      detail: subscriptions.length > 0 ? `${subscriptions.map(sub => sourceHost(sub.url)).slice(0, 2).join(' / ')}${subscriptions.length > 2 ? ' ...' : ''}` : '等待添加来源',
      tone: subscriptions.length > 0 ? 'success' : 'warning',
      icon: <CloudIcon className="h-4 w-4" />,
    },
    {
      label: '节点库存',
      value: String(proxyCount),
      detail: activeProxy ? `当前 ${activeProxy}` : '同步后选择节点',
      tone: proxyCount > 0 ? 'success' : 'warning',
      icon: <WifiIcon className="h-4 w-4" />,
    },
    {
      label: '同步策略',
      value: replace ? '覆盖更新' : '增量保留',
      detail: syncCheck ? '同步后测速选优' : '同步后不测速',
      tone: syncCheck ? 'success' : 'neutral',
      icon: <RefreshIcon className="h-4 w-4" />,
    },
    {
      label: '最近结果',
      value: lastResult?.title || '暂无',
      detail: lastResult?.detail || '同步或保存后显示',
      tone: lastResult?.tone || 'neutral',
      icon: <CheckIcon className="h-4 w-4" />,
    },
  ]
  const sourceSummary = useMemo(() => {
    const groups = subscriptions.reduce<Record<string, number>>((acc, sub) => {
      const key = sourceType(sub.url)
      acc[key] = (acc[key] || 0) + 1
      return acc
    }, {})
    return Object.entries(groups)
  }, [subscriptions])

  const copyText = async (label: string, text: string) => {
    if (!text.trim() || text.startsWith('保存订阅后')) {
      setMessage(`${label} 暂不可复制`)
      return
    }
    try {
      let ok = false
      if ((window as any).runtime?.ClipboardSetText) {
        ok = await ClipboardSetText(text)
      } else if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
        ok = true
      }
      setMessage(ok ? `已复制 ${label}` : `复制 ${label} 失败`)
    } catch (err: any) {
      setMessage(err.message || `复制 ${label} 失败`)
    }
  }

  const handleSave = async () => {
    const target = name.trim()
    if (!target) {
      setMessage('订阅名称不能为空')
      return
    }
    if (!url.trim()) {
      setMessage('订阅 URL 不能为空')
      return
    }
    try {
      setBusy('save-sync')
      await saveSubscription({name: target, url: url.trim(), replace})
      const result = await syncSubscription(target, replace)
      const summary = syncResultDetail(result)
      const prefix = `订阅已保存并同步：${syncResultTitle(result)}；${summary}`
      const text = shouldCheckAfterSync(result, syncCheck) ? await runBestCheck(prefix) : prefix
      setLastResult({title: syncResultTitle(result), detail: summary, tone: syncResultTone(result)})
      setMessage(text)
    } catch (err: any) {
      setLastResult({title: '保存或同步失败', detail: err.message, tone: 'danger'})
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  const runBestCheck = async (prefix: string) => {
    if (!syncCheck) return `${prefix}`
    try {
      const result = await checkBestProxy(10)
      return `${prefix}；${result?.message || '测速选优完成'}`
    } catch (checkErr: any) {
      return `${prefix}；测速选优失败：${checkErr.message}`
    }
  }

  const handleSync = async (subName = name) => {
    const target = subName.trim()
    if (!target) {
      setMessage('请选择要同步的订阅')
      return
    }
    try {
      setBusy(`sync:${target}`)
      const result = await syncSubscription(target, replace)
      const summary = syncResultDetail(result)
      const prefix = `订阅已同步：${syncResultTitle(result)}；${summary}`
      const text = shouldCheckAfterSync(result, syncCheck) ? await runBestCheck(prefix) : prefix
      setLastResult({title: syncResultTitle(result), detail: summary, tone: syncResultTone(result)})
      setMessage(text)
    } catch (err: any) {
      setLastResult({title: '同步失败', detail: err.message, tone: 'danger'})
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
      const warnings: string[] = []
      let autoSelectable = 0
      for (const sub of subscriptions) {
        const result = await syncSubscription(sub.name, replace)
        total += result.imported
        autoSelectable += result.imported_auto_selectable_proxy_profiles ?? result.auto_selectable_proxy_profiles
        if (result.warnings?.length) {
          warnings.push(`${sub.name}: ${result.warnings.join('；')}`)
        }
      }
      const summary = warnings.length ? warnings.join('；') : `${subscriptions.length} 个来源`
      const prefix = `已同步 ${subscriptions.length} 个订阅，共 ${total} 个项目`
      const text = syncCheck && autoSelectable > 0 ? await runBestCheck(prefix) : `${prefix}；${summary}`
      setLastResult({title: `${total} 个项目`, detail: summary, tone: warnings.length ? 'warning' : 'success'})
      setMessage(text)
    } catch (err: any) {
      setLastResult({title: '同步失败', detail: err.message, tone: 'danger'})
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
      setLastResult({title: '已删除', detail: subName, tone: 'neutral'})
      setMessage('订阅已删除')
    } catch (err: any) {
      setLastResult({title: '删除失败', detail: err.message, tone: 'danger'})
      setMessage(err.message)
    } finally {
      setBusy(null)
    }
  }

  const editSubscription = (sub: Subscription) => {
    setName(sub.name)
    setURL(sub.url)
    setMessage('')
  }

  const clearEditor = () => {
    setName('')
    setURL('')
    setMessage('')
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 lg:grid-cols-4">
        {statCards.map(item => <StatTile key={item.label} item={item} />)}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(22rem,0.82fr)_minmax(0,1.18fr)]">
        <div className="space-y-6">
          <div className="panel p-5">
            <SectionHeader
              icon={<ServerIcon className="h-4 w-4" />}
              title="同步控制台"
              detail="统一控制覆盖策略、同步范围和同步后选优"
              action={<span className={`rounded-full border px-2.5 py-1 text-xs ${busy ? toneClasses.warning : toneClasses.success}`}>{busy ? '处理中' : '就绪'}</span>}
            />
            <div className="space-y-3">
              <div className="row-surface flex items-center justify-between gap-3 p-3">
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-main">同步全部来源</div>
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

              <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
                <span>
                  <span className="block font-medium text-main">覆盖同名节点</span>
                  <span className="mt-1 block text-xs text-subtle">关闭后保留已有同名节点</span>
                </span>
                <input type="checkbox" checked={replace} onChange={e => setReplace(e.target.checked)} className="h-4 w-4 accent-[#0b8a7e]" />
              </label>

              <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
                <span>
                  <span className="block font-medium text-main">同步后测速选优</span>
                  <span className="mt-1 block text-xs text-subtle">自动选择最快可用国外节点</span>
                </span>
                <input type="checkbox" checked={syncCheck} onChange={e => setSyncCheck(e.target.checked)} className="h-4 w-4 accent-[#0b8a7e]" />
              </label>

              <button
                onClick={() => copyText('同步命令', syncCommand)}
                disabled={!selectedSubscription}
                className="row-surface group flex w-full items-start justify-between gap-3 p-3 text-left transition disabled:cursor-not-allowed disabled:opacity-60"
              >
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-main">CLI 同步命令</div>
                  <div className="mt-1 truncate font-mono text-xs text-subtle">{syncCommand}</div>
                </div>
                <CopyIcon className="mt-0.5 h-4 w-4 shrink-0 text-faint group-hover:text-emerald-700" />
              </button>
            </div>
          </div>

          <div className="panel p-5">
            <SectionHeader
              icon={<SaveIcon className="h-4 w-4" />}
              title="订阅编辑器"
              detail={selectedSubscription ? '正在编辑已保存来源' : '添加或保存一个订阅来源'}
              action={
                <button
                  onClick={clearEditor}
                  disabled={busy !== null}
                  className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                >
                  <PlusIcon className="h-4 w-4" />
                  新建
                </button>
              }
            />
            <div className="space-y-4">
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
              {activeSource && (
                <div className="rounded-lg border border-[#dbe1eb] bg-white/58 p-3 text-sm text-subtle dark:border-white/10 dark:bg-white/5">
                  <div className="flex items-center justify-between gap-3">
                    <span>{sourceType(activeSource.url)}</span>
                    <span className="max-w-52 truncate font-medium text-main">{sourceHost(activeSource.url)}</span>
                  </div>
                </div>
              )}
              <div className="flex flex-wrap justify-end gap-3">
                <button
                  onClick={handleSave}
                  disabled={!canSave}
                  className="primary-button px-4 py-2 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400 disabled:shadow-none"
                >
                  {busy === 'save-sync' ? <RefreshIcon className="h-4 w-4" /> : <SaveIcon className="h-4 w-4" />}
                  {busy === 'save-sync' ? '保存同步中' : '保存并同步'}
                </button>
                <button
                  onClick={() => handleSync()}
                  disabled={!canSyncCurrent}
                  className="secondary-button px-4 py-2 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                >
                  <RefreshIcon className="h-4 w-4" />
                  {busy?.startsWith('sync:') ? '同步中' : '同步当前'}
                </button>
              </div>
            </div>
          </div>
        </div>

        <div className="space-y-6">
          <div className="panel p-5">
            <SectionHeader
              icon={<CloudIcon className="h-4 w-4" />}
              title="订阅来源"
              detail="维护节点来源，同步后进入节点页确认当前候选"
              action={<span className="pill px-2.5 py-1 text-xs">{subscriptions.length} 个来源</span>}
            />

            {subscriptions.length === 0 ? (
              <div className="rounded-lg border border-dashed border-[#cfd6e3] p-8 text-center text-subtle dark:border-white/10">
                没有节点订阅。先在左侧保存一个订阅来源。
              </div>
            ) : (
              <div className="space-y-3">
                {subscriptions.map(sub => {
                  const isSelected = sub.name === name
                  const host = sourceHost(sub.url)
                  const source = sourceType(sub.url)
                  return (
                    <div key={sub.name} className={`row-surface p-3 ${isSelected ? 'ring-2 ring-[#0b8a7e]/20' : ''}`}>
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex min-w-0 items-start gap-3">
                          <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-lg border ${isSelected ? toneClasses.success : toneClasses.neutral}`}>
                            <CloudIcon className="h-4 w-4" />
                          </div>
                          <div className="min-w-0">
                            <div className="flex flex-wrap items-center gap-2">
                              <p className="truncate font-semibold text-main">{sub.name}</p>
                              <span className="rounded-full border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs text-subtle dark:border-white/10 dark:bg-white/5">{source}</span>
                              <span className="max-w-52 truncate rounded-full border border-emerald-500/20 bg-emerald-50 px-2 py-0.5 text-xs text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200">{host}</span>
                            </div>
                            <p className="mt-2 truncate text-sm text-subtle">{sub.url}</p>
                          </div>
                        </div>
                      </div>
                      <div className="mt-3 flex flex-wrap justify-end gap-2">
                        <button
                          onClick={() => editSubscription(sub)}
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
                          {busy === `sync:${sub.name}` ? '同步中' : '同步'}
                        </button>
                        <button
                          onClick={() => handleDelete(sub.name)}
                          disabled={busy !== null}
                          className="danger-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                        >
                          <TrashIcon className="h-4 w-4" />
                          {busy === `delete:${sub.name}` ? '删除中' : '删除'}
                        </button>
                      </div>
                    </div>
                  )
                })}
              </div>
            )}
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <div className="panel p-5">
              <SectionHeader
                icon={<ShieldIcon className="h-4 w-4" />}
                title="来源分布"
                detail="快速判断订阅来源类型"
              />
              <div className="space-y-2">
                {sourceSummary.length === 0 ? (
                  <div className="rounded-lg border border-dashed border-[#cfd6e3] p-4 text-sm text-subtle dark:border-white/10">暂无来源</div>
                ) : sourceSummary.map(([label, count]) => (
                  <div key={label} className="row-surface flex items-center justify-between gap-3 p-3">
                    <span className="text-sm text-main">{label}</span>
                    <span className="pill px-2.5 py-1 text-xs">{count}</span>
                  </div>
                ))}
              </div>
            </div>

            <div className="panel p-5">
              <SectionHeader
                icon={<FileIcon className="h-4 w-4" />}
                title="共享配置"
                detail="桌面端和 CLI 使用同一份配置"
              />
              <div className="space-y-2">
                <div className="row-surface p-3">
                  <div className="text-xs text-faint">配置文件</div>
                  <div className="mt-2 break-all font-mono text-xs text-subtle">{state?.config_path || '未加载'}</div>
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
        </div>
      </div>

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
