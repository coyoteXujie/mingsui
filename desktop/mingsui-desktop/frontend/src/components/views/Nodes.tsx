import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiCheck, FiSearch, FiTrash2, FiWifi, FiZap} from 'react-icons/fi'
import {useDesktop, ProxyProfile} from '../../hooks/useDesktop'

const CheckIcon = FiCheck as ComponentType<{className?: string}>
const SearchIcon = FiSearch as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>
const WifiIcon = FiWifi as ComponentType<{className?: string}>
const ZapIcon = FiZap as ComponentType<{className?: string}>

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

  if (loading) return <div className="flex h-64 items-center justify-center text-[#8b949e]">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 md:grid-cols-4">
        {[
          ['全部节点', counts.all],
          ['可连接', counts.usable],
          ['当前', counts.current],
          ['不支持', counts.unsupported],
        ].map(([label, value]) => (
          <div key={label} className="rounded-lg border border-white/10 bg-[#17191c] p-4">
            <div className="text-xs text-[#6e7681]">{label}</div>
            <div className="mt-2 text-2xl font-semibold text-white">{value}</div>
          </div>
        ))}
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-white/10 bg-[#17191c] p-4">
        <div className="min-w-0">
          <div className="text-sm font-medium text-white">节点列表</div>
          <div className="mt-1 text-xs text-[#8b949e]">当前选择：{config?.active_proxy_profile || '未选择'}</div>
        </div>
        <button
          onClick={handleCheckBest}
          disabled={checkingName !== null}
          className="inline-flex items-center gap-2 rounded-lg bg-[#0b6f65] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[#0a5f57] disabled:bg-white/10 disabled:text-[#6e7681]"
        >
          <ZapIcon className="h-4 w-4" />
          {checkingName === '__best__' ? '测速中...' : '测速选优'}
        </button>
      </div>

      <div className="rounded-lg border border-white/10 bg-[#17191c] p-4">
        <div className="flex flex-wrap items-center gap-3">
          <label className="relative min-w-64 flex-1 md:flex-none">
            <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-[#6e7681]" />
            <input
              placeholder="搜索节点或协议"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="w-full rounded-lg border border-white/10 bg-black/20 py-2 pl-9 pr-3 text-sm text-white placeholder:text-[#6e7681] focus:border-[#0b6f65] focus:outline-none"
            />
          </label>
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
              className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                filter === f.id
                  ? 'border-[#0b6f65] bg-[#0b6f65] text-white'
                  : 'border-white/10 bg-white/5 text-[#c9d1d9] hover:bg-white/10 hover:text-white'
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {filteredProfiles.length === 0 ? (
          <div className="rounded-lg border border-white/10 bg-[#17191c] p-8 text-center text-[#8b949e]">没有匹配的机场节点</div>
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
              <div
                key={profile.name}
                className={`rounded-lg border p-4 transition-colors ${
                  isCurrent
                    ? 'border-[#0b6f65]/60 bg-[#10211f]'
                    : 'border-white/10 bg-[#17191c] hover:border-white/20'
                }`}
              >
                <div className="flex items-center justify-between gap-4">
                  <div className="min-w-0">
                    <div className="flex items-center gap-3">
                      <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-lg border ${
                        exportable ? 'border-[#0b6f65]/30 bg-[#0b6f65]/10 text-[#3fb950]' : 'border-white/10 bg-white/5 text-[#6e7681]'
                      }`}>
                        <WifiIcon className="h-4 w-4" />
                      </div>
                      <div className="min-w-0">
                        <p className="truncate font-medium text-white">
                          {profile.name}
                          {isCurrent && <span className="ml-2 rounded-full bg-[#0b6f65] px-2 py-0.5 text-xs text-white">当前</span>}
                        </p>
                        <p className={`mt-1 text-sm ${exportable && autoSelectable ? 'text-[#8b949e]' : 'text-amber-300'}`}>
                          {(profile.protocol || '-').toUpperCase()} · {compatibility}
                        </p>
                      </div>
                    </div>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <button
                      onClick={() => handleSelect(profile.name)}
                      disabled={!exportable}
                      className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors ${
                        exportable ? 'bg-[#0b6f65] text-white hover:bg-[#0a5f57]' : 'cursor-not-allowed bg-white/10 text-[#6e7681]'
                      }`}
                    >
                      <CheckIcon className="h-4 w-4" />
                      选择
                    </button>
                    <button
                      onClick={() => handleCheck(profile.name)}
                      disabled={!exportable || checkingName !== null}
                      className={`inline-flex items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                        exportable
                          ? 'border-white/10 bg-white/5 text-white hover:bg-white/10 disabled:bg-white/10 disabled:text-[#6e7681]'
                          : 'cursor-not-allowed border-white/10 bg-white/10 text-[#6e7681]'
                      }`}
                    >
                      <ZapIcon className="h-4 w-4" />
                      {checkingName === profile.name ? '检测中...' : '检测'}
                    </button>
                    <button
                      onClick={() => handleDelete(profile.name)}
                      className="inline-flex items-center gap-1.5 rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-1.5 text-sm text-red-200 transition-colors hover:bg-red-500/20"
                    >
                      <TrashIcon className="h-4 w-4" />
                      删除
                    </button>
                  </div>
                </div>
              </div>
            )
          })
        )}
      </div>

      {message && <div className="fixed bottom-4 right-4 rounded-lg border border-white/10 bg-[#17191c] px-4 py-2 text-white shadow-2xl shadow-black/30">{message}</div>}
    </div>
  )
}
