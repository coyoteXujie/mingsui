import type {ComponentType} from 'react'
import {FiActivity, FiCloud, FiGlobe, FiHome, FiSettings, FiTerminal} from 'react-icons/fi'
import type {AppState} from '../../hooks/useDesktop'

const asIcon = (Icon: unknown) => Icon as ComponentType<{className?: string}>
const BrandIcon = asIcon(FiActivity)

function Sidebar({activeView, onViewChange, state}: {activeView: string; onViewChange: (v: string) => void; state: AppState | null}) {
  const navItems = [
    {id: 'overview', label: '总览', icon: asIcon(FiHome)},
    {id: 'nodes', label: '节点', icon: asIcon(FiGlobe)},
    {id: 'subscriptions', label: '订阅', icon: asIcon(FiCloud)},
    {id: 'logs', label: '日志', icon: asIcon(FiTerminal)},
    {id: 'settings', label: '设置', icon: asIcon(FiSettings)},
  ]

  return (
    <nav className="sidebar-shell flex h-screen w-60 flex-col">
      <div className="border-b border-[#ded8f5] p-4">
        <div className="flex items-center gap-2">
          <div className="grid h-9 w-9 place-items-center rounded-lg bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15">
            <BrandIcon />
          </div>
          <div>
            <h1 className="text-xl font-bold text-main">明隧</h1>
            <p className="mt-1 text-xs text-subtle">AI proxy toolkit</p>
          </div>
        </div>
      </div>
      <div className="flex-1 space-y-1 p-2">
        {navItems.map(item => {
          const Icon = item.icon
          return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm transition-colors ${
              activeView === item.id
                ? 'bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15'
                : 'text-[#5b6477] hover:bg-white/70 hover:text-[#253044]'
              }`}
          >
            <Icon className="h-4 w-4" />
            <span>{item.label}</span>
          </button>
        )})}
      </div>
      <div className="border-t border-[#ded8f5] p-4">
        <div className="panel-soft space-y-3 p-3 text-xs">
          <div>
            <div className="mb-1 text-faint">配置</div>
            <div className="break-all text-subtle">{state?.config_path || '未加载'}</div>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <div>
              <div className="text-faint">HTTP</div>
              <div className="mt-1 text-main">{state?.status?.http_addr || '-'}</div>
            </div>
            <div>
              <div className="text-faint">SOCKS5</div>
              <div className="mt-1 text-main">{state?.status?.local_addr || '-'}</div>
            </div>
          </div>
          <div className="flex items-center justify-between border-t border-[#ded8f5] pt-3">
            <span className="text-faint">系统代理</span>
            <span className={state?.system_proxy?.enabled ? 'text-emerald-700' : 'text-subtle'}>
              {state?.system_proxy?.supported ? (state.system_proxy.enabled ? '已开启' : '未开启') : '不支持'}
            </span>
          </div>
        </div>
      </div>
    </nav>
  )
}

export {Sidebar}
