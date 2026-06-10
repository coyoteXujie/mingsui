import {useState, useEffect} from 'react'
import type {ComponentType} from 'react'
import {useTheme} from 'next-themes'
import {FiMoon, FiSun} from 'react-icons/fi'

const MoonIcon = FiMoon as ComponentType<{className?: string}>
const SunIcon = FiSun as ComponentType<{className?: string}>

function ThemeToggle() {
  const {theme, setTheme} = useTheme()
  const [mounted, setMounted] = useState(false)

  useEffect(() => setMounted(true), [])
  if (!mounted) return <div className="w-10 h-10" />

  return (
    <button
      onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
      className="flex h-10 w-10 items-center justify-center rounded-lg border border-white/10 bg-white/5 text-white transition-colors hover:bg-white/10"
      title={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'}
    >
      {theme === 'dark' ? <SunIcon className="h-4 w-4" /> : <MoonIcon className="h-4 w-4" />}
    </button>
  )
}

export {ThemeToggle}
