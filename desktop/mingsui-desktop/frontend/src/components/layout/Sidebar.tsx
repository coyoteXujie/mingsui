import type {ComponentType} from 'react'
import {FiActivity, FiCloud, FiGlobe, FiHome, FiSettings, FiTerminal} from 'react-icons/fi'

const asIcon = (Icon: unknown) => Icon as ComponentType<{className?: string}>
const BrandIcon = asIcon(FiActivity)

function Sidebar({activeView, onViewChange}: {activeView: string; onViewChange: (v: string) => void}) {
  const navItems = [
    {id: 'overview', label: '总览', icon: asIcon(FiHome)},
    {id: 'nodes', label: '节点', icon: asIcon(FiGlobe)},
    {id: 'subscriptions', label: '订阅', icon: asIcon(FiCloud)},
    {id: 'logs', label: '日志', icon: asIcon(FiTerminal)},
    {id: 'settings', label: '设置', icon: asIcon(FiSettings)},
  ]

  return (
    <nav className="w-56 h-screen bg-[#1e1e1e] border-r border-[#333] flex flex-col">
      <div className="p-4 border-b border-[#333]">
        <div className="flex items-center gap-2">
          <div className="grid h-8 w-8 place-items-center rounded-lg bg-[#0b6f65] text-white">
            <BrandIcon />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">明隧</h1>
            <p className="text-xs text-gray-400 mt-1">AI proxy toolkit</p>
          </div>
        </div>
      </div>
      <div className="flex-1 py-2">
        {navItems.map(item => {
          const Icon = item.icon
          return (
          <button
            key={item.id}
            onClick={() => onViewChange(item.id)}
            className={`w-full px-4 py-3 flex items-center gap-3 text-left transition-colors ${
              activeView === item.id
                ? 'bg-[#0b6f65] text-white'
                : 'text-gray-300 hover:bg-[#2a2a2a]'
              }`}
          >
            <Icon className="h-4 w-4" />
            <span>{item.label}</span>
          </button>
        )})}
      </div>
    </nav>
  )
}

export {Sidebar}
