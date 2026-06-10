import {useState, useEffect, useMemo} from 'react'
import type {ComponentType} from 'react'
import {FiAlertCircle, FiCheckCircle, FiCopy, FiPauseCircle, FiPlayCircle, FiRefreshCw, FiSearch, FiTerminal} from 'react-icons/fi'
import {ClipboardSetText} from '../../../wailsjs/runtime/runtime'

const AlertIcon = FiAlertCircle as ComponentType<{className?: string}>
const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const CopyIcon = FiCopy as ComponentType<{className?: string}>
const PauseIcon = FiPauseCircle as ComponentType<{className?: string}>
const PlayIcon = FiPlayCircle as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const SearchIcon = FiSearch as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>

type LogFilter = 'all' | 'errors' | 'warnings' | 'connections' | 'mihomo'

function logLevel(line: string): 'error' | 'warning' | 'info' {
  const text = line.toLowerCase()
  if (text.includes('error') || text.includes('failed') || text.includes('invalid') || text.includes('rejected') || text.includes('失败')) return 'error'
  if (text.includes('warn') || text.includes('temporary') || text.includes('跳过') || text.includes('警告')) return 'warning'
  return 'info'
}

function matchesFilter(line: string, filter: LogFilter) {
  const text = line.toLowerCase()
  if (filter === 'all') return true
  if (filter === 'errors') return logLevel(line) === 'error'
  if (filter === 'warnings') return logLevel(line) === 'warning'
  if (filter === 'connections') return text.includes('connect') || text.includes('listening') || text.includes('target=') || text.includes('连接')
  if (filter === 'mihomo') return text.includes('mihomo') || text.includes('内核')
  return true
}

function levelBadge(line: string) {
  const level = logLevel(line)
  if (level === 'error') return {label: '错误', className: 'border-red-200 bg-red-50 text-red-700'}
  if (level === 'warning') return {label: '警告', className: 'border-amber-200 bg-amber-50 text-amber-700'}
  return {label: '信息', className: 'border-slate-200 bg-slate-50 text-slate-700'}
}

export function Logs() {
  const [logs, setLogs] = useState<string[]>([])
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<LogFilter>('all')
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [message, setMessage] = useState('')

  const refreshLogs = async () => {
    try {
      const lines = await window.go.main.App.GetLogs()
      setLogs(lines || [])
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
  const query = search.trim().toLowerCase()
  const errorCount = logs.filter(line => logLevel(line) === 'error').length
  const warningCount = logs.filter(line => logLevel(line) === 'warning').length
  const visibleLogs = useMemo(() => logs.filter(line => {
    if (!matchesFilter(line, filter)) return false
    if (query && !line.toLowerCase().includes(query)) return false
    return true
  }), [logs, filter, query])
  const filters: Array<{id: LogFilter; label: string; count: number}> = [
    {id: 'all', label: '全部', count: logs.length},
    {id: 'errors', label: '错误', count: errorCount},
    {id: 'warnings', label: '警告', count: warningCount},
    {id: 'connections', label: '连接', count: logs.filter(line => matchesFilter(line, 'connections')).length},
    {id: 'mihomo', label: '内核', count: logs.filter(line => matchesFilter(line, 'mihomo')).length},
  ]

  const copyLogs = async () => {
    const text = visibleLogs.join('\n')
    if (!text) {
      setMessage('没有可复制的日志')
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
      setMessage(ok ? `已复制 ${visibleLogs.length} 行日志` : '复制日志失败')
    } catch (err: any) {
      setMessage(err.message || '复制日志失败')
    }
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-4">
        <div className="panel p-4">
          <div className="text-xs text-faint">日志行数</div>
          <div className="mt-2 text-2xl font-semibold text-main">{logs.length}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">错误 / 警告</div>
          <div className="mt-2 text-2xl font-semibold text-main">{errorCount} / {warningCount}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">筛选结果</div>
          <div className="mt-2 text-2xl font-semibold text-main">{visibleLogs.length}</div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">自动刷新</div>
          <div className={`mt-2 text-sm font-medium ${autoRefresh ? 'text-emerald-700' : 'text-subtle'}`}>{autoRefresh ? '每 3 秒刷新' : '已暂停'}</div>
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

      <div className="grid gap-3 md:grid-cols-[1fr_2fr]">
        <div className="panel p-4">
          <div className="text-xs text-faint">状态摘要</div>
          <div className="mt-3 space-y-2">
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div className="flex items-center gap-2 text-sm text-main">
                {errorCount > 0 ? <AlertIcon className="h-4 w-4 text-red-700" /> : <CheckIcon className="h-4 w-4 text-emerald-700" />}
                运行健康
              </div>
              <span className={errorCount > 0 ? 'text-sm text-red-700' : 'text-sm text-emerald-700'}>
                {errorCount > 0 ? '需要关注' : '未见错误'}
              </span>
            </div>
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div className="text-sm text-main">当前视图</div>
              <span className="text-sm text-subtle">{filters.find(item => item.id === filter)?.label || '全部'}</span>
            </div>
          </div>
        </div>
        <div className="panel p-4">
          <div className="text-xs text-faint">最新记录</div>
          <div className="mt-2 line-clamp-3 font-mono text-sm leading-6 text-subtle">{logs[logs.length - 1] || '暂无日志'}</div>
        </div>
      </div>

      <div className="panel overflow-hidden">
        <div className="flex items-center justify-between gap-3 border-b border-[#ded8f5] px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="icon-tile h-8 w-8 text-emerald-700"><TerminalIcon className="h-4 w-4" /></span>
            <h3 className="text-base font-semibold text-main">运行日志</h3>
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
              onClick={handleRefresh}
              className="secondary-button px-3 py-1.5 text-sm"
            >
              <RefreshIcon className="h-4 w-4" />
              刷新
            </button>
          </div>
        </div>
        <div className="max-h-[calc(100vh-320px)] min-h-96 overflow-auto bg-white/45">
          {visibleLogs.length > 0 ? (
            visibleLogs.map((line, index) => {
              const badge = levelBadge(line)
              return (
                <div key={`${index}-${line}`} className="grid grid-cols-[3.5rem_4.5rem_minmax(0,1fr)] gap-3 border-b border-[#ded8f5] px-4 py-2 font-mono text-sm leading-6 last:border-b-0">
                  <div className="text-right text-xs text-faint">{index + 1}</div>
                  <div>
                    <span className={`rounded-full border px-2 py-0.5 text-xs ${badge.className}`}>{badge.label}</span>
                  </div>
                  <div className="break-all text-[#465064]">{line}</div>
                </div>
              )
            })
          ) : (
            <div className="p-8 text-center text-subtle">{logs.length > 0 ? '没有匹配的日志' : '暂无日志'}</div>
          )}
        </div>
      </div>

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
