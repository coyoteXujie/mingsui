import {useState} from 'react'
import type {ComponentType, ReactNode} from 'react'
import {
  FiActivity,
  FiAlertCircle,
  FiBox,
  FiCheckCircle,
  FiCode,
  FiCommand,
  FiCopy,
  FiCpu,
  FiFileText,
  FiGlobe,
  FiHardDrive,
  FiPower,
  FiRefreshCw,
  FiShield,
  FiTerminal,
  FiUploadCloud,
  FiWifi,
  FiXCircle,
  FiZap,
} from 'react-icons/fi'
import {ClipboardSetText} from '../../../wailsjs/runtime/runtime'
import {useDesktop} from '../../hooks/useDesktop'
import type {ClientConfig, ReadinessAction, ReadinessStatus, RuntimeStatus, SubscriptionSyncResult} from '../../hooks/useDesktop'

const ActivityIcon = FiActivity as ComponentType<{className?: string}>
const AlertIcon = FiAlertCircle as ComponentType<{className?: string}>
const BoxIcon = FiBox as ComponentType<{className?: string}>
const CheckIcon = FiCheckCircle as ComponentType<{className?: string}>
const CodeIcon = FiCode as ComponentType<{className?: string}>
const CommandIcon = FiCommand as ComponentType<{className?: string}>
const CopyIcon = FiCopy as ComponentType<{className?: string}>
const CpuIcon = FiCpu as ComponentType<{className?: string}>
const FileIcon = FiFileText as ComponentType<{className?: string}>
const GlobeIcon = FiGlobe as ComponentType<{className?: string}>
const HardDriveIcon = FiHardDrive as ComponentType<{className?: string}>
const PowerIcon = FiPower as ComponentType<{className?: string}>
const RefreshIcon = FiRefreshCw as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>
const ImportIcon = FiUploadCloud as ComponentType<{className?: string}>
const WifiIcon = FiWifi as ComponentType<{className?: string}>
const XIcon = FiXCircle as ComponentType<{className?: string}>
const ZapIcon = FiZap as ComponentType<{className?: string}>

type Tone = 'success' | 'warning' | 'danger' | 'neutral'

interface StatItem {
  label: string
  value: string
  detail: string
  icon: ReactNode
  tone?: Tone
}

interface TerminalAction {
  label: string
  description: string
  command: string
  icon: ReactNode
  badge: string
  ready: boolean
}

const toneClasses: Record<Tone, string> = {
  success: 'border-emerald-500/20 bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200',
  warning: 'border-amber-500/25 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  danger: 'border-red-500/25 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
  neutral: 'border-slate-200 bg-white/60 text-subtle dark:border-white/10 dark:bg-white/5',
}

