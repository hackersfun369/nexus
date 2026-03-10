import { useEffect } from 'react'
import { useAppStore } from '@/store/app'
import HomePage from '@/pages/HomePage'
import BuilderPage from '@/pages/BuilderPage'
import WizardPage from '@/pages/WizardPage'

export type Page = 'home' | 'wizard' | 'builder'

export default function App() {
  const { page } = useAppStore()

  useEffect(() => {
    document.title = 'NEXUS — Intelligent Development System'
  }, [])

  return (
    <div className="h-screen w-screen overflow-hidden bg-[#0d1117] text-[#e6edf3]">
      {page === 'home'    && <HomePage />}
      {page === 'wizard'  && <WizardPage />}
      {page === 'builder' && <BuilderPage />}
    </div>
  )
}
