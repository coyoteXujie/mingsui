import {useState, useEffect, useMemo} from 'react'
import type {ComponentType, ReactNode} from 'react'
import {
  FiActivity,
  FiAlertCircle,
  FiCheckCircle,
  FiClipboard,
  FiCopy,
  FiFileText,
  FiPauseCircle,
  FiPlayCircle,
  FiRefreshCw,
  FiSearch,
  FiShield,
  FiTerminal,
  FiTrash2,
  FiWifi,
} from 'react-icons/fi'
import {ClipboardSetText} from '../../../wailsjs/runtime/runtime'
import {useDesktop} from '../../hooks/useDesktop'

const ActivityIcon = FiActivity as ComponentType<{className?: string}>
const AlertIcon = FiAlertCircle as ComponentType<{className?: string}>
const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const ClipboardIcon = FiClipboard as ComponentType<{className?: string}>
const CopyIcon = FiCopy as ComponentType<{className?: string}>
const FileIcon = FiFileText as ComponentType<{className?: string}>
const PauseIcon = FiPauseCircle as ComponentType<{className?: string}>
const PlayIcon = FiPlayCircle as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const SearchIcon = FiSearch as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>
const WifiIcon = FiWifi as ComponentType<{className?: string}>

type LogFilter = 'all' | 'errors' | 'warnings' | 'connections' | 'proxy' | 'mihomo'
type Tone = 'success' | 'warning' | 'danger' | 'neutral'

interface StatusCard {
  label: string
  value: string
  detail: string
  tone: Tone
  icon: ReactNode
}

interface DiagnosticCommand {
  label: string
  detail: string
  command: string
}

const toneClasses: Record<Tone, string> = {
  success: 'border-emerald-500/20 bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200',
  warning: 'border-amber-500/25 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  danger: 'border-red-500/25 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
  neutral: 'border-slate-200 bg-white/60 text-subtle dark:border-white/10 dark:bg-white/5',
}

function logLevel(line: string): 'error' | 'warning' | 'info' {
  const text = line.toLowerCase()
  if (
    text.includes('error') ||
    text.includes('failed') ||
    text.includes('invalid') ||
    text.includes('panic') ||
    text.includes('rejected') ||
    text.includes('失败')
  ) return 'error'
  if (text.includes('warn') || text.includes('temporary') || text.includes('跳过') || text.includes('警告')) return 'warning'
  return 'info'
}

function matchesFilter(line: string, filter: LogFilter) {
  const text = line.toLowerCase()
  if (filter === 'all') return true
  if (filter === 'errors') return logLevel(line) === 'error'
  if (filter === 'warnings') return logLevel(line) === 'warning'
  if (filter === 'connections') return text.includes('connect') || text.includes('listening') || text.includes('target=') || text.includes('连接')
  if (filter === 'proxy') return text.includes('proxy') || text.includes('socks') || text.includes('http') || text.includes('系统代理')
  if (filter === 'mihomo') return text.includes('mihomo') || text.includes('内核')
  return true
}

function levelBadge(line: string) {
  const level = logLevel(line)
  if (level === 'error') return {label: '错误', className: 'border-red-200 bg-red-50 text-red-700 dark:border-red-500/20 dark:bg-red-500/10 dark:text-red-200'}
  if (level === 'warning') return {label: '警告', className: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/20 dark:bg-amber-500/10 dark:text-amber-200'}
  return {label: '信息', className: 'border-slate-200 bg-slate-50 text-slate-700 dark:border-white/10 dark:bg-white/5 dark:text-slate-200'}
}

function quotePath(path: string) {
  if (!path.trim()) return ''
  return `"${path.replace(/(["\\$`])/g, '\\$1')}"`
}

