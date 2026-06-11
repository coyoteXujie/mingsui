import {useEffect, useMemo, useState} from 'react'
import type {ComponentType, ReactNode} from 'react'
import {
  FiAlertTriangle,
  FiCheckCircle,
  FiCopy,
  FiClock,
  FiFileText,
  FiGlobe,
  FiHardDrive,
  FiInfo,
  FiKey,
  FiLock,
  FiSave,
  FiServer,
  FiShield,
  FiSliders,
  FiTerminal,
} from 'react-icons/fi'
import {ClipboardSetText} from '../../../wailsjs/runtime/runtime'
import {useDesktop} from '../../hooks/useDesktop'

const AlertIcon = FiAlertTriangle as ComponentType<{className?: string}>
const CheckCircleIcon = FiCheckCircle as ComponentType<{className?: string}>
const CopyIcon = FiCopy as ComponentType<{className?: string}>
const ClockIcon = FiClock as ComponentType<{className?: string}>
const FileIcon = FiFileText as ComponentType<{className?: string}>
const GlobeIcon = FiGlobe as ComponentType<{className?: string}>
const HardDriveIcon = FiHardDrive as ComponentType<{className?: string}>
const InfoIcon = FiInfo as ComponentType<{className?: string}>
const KeyIcon = FiKey as ComponentType<{className?: string}>
const LockIcon = FiLock as ComponentType<{className?: string}>
const SaveIcon = FiSave as ComponentType<{className?: string}>
const ServerIcon = FiServer as ComponentType<{className?: string}>
const ShieldIcon = FiShield as ComponentType<{className?: string}>
const SlidersIcon = FiSliders as ComponentType<{className?: string}>
const TerminalIcon = FiTerminal as ComponentType<{className?: string}>

type IssueSeverity = 'error' | 'warning' | 'info'
type StatusTone = 'ready' | 'warning' | 'danger' | 'muted'

interface SettingIssue {
  id: string
  severity: IssueSeverity
  title: string
  detail: string
}

interface StatusCardProps {
  label: string
  value: string
  detail: string
  tone: StatusTone
  icon: ReactNode
}

interface CatalogItem {
  label: string
  value: string
  detail: string
  tone: StatusTone
  icon: ReactNode
}

interface CommandItem {
  label: string
  detail: string
  command: string
}

const toneClasses: Record<StatusTone, string> = {
  ready: 'border-emerald-500/20 bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200',
  warning: 'border-amber-500/25 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  danger: 'border-red-500/25 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
  muted: 'border-slate-200 bg-white/60 text-subtle dark:border-white/10 dark:bg-white/5',
}

const issueClasses: Record<IssueSeverity, string> = {
  error: 'border-red-500/20 bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-200',
  warning: 'border-amber-500/20 bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-200',
  info: 'border-sky-500/20 bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-200',
}

function looksLikeHostPort(value: string): boolean {
  return /^[^\s:]+:\d{1,5}$/.test(value.trim())
}

function maskSecret(value: string): string {
  const trimmed = value.trim()
  if (!trimmed) return '未设置'
  if (trimmed.length <= 6) return '已设置'
  return `${trimmed.slice(0, 2)}...${trimmed.slice(-2)}`
}

