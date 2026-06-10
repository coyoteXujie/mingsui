import {useState} from 'react'
import type {ComponentType} from 'react'
import {FiAlertCircle, FiCheck, FiCpu, FiEdit3, FiLock, FiPlus, FiSave, FiSearch, FiServer, FiShield, FiTrash2, FiWifi, FiZap} from 'react-icons/fi'
import {useDesktop, ProxyProfile, RelayProfile} from '../../hooks/useDesktop'

const AlertIcon = FiAlertCircle as ComponentType<{className?: string}>
const CheckIcon = FiCheck as ComponentType<{className?: string}>
const CpuIcon = FiCpu as ComponentType<{className?: string}>
const EditIcon = FiEdit3 as ComponentType<{className?: string}>
const LockIcon = FiLock as ComponentType<{className?: string}>
const PlusIcon = FiPlus as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const SearchIcon = FiSearch as ComponentType<{className?: string}>
const ServerIcon = FiServer as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const TrashIcon = FiTrash2 as ComponentType<{className?: string}>
const WifiIcon = FiWifi as ComponentType<{className?: string}>
const ZapIcon = FiZap as ComponentType<{className?: string}>

type NodeFilter = 'all' | 'usable' | 'current' | 'domestic' | 'unsupported'

function protocolTone(protocol: string) {
  const key = protocol.toLowerCase()
  if (key.includes('ss')) return 'bg-sky-50 text-sky-700 border-sky-200'
  if (key.includes('trojan')) return 'bg-purple-50 text-purple-700 border-purple-200'
  if (key.includes('tuic') || key.includes('hysteria')) return 'bg-amber-50 text-amber-700 border-amber-200'
  return 'bg-slate-50 text-slate-700 border-slate-200'
}