const actionToneClasses: Record<ReadinessAction['severity'], string> = {
  info: 'border-sky-500/20 bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-200',
  warning: 'border-amber-500/20 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  error: 'border-red-500/20 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatRuntime(startedAt: string): string {
  const started = Date.parse(startedAt)
  if (!started || Number.isNaN(started)) return '-'
  const seconds = Math.max(0, Math.floor((Date.now() - started) / 1000))
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  return `${hours}h ${minutes % 60}m`
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

function readinessTone(readiness?: ReadinessStatus): Tone {
  if (!readiness) return 'neutral'
  if (!readiness.ok) return 'danger'
  if (readiness.readiness === 'needs_setup') return 'warning'
  return 'success'
}

function readinessLabel(readiness?: ReadinessStatus): string {
  if (!readiness) return '等待检查'
  if (readiness.readiness === 'ready') return '就绪'
  if (readiness.readiness === 'needs_setup') return '待配置'
  return readiness.readiness
}

function modeLabel(readiness: ReadinessStatus | undefined, hasProxy: boolean, hasRelay: boolean): string {
  if (readiness?.mode === 'proxy') return '机场节点模式'
  if (readiness?.mode === 'relay') return 'Relay 模式'
  if (hasProxy) return '机场节点模式'
  if (hasRelay) return 'Relay 模式'
  return '未选择模式'
}

function importResultTitle(result: SubscriptionSyncResult): string {
  if (result.kind === 'proxy') return `${result.imported} 个机场节点`
  if (result.kind === 'relay') return `${result.imported} 个 relay profile`
  return `${result.imported} 个节点`
}

function importResultDetail(result: SubscriptionSyncResult): string {
  const parts: string[] = []
  if (result.kind === 'proxy') {
    const exportable = result.imported_exportable_proxy_profiles ?? result.exportable_proxy_profiles
    const autoSelectable = result.imported_auto_selectable_proxy_profiles ?? result.auto_selectable_proxy_profiles
    parts.push(`${exportable} 可连接`)
    parts.push(`${autoSelectable} 自动候选`)
  } else if (result.kind === 'relay') {
    parts.push(`${result.relay_profiles} 个 relay profile`)
  }
  if (result.selected) {
    parts.push(`当前 ${result.selected}`)
  }
  if (result.warnings?.length) {
    parts.push(result.warnings.join('；'))
  }
  return parts.join(' · ')
}

function shouldCheckImport(result: SubscriptionSyncResult, importCheck: boolean, importSelect: string): boolean {
  const autoSelectable = result.imported_auto_selectable_proxy_profiles ?? result.auto_selectable_proxy_profiles
  return importCheck && !importSelect.trim() && result.kind === 'proxy' && autoSelectable > 0
}

function StatCard({item}: {item: StatItem}) {
  const tone = item.tone || 'neutral'
  return (
    <div className="row-surface p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs text-faint">{item.label}</div>
          <div className="mt-1 truncate text-sm font-semibold text-main">{item.value}</div>
          <div className="mt-1 truncate text-xs text-subtle">{item.detail}</div>
        </div>
        <span className={`grid h-8 w-8 shrink-0 place-items-center rounded-lg border ${toneClasses[tone]}`}>
          {item.icon}
        </span>
      </div>
    </div>
  )
}

function SectionHeader({icon, title, detail, action}: {icon: ReactNode; title: string; detail: string; action?: ReactNode}) {
  return (
    <div className="mb-4 flex items-start justify-between gap-3">
      <div className="flex items-start gap-3">
        <span className="icon-tile h-9 w-9 shrink-0">{icon}</span>
        <div>
          <h3 className="text-base font-semibold text-main">{title}</h3>
          <p className="mt-1 text-xs text-subtle">{detail}</p>
        </div>
      </div>
      {action}
    </div>
  )
}

export function Overview() {
  const {
    state,
    loading,
    start,
    stop,
    importProfiles,
    checkBestProxy,
    enableSystemProxy,
    disableSystemProxy,
  } = useDesktop()
  const [importContent, setImportContent] = useState('')
  const [importSelect, setImportSelect] = useState('')
  const [importReplace, setImportReplace] = useState(true)
  const [importCheck, setImportCheck] = useState(true)
  const [message, setMessage] = useState('')
  const [connecting, setConnecting] = useState(false)
  const [importing, setImporting] = useState(false)
  const [checkingBest, setCheckingBest] = useState(false)
  const [switchingProxy, setSwitchingProxy] = useState(false)

  const status: RuntimeStatus = state?.status || {running: false, local_addr: '', http_addr: '', relay_addr: '', started_at: '', last_error: ''}
  const config: ClientConfig = state?.config || {
    local_addr: '', http_addr: '', relay_addr: '', token: '', dial_timeout_seconds: 10,
    local_auth: {enabled: false, username: '', password: ''},
    tls: {enabled: false, server_name: '', ca_file: '', insecure_skip_verify: false},
    profiles: [], proxy_profiles: [], subscriptions: [], active_profile: '', active_proxy_profile: ''
  }
  const systemProxy = state?.system_proxy || {supported: false, enabled: false, message: ''}

  const activeProxy = config.proxy_profiles.find(p => p.name === config.active_proxy_profile)
  const nodeLabel = activeProxy ? activeProxy.name : config.active_profile || '未选择节点'
  const metrics = status.metrics || {active_connections: 0, total_connections: 0, upload_bytes: 0, download_bytes: 0}
  const httpAddr = status.http_addr || config.http_addr
  const socksAddr = status.local_addr || config.local_addr
  const relayAddr = status.relay_addr || config.relay_addr
  const httpProxy = httpAddr ? `http://${httpAddr}` : ''
  const socksProxy = socksAddr ? `socks5://${socksAddr}` : ''
  const configFlag = state?.config_path ? ` -config ${shellQuote(state.config_path)}` : ''
  const envCommand = `eval "$(mingsui env${configFlag})"`
  const execTemplate = `mingsui exec${configFlag} -connect -- <command>`
  const testCommand = `mingsui exec${configFlag} -connect -- curl -I https://www.google.com/generate_204`
  const doctorCommand = `mingsui doctor${configFlag} -json`
  const readiness = state?.readiness
  const readinessActions = readiness?.actions?.slice(0, 4) || []
  const warnings = readiness?.warnings || []
  const mode = modeLabel(readiness, Boolean(activeProxy), Boolean(config.active_profile || relayAddr))
  const protocolLabel = activeProxy?.protocol ? activeProxy.protocol.toUpperCase() : config.active_profile ? 'RELAY' : 'IDLE'
  const readyTone = readinessTone(readiness)
  const canImport = Boolean(importContent.trim()) && !importing

  const proxyEnvBlock = [
    httpAddr ? `export HTTP_PROXY="${httpProxy}"` : '',
    httpAddr ? `export HTTPS_PROXY="${httpProxy}"` : '',
    socksAddr ? `export ALL_PROXY="${socksProxy}"` : '',
    httpAddr || socksAddr ? 'export NO_PROXY="localhost,127.0.0.1,::1"' : '',
    httpAddr ? `export http_proxy="${httpProxy}"` : '',
    httpAddr ? `export https_proxy="${httpProxy}"` : '',
    socksAddr ? `export all_proxy="${socksProxy}"` : '',
    httpAddr || socksAddr ? 'export no_proxy="localhost,127.0.0.1,::1"' : '',
  ].filter(Boolean).join('\n')
  const terminalActions: TerminalAction[] = [
    {
      label: '接管当前 Shell',
      description: '适合连续运行 curl、npm、git 或 AI CLI',
      command: envCommand,
      icon: <TerminalIcon className="h-4 w-4" />,
      badge: '持久会话',
      ready: Boolean(httpAddr || socksAddr),
    },
    {
      label: 'AI Agent 单次执行',
      description: '运行前临时连接，命令结束后自动释放',
      command: execTemplate,
      icon: <CommandIcon className="h-4 w-4" />,
      badge: '推荐',
      ready: Boolean(config.active_proxy_profile || config.active_profile || relayAddr),
    },
    {
      label: '连通性测试',
      description: '快速确认当前节点是否能访问外网',
      command: testCommand,
      icon: <CodeIcon className="h-4 w-4" />,
      badge: '诊断',
      ready: Boolean(config.active_proxy_profile || config.active_profile || relayAddr),
    },
    {
      label: 'CLI 诊断',
      description: '输出 readiness、端口和配置问题',
      command: doctorCommand,
      icon: <BoxIcon className="h-4 w-4" />,
      badge: readinessLabel(readiness),
      ready: true,
    },
  ]
  const stats: StatItem[] = [
    {label: '当前节点', value: nodeLabel, detail: protocolLabel, icon: <WifiIcon className="h-4 w-4" />, tone: activeProxy || config.active_profile ? 'success' : 'neutral'},
    {label: 'SOCKS5', value: socksAddr || '-', detail: 'ALL_PROXY', icon: <HardDriveIcon className="h-4 w-4" />, tone: socksAddr ? 'success' : 'warning'},
    {label: 'HTTP', value: httpAddr || '-', detail: 'HTTP_PROXY / HTTPS_PROXY', icon: <GlobeIcon className="h-4 w-4" />, tone: httpAddr ? 'success' : 'warning'},
    {label: '系统代理', value: systemProxy.supported ? (systemProxy.enabled ? '已开启' : '未开启') : '不支持', detail: systemProxy.message || '桌面一键切换', icon: <ShieldIcon className="h-4 w-4" />, tone: systemProxy.enabled ? 'success' : systemProxy.supported ? 'neutral' : 'warning'},
    {label: '活跃连接', value: String(metrics.active_connections), detail: `总计 ${metrics.total_connections}`, icon: <ActivityIcon className="h-4 w-4" />, tone: metrics.active_connections > 0 ? 'success' : 'neutral'},
    {label: '总流量', value: `${formatBytes(metrics.upload_bytes)} / ${formatBytes(metrics.download_bytes)}`, detail: '上传 / 下载', icon: <ZapIcon className="h-4 w-4" />, tone: 'neutral'},
  ]

  const handleConnect = async () => {
    try {
      setConnecting(true)
      if (status.running) {
        await stop()
        setMessage('已断开')
      } else {
        await start()
        setMessage('已连接')
      }
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setConnecting(false)
    }
  }

  const handleSystemProxyToggle = async () => {
    if (!systemProxy.supported) {
      setMessage(systemProxy.message || '当前系统不支持自动切换系统代理')
      return
    }
    try {
      setSwitchingProxy(true)
      if (systemProxy.enabled) {
        await disableSystemProxy()
        setMessage('系统代理已关闭')
      } else {
        await enableSystemProxy()
        setMessage('系统代理已开启')
      }
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setSwitchingProxy(false)
    }
  }

  const handleCheckBest = async () => {
    try {
      setCheckingBest(true)
      const result = await checkBestProxy(10)
      setMessage(result?.message || '测速选优完成')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setCheckingBest(false)
    }
  }

  const handleImport = async () => {
    if (!importContent.trim()) {
      setMessage('请输入订阅内容')
      return
    }
    try {
      setImporting(true)
      const result = await importProfiles(importContent, importReplace, importSelect)
      setImportContent('')
      const summary = importResultDetail(result)
      const prefix = `已导入 ${importResultTitle(result)}${summary ? `；${summary}` : ''}`
      if (!shouldCheckImport(result, importCheck, importSelect)) {
        setMessage(prefix)
        return
      }
      try {
        setCheckingBest(true)
        const checkResult = await checkBestProxy(10)
        setMessage(`${prefix}；${checkResult?.message || '测速选优完成'}`)
      } catch (checkErr: any) {
        setMessage(`${prefix}；测速选优失败：${checkErr.message}`)
      } finally {
        setCheckingBest(false)
      }
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setImporting(false)
    }
  }

  const copyText = async (label: string, text: string) => {
    if (!text.trim()) {
      setMessage(`${label} 暂不可复制`)
      return
    }
    try {
      let ok = false
      if ((window as any).runtime?.ClipboardSetText) {
        ok = await ClipboardSetText(text)
      } else if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(text)
        ok = true
      }
      setMessage(ok ? `已复制 ${label}` : `复制 ${label} 失败`)
    } catch (err: any) {
      setMessage(err.message || `复制 ${label} 失败`)
    }
  }

  if (loading) {
    return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 xl:grid-cols-[minmax(0,1.22fr)_minmax(24rem,0.78fr)]">
        <div className="panel p-5">
          <div className="flex flex-wrap items-start justify-between gap-5">
            <div className="flex min-w-0 items-start gap-4">
              <div className={`grid h-12 w-12 shrink-0 place-items-center rounded-lg border ${
                status.running ? 'border-emerald-500/25 bg-emerald-50 text-emerald-700' : 'border-slate-200 bg-slate-50 text-slate-400'
              }`}>
                {status.running ? <CheckIcon className="h-7 w-7" /> : <XIcon className="h-7 w-7" />}
              </div>
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <h2 className="text-2xl font-semibold text-main">{status.running ? '代理运行中' : '代理未启动'}</h2>
                  <span className="pill px-2.5 py-1 text-xs">{protocolLabel}</span>
                  <span className={`rounded-full border px-2.5 py-1 text-xs ${toneClasses[readyTone]}`}>
                    {readinessLabel(readiness)}
                  </span>
                </div>
                <p className="mt-2 truncate text-sm text-subtle">{nodeLabel} · {mode}</p>
              </div>
            </div>

            <div className="flex shrink-0 flex-wrap gap-2">
              <button
                onClick={handleSystemProxyToggle}
                disabled={switchingProxy || !systemProxy.supported}
                className="secondary-button px-4 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-50"
              >
                <ShieldIcon className="h-4 w-4" />
                {switchingProxy ? '切换中' : systemProxy.enabled ? '关闭系统代理' : '开启系统代理'}
              </button>
              <button
                onClick={handleConnect}
                disabled={connecting}
                className={`inline-flex min-w-28 items-center justify-center gap-2 rounded-lg px-5 py-2 text-sm font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-70 ${
                  status.running
                    ? 'border border-red-500/20 bg-red-50 text-red-700 hover:bg-red-100'
                    : 'bg-[#0b8a7e] text-white shadow-lg shadow-teal-700/15 hover:bg-[#08766d]'
                }`}
              >
                <PowerIcon className="h-4 w-4" />
                {connecting ? '处理中' : status.running ? '断开' : '连接'}
              </button>
            </div>
          </div>

          <div className="mt-5 grid gap-3 md:grid-cols-3">
            <div className="row-surface p-3">
              <div className="text-xs text-faint">运行时长</div>
              <div className="mt-1 text-lg font-semibold text-main">{status.running ? formatRuntime(status.started_at) : '-'}</div>
              <div className="mt-1 text-xs text-subtle">{status.running ? '本地监听已打开' : '等待启动'}</div>
            </div>
            <div className="row-surface p-3">
              <div className="text-xs text-faint">系统代理</div>
              <div className="mt-1 text-lg font-semibold text-main">{systemProxy.enabled ? '已接管' : '未接管'}</div>
              <div className="mt-1 truncate text-xs text-subtle">{systemProxy.message || '桌面端一键切换'}</div>
            </div>
            <div className="row-surface p-3">
              <div className="text-xs text-faint">当前策略</div>
              <div className="mt-1 truncate text-lg font-semibold text-main">{readiness?.selected_proxy || readiness?.selected_profile || nodeLabel}</div>
              <div className="mt-1 text-xs text-subtle">{relayAddr || httpAddr || '等待配置'}</div>
            </div>
          </div>
        </div>

        <div className="panel p-5">
          <div className="flex items-start justify-between gap-3">
            <div>
              <div className="text-xs text-faint">控制台摘要</div>
              <div className="mt-1 text-base font-semibold text-main">节点、订阅和 AI 终端共用同一配置</div>
            </div>
            <span className={`rounded-full border px-2.5 py-1 text-xs ${status.running ? toneClasses.success : toneClasses.neutral}`}>
              {status.running ? '在线' : '离线'}
            </span>
          </div>
          <div className="mt-4 space-y-2">
            {[
              ['机场节点', `${config.proxy_profiles.length} 个`],
              ['订阅来源', `${config.subscriptions.length} 个`],
              ['Relay profile', `${config.profiles.length} 个`],
              ['活跃连接', `${metrics.active_connections} / ${metrics.total_connections}`],
            ].map(([label, value]) => (
              <div key={label} className="row-surface flex items-center justify-between gap-3 px-3 py-2.5">
                <span className="text-sm text-subtle">{label}</span>
                <span className="font-mono text-sm font-semibold text-main">{value}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-6">
        {stats.map(item => <StatCard key={item.label} item={item} />)}
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.35fr)_minmax(22rem,0.65fr)]">
        <div className="panel p-5">
          <SectionHeader
            icon={<AlertIcon className="h-4 w-4" />}
            title="连接检查"
            detail="把配置问题、可执行动作和系统代理状态集中到首页"
            action={
              <span className={`rounded-full border px-2.5 py-1 text-xs ${toneClasses[readyTone]}`}>{readinessLabel(readiness)}</span>
            }
          />

          <div className="rounded-lg border border-[#dbe1eb] bg-white/58 p-4 dark:border-white/10 dark:bg-white/5">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <span className={`grid h-8 w-8 place-items-center rounded-lg border ${toneClasses[readyTone]}`}>
                    {readyTone === 'danger' ? <XIcon className="h-4 w-4" /> : readyTone === 'success' ? <CheckIcon className="h-4 w-4" /> : <AlertIcon className="h-4 w-4" />}
                  </span>
                  <div className="text-sm font-semibold text-main">{mode}</div>
                  <span className="pill px-2.5 py-1 text-xs">{readiness?.selected_proxy || readiness?.selected_profile || nodeLabel}</span>
                </div>
                <p className="mt-3 text-sm leading-6 text-subtle">{readiness?.message || '桌面端正在等待后端 readiness 状态。'}</p>
              </div>
              <button
                onClick={handleCheckBest}
                disabled={checkingBest || config.proxy_profiles.length === 0}
                className="secondary-button px-4 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-60"
              >
                <ZapIcon className="h-4 w-4" />
                {checkingBest ? '测速中' : '测速选优'}
              </button>
            </div>

            {warnings.length > 0 && (
              <div className="mt-4 space-y-2">
                {warnings.map(warning => (
                  <div key={warning} className="rounded-lg border border-amber-500/20 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                    {warning}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="mt-4 grid gap-3 lg:grid-cols-2">
            {readinessActions.length === 0 ? (
              <div className="rounded-lg border border-dashed border-[#cfd6e3] p-4 text-sm text-subtle dark:border-white/10">
                当前没有必须处理的动作。
              </div>
            ) : readinessActions.map(action => (
              <div key={action.id} className={`rounded-lg border p-3 ${actionToneClasses[action.severity]}`}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="text-sm font-semibold">{action.label}</div>
                    {action.description && <div className="mt-1 text-xs leading-5 opacity-85">{action.description}</div>}
                  </div>
                  {action.command && (
                    <button
                      onClick={() => copyText(action.label, action.command || '')}
                      className="shrink-0 rounded-lg border border-current/20 px-2.5 py-1 text-xs transition hover:bg-white/40 dark:hover:bg-white/10"
                    >
                      复制
                    </button>
                  )}
                </div>
                {action.command && <div className="mt-3 truncate font-mono text-xs opacity-80">{action.command}</div>}
              </div>
            ))}
          </div>
        </div>

        <div className="panel p-5">
          <SectionHeader
            icon={<TerminalIcon className="h-4 w-4" />}
            title="AI / CLI 启动台"
            detail="从桌面端直接复制终端和 Agent 的真实入口"
            action={<span className={`rounded-full border px-2.5 py-1 text-xs ${status.running ? toneClasses.success : toneClasses.neutral}`}>{status.running ? '代理在线' : '待连接'}</span>}
          />

          <div className="grid gap-3 sm:grid-cols-2">
            {terminalActions.map(action => (
              <button
                key={action.label}
                onClick={() => copyText(action.label, action.command)}
                disabled={!action.command || !action.ready}
                title={action.command || '当前没有可复制内容'}
                className={`row-surface group flex min-h-36 w-full flex-col justify-between gap-3 p-3 text-left transition disabled:cursor-not-allowed disabled:opacity-60 ${
                  action.ready ? 'hover:border-[#0b8a7e]/30 hover:bg-white/88' : ''
                }`}
              >
                <div className="flex w-full items-start justify-between gap-3">
                  <span className={`grid h-9 w-9 shrink-0 place-items-center rounded-lg border ${action.ready ? toneClasses.success : toneClasses.warning}`}>
                    {action.icon}
                  </span>
                  <span className="rounded-full border border-[#dbe1eb] bg-white/68 px-2 py-0.5 text-xs text-subtle dark:border-white/10 dark:bg-white/5">
                    {action.ready ? action.badge : '待配置'}
                  </span>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-semibold text-main">{action.label}</div>
                  <div className="mt-1 min-h-8 text-xs leading-4 text-subtle">{action.description}</div>
                  <div className="mt-3 line-clamp-2 break-all font-mono text-xs leading-5 text-faint">{action.command}</div>
                </div>
                <div className="flex w-full items-center justify-between border-t border-[#dbe1eb] pt-3 text-xs text-faint dark:border-white/10">
                  <span>{action.ready ? '点击复制' : '先完成配置'}</span>
                  <CopyIcon className="h-4 w-4 shrink-0 group-hover:text-emerald-700" />
                </div>
              </button>
            ))}
          </div>

          <div className="mt-4 rounded-lg border border-[#dbe1eb] bg-[#101828] p-4 text-white shadow-sm dark:border-white/10">
            <div className="mb-3 flex items-center justify-between gap-3">
              <div className="min-w-0">
                <div className="text-xs text-slate-400">完整代理环境</div>
                <div className="mt-1 truncate text-sm font-semibold text-white">{httpProxy || socksProxy || '等待本地监听地址'}</div>
              </div>
              <button
                onClick={() => copyText('完整代理环境', proxyEnvBlock)}
                disabled={!proxyEnvBlock.trim()}
                className="inline-flex shrink-0 items-center gap-2 rounded-lg border border-white/12 bg-white/10 px-3 py-1.5 text-sm text-slate-100 transition hover:bg-white/15 disabled:cursor-not-allowed disabled:opacity-50"
              >
                <CopyIcon className="h-4 w-4" />
                复制
              </button>
            </div>
            <pre className="max-h-40 overflow-auto whitespace-pre-wrap rounded-lg border border-white/10 bg-black/20 p-3 font-mono text-xs leading-5 text-slate-200">
              {proxyEnvBlock || '连接后显示 HTTP_PROXY / ALL_PROXY / NO_PROXY'}
            </pre>
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.05fr)_minmax(22rem,0.95fr)]">
        <div className="panel p-5">
          <SectionHeader
            icon={<ImportIcon className="h-4 w-4" />}
            title="快速导入"
            detail="粘贴订阅 URL 或节点内容，导入后可自动测速选择"
          />
          <textarea
            placeholder="粘贴机场订阅 URL 或节点内容"
            value={importContent}
            onChange={e => setImportContent(e.target.value)}
            className="form-control mb-3 h-28 w-full resize-none p-3"
          />
          <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_9rem_10rem]">
            <input
              placeholder="默认节点名称（可选）"
              value={importSelect}
              onChange={e => setImportSelect(e.target.value)}
              className="form-control min-w-0 px-3 py-2"
            />
            <label className="row-surface flex items-center justify-between gap-2 px-3 py-2 text-sm text-subtle">
              覆盖同名
              <input type="checkbox" checked={importReplace} onChange={e => setImportReplace(e.target.checked)} className="h-4 w-4 accent-[#0b8a7e]" />
            </label>
            <label className="row-surface flex items-center justify-between gap-2 px-3 py-2 text-sm text-subtle">
              测速选优
              <input
                type="checkbox"
                checked={importCheck && !importSelect.trim()}
                disabled={Boolean(importSelect.trim())}
                onChange={e => setImportCheck(e.target.checked)}
                className="h-4 w-4 accent-[#0b8a7e] disabled:opacity-50"
              />
            </label>
          </div>
          <div className="mt-4 flex flex-wrap items-center justify-between gap-3">
            <div className="text-xs text-subtle">
              已有 {config.proxy_profiles.length} 个节点，{config.subscriptions.length} 个订阅来源
            </div>
            <button
              onClick={handleImport}
              disabled={!canImport}
              className="primary-button px-4 py-2 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400 disabled:shadow-none"
            >
              <ImportIcon className="h-4 w-4" />
              {importing ? '导入中' : importCheck && !importSelect.trim() ? '导入并选优' : '导入'}
            </button>
          </div>
        </div>

        <div className="panel p-5">
          <SectionHeader
            icon={<FileIcon className="h-4 w-4" />}
            title="运行摘要"
            detail="确认桌面端、CLI 和系统代理看到的是同一份配置"
          />
          <div className="space-y-3">
            {[
              ['配置文件', state?.config_path || '未加载'],
              ['本地 SOCKS5', socksAddr || '-'],
              ['本地 HTTP', httpAddr || '-'],
              ['Relay', relayAddr || '-'],
              ['上次错误', status.last_error || '-'],
            ].map(([label, value]) => (
              <div key={label} className="row-surface flex items-start justify-between gap-3 p-3">
                <div className="text-xs text-faint">{label}</div>
                <div className="max-w-[70%] break-all text-right font-mono text-xs text-subtle">{value}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {message && (
        <div className="toast fixed bottom-4 right-4 px-4 py-2">
          {message}
        </div>
      )}
    </div>
  )
}