function shellQuote(value: string): string {
  return `'${value.replace(/'/g, `'\\''`)}'`
}

function sectionTone(hasError: boolean, hasWarning: boolean): StatusTone {
  if (hasError) return 'danger'
  if (hasWarning) return 'warning'
  return 'ready'
}

function issueIcon(severity: IssueSeverity) {
  if (severity === 'error') return <AlertIcon className="h-4 w-4" />
  if (severity === 'warning') return <AlertIcon className="h-4 w-4" />
  return <InfoIcon className="h-4 w-4" />
}

function StatusCard({label, value, detail, tone, icon}: StatusCardProps) {
  return (
    <div className="panel p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs text-faint">{label}</div>
          <div className="mt-2 truncate text-sm font-semibold text-main">{value}</div>
          <div className="mt-1 truncate text-xs text-subtle">{detail}</div>
        </div>
        <span className={`grid h-9 w-9 shrink-0 place-items-center rounded-lg border ${toneClasses[tone]}`}>
          {icon}
        </span>
      </div>
    </div>
  )
}

function SectionTitle({icon, title, detail}: {icon: ReactNode; title: string; detail: string}) {
  return (
    <div className="mb-5 flex items-start justify-between gap-3">
      <div className="flex items-start gap-3">
        <span className="icon-tile h-9 w-9 shrink-0">{icon}</span>
        <div>
          <h3 className="text-base font-semibold text-main">{title}</h3>
          <p className="mt-1 text-xs text-subtle">{detail}</p>
        </div>
      </div>
    </div>
  )
}

export function Settings() {
  const {state, loading, saveConfig} = useDesktop()
  const [message, setMessage] = useState('')
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  const config = state?.config
  const [localAddr, setLocalAddr] = useState(config?.local_addr || '')
  const [httpAddr, setHttpAddr] = useState(config?.http_addr || '')
  const [relayAddr, setRelayAddr] = useState(config?.relay_addr || '')
  const [token, setToken] = useState(config?.token || '')
  const [timeoutSeconds, setTimeoutSeconds] = useState(config?.dial_timeout_seconds || 10)
  const [authEnabled, setAuthEnabled] = useState(config?.local_auth?.enabled || false)
  const [authUser, setAuthUser] = useState(config?.local_auth?.username || '')
  const [authPass, setAuthPass] = useState(config?.local_auth?.password || '')
  const [tlsEnabled, setTlsEnabled] = useState(config?.tls?.enabled || false)
  const [tlsServerName, setTlsServerName] = useState(config?.tls?.server_name || '')
  const [tlsCAFile, setTlsCAFile] = useState(config?.tls?.ca_file || '')
  const [tlsInsecure, setTlsInsecure] = useState(config?.tls?.insecure_skip_verify || false)

  useEffect(() => {
    if (!config || dirty) return
    setLocalAddr(config.local_addr || '')
    setHttpAddr(config.http_addr || '')
    setRelayAddr(config.relay_addr || '')
    setToken(config.token || '')
    setTimeoutSeconds(config.dial_timeout_seconds || 10)
    setAuthEnabled(Boolean(config.local_auth?.enabled))
    setAuthUser(config.local_auth?.username || '')
    setAuthPass(config.local_auth?.password || '')
    setTlsEnabled(Boolean(config.tls?.enabled))
    setTlsServerName(config.tls?.server_name || '')
    setTlsCAFile(config.tls?.ca_file || '')
    setTlsInsecure(Boolean(config.tls?.insecure_skip_verify))
  }, [config, dirty])

  const update = <T,>(setter: (value: T) => void, value: T) => {
    setDirty(true)
    setMessage('')
    setter(value)
  }

  const issues = useMemo<SettingIssue[]>(() => {
    const next: SettingIssue[] = []

    if (!localAddr.trim()) {
      next.push({id: 'local-required', severity: 'error', title: 'SOCKS5 监听不能为空', detail: '本地 SOCKS5 入口是 CLI、AI 工具和系统代理的基础能力。'})
    } else if (!looksLikeHostPort(localAddr)) {
      next.push({id: 'local-format', severity: 'warning', title: 'SOCKS5 监听格式可疑', detail: '推荐使用 127.0.0.1:1080 这类 host:port 格式。'})
    }

    if (!httpAddr.trim()) {
      next.push({id: 'http-required', severity: 'error', title: 'HTTP 监听不能为空', detail: 'AI Agent 和多数终端工具会优先使用 HTTP_PROXY。'})
    } else if (!looksLikeHostPort(httpAddr)) {
      next.push({id: 'http-format', severity: 'warning', title: 'HTTP 监听格式可疑', detail: '推荐使用 127.0.0.1:8080 这类 host:port 格式。'})
    }

    if (timeoutSeconds <= 0 || Number.isNaN(timeoutSeconds)) {
      next.push({id: 'timeout-required', severity: 'error', title: '连接超时必须大于 0', detail: '过小或无效的超时会让测速、导入检查和连接启动不稳定。'})
    } else if (timeoutSeconds > 60) {
      next.push({id: 'timeout-long', severity: 'warning', title: '连接超时偏长', detail: '超过 60 秒会让自动选优和失败反馈明显变慢。'})
    }

    if (!relayAddr.trim()) {
      next.push({id: 'relay-empty', severity: 'error', title: '默认 Relay 地址不能为空', detail: '机场节点模式不会使用它，但共用配置仍需要保留一个有效的 Relay 地址作为回退。'})
    } else if (!looksLikeHostPort(relayAddr)) {
      next.push({id: 'relay-format', severity: 'warning', title: 'Relay 地址格式可疑', detail: '推荐使用 relay.example.com:443 这类 host:port 格式。'})
    }

    if (!token.trim() || token.trim() === 'change-me') {
      next.push({id: 'token-default', severity: 'warning', title: 'Relay token 需要更换', detail: '默认或空 token 不适合作为公网 Relay 的认证凭据。'})
    }

    if (authEnabled && !authUser.trim()) {
      next.push({id: 'auth-user', severity: 'error', title: '本地认证缺少用户名', detail: '启用本地代理认证后，用户名和密码必须同时配置。'})
    }

    if (authEnabled && !authPass) {
      next.push({id: 'auth-pass', severity: 'error', title: '本地认证缺少密码', detail: '启用本地代理认证后，空密码会导致客户端连接不可控。'})
    }

    if (tlsEnabled && !tlsServerName.trim()) {
      next.push({id: 'tls-server-name', severity: 'info', title: 'TLS ServerName 未设置', detail: '公网域名证书通常需要 ServerName 才能完成校验。'})
    }

    if (tlsEnabled && !tlsCAFile.trim() && !tlsInsecure) {
      next.push({id: 'tls-ca', severity: 'info', title: 'TLS CA 文件未设置', detail: '使用系统根证书时可以为空；自签证书建议指定 CA 文件。'})
    }

    if (tlsEnabled && tlsInsecure) {
      next.push({id: 'tls-insecure', severity: 'warning', title: 'TLS 正在跳过证书校验', detail: '这个选项只适合临时调试，不建议用于长期连接公网 Relay。'})
    }

    return next
  }, [authEnabled, authPass, authUser, httpAddr, localAddr, relayAddr, timeoutSeconds, tlsCAFile, tlsEnabled, tlsInsecure, tlsServerName, token])

  const errorCount = issues.filter(issue => issue.severity === 'error').length
  const warningCount = issues.filter(issue => issue.severity === 'warning').length
  const infoCount = issues.filter(issue => issue.severity === 'info').length
  const canSave = Boolean(config) && dirty && errorCount === 0 && !saving
  const configPath = state?.config_path || '未加载'
  const topLevelConfigFlag = state?.config_path ? ` -config ${shellQuote(state.config_path)}` : ''
  const configCommandFlag = state?.config_path ? ` -path ${shellQuote(state.config_path)}` : ''
  const issueMatches = (issue: SettingIssue, prefixes: string[]) => prefixes.some(prefix => issue.id.startsWith(prefix))

  const localPrefixes = ['local', 'http', 'timeout']
  const localTone = sectionTone(
    issues.some(issue => issueMatches(issue, localPrefixes) && issue.severity === 'error'),
    issues.some(issue => issueMatches(issue, localPrefixes) && issue.severity === 'warning'),
  )
  const relayHasWarning = issues.some(issue => issue.id.startsWith('relay') && issue.severity === 'warning')
  const relayTone: StatusTone = relayAddr.trim() ? sectionTone(false, relayHasWarning) : 'muted'
  const authTone = sectionTone(
    issues.some(issue => issue.id.startsWith('auth') && issue.severity === 'error'),
    issues.some(issue => issue.id === 'token-default' && issue.severity === 'warning'),
  )
  const tlsHasWarning = issues.some(issue => issue.id.startsWith('tls') && issue.severity === 'warning')
  const tlsTone: StatusTone = tlsEnabled ? sectionTone(false, tlsHasWarning) : 'muted'

  const catalogItems: CatalogItem[] = [
    {
      label: '本地入口',
      value: localTone === 'ready' ? '就绪' : localTone === 'danger' ? '需修正' : '需确认',
      detail: `${localAddr || '-'} / ${httpAddr || '-'}`,
      tone: localTone,
      icon: <HardDriveIcon className="h-4 w-4" />,
    },
    {
      label: 'Relay 默认连接',
      value: relayAddr ? '已配置' : '需填写',
      detail: relayAddr || '保留默认地址作为 relay 模式回退',
      tone: relayTone,
      icon: <ServerIcon className="h-4 w-4" />,
    },
    {
      label: '安全认证',
      value: authEnabled ? '本地认证开启' : '本地认证关闭',
      detail: `Token ${maskSecret(token)}`,
      tone: authTone,
      icon: <ShieldIcon className="h-4 w-4" />,
    },
    {
      label: 'TLS 传输',
      value: tlsEnabled ? (tlsInsecure ? '跳过校验' : '已启用') : '未启用',
      detail: tlsEnabled ? (tlsServerName || '未设置 ServerName') : 'Relay 明文连接',
      tone: tlsTone,
      icon: <LockIcon className="h-4 w-4" />,
    },
  ]
  const commandItems: CommandItem[] = [
    {
      label: '查看状态',
      detail: '确认 CLI 读取的是桌面端同一份配置',
      command: `mingsui status${topLevelConfigFlag} -json`,
    },
    {
      label: '运行诊断',
      detail: '输出结构化配置和连接诊断',
      command: `mingsui doctor${topLevelConfigFlag} -json`,
    },
    {
      label: '查看配置',
      detail: '在终端检查完整客户端配置',
      command: `mingsui config show${configCommandFlag}`,
    },
    {
      label: '导出环境',
      detail: '让当前终端使用桌面端本地代理',
      command: `eval "$(mingsui env${topLevelConfigFlag})"`,
    },
  ]

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

  const handleSave = async () => {
    if (!config) {
      setMessage('配置尚未加载')
      return
    }
    if (errorCount > 0) {
      setMessage('请先修正红色错误项')
      return
    }
    try {
      setSaving(true)
      await saveConfig({
        ...config,
        local_addr: localAddr.trim(),
        http_addr: httpAddr.trim(),
        relay_addr: relayAddr.trim(),
        token: token.trim(),
        dial_timeout_seconds: Math.max(1, Math.floor(timeoutSeconds)),
        local_auth: {
          enabled: authEnabled,
          username: authUser.trim(),
          password: authPass,
        },
        tls: {
          enabled: tlsEnabled,
          server_name: tlsServerName.trim(),
          ca_file: tlsCAFile.trim(),
          insecure_skip_verify: tlsInsecure,
        },
        profiles: config.profiles || [],
        proxy_profiles: config.proxy_profiles || [],
        subscriptions: config.subscriptions || [],
      })
      setDirty(false)
      setMessage('配置已保存')
    } catch (err: any) {
      setMessage(err.message)
    } finally {
      setSaving(false)
    }
  }

  if (loading) return <div className="flex h-64 items-center justify-center text-subtle">加载中...</div>

  return (
    <div className="space-y-6">
      <div className="grid gap-3 lg:grid-cols-4">
        <StatusCard
          label="SOCKS5 入口"
          value={localAddr || '未设置'}
          detail="供浏览器、CLI 和 AI 工具使用"
          tone={localAddr ? localTone : 'danger'}
          icon={<HardDriveIcon className="h-4 w-4" />}
        />
        <StatusCard
          label="HTTP 入口"
          value={httpAddr || '未设置'}
          detail="HTTP_PROXY / HTTPS_PROXY"
          tone={httpAddr ? localTone : 'danger'}
          icon={<GlobeIcon className="h-4 w-4" />}
        />
        <StatusCard
          label="Relay"
          value={relayAddr || '未设置'}
          detail={tlsEnabled ? 'TLS 连接已启用' : '默认明文连接'}
          tone={relayTone}
          icon={<ServerIcon className="h-4 w-4" />}
        />
        <StatusCard
          label="保存状态"
          value={dirty ? '未保存' : '已同步'}
          detail={errorCount > 0 ? `${errorCount} 个错误待处理` : `${warningCount} 个风险提示`}
          tone={errorCount > 0 ? 'danger' : dirty ? 'warning' : 'ready'}
          icon={dirty ? <SaveIcon className="h-4 w-4" /> : <CheckCircleIcon className="h-4 w-4" />}
        />
      </div>

      <div className="grid gap-6 xl:grid-cols-[20rem_minmax(0,1fr)]">
        <aside className="space-y-6">
          <div className="panel p-5">
            <SectionTitle
              icon={<SlidersIcon className="h-4 w-4" />}
              title="设置中心"
              detail="按桌面端和 CLI 共用的配置边界组织"
            />

            <div className="mb-4 rounded-lg border border-dashed border-[#cfd6e3] p-3 text-left dark:border-white/10">
              <div className="flex items-center gap-2 text-xs text-faint">
                <FileIcon className="h-4 w-4" />
                配置文件
              </div>
              <div className="mt-2 break-all font-mono text-xs text-subtle">{configPath}</div>
            </div>

            <div className="space-y-2">
              {catalogItems.map(item => (
                <div key={item.label} className="row-surface flex items-center justify-between gap-3 p-3">
                  <div className="flex min-w-0 items-center gap-3">
                    <span className={`grid h-8 w-8 shrink-0 place-items-center rounded-lg border ${toneClasses[item.tone]}`}>
                      {item.icon}
                    </span>
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-main">{item.label}</div>
                      <div className="mt-1 truncate text-xs text-subtle">{item.detail}</div>
                    </div>
                  </div>
                  <span className={`shrink-0 rounded-full border px-2.5 py-1 text-xs ${toneClasses[item.tone]}`}>{item.value}</span>
                </div>
              ))}
            </div>
          </div>

          <div className="panel p-5">
            <div className="mb-4 flex items-center justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-main">配置检查</h3>
                <p className="mt-1 text-xs text-subtle">{errorCount} 错误 · {warningCount} 风险 · {infoCount} 提示</p>
              </div>
              <span className={`grid h-9 w-9 place-items-center rounded-lg border ${errorCount ? toneClasses.danger : toneClasses.ready}`}>
                {errorCount ? <AlertIcon className="h-4 w-4" /> : <CheckCircleIcon className="h-4 w-4" />}
              </span>
            </div>

            {issues.length === 0 ? (
              <div className="rounded-lg border border-emerald-500/20 bg-emerald-50 p-4 text-sm text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-200">
                当前配置没有发现明显问题。
              </div>
            ) : (
              <div className="space-y-2">
                {issues.map(issue => (
                  <div key={issue.id} className={`rounded-lg border p-3 ${issueClasses[issue.severity]}`}>
                    <div className="flex items-center gap-2 text-sm font-medium">
                      {issueIcon(issue.severity)}
                      {issue.title}
                    </div>
                    <div className="mt-1 text-xs leading-5 opacity-85">{issue.detail}</div>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="panel p-5">
            <SectionTitle
              icon={<TerminalIcon className="h-4 w-4" />}
              title="CLI 等价入口"
              detail="复制命令验证桌面端正在使用的同一份配置"
            />
            <div className="space-y-2">
              {commandItems.map(item => (
                <button
                  key={item.label}
                  onClick={() => copyText(item.label, item.command)}
                  className="row-surface group flex w-full items-start justify-between gap-3 p-3 text-left transition hover:border-[#0b8a7e]/30"
                >
                  <div className="min-w-0">
                    <div className="text-sm font-semibold text-main">{item.label}</div>
                    <div className="mt-1 text-xs text-subtle">{item.detail}</div>
                    <div className="mt-2 truncate font-mono text-xs text-faint">{item.command}</div>
                  </div>
                  <CopyIcon className="mt-0.5 h-4 w-4 shrink-0 text-faint group-hover:text-emerald-700" />
                </button>
              ))}
            </div>
          </div>
        </aside>

        <div className="space-y-6">
          <div className="panel p-5">
            <SectionTitle
              icon={<HardDriveIcon className="h-4 w-4" />}
              title="本地代理入口"
              detail="桌面端、CLI、系统代理和 AI 工具都会依赖这里的监听地址"
            />
            <div className="grid gap-4 lg:grid-cols-[1fr_1fr_12rem]">
              <label className="block">
                <span className="mb-1 block text-sm text-subtle">SOCKS5 监听</span>
                <input
                  placeholder="127.0.0.1:1080"
                  value={localAddr}
                  onChange={e => update(setLocalAddr, e.target.value)}
                  className="form-control w-full px-3 py-2"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm text-subtle">HTTP 监听</span>
                <input
                  placeholder="127.0.0.1:8080"
                  value={httpAddr}
                  onChange={e => update(setHttpAddr, e.target.value)}
                  className="form-control w-full px-3 py-2"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm text-subtle">超时秒数</span>
                <div className="relative">
                  <ClockIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
                  <input
                    type="number"
                    min={1}
                    value={Number.isNaN(timeoutSeconds) ? '' : timeoutSeconds}
                    onChange={e => update(setTimeoutSeconds, Number(e.target.value))}
                    className="form-control w-full py-2 pl-9 pr-3"
                  />
                </div>
              </label>
            </div>
          </div>

          <div className="panel p-5">
            <SectionTitle
              icon={<ServerIcon className="h-4 w-4" />}
              title="Relay 默认连接"
              detail="Relay 模式和远端转发会使用这组默认连接信息"
            />
            <div className="grid gap-4 lg:grid-cols-[1fr_1fr]">
              <label className="block">
                <span className="mb-1 block text-sm text-subtle">默认 Relay</span>
                <input
                  placeholder="relay.example.com:443"
                  value={relayAddr}
                  onChange={e => update(setRelayAddr, e.target.value)}
                  className="form-control w-full px-3 py-2"
                />
              </label>
              <label className="block">
                <span className="mb-1 block text-sm text-subtle">Relay Token</span>
                <div className="relative">
                  <KeyIcon className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-faint" />
                  <input
                    type="password"
                    value={token}
                    onChange={e => update(setToken, e.target.value)}
                    className="form-control w-full py-2 pl-9 pr-3"
                  />
                </div>
              </label>
            </div>
          </div>

          <div className="grid gap-6 lg:grid-cols-2">
            <div className="panel p-5">
              <SectionTitle
                icon={<ShieldIcon className="h-4 w-4" />}
                title="本地代理认证"
                detail="需要把本地端口暴露给局域网时再开启"
              />
              <label className="row-surface mb-4 flex items-center justify-between gap-3 p-3">
                <div>
                  <div className="text-sm font-medium text-main">启用用户名密码</div>
                  <div className="mt-1 text-xs text-subtle">HTTP 和 SOCKS5 都会要求本地认证</div>
                </div>
                <input
                  type="checkbox"
                  checked={authEnabled}
                  onChange={e => update(setAuthEnabled, e.target.checked)}
                  className="h-4 w-4 accent-[#0b8a7e]"
                />
              </label>
              {authEnabled ? (
                <div className="grid gap-4 md:grid-cols-2">
                  <input
                    placeholder="认证用户名"
                    value={authUser}
                    onChange={e => update(setAuthUser, e.target.value)}
                    className="form-control px-3 py-2"
                  />
                  <input
                    type="password"
                    placeholder="认证密码"
                    value={authPass}
                    onChange={e => update(setAuthPass, e.target.value)}
                    className="form-control px-3 py-2"
                  />
                </div>
              ) : (
                <div className="rounded-lg border border-dashed border-[#cfd6e3] p-4 text-sm text-subtle dark:border-white/10">
                  当前本机 HTTP/SOCKS5 代理无需用户名密码。
                </div>
              )}
            </div>

            <div className="panel p-5">
              <SectionTitle
                icon={<LockIcon className="h-4 w-4" />}
                title="Relay TLS"
                detail="公网 Relay 建议启用 TLS 并完成证书校验"
              />
              <label className="row-surface mb-4 flex items-center justify-between gap-3 p-3">
                <div>
                  <div className="text-sm font-medium text-main">启用 TLS</div>
                  <div className="mt-1 text-xs text-subtle">连接 Relay 时使用 TLS 传输</div>
                </div>
                <input
                  type="checkbox"
                  checked={tlsEnabled}
                  onChange={e => update(setTlsEnabled, e.target.checked)}
                  className="h-4 w-4 accent-[#0b8a7e]"
                />
              </label>
              {tlsEnabled ? (
                <div className="space-y-3">
                  <input
                    placeholder="TLS ServerName"
                    value={tlsServerName}
                    onChange={e => update(setTlsServerName, e.target.value)}
                    className="form-control w-full px-3 py-2"
                  />
                  <input
                    placeholder="TLS CA 文件"
                    value={tlsCAFile}
                    onChange={e => update(setTlsCAFile, e.target.value)}
                    className="form-control w-full px-3 py-2"
                  />
                  <label className="flex items-center justify-between gap-3 rounded-lg border border-amber-500/20 bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-500/10 dark:text-amber-200">
                    <span>跳过证书校验</span>
                    <input
                      type="checkbox"
                      checked={tlsInsecure}
                      onChange={e => update(setTlsInsecure, e.target.checked)}
                      className="h-4 w-4 accent-[#d97706]"
                    />
                  </label>
                </div>
              ) : (
                <div className="rounded-lg border border-dashed border-[#cfd6e3] p-4 text-sm text-subtle dark:border-white/10">
                  默认使用明文 Relay 连接；公网部署时建议启用 TLS。
                </div>
              )}
            </div>
          </div>

          <div className="panel flex flex-wrap items-center justify-between gap-4 p-4">
            <div className="flex flex-wrap items-center gap-2">
              <span className={`rounded-full border px-2.5 py-1 text-xs ${dirty ? toneClasses.warning : toneClasses.ready}`}>
                {dirty ? '有未保存修改' : '配置已同步'}
              </span>
              <span className={`rounded-full border px-2.5 py-1 text-xs ${errorCount ? toneClasses.danger : toneClasses.muted}`}>
                {errorCount ? `${errorCount} 个错误阻止保存` : dirty ? '可保存' : '等待修改'}
              </span>
              {warningCount > 0 && (
                <span className={`rounded-full border px-2.5 py-1 text-xs ${toneClasses.warning}`}>{warningCount} 个风险提示</span>
              )}
            </div>
            <button
              onClick={handleSave}
              disabled={!canSave}
              className="primary-button px-5 py-2.5 text-sm font-medium disabled:bg-slate-200 disabled:text-slate-400 disabled:shadow-none"
            >
              <SaveIcon className="h-4 w-4" />
              {saving ? '保存中...' : '保存配置'}
            </button>
          </div>
        </div>
      </div>

      {message && <div className="toast fixed bottom-4 right-4 px-4 py-2">{message}</div>}
    </div>
  )
}