export function Nodes() {
  const {
    state,
    loading,
    selectProxy,
    deleteProxy,
    checkProxy,
    checkBestProxy,
    saveRelayProfile,
    deleteRelayProfile,
    selectRelayProfile,
    checkRelayProfile,
  } = useDesktop()
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<NodeFilter>('all')
  const [message, setMessage] = useState('')
  const [checkingName, setCheckingName] = useState<string | null>(null)
  const [relayBusy, setRelayBusy] = useState<string | null>(null)
  const [relayName, setRelayName] = useState('')
  const [relayAddr, setRelayAddr] = useState('')
  const [relayToken, setRelayToken] = useState('')
  const [relayTLSEnabled, setRelayTLSEnabled] = useState(false)
  const [relayServerName, setRelayServerName] = useState('')
  const [relayCAFile, setRelayCAFile] = useState('')
  const [relayInsecure, setRelayInsecure] = useState(false)
  const [relayReplace, setRelayReplace] = useState(true)

  const config = state?.config
  const profiles: ProxyProfile[] = config?.proxy_profiles || []
  const relayProfiles: RelayProfile[] = config?.profiles || []
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
  const autoCandidateCount = profiles.filter(p => {
    const cap = capabilityMap.get(p.name)
    return cap?.exportable !== false && cap?.auto_selectable !== false
  }).length
  const activeProfile = profiles.find(p => p.name === config?.active_proxy_profile)
  const activeRelayProfile = relayProfiles.find(p => p.name === config?.active_profile)
  const selectedRelayProfile = relayProfiles.find(p => p.name === relayName)
  const canSaveRelay = Boolean(relayName.trim() && relayAddr.trim() && relayToken.trim() && relayBusy === null)
  const protocolCounts = profiles.reduce<Record<string, number>>((acc, profile) => {
    const protocol = (profile.protocol || 'unknown').toUpperCase()
    acc[protocol] = (acc[protocol] || 0) + 1
    return acc
  }, {})
  const protocolSummary = Object.entries(protocolCounts)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 4)
  const filters: Array<{id: NodeFilter; label: string; count: number}> = [
    {id: 'all', label: '全部', count: counts.all},
    {id: 'usable', label: '可连接', count: counts.usable},
    {id: 'current', label: '当前', count: counts.current},
    {id: 'domestic', label: '国内', count: counts.domestic},
    {id: 'unsupported', label: '不支持', count: counts.unsupported},
  ]

  const clearRelayEditor = () => {
    setRelayName('')
    setRelayAddr('')
    setRelayToken('')
    setRelayTLSEnabled(false)
    setRelayServerName('')
    setRelayCAFile('')
    setRelayInsecure(false)
  }

  const editRelayProfile = (profile: RelayProfile) => {
    setRelayName(profile.name)
    setRelayAddr(profile.relay_addr)
    setRelayToken(profile.token)
    setRelayTLSEnabled(Boolean(profile.tls?.enabled))
    setRelayServerName(profile.tls?.server_name || '')
    setRelayCAFile(profile.tls?.ca_file || '')
    setRelayInsecure(Boolean(profile.tls?.insecure_skip_verify))
  }

  const handleSaveRelay = async () => {
    if (!relayName.trim()) {
      setMessage('Relay profile 名称不能为空')
      return
    }
    if (!relayAddr.trim()) {
      setMessage('Relay 地址不能为空')
      return
    }
    if (!relayToken.trim()) {
      setMessage('Relay token 不能为空')
      return
    }
    try {
      setRelayBusy('save')
      await saveRelayProfile({
        name: relayName.trim(),
        relay_addr: relayAddr.trim(),
        token: relayToken,
        replace: relayReplace,
        tls: {
          enabled: relayTLSEnabled,
          server_name: relayServerName.trim(),
          ca_file: relayCAFile.trim(),
          insecure_skip_verify: relayInsecure,
        },
      })
      setMessage('Relay profile 已保存')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setRelayBusy(null)
    }
  }

  const handleSelectRelay = async (name: string) => {
    try {
      setRelayBusy(`select:${name}`)
      await selectRelayProfile(name)
      setMessage(`已选择 Relay profile ${name}`)
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setRelayBusy(null)
    }
  }

  const handleCheckRelay = async (name: string) => {
    try {
      setRelayBusy(`check:${name}`)
      const result = await checkRelayProfile(name)
      setMessage(result?.message || `${name} 检测完成`)
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setRelayBusy(null)
    }
  }

  const handleDeleteRelay = async (name: string) => {
    if (!confirm(`删除 Relay profile ${name}？`)) return
    try {
      setRelayBusy(`delete:${name}`)
      await deleteRelayProfile(name)
      if (relayName === name) {
        clearRelayEditor()
      }
      setMessage(`已删除 Relay profile ${name}`)
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setRelayBusy(null)
    }
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-6 xl:grid-cols-[1.1fr_1.9fr]">
        <div className="panel p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="icon-tile h-8 w-8 text-emerald-700"><CpuIcon className="h-4 w-4" /></span>
              <h3 className="text-base font-semibold text-main">代理模式</h3>
            </div>
            <span className="pill px-2.5 py-1 text-xs">{autoCandidateCount} 个候选</span>
          </div>
          <div className="space-y-2">
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <div className="text-sm font-medium text-main">自动选择</div>
                <div className="mt-1 text-xs text-subtle">测速后切换到最快可连接节点</div>
              </div>
              <button
                onClick={handleCheckBest}
                disabled={checkingName !== null || autoCandidateCount === 0}
                className="primary-button shrink-0 px-3 py-1.5 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400"
              >
                <ZapIcon className="h-4 w-4" />
                {checkingName === '__best__' ? '测速中' : '测速'}
              </button>
            </div>
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <div className="text-sm font-medium text-main">手动选择</div>
                <div className="mt-1 truncate text-xs text-subtle">{activeProfile?.name || config?.active_proxy_profile || '未选择节点'}</div>
              </div>
              <span className="rounded-full border border-emerald-500/20 bg-emerald-50 px-2.5 py-1 text-xs text-emerald-700">
                {activeProfile?.protocol ? activeProfile.protocol.toUpperCase() : 'IDLE'}
              </span>
            </div>
            <div className="row-surface flex items-center justify-between gap-3 p-3">
              <div className="min-w-0">
                <div className="text-sm font-medium text-main">兼容性</div>
                <div className="mt-1 text-xs text-subtle">{counts.usable} 可连接 · {counts.unsupported} 不支持</div>
              </div>
              <ShieldIcon className="h-4 w-4 text-faint" />
            </div>
          </div>
        </div>

        <div className="panel p-5">
          <div className="mb-4 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold text-main">节点概览</h3>
              <p className="mt-1 text-xs text-subtle">{filteredProfiles.length} / {profiles.length} 个节点正在显示</p>
            </div>
            <span className="pill px-2.5 py-1 text-xs">{config?.active_proxy_profile || '未选择'}</span>
          </div>
          <div className="grid gap-3 md:grid-cols-5">
            {[
              ['全部节点', counts.all],
              ['可连接', counts.usable],
              ['自动候选', autoCandidateCount],
              ['不支持', counts.unsupported],
              ['Relay', relayProfiles.length],
            ].map(([label, value]) => (
              <div key={label} className="panel-soft p-3">
                <div className="text-xs text-faint">{label}</div>
                <div className="mt-2 text-2xl font-semibold text-main">{value}</div>
              </div>
            ))}
          </div>
          <div className="mt-4 flex flex-wrap gap-2">
            {protocolSummary.length === 0 ? (
              <span className="pill px-2.5 py-1 text-xs">暂无协议</span>
            ) : protocolSummary.map(([protocol, count]) => (
              <span key={protocol} className={`rounded-full border px-2.5 py-1 text-xs ${protocolTone(protocol)}`}>
                {protocol} {count}
              </span>
            ))}
          </div>
        </div>
      </div>

      <div className="panel p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <label className="relative min-w-64 flex-1 md:flex-none">
            <SearchIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
            <input
              placeholder="搜索节点或协议"
              value={search}
              onChange={e => setSearch(e.target.value)}
              className="form-control w-full py-2 pl-9 pr-3 text-sm"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            {filters.map(f => (
              <button
                key={f.id}
                onClick={() => setFilter(f.id)}
                className={`rounded-full border px-3 py-1.5 text-sm transition-colors ${
                  filter === f.id
                    ? 'border-[#0b8a7e] bg-[#0b8a7e] text-white'
                    : 'pill hover:bg-white/90'
                }`}
              >
                {f.label} {f.count}
              </button>
            ))}
          </div>
        </div>
      </div>

      <div className="panel overflow-x-auto">
        <div className="grid grid-cols-[minmax(18rem,1fr)_7rem_8rem_8rem_17rem] gap-4 border-b border-[#ded8f5] px-4 py-3 text-xs font-medium text-faint">
          <div>节点</div>
          <div>协议</div>
          <div>连接能力</div>
          <div>自动选优</div>
          <div className="text-right">操作</div>
        </div>
        {filteredProfiles.length === 0 ? (
          <div className="p-8 text-center text-subtle">没有匹配的机场节点</div>
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
            const protocol = (profile.protocol || '-').toUpperCase()

            return (
              <div
                key={profile.name}
                className={`grid grid-cols-[minmax(18rem,1fr)_7rem_8rem_8rem_17rem] items-center gap-4 border-b border-[#ded8f5] px-4 py-3 transition-colors last:border-b-0 ${
                  isCurrent
                    ? 'bg-emerald-50/80'
                    : 'hover:bg-white/55'
                }`}
              >
                <div className="flex min-w-0 items-center gap-3">
                  <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-lg border ${
                    exportable ? 'border-[#0b8a7e]/25 bg-emerald-50 text-emerald-700' : 'border-slate-200 bg-white/60 text-faint'
                  }`}>
                    <WifiIcon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate font-medium text-main">
                      {profile.name}
                      {isCurrent && <span className="ml-2 rounded-full bg-[#0b8a7e] px-2 py-0.5 text-xs text-white">当前</span>}
                    </p>
                    <p className={`mt-1 truncate text-xs ${exportable && autoSelectable ? 'text-subtle' : 'text-amber-700'}`}>
                      {compatibility}
                    </p>
                  </div>
                </div>
                <div>
                  <span className={`rounded-full border px-2.5 py-1 text-xs ${protocolTone(protocol)}`}>{protocol}</span>
                </div>
                <div className={exportable ? 'text-sm text-emerald-700' : 'flex items-center gap-1.5 text-sm text-amber-700'}>
                  {!exportable && <AlertIcon className="h-4 w-4" />}
                  {exportable ? '可连接' : '不支持'}
                </div>
                <div className={autoSelectable ? 'text-sm text-main' : 'text-sm text-subtle'}>
                  {autoSelectable ? '参与' : '跳过'}
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => handleSelect(profile.name)}
                    disabled={!exportable}
                    className={`inline-flex min-w-20 items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm transition-colors ${
                      exportable ? 'primary-button' : 'cursor-not-allowed bg-slate-100 text-slate-400'
                    }`}
                  >
                    <CheckIcon className="h-4 w-4" />
                    选择
                  </button>
                  <button
                    onClick={() => handleCheck(profile.name)}
                    disabled={!exportable || checkingName !== null}
                    className={`inline-flex min-w-20 items-center gap-1.5 rounded-lg border px-3 py-1.5 text-sm transition-colors ${
                      exportable
                        ? 'secondary-button disabled:bg-slate-100 disabled:text-slate-400'
                        : 'cursor-not-allowed border-slate-200 bg-slate-100 text-slate-400'
                    }`}
                  >
                    <ZapIcon className="h-4 w-4" />
                    {checkingName === profile.name ? '检测中' : '检测'}
                  </button>
                  <button
                    onClick={() => handleDelete(profile.name)}
                    className="danger-button px-3 py-1.5 text-sm"
                  >
                    <TrashIcon className="h-4 w-4" />
                    删除
                  </button>
                </div>
              </div>
            )
          })
        )}
      </div>

      <div className="grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <div className="panel p-5">
          <div className="mb-5 flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <span className="icon-tile h-8 w-8 text-emerald-700"><ServerIcon className="h-4 w-4" /></span>
              <div>
                <h3 className="text-base font-semibold text-main">Relay profile</h3>
                <p className="mt-1 text-xs text-subtle">保存自建 relay，和 CLI 共用同一份配置</p>
              </div>
            </div>
            <button
              onClick={clearRelayEditor}
              disabled={relayBusy !== null}
              className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
            >
              <PlusIcon className="h-4 w-4" />
              新建
            </button>
          </div>

          <div className="grid gap-4 md:grid-cols-[0.8fr_1.2fr]">
            <label className="block">
              <span className="mb-1 block text-sm text-subtle">名称</span>
              <input
                placeholder="tokyo"
                value={relayName}
                onChange={e => setRelayName(e.target.value)}
                className="form-control w-full px-3 py-2"
              />
            </label>
            <label className="block">
              <span className="mb-1 block text-sm text-subtle">Relay 地址</span>
              <input
                placeholder="relay.example.com:9443"
                value={relayAddr}
                onChange={e => setRelayAddr(e.target.value)}
                className="form-control w-full px-3 py-2"
              />
            </label>
          </div>

          <div className="mt-4">
            <label className="block">
              <span className="mb-1 block text-sm text-subtle">Token</span>
              <div className="relative">
                <LockIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
                <input
                  type="password"
                  value={relayToken}
                  onChange={e => setRelayToken(e.target.value)}
                  className="form-control w-full py-2 pl-9 pr-3"
                />
              </div>
            </label>
          </div>

          <div className="mt-4 grid gap-3 md:grid-cols-2">
            <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
              <span>
                <span className="block font-medium text-main">启用 TLS</span>
                <span className="mt-1 block text-xs text-subtle">公网 relay 建议开启</span>
              </span>
              <input type="checkbox" checked={relayTLSEnabled} onChange={e => setRelayTLSEnabled(e.target.checked)} className="h-4 w-4 accent-[#0b8a7e]" />
            </label>
            <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle">
              <span>
                <span className="block font-medium text-main">覆盖同名</span>
                <span className="mt-1 block text-xs text-subtle">保存时更新已有 profile</span>
              </span>
              <input type="checkbox" checked={relayReplace} onChange={e => setRelayReplace(e.target.checked)} className="h-4 w-4 accent-[#0b8a7e]" />
            </label>
          </div>

          {relayTLSEnabled && (
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              <input
                placeholder="TLS ServerName"
                value={relayServerName}
                onChange={e => setRelayServerName(e.target.value)}
                className="form-control px-3 py-2"
              />
              <input
                placeholder="TLS CA 文件"
                value={relayCAFile}
                onChange={e => setRelayCAFile(e.target.value)}
                className="form-control px-3 py-2"
              />
              <label className="row-surface flex items-center justify-between gap-3 p-3 text-sm text-subtle md:col-span-2">
                <span>
                  <span className="block font-medium text-main">跳过证书校验</span>
                  <span className="mt-1 block text-xs text-subtle">只建议临时调试使用</span>
                </span>
                <input type="checkbox" checked={relayInsecure} onChange={e => setRelayInsecure(e.target.checked)} className="h-4 w-4 accent-[#d97706]" />
              </label>
            </div>
          )}

          {selectedRelayProfile && (
            <div className="mt-4 rounded-lg border border-emerald-500/20 bg-emerald-50 p-3 text-sm text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200">
              正在编辑已保存 profile：{selectedRelayProfile.name}
            </div>
          )}

          <div className="mt-5 flex flex-wrap justify-end gap-3">
            <button
              onClick={handleSaveRelay}
              disabled={!canSaveRelay}
              className="primary-button px-4 py-2 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400 disabled:shadow-none"
            >
              <SaveIcon className="h-4 w-4" />
              {relayBusy === 'save' ? '保存中' : '保存 profile'}
            </button>
            <button
              onClick={() => relayName && handleSelectRelay(relayName)}
              disabled={!selectedRelayProfile || relayBusy !== null}
              className="secondary-button px-4 py-2 text-sm disabled:bg-slate-200 disabled:text-slate-400"
            >
              <CheckIcon className="h-4 w-4" />
              选择当前
            </button>
          </div>
        </div>

        <div className="panel p-5">
          <div className="mb-5 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold text-main">已保存 Relay</h3>
              <p className="mt-1 text-xs text-subtle">{activeRelayProfile ? `当前 ${activeRelayProfile.name}` : '未选择 relay profile'}</p>
            </div>
            <span className="pill px-2.5 py-1 text-xs">{relayProfiles.length} 个 profile</span>
          </div>

          {relayProfiles.length === 0 ? (
            <div className="rounded-lg border border-dashed border-[#cfd6e3] p-8 text-center text-subtle dark:border-white/10">
              暂无 Relay profile。使用机场订阅时可以先不配置。
            </div>
          ) : (
            <div className="space-y-3">
              {relayProfiles.map(profile => {
                const isActive = profile.name === config?.active_profile
                const tlsEnabled = Boolean(profile.tls?.enabled)
                return (
                  <div key={profile.name} className={`row-surface p-3 ${isActive ? 'ring-2 ring-[#0b8a7e]/20' : ''}`}>
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex min-w-0 items-start gap-3">
                        <div className={`grid h-10 w-10 shrink-0 place-items-center rounded-lg border ${
                          isActive ? 'border-[#0b8a7e]/25 bg-emerald-50 text-emerald-700' : 'border-slate-200 bg-white/60 text-faint'
                        }`}>
                          <ServerIcon className="h-4 w-4" />
                        </div>
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="truncate font-semibold text-main">{profile.name}</p>
                            {isActive && <span className="rounded-full bg-[#0b8a7e] px-2 py-0.5 text-xs text-white">当前</span>}
                            <span className={`rounded-full border px-2 py-0.5 text-xs ${
                              tlsEnabled
                                ? 'border-emerald-500/20 bg-emerald-50 text-emerald-700'
                                : 'border-slate-200 bg-slate-50 text-subtle'
                            }`}>
                              {tlsEnabled ? 'TLS' : '明文'}
                            </span>
                          </div>
                          <p className="mt-2 truncate text-sm text-subtle">{profile.relay_addr}</p>
                        </div>
                      </div>
                    </div>
                    <div className="mt-3 flex flex-wrap justify-end gap-2">
                      <button
                        onClick={() => editRelayProfile(profile)}
                        className="secondary-button px-3 py-1.5 text-sm"
                      >
                        <EditIcon className="h-4 w-4" />
                        编辑
                      </button>
                      <button
                        onClick={() => handleSelectRelay(profile.name)}
                        disabled={relayBusy !== null}
                        className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                      >
                        <CheckIcon className="h-4 w-4" />
                        {relayBusy === `select:${profile.name}` ? '选择中' : '选择'}
                      </button>
                      <button
                        onClick={() => handleCheckRelay(profile.name)}
                        disabled={relayBusy !== null}
                        className="secondary-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                      >
                        <ZapIcon className="h-4 w-4" />
                        {relayBusy === `check:${profile.name}` ? '检测中' : '检测'}
                      </button>
                      <button
                        onClick={() => handleDeleteRelay(profile.name)}
                        disabled={relayBusy !== null}
                        className="danger-button px-3 py-1.5 text-sm disabled:bg-slate-200 disabled:text-slate-400"
                      >
                        <TrashIcon className="h-4 w-4" />
                        {relayBusy === `delete:${profile.name}` ? '删除中' : '删除'}
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
