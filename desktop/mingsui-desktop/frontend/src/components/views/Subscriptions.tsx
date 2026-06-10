import {useState} from 'react'
import {useDesktop} from '../../hooks/useDesktop'

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

  if (loading) return <div className="flex items-center justify-center h-64"><div className="text-gray-400">加载中...</div></div>

  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold text-white">订阅</h2>

      <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
        <h3 className="text-lg font-semibold text-white mb-4">节点订阅</h3>
        <div className="grid md:grid-cols-2 gap-4 mb-4">
          <input
            placeholder="名称"
            value={name}
            onChange={e => setName(e.target.value)}
            className="bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white"
          />
          <input
            placeholder="https://example.com/sub"
            value={url}
            onChange={e => setURL(e.target.value)}
            className="bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white"
          />
        </div>
        <label className="flex items-center gap-2 text-gray-400 mb-4">
          <input type="checkbox" checked={replace} onChange={e => setReplace(e.target.checked)} />
          覆盖同名节点
        </label>
        <div className="flex gap-3">
          <button
            onClick={handleSave}
            disabled={busy !== null}
            className="px-4 py-2 bg-[#0b6f65] hover:bg-[#0a5f57] disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg"
          >
            {busy === 'save' ? '保存中...' : '保存订阅'}
          </button>
          <button
            onClick={() => handleSync()}
            disabled={busy !== null}
            className="px-4 py-2 bg-[#2a2a2a] hover:bg-[#3a3a3a] disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg"
          >
            {busy?.startsWith('sync:') ? '同步中...' : '同步'}
          </button>
        </div>
      </div>

      <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
        <h3 className="text-lg font-semibold text-white mb-4">已保存订阅</h3>
        {subscriptions.length === 0 ? (
          <p className="text-gray-400">没有节点订阅</p>
        ) : (
          <div className="space-y-3">
            {subscriptions.map(sub => (
              <div key={sub.name} className="flex items-center justify-between p-3 bg-[#1a1a1a] rounded-lg">
                <div>
                  <p className="text-white font-medium">{sub.name}</p>
                  <p className="text-gray-400 text-sm">{sub.url}</p>
                </div>
                <div className="flex gap-2">
                  <button onClick={() => { setName(sub.name); setURL(sub.url) }} className="px-3 py-1.5 bg-[#2a2a2a] hover:bg-[#3a3a3a] text-white rounded-lg text-sm">编辑</button>
                  <button
                    onClick={() => handleSync(sub.name)}
                    disabled={busy !== null}
                    className="px-3 py-1.5 bg-[#2a2a2a] hover:bg-[#3a3a3a] disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg text-sm"
                  >
                    {busy === `sync:${sub.name}` ? '同步中...' : '同步'}
                  </button>
                  <button
                    onClick={() => handleDelete(sub.name)}
                    disabled={busy !== null}
                    className="px-3 py-1.5 bg-red-600 hover:bg-red-700 disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg text-sm"
                  >
                    {busy === `delete:${sub.name}` ? '删除中...' : '删除'}
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {message && <div className="fixed bottom-4 right-4 bg-[#333] text-white px-4 py-2 rounded-lg">{message}</div>}
    </div>
  )
}
