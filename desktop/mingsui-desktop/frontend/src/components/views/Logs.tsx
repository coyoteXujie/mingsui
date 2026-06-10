import {useState, useEffect} from 'react'
import type {ComponentType} from 'react'
import {FiRefreshCw, FiTerminal} from 'react-icons/fi'

const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>

export function Logs() {
  const [logs, setLogs] = useState<string[]>([])
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
    const timer = setInterval(refreshLogs, 3000)
    return () => clearInterval(timer)
  }, [])

  const handleRefresh = () => refreshLogs()

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-3">
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-4">
          <div className="text-xs text-[#6e7681]">日志行数</div>
          <div className="mt-2 text-2xl font-semibold text-white">{logs.length}</div>
        </div>
        <div className="rounded-lg border border-white/10 bg-[#17191c] p-4 md:col-span-2">
          <div className="text-xs text-[#6e7681]">最新记录</div>
          <div className="mt-2 truncate font-mono text-sm text-[#c9d1d9]">{logs[logs.length - 1] || '暂无日志'}</div>
        </div>
      </div>

      <div className="rounded-lg border border-white/10 bg-[#17191c]">
        <div className="flex items-center justify-between gap-3 border-b border-white/10 px-4 py-3">
          <div className="flex items-center gap-2">
            <TerminalIcon className="h-4 w-4 text-[#3fb950]" />
            <h3 className="text-base font-semibold text-white">运行日志</h3>
          </div>
          <button
            onClick={handleRefresh}
            className="inline-flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-3 py-1.5 text-sm text-white transition-colors hover:bg-white/10"
          >
            <RefreshIcon className="h-4 w-4" />
            刷新
          </button>
        </div>
        <pre className="max-h-[calc(100vh-260px)] min-h-96 overflow-auto p-4 font-mono text-sm leading-6 text-[#c9d1d9]">
          {logs.length > 0 ? logs.join('\n') : '暂无日志'}
        </pre>
      </div>

      {message && <div className="fixed bottom-4 right-4 rounded-lg border border-white/10 bg-[#17191c] px-4 py-2 text-white shadow-2xl shadow-black/30">{message}</div>}
    </div>
  )
}
