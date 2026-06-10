import React from 'react'
import {createRoot} from 'react-dom/client'
import {ThemeProvider} from './lib/theme-provider'
import './index.css'
import App from './App'

class AppErrorBoundary extends React.Component<{children: React.ReactNode}, {error: Error | null}> {
  state = {error: null}

  static getDerivedStateFromError(error: Error) {
    return {error}
  }

  render() {
    if (!this.state.error) return this.props.children
    return <FatalError error={this.state.error} />
  }
}

function FatalError({error}: {error: unknown}) {
  const message = error instanceof Error ? error.message : String(error)
  return (
    <div className="min-h-screen bg-[#f8fafc] p-6 text-[#202735]">
      <div className="mx-auto mt-16 max-w-2xl rounded-lg border border-red-200 bg-white p-5 shadow-sm">
        <div className="text-base font-semibold text-red-700">桌面端前端启动失败</div>
        <pre className="mt-3 whitespace-pre-wrap rounded-lg bg-red-50 p-3 font-mono text-sm text-red-800">{message}</pre>
      </div>
    </div>
  )
}

function renderFatalError(error: unknown) {
  const root = document.getElementById('root')
  if (!root) return
  createRoot(root).render(<FatalError error={error} />)
}

window.addEventListener('error', event => {
  renderFatalError(event.error || event.message)
})

window.addEventListener('unhandledrejection', event => {
  renderFatalError(event.reason)
})

const container = document.getElementById('root')

if (!container) {
  throw new Error('Missing #root container')
}

const root = createRoot(container)

root.render(
    <React.StrictMode>
        <ThemeProvider>
            <AppErrorBoundary>
                <App/>
            </AppErrorBoundary>
        </ThemeProvider>
    </React.StrictMode>
)
