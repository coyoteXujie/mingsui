import type {ComponentType} from 'react'
import {FiActivity, FiCloud, FiCpu, FiGlobe, FiHome, FiSettings, FiTerminal, FiWifi, FiZap} from 'react-icons/fi'
import type {AppState} from '../../hooks/useDesktop'

const asIcon = (Icon: unknown) => Icon as ComponentType<{className?: string}>
const BrandIcon = asIcon(FiActivity)
const CpuIcon = asIcon(FiCpu)
const WifiIcon = asIcon(FiWifi)
const ZapIcon = asIcon(FiZap)

function Sidebar({activeView, onViewChange, state}: {activeView: string; onViewChange: (v: string) => void; state: AppState | null}) {
  const navItems = [
    {id: 'overview', label: '总览', icon: asIcon(FiHome)},
    {id: 'nodes', label: '节点', icon: asIcon(FiGlobe)},
    {id: 'subscriptions', label: '订阅', icon: asIcon(FiCloud)},
    {id: 'logs', label: '诊断', icon: asIcon(FiTerminal)},
    {id: 'settings', label: '设置', icon: asIcon(FiSettings)},
  ]

  const status = state?.status
  const metrics = status?.metrics
  const running = Boolean(status?.running)
  const currentNode = state?.config?.active_proxy_profile || state?.config?.active_profile || '未选择'
  const systemProxyEnabled = Boolean(state?.system_proxy?.enabled)

  return (
    <nav className="sidebar-shell flex h-screen w-72 flex-col">
      <div className="px-4 py-4">
        <div className="flex items-center gap-3">
          <div className="grid h-9 w-9 place-items-center rounded-lg bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15">
            <BrandIcon className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <h1 className="text-base font-semibold text-main">明隧</h1>
            <p className="mt-0.5 truncate text-xs text-subtle">AI Proxy Desktop</p>
          </div>
        </div>

        <div className="mt-4 rounded-lg border border-[#d4dbe8] bg-white/80 p-3 shadow-sm dark:border-white/10 dark:bg-white/10">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2 text-sm font-semibold text-main">
                <span className={`h-2.5 w-2.5 rounded-full ${running ? 'bg-emerald-500 shadow-[0_0_0_4px_rgba(16,185,129,0.12)]' : 'bg-slate-300'}`} />
                {running ? '代理运行中' : '代理未启动'}
              </div>
              <div className="mt-2 truncate text-xs text-subtle">{currentNode}</div>
            </div>
            <span className={`rounded-full border px-2 py-1 text-xs ${
              systemProxyEnabled
                ? 'bg-emerald-50 text-emerald-700'
                : 'bg-slate-100 text-subtle dark:bg-white/10'
            }`}>
              {systemProxyEnabled ? '系统' : '手动'}
            </span>
          </div>
          <div className="mt-3 grid grid-cols-2 gap-2 text-xs">
            <div className="rounded-lg border border-[#e0e5ee] bg-[#f8fafc] p-2 dark:border-white/10 dark:bg-white/5">
              <div className="text-faint">HTTP</div>
              <div className="mt-1 truncate font-medium text-main">{status?.http_addr || state?.config?.http_addr || '-'}</div>
            </div>
            <div className="rounded-lg border border-[#e0e5ee] bg-[#f8fafc] p-2 dark:border-white/10 dark:bg-white/5">
              <div className="text-faint">SOCKS</div>
              <div className="mt-1 truncate font-medium text-main">{status?.local_addr || state?.config?.local_addr || '-'}</div>
            </div>
          </div>
        </div>
      </div>

      <div className="px-3 pb-3">
        <div className="mb-2 px-2 text-xs font-medium text-faint">工作区</div>
        <div className="space-y-1">
        {navItems.map(item => {
          const Icon = item.icon
          const active = activeView === item.id
          return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`group flex h-10 w-full items-center gap-3 rounded-lg border px-3 text-left text-sm transition-colors ${
              active
                ? 'border-[#0b8a7e]/45 bg-[#e7f7f3] text-[#0b6f65] shadow-sm dark:bg-white/10 dark:text-white'
                : 'border-transparent text-[#5b6477] hover:border-[#dbe1eb] hover:bg-white/62 hover:text-[#253044] dark:text-[#aeb7c7] dark:hover:border-white/10 dark:hover:bg-white/8 dark:hover:text-white'
              }`}
          >
            <span className={`grid h-7 w-7 shrink-0 place-items-center rounded-md ${
              active ? 'bg-[#0b8a7e] text-white' : 'bg-white/70 text-[#788197] group-hover:text-[#0b8a7e] dark:bg-white/6'
            }`}>
              <Icon className="h-4 w-4" />
            </span>
            <span className="font-medium">{item.label}</span>
          </button>
        )})}
        </div>
      </div>

      <div className="flex-1 overflow-auto px-3">
        <div className="mb-2 px-2 text-xs font-medium text-faint">实时状态</div>
        <div className="space-y-2">
          <div className="rounded-lg border border-[#dbe1eb] bg-white/60 p-3 dark:border-white/10 dark:bg-white/5">
            <div className="mb-3 flex items-center gap-2 text-xs text-faint">
              <ZapIcon className="h-4 w-4" />
              连接概览
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div>
                <div className="text-faint">活跃</div>
                <div className="mt-1 font-semibold text-main">{metrics?.active_connections || 0}</div>
              </div>
              <div>
                <div className="text-faint">总计</div>
                <div className="mt-1 font-semibold text-main">{metrics?.total_connections || 0}</div>
              </div>
              <div>
                <div className="text-faint">状态</div>
                <div className="mt-1 font-semibold text-main">{running ? 'ON' : 'OFF'}</div>
              </div>
            </div>
          </div>
          <div className="rounded-lg border border-[#dbe1eb] bg-white/60 p-3 dark:border-white/10 dark:bg-white/5">
            <div className="mb-3 flex items-center gap-2 text-xs text-faint">
              <WifiIcon className="h-4 w-4" />
              节点来源
            </div>
            <div className="grid grid-cols-3 gap-2 text-xs">
              <div>
                <div className="text-faint">节点</div>
                <div className="mt-1 font-semibold text-main">{state?.config?.proxy_profiles?.length || 0}</div>
              </div>
              <div>
                <div className="text-faint">订阅</div>
                <div className="mt-1 font-semibold text-main">{state?.config?.subscriptions?.length || 0}</div>
              </div>
              <div>
                <div className="text-faint">Relay</div>
                <div className="mt-1 font-semibold text-main">{state?.config?.profiles?.length || 0}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="border-t border-[#dbe1eb] p-4 dark:border-white/10">
        <div className="flex items-center gap-2 text-xs text-faint">
          <CpuIcon className="h-4 w-4" />
          共享配置
        </div>
        <div className="mt-2 line-clamp-2 break-all text-xs text-subtle">{state?.config_path || '未加载'}</div>
      </div>
    </nav>
  )
}

export {Sidebar}
