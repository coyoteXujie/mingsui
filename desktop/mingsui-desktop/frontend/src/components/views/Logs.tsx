import {useState, useEffect} from 'react'

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
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-white">日志</h2>
        <button onClick={handleRefresh} className="px-4 py-2 bg-[#2a2a2a] hover:bg-[#3a3a3a] text-white rounded-lg">
          刷新
        </button>
      </div>

      <div className="bg-[#1a1a1a] border border-[#333] rounded-lg">
        <pre className="p-4 text-sm text-gray-300 font-mono overflow-auto max-h-[calc(100vh-200px)]">
          {logs.length > 0 ? logs.join('\n') : '暂无日志'}
        </pre>
      </div>

      {message && <div className="fixed bottom-4 right-4 bg-[#333] text-white px-4 py-2 rounded-lg">{message}</div>}
    </div>
  )
}
