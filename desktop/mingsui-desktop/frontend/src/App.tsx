import {useState} from 'react'
import {Sidebar} from './components/layout/Sidebar'
import {ThemeToggle} from './components/common/ThemeToggle'
import {Overview} from './components/views/Overview'
import {Nodes} from './components/views/Nodes'
import {Subscriptions} from './components/views/Subscriptions'
import {Logs} from './components/views/Logs'
import {Settings} from './components/views/Settings'
import {useDesktop} from './hooks/useDesktop'

type ViewType = 'overview' | 'nodes' | 'subscriptions' | 'logs' | 'settings'

const viewTitles: Record<ViewType, {title: string; detail: string}> = {
  overview: {title: '总览', detail: '连接状态、系统代理和快速导入'},
  nodes: {title: '节点', detail: '机场节点、Relay profile 和测速选优'},
  subscriptions: {title: '订阅', detail: '保存、同步和维护节点来源'},
  logs: {title: '日志', detail: '桌面端运行日志和内核输出'},
  settings: {title: '设置', detail: '本地监听、认证、TLS 和默认 relay'},
}

function App() {
  const [activeView, setActiveView] = useState<ViewType>('overview')
  const {state} = useDesktop()
  const view = viewTitles[activeView]

  const renderView = () => {
    switch (activeView) {
      case 'overview': return <Overview />
      case 'nodes': return <Nodes />
      case 'subscriptions': return <Subscriptions />
      case 'logs': return <Logs />
      case 'settings': return <Settings />
      default: return <Overview />
    }
  }

  return (
    <div className="app-shell">
      <div className="flex min-h-screen">
        <Sidebar activeView={activeView} onViewChange={(view) => setActiveView(view as ViewType)} state={state} />
        <main className="min-w-0 flex-1 overflow-auto">
          <header className="topbar sticky top-0 z-10 px-6 py-4">
            <div className="flex items-center justify-between gap-4">
              <div>
                <h1 className="text-xl font-semibold text-main">{view.title}</h1>
                <p className="mt-1 text-sm text-subtle">{view.detail}</p>
              </div>
              <div className="flex items-center gap-3">
                <div className={`rounded-full border px-3 py-1 text-sm ${
                  state?.status?.running
                    ? 'border-emerald-500/30 bg-emerald-50 text-emerald-700'
                    : 'border-slate-200 bg-white/60 text-subtle'
                }`}>
                  {state?.status?.running ? '已连接' : '未连接'}
                </div>
                <ThemeToggle />
              </div>
            </div>
          </header>
          <div className="p-6">
            {renderView()}
          </div>
        </main>
      </div>
    </div>
  )
}

export default App
