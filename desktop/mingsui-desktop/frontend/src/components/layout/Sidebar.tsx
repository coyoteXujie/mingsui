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
    {id: 'logs', label: '日志', icon: asIcon(FiTerminal)},
    {id: 'settings', label: '设置', icon: asIcon(FiSettings)},
  ]

  const status = state?.status
  const metrics = status?.metrics
  const running = Boolean(status?.running)
  const currentNode = state?.config?.active_proxy_profile || state?.config?.active_profile || '未选择'

  return (
    <nav className="sidebar-shell flex h-screen w-64 flex-col">
      <div className="p-4">
        <div className="mb-4 flex items-center gap-3">
          <div className="grid h-10 w-10 place-items-center rounded-lg bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15">
            <BrandIcon className="h-5 w-5" />
          </div>
          <div className="min-w-0">
            <h1 className="text-xl font-bold text-main">明隧</h1>
            <p className="mt-0.5 truncate text-xs text-subtle">MingSui Desktop</p>
          </div>
        </div>

        <div className="rounded-lg border border-[#d4dbe8] bg-white/80 p-3 shadow-sm dark:border-white/10 dark:bg-white/10">
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className={`h-2.5 w-2.5 rounded-full ${running ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                <span className="text-sm font-semibold text-main">{running ? '已连接' : '未连接'}</span>
              </div>
              <div className="mt-1 truncate text-xs text-subtle">{currentNode}</div>
            </div>
            <span className={`rounded-full px-2 py-1 text-xs ${
              state?.system_proxy?.enabled
                ? 'bg-emerald-50 text-emerald-700'
                : 'bg-slate-100 text-subtle dark:bg-white/10'
            }`}>
              {state?.system_proxy?.enabled ? '系统代理' : '手动'}
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

      <div className="grid grid-cols-2 gap-2 px-3 pb-3">
        {navItems.map(item => {
          const Icon = item.icon
          return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`flex min-h-20 flex-col items-start justify-between rounded-lg border p-3 text-left text-sm transition-colors ${
              activeView === item.id
                ? 'border-[#0b8a7e] bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15'
                : 'border-[#dbe1eb] bg-white/60 text-[#5b6477] hover:border-[#0b8a7e]/35 hover:bg-white hover:text-[#253044] dark:border-white/10 dark:bg-white/5 dark:text-[#aeb7c7] dark:hover:bg-white/10 dark:hover:text-white'
              }`}
          >
            <Icon className="h-4 w-4" />
            <span>{item.label}</span>
          </button>
        )})}
      </div>

      <div className="flex-1 px-3">
        <div className="grid gap-2">
          <div className="rounded-lg border border-[#dbe1eb] bg-white/60 p-3 dark:border-white/10 dark:bg-white/5">
            <div className="mb-2 flex items-center gap-2 text-xs text-faint">
              <ZapIcon className="h-4 w-4" />
              连接概览
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <div className="text-faint">活跃</div>
                <div className="mt-1 font-semibold text-main">{metrics?.active_connections || 0}</div>
              </div>
              <div>
                <div className="text-faint">总计</div>
                <div className="mt-1 font-semibold text-main">{metrics?.total_connections || 0}</div>
              </div>
            </div>
          </div>
          <div className="rounded-lg border border-[#dbe1eb] bg-white/60 p-3 dark:border-white/10 dark:bg-white/5">
            <div className="mb-2 flex items-center gap-2 text-xs text-faint">
              <WifiIcon className="h-4 w-4" />
              节点来源
            </div>
            <div className="grid grid-cols-2 gap-2 text-xs">
              <div>
                <div className="text-faint">节点</div>
                <div className="mt-1 font-semibold text-main">{state?.config?.proxy_profiles?.length || 0}</div>
              </div>
              <div>
                <div className="text-faint">订阅</div>
                <div className="mt-1 font-semibold text-main">{state?.config?.subscriptions?.length || 0}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="border-t border-[#dbe1eb] p-4 dark:border-white/10">
        <div className="flex items-center gap-2 text-xs text-faint">
          <CpuIcon className="h-4 w-4" />
          配置
        </div>
        <div className="mt-2 line-clamp-2 break-all text-xs text-subtle">{state?.config_path || '未加载'}</div>
      </div>
    </nav>
  )
}

export {Sidebar}
