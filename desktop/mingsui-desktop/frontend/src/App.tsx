import {useState} from 'react'
import {useTheme} from 'next-themes'
import {Sidebar} from './components/layout/Sidebar'
import {ThemeToggle} from './components/common/ThemeToggle'
import {Overview} from './components/views/Overview'
import {Nodes} from './components/views/Nodes'
import {Subscriptions} from './components/views/Subscriptions'
import {Logs} from './components/views/Logs'
import {Settings} from './components/views/Settings'

type ViewType = 'overview' | 'nodes' | 'subscriptions' | 'logs' | 'settings'

function App() {
  const {theme} = useTheme()
  const [activeView, setActiveView] = useState<ViewType>('overview')

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
    <div className={`min-h-screen ${theme === 'dark' ? 'bg-[#121212]' : 'bg-gray-100'}`}>
      <div className="flex">
        <Sidebar activeView={activeView} onViewChange={(view) => setActiveView(view as ViewType)} />
        <main className="flex-1 p-6 overflow-auto">
          <div className="flex items-center justify-between mb-6">
            <div />
            <ThemeToggle />
          </div>
          {renderView()}
        </main>
      </div>
    </div>
  )
}

export default App