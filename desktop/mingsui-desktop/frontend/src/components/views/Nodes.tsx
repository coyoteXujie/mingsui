import {useState} from 'react'
import {useDesktop, ProxyProfile} from '../../hooks/useDesktop'

export function Nodes() {
  const {state, loading, selectProxy, deleteProxy, checkProxy, checkBestProxy} = useDesktop()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState('all')
  const [message, setMessage] = useState('')
  const [checkingName, setCheckingName] = useState<string | null>(null)

  const config = state?.config
  const profiles: ProxyProfile[] = config?.proxy_profiles || []
  const capabilities = state?.proxy_capabilities || []
  const capabilityMap = new Map(capabilities.map(c => [c.name, c]))

  const filteredProfiles = profiles.filter(p => {
    const cap = capabilityMap.get(p.name)
    const exportable = cap?.exportable !== false
    const autoSelectable = cap?.auto_selectable !== false
    const query = search.trim().toLowerCase()
    if (query && !`${p.name} ${p.protocol}`.toLowerCase().includes(query)) return false
    if (filter === 'usable' && !exportable) return false
    if (filter === 'current' && p.name !== config?.active_proxy_profile) return false
    if (filter === 'domestic' && !(exportable && !autoSelectable)) return false
    if (filter === 'unsupported' && exportable) return false
    return true
  })

  const handleSelect = async (name: string) => {
    try {
      await selectProxy(name)
      setMessage(`已选择 ${name}`)
    } catch (err: any) {
      setMessage(err.message)
    }
  }

  const handleDelete = async (name: string) => {
    if (!confirm(`删除机场节点 ${name}？`)) return
    try {
      await deleteProxy(name)
      setMessage(`已删除 ${name}`)
    } catch (err: any) {
      setMessage(err.message)
    }
  }

  const handleCheck = async (name: string) => {
    try {
      setCheckingName(name)
      const result = await checkProxy(name, 10)
      setMessage(result?.message || `${name} 检测完成`)
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setCheckingName(null)
    }
  }

  const handleCheckBest = async () => {
    try {
      setCheckingName('__best__')
      const result = await checkBestProxy(10)
      setMessage(result?.message || '测速选优完成')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setCheckingName(null)
    }
  }

  const counts = {
    all: profiles.length,
    usable: profiles.filter(p => capabilityMap.get(p.name)?.exportable !== false).length,
    current: profiles.filter(p => p.name === config?.active_proxy_profile).length,
    domestic: profiles.filter(p => {
      const cap = capabilityMap.get(p.name)
      return cap?.exportable !== false && cap?.auto_selectable === false
    }).length,
    unsupported: profiles.filter(p => capabilityMap.get(p.name)?.exportable === false).length,
  }

  if (loading) return <div className="flex items-center justify-center h-64"><div className="text-gray-400">加载中...</div></div>

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-bold text-white">节点</h2>
        <button
          onClick={handleCheckBest}
          disabled={checkingName !== null}
          className="px-4 py-2 bg-[#0b6f65] hover:bg-[#0a5f57] disabled:bg-[#333] disabled:text-gray-500 text-white rounded-lg"
        >
          {checkingName === '__best__' ? '测速中...' : '测速选优'}
        </button>
      </div>

      <div className="bg-[#252525] border border-[#333] rounded-lg p-4">
        <div className="flex flex-wrap gap-4 items-center">
          <input
            placeholder="搜索节点"
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="w-64 bg-[#1a1a1a] border border-[#333] rounded-lg px-3 py-2 text-white"
          />
          <div className="flex gap-2">
            {[
              {id: 'all', label: `全部 ${counts.all}`},
              {id: 'usable', label: `可连接 ${counts.usable}`},
              {id: 'current', label: `当前 ${counts.current}`},
              {id: 'domestic', label: `国内 ${counts.domestic}`},
              {id: 'unsupported', label: `不支持 ${counts.unsupported}`},
            ].map(f => (
              <button
                key={f.id}
                onClick={() => setFilter(f.id)}
                className={`px-3 py-1 rounded-full text-sm transition-colors ${
                  filter === f.id ? 'bg-[#0b6f65] text-white' : 'bg-[#2a2a2a] text-gray-300 hover:bg-[#3a3a3a]'
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="space-y-3">
        {filteredProfiles.length === 0 ? (
          <div className="bg-[#252525] border border-[#333] rounded-lg p-4 text-center text-gray-400">没有匹配的机场节点</div>
        ) : (
          filteredProfiles.map(profile => {
            const cap = capabilityMap.get(profile.name)
            const isCurrent = profile.name === config?.active_proxy_profile
            const exportable = cap?.exportable !== false
            const autoSelectable = cap?.auto_selectable !== false
            const compatibility = !exportable
              ? '暂不支持直接连接'
              : autoSelectable
                ? '可连接'
                : '可连接，国内节点不自动选择'

            return (
              <div key={profile.name} className="bg-[#252525] border border-[#333] rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-white font-medium">
                      {profile.name}
                      {isCurrent && <span className="ml-2 px-2 py-0.5 bg-[#0b6f65] text-white text-xs rounded-full">当前</span>}
                    </p>
                    <p className={`text-sm mt-1 ${exportable && autoSelectable ? 'text-gray-400' : 'text-amber-400'}`}>
                      {(profile.protocol || '-').toUpperCase()} · {compatibility}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <button
                      onClick={() => handleSelect(profile.name)}
                      disabled={!exportable}
                      className={`px-3 py-1.5 rounded-lg text-sm ${
                        exportable ? 'bg-[#0b6f65] hover:bg-[#0a5f57] text-white' : 'bg-[#333] text-gray-500 cursor-not-allowed'
                      }`}
                    >
                      选择
                    </button>
                    <button
                      onClick={() => handleCheck(profile.name)}
                      disabled={!exportable || checkingName !== null}
                      className={`px-3 py-1.5 rounded-lg text-sm ${
                        exportable ? 'bg-[#2a2a2a] hover:bg-[#3a3a3a] disabled:bg-[#333] disabled:text-gray-500 text-white' : 'bg-[#333] text-gray-500 cursor-not-allowed'
                      }`}
                    >
                      {checkingName === profile.name ? '检测中...' : '检测'}
                    </button>
                    <button
                      onClick={() => handleDelete(profile.name)}
                      className="px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white rounded-lg text-sm"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {message && <div className="fixed bottom-4 right-4 bg-[#333] text-white px-4 py-2 rounded-lg">{message}</div>}
    </div>
  )
}
