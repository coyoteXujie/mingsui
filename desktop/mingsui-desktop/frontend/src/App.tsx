import {useState} from 'react'
import {Sidebar} from './components/layout/Sidebar'
import {ThemeToggle} from './components/common/ThemeToggle'
import {Overview} from './components/views/Overview'
import {Nodes} from './components/views/Nodes'
import {Subscriptions} from './components/views/Subscriptions'
import {Logs} from './components/views/Logs'
import {Settings} from './components/views/Settings'
import {useDesktop} from './hooks/useDesktop'
import {FiAlertCircle, FiRefreshCw} from 'react-icons/fi'
import type {ComponentType} from 'react'

type ViewType = 'overview' | 'nodes' | 'subscriptions' | 'logs' | 'settings'

const AlertIcon = FiAlertCircle as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>

const viewTitles: Record<ViewType, {title: string; detail: string}> = {
  overview: {title: '总览', detail: '连接状态、系统代理和快速导入'},
  nodes: {title: '节点', detail: '机场节点、Relay profile 和测速选优'},
  subscriptions: {title: '订阅', detail: '保存、同步和维护节点来源'},
  logs: {title: '诊断', detail: '运行状态、日志和 CLI 排障命令'},
  settings: {title: '设置', detail: '本地监听、认证、TLS 和默认 relay'},
}

function BackendErrorBanner({error, loading, onRetry}: {error: string; loading: boolean; onRetry: () => void}) {
  return (
    <div className="mb-5 rounded-lg border border-red-500/20 bg-red-50 p-4 text-red-800 shadow-sm dark:bg-red-500/10 dark:text-red-200">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <span className="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-red-500/25 bg-white/70 dark:bg-white/10">
            <AlertIcon className="h-4 w-4" />
          </span>
          <div className="min-w-0">
            <div className="text-sm font-semibold">桌面后端连接异常</div>
            <div className="mt-1 break-words text-sm leading-6 opacity-90">{error}</div>
          </div>
        </div>
        <button
          onClick={onRetry}
          disabled={loading}
          className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-current/20 bg-white/70 px-3 py-1.5 text-sm font-medium transition hover:bg-white disabled:cursor-not-allowed disabled:opacity-60 dark:bg-white/10 dark:hover:bg-white/15"
        >
          <RefreshIcon className="h-4 w-4" />
          {loading ? '重试中' : '重试'}
        </button>
      </div>
    </div>
  )
}

function App() {
  const [activeView, setActiveView] = useState<ViewType>('overview')
  const {state, error, loading, refresh} = useDesktop()
  const view = viewTitles[activeView]
  const status = state?.status
  const config = state?.config
  const currentNode = config?.active_proxy_profile || config?.active_profile || '未选择节点'
  const nodeCount = config?.proxy_profiles?.length || 0
  const subscriptionCount = config?.subscriptions?.length || 0

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
          <header className="topbar sticky top-0 z-10 px-6 py-3.5">
            <div className="flex items-center justify-between gap-4">
              <div className="min-w-0">
                <div className="flex items-center gap-2 text-xs text-faint">
                  <span>明隧桌面端</span>
                  <span>/</span>
                  <span>{view.title}</span>
                </div>
                <div>
                  <h1 className="mt-1 text-lg font-semibold text-main">{view.title}</h1>
                  <p className="mt-0.5 truncate text-xs text-subtle">{view.detail}</p>
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <div className="hidden items-center gap-2 rounded-lg border border-[#dbe1eb] bg-white/60 px-3 py-2 text-xs text-subtle dark:border-white/10 dark:bg-white/5 xl:flex">
                  <span className={`h-2 w-2 rounded-full ${status?.running ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                  <span className="max-w-52 truncate text-main">{currentNode}</span>
                </div>
                <div className="hidden gap-2 text-xs lg:flex">
                  <span className="rounded-lg border border-[#dbe1eb] bg-white/60 px-2.5 py-2 text-subtle dark:border-white/10 dark:bg-white/5">{nodeCount} 节点</span>
                  <span className="rounded-lg border border-[#dbe1eb] bg-white/60 px-2.5 py-2 text-subtle dark:border-white/10 dark:bg-white/5">{subscriptionCount} 订阅</span>
                </div>
                <button
                  onClick={() => refresh(true)}
                  disabled={loading}
                  title="刷新桌面状态"
                  className="secondary-button h-9 w-9 p-0 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <RefreshIcon className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
                </button>
                <div className={`rounded-lg border px-3 py-2 text-sm ${
                  status?.running
                    ? 'border-emerald-500/30 bg-emerald-50 text-emerald-700'
                    : 'border-slate-200 bg-white/60 text-subtle'
                }`}>
                  {status?.running ? '已连接' : '未连接'}
                </div>
                <ThemeToggle />
              </div>
            </div>
          </header>
          <div className="p-6">
            {error && <BackendErrorBanner error={error} loading={loading} onRetry={() => refresh(true)} />}
            {renderView()}
          </div>
        </main>
      </div>
    </div>
  )
}

export default App