function readinessLabel(readiness?: string) {
  if (!readiness) return '等待检查'
  if (readiness === 'ready') return '就绪'
  if (readiness === 'needs_setup') return '待配置'
  if (readiness === 'blocked') return '阻塞'
  return readiness
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

function StatusTile({item}: {item: StatusCard}) {
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

export function Logs() {
  const {state, getLogs} = useDesktop()
  const [logs, setLogs] = useState<string[]>([])
  const [clearedCount, setClearedCount] = useState(0)
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<LogFilter>('all')
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [message, setMessage] = useState('')

  const refreshLogs = async () => {
    try {
      const lines = await getLogs()
      const next = lines || []
      setLogs(next)
      setClearedCount(current => current > next.length ? 0 : current)
      setMessage('')
    } catch (err: any) {
      setMessage(err.message)
    }
  }

  useEffect(() => {
    refreshLogs()
  }, [])

  useEffect(() => {
    if (!autoRefresh) return
    const timer = setInterval(refreshLogs, 3000)
    return () => clearInterval(timer)
  }, [autoRefresh])

  const handleRefresh = () => refreshLogs()
  const scopedLogs = logs.slice(clearedCount)
  const query = search.trim().toLowerCase()
  const errorCount = scopedLogs.filter(line => logLevel(line) === 'error').length
  const warningCount = scopedLogs.filter(line => logLevel(line) === 'warning').length
  const visibleLogs = useMemo(() => scopedLogs.filter(line => {
    if (!matchesFilter(line, filter)) return false
    if (query && !line.toLowerCase().includes(query)) return false
    return true
  }), [scopedLogs, filter, query])

  const status = state?.status
  const readiness = state?.readiness
  const config = state?.config
  const systemProxy = state?.system_proxy
  const activeNode = config?.active_proxy_profile || config?.active_profile || '未选择'
  const readinessTone: Tone = !readiness ? 'neutral' : !readiness.ok ? 'danger' : readiness.readiness === 'needs_setup' ? 'warning' : 'success'
  const healthTone: Tone = errorCount > 0 || readinessTone === 'danger' ? 'danger' : warningCount > 0 || readinessTone === 'warning' ? 'warning' : 'success'
  const configArg = state?.config_path ? ` -config ${quotePath(state.config_path)}` : ''
  const diagnosticCommands: DiagnosticCommand[] = [
    {label: '状态 JSON', detail: '查看 readiness、当前节点和配置路径', command: `mingsui status${configArg} -json`},
    {label: '连接诊断', detail: '检查本地端口、Relay 和配置问题', command: `mingsui doctor${configArg}`},
    {label: '诊断 JSON', detail: '给 issue 或 AI 分析使用的结构化报告', command: `mingsui doctor${configArg} -json`},
    {label: '系统代理', detail: '确认系统代理是否被 MingSui 接管', command: 'mingsui system-proxy status -json'},
  ]
  const readinessActions = readiness?.actions || []
  const cards: StatusCard[] = [
    {
      label: '运行健康',
      value: healthTone === 'danger' ? '需要关注' : healthTone === 'warning' ? '有风险' : '正常',
      detail: `${errorCount} 错误 · ${warningCount} 警告`,
      tone: healthTone,
      icon: healthTone === 'danger' ? <AlertIcon className="h-4 w-4" /> : <CheckIcon className="h-4 w-4" />,
    },
    {
      label: '连接状态',
      value: status?.running ? '已连接' : '未连接',
      detail: activeNode,
      tone: status?.running ? 'success' : 'neutral',
      icon: <ActivityIcon className="h-4 w-4" />,
    },
    {
      label: 'Readiness',
      value: readinessLabel(readiness?.readiness),
      detail: readiness?.mode === 'proxy' ? '机场节点模式' : readiness?.mode === 'relay' ? 'Relay 模式' : '等待状态',
      tone: readinessTone,
      icon: <ShieldIcon className="h-4 w-4" />,
    },
    {
      label: '当前视图',
      value: String(visibleLogs.length),
      detail: clearedCount > 0 ? `已隐藏 ${clearedCount} 行旧日志` : autoRefresh ? '每 3 秒刷新' : '自动刷新暂停',
      tone: autoRefresh ? 'success' : 'neutral',
      icon: autoRefresh ? <RefreshIcon className="h-4 w-4" /> : <PauseIcon className="h-4 w-4" />,
    },
  ]
  const filters: Array<{id: LogFilter; label: string; count: number}> = [
    {id: 'all', label: '全部', count: scopedLogs.length},
    {id: 'errors', label: '错误', count: errorCount},
    {id: 'warnings', label: '警告', count: warningCount},
    {id: 'connections', label: '连接', count: scopedLogs.filter(line => matchesFilter(line, 'connections')).length},
    {id: 'proxy', label: '代理', count: scopedLogs.filter(line => matchesFilter(line, 'proxy')).length},
    {id: 'mihomo', label: '内核', count: scopedLogs.filter(line => matchesFilter(line, 'mihomo')).length},
  ]

  const copyText = async (label: string, text: string) => {
    if (!text.trim()) {
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

  const copyLogs = async () => {
    await copyText(`${visibleLogs.length} 行日志`, visibleLogs.join('\n'))
  }

  const copyDiagnosticBundle = async () => {
    const bundle = [
      'MingSui Desktop 诊断摘要',
      `配置文件: ${state?.config_path || '未加载'}`,
      `运行状态: ${status?.running ? '已连接' : '未连接'}`,
      `当前节点: ${activeNode}`,
      `Readiness: ${readinessLabel(readiness?.readiness)}`,
      `系统代理: ${systemProxy?.supported ? (systemProxy.enabled ? '已开启' : '未开启') : '不支持'}`,
      `本地 SOCKS5: ${status?.local_addr || config?.local_addr || '-'}`,
      `本地 HTTP: ${status?.http_addr || config?.http_addr || '-'}`,
      `Relay: ${status?.relay_addr || config?.relay_addr || '-'}`,
      '',
      '可见日志:',
      visibleLogs.join('\n') || '暂无可见日志',
    ].join('\n')
    await copyText('诊断摘要', bundle)
  }

  const clearView = () => {
    setClearedCount(logs.length)
    setMessage(`已清空当前视图 ${logs.length} 行；后续新日志会继续显示`)
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-3 lg:grid-cols-4">
        {cards.map(item => <StatusTile key={item.label} item={item} />)}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(24rem,0.85fr)]">
        <div className="panel p-5">
          <SectionHeader
            icon={<ShieldIcon className="h-4 w-4" />}
            title="诊断摘要"
            detail="把 readiness、系统代理、配置路径和最近错误集中到一处"
            action={<span className={`rounded-full border px-2.5 py-1 text-xs ${toneClasses[healthTone]}`}>{cards[0].value}</span>}
          />
          <div className="grid gap-3 lg:grid-cols-2">
            {[
              ['配置文件', state?.config_path || '未加载'],
              ['当前节点', activeNode],
              ['系统代理', systemProxy?.supported ? (systemProxy.enabled ? '已开启' : '未开启') : systemProxy?.message || '不支持'],
              ['最近错误', status?.last_error || '暂无'],
            ].map(([label, value]) => (
              <div key={label} className="row-surface p-3">
                <div className="text-xs text-faint">{label}</div>
                <div className="mt-2 break-all text-sm font-medium text-main">{value}</div>
              </div>
            ))}
          </div>

          {readinessActions.length > 0 && (
            <div className="mt-4">
              <div className="mb-2 text-xs font-medium text-faint">建议动作</div>
              <div className="grid gap-2 lg:grid-cols-2">
                {readinessActions.slice(0, 4).map(action => (
                  <button
                    key={action.id}
                    onClick={() => copyText(action.label, action.command || '')}
                    disabled={!action.command}
                    className="row-surface group flex min-h-24 items-start justify-between gap-3 p-3 text-left transition disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    <div className="min-w-0">
                      <div className="text-sm font-semibold text-main">{action.label}</div>
                      <div className="mt-1 text-xs leading-5 text-subtle">{action.description || '来自 readiness 的建议动作'}</div>
                      {action.command && <div className="mt-2 truncate font-mono text-xs text-faint">{action.command}</div>}
                    </div>
                    <CopyIcon className="mt-0.5 h-4 w-4 shrink-0 text-faint group-hover:text-emerald-700" />
                  </button>
                ))}
              </div>
            </div>
          )}

          <div className="mt-4 rounded-lg border border-[#dbe1eb] bg-white/58 p-4 dark:border-white/10 dark:bg-white/5">
            <div className="flex items-start gap-3">
              <span className={`grid h-8 w-8 shrink-0 place-items-center rounded-lg border ${toneClasses[readinessTone]}`}>
                {readinessTone === 'danger' ? <AlertIcon className="h-4 w-4" /> : <CheckIcon className="h-4 w-4" />}
              </span>
              <div className="min-w-0">
                <div className="text-sm font-semibold text-main">{readinessLabel(readiness?.readiness)}</div>
                <div className="mt-1 text-sm leading-6 text-subtle">{readiness?.message || '桌面端正在等待 readiness 状态。'}</div>
              </div>
            </div>
            {readiness?.warnings && readiness.warnings.length > 0 && (
              <div className="mt-3 space-y-2">
                {readiness.warnings.map(warning => (
                  <div key={warning} className="rounded-lg border border-amber-500/20 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                    {warning}
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        <div className="panel p-5">
          <SectionHeader
            icon={<ClipboardIcon className="h-4 w-4" />}
            title="CLI 诊断命令"
            detail="复制到终端或发给 AI 继续分析"
            action={
              <button onClick={copyDiagnosticBundle} className="secondary-button px-3 py-1.5 text-sm">
                <CopyIcon className="h-4 w-4" />
                诊断摘要
              </button>
            }
          />
          <div className="space-y-3">
            {diagnosticCommands.map(item => (
              <button
                key={item.label}
                onClick={() => copyText(item.label, item.command)}
                className="row-surface group flex w-full items-start justify-between gap-3 p-3 text-left transition"
              >
                <div className="min-w-0">
                  <div className="text-sm font-semibold text-main">{item.label}</div>
                  <div className="mt-1 text-xs text-subtle">{item.detail}</div>
                  <div className="mt-2 truncate font-mono text-xs text-faint">{item.command}</div>
                </div>
                <CopyIcon className="mt-0.5 h-4 w-4 shrink-0 text-faint group-hover:text-emerald-700" />
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="panel p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="relative min-w-72 flex-1 md:flex-none">
            <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
            <input
              placeholder="搜索日志内容、目标地址或错误关键字"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="form-control w-full py-2 pl-9 pr-3 text-sm"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            {filters.map(item => (
              <button
                key={item.id}
                onClick={() => setFilter(item.id)}
                className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                  filter === item.id
                    ? 'border-[#0b8a7e] bg-[#0b8a7e] text-white'
                    : 'pill hover:bg-white/90'
                }`}
              >
                {item.label} {item.count}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="panel overflow-hidden">
        <div className="flex items-center justify-between gap-3 border-b border-[#dbe1eb] px-4 py-3 dark:border-white/10">
          <div className="flex items-center gap-2">
            <span className="icon-tile h-8 w-8 text-emerald-700"><TerminalIcon className="h-4 w-4" /></span>
            <div>
              <h3 className="text-base font-semibold text-main">运行日志</h3>
              <p className="mt-0.5 text-xs text-subtle">{visibleLogs.length} / {scopedLogs.length} 行正在显示</p>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setAutoRefresh(value => !value)}
              className="secondary-button px-3 py-1.5 text-sm"
            >
              {autoRefresh ? <PauseIcon className="h-4 w-4" /> : <PlayIcon className="h-4 w-4" />}
              {autoRefresh ? '暂停' : '继续'}
            </button>
            <button
              onClick={copyLogs}
              className="secondary-button px-3 py-1.5 text-sm"
            >
              <CopyIcon className="h-4 w-4" />
              复制
            </button>
            <button
              onClick={clearView}
              className="secondary-button px-3 py-1.5 text-sm"
            >
              <TrashIcon className="h-4 w-4" />
              清空视图
            </button>
            <button
              onClick={handleRefresh}
              className="secondary-button px-3 py-1.5 text-sm"
            >
              <RefreshIcon className="h-4 w-4" />
              刷新
            </button>
          </div>
        </div>
        <div className="max-h-[calc(100vh-360px)] min-h-96 overflow-auto bg-white/45 dark:bg-white/[0.03]">
          {visibleLogs.length > 0 ? (
            visibleLogs.map((line, index) => {
              const badge = levelBadge(line)
              return (
                <div key={`${index}-${line}`} className="grid grid-cols-[3.5rem_4.5rem_minmax(0,1fr)] gap-3 border-b border-[#dbe1eb] px-4 py-2 font-mono text-sm leading-6 last:border-b-0 dark:border-white/10">
                  <div className="text-right text-xs text-faint">{index + 1}</div>
                  <div>
                    <span className={`rounded-full border px-2 py-0.5 text-xs ${badge.className}`}>{badge.label}</span>
                  </div>
                  <div className="break-all text-[#465064] dark:text-slate-300">{line}</div>
                </div>
              )
            })
          ) : (
            <div className="p-8 text-center text-subtle">{scopedLogs.length > 0 ? '没有匹配的日志' : clearedCount > 0 ? '当前视图已清空，等待新日志' : '暂无日志'}</div>
          )}
        </div>
      </div>

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
