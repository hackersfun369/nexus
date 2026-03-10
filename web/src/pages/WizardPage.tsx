import { useAppStore, Platform, DeployTarget } from '@/store/app'
import { Smartphone, Globe, Monitor, Apple, Terminal, Server, Layers, Cloud, HardDrive } from 'lucide-react'

const PLATFORMS: { id: Platform; label: string; icon: React.ReactNode; desc: string }[] = [
  { id: 'android',  label: 'Android',  icon: <Smartphone className="w-5 h-5" />, desc: 'Kotlin / Java / Flutter' },
  { id: 'web',      label: 'Web',      icon: <Globe className="w-5 h-5" />,      desc: 'React / Vue / Svelte' },
  { id: 'windows',  label: 'Windows',  icon: <Monitor className="w-5 h-5" />,    desc: 'Electron / .NET / WPF' },
  { id: 'mac',      label: 'macOS',    icon: <Apple className="w-5 h-5" />,      desc: 'Swift / Electron' },
  { id: 'cli',      label: 'CLI Tool', icon: <Terminal className="w-5 h-5" />,   desc: 'Go / Python / Rust' },
  { id: 'backend',  label: 'Backend',  icon: <Server className="w-5 h-5" />,     desc: 'Go / Python / Node' },
  { id: 'all',      label: 'All',      icon: <Layers className="w-5 h-5" />,     desc: 'Every platform' },
]

const LANGUAGES = [
  'Python', 'TypeScript', 'JavaScript', 'Kotlin', 'Java',
  'Go', 'Rust', 'Swift', 'Dart', 'C#', 'C++', 'C',
  'NEXUS decides',
]

const DEPLOY_TARGETS: { id: DeployTarget; label: string; icon: React.ReactNode; desc: string }[] = [
  { id: 'local', label: 'Local',  icon: <HardDrive className="w-5 h-5" />, desc: 'Runs on your machine' },
  { id: 'cloud', label: 'Cloud',  icon: <Cloud className="w-5 h-5" />,     desc: 'Deploy to the web' },
  { id: 'both',  label: 'Both',   icon: <Layers className="w-5 h-5" />,    desc: 'Local + cloud deploy' },
]

export default function WizardPage() {
  const {
    selectedPlatforms, togglePlatform,
    selectedLanguages, toggleLanguage,
    deployTarget, setDeployTarget,
    setPage, messages,
  } = useAppStore()

  const firstMessage = messages[0]?.content ?? ''
  const canContinue = selectedPlatforms.length > 0

  const handleContinue = () => {
    setPage('builder')
  }

  return (
    <div className="h-full flex flex-col items-center justify-center px-4 overflow-y-auto py-8">
      <div className="w-full max-w-2xl">

        {/* User's prompt */}
        {firstMessage && (
          <div className="mb-8 p-4 bg-[#161b22] border border-[#30363d] rounded-xl">
            <p className="text-[#7d8590] text-xs uppercase tracking-wider mb-1">Your idea</p>
            <p className="text-[#e6edf3]">{firstMessage}</p>
          </div>
        )}

        {/* Platform picker */}
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-1">Which platforms?</h2>
          <p className="text-[#7d8590] text-sm mb-4">Select all that apply</p>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
            {PLATFORMS.map(({ id, label, icon, desc }) => {
              const selected = selectedPlatforms.includes(id)
              return (
                <button
                  key={id}
                  onClick={() => togglePlatform(id)}
                  className={`p-4 rounded-xl border text-left transition-all
                    ${selected
                      ? 'border-[#58a6ff] bg-[#58a6ff1a] text-[#e6edf3]'
                      : 'border-[#30363d] bg-[#161b22] text-[#7d8590] hover:border-[#58a6ff] hover:text-[#e6edf3]'
                    }`}
                >
                  <div className={`mb-2 ${selected ? 'text-[#58a6ff]' : ''}`}>{icon}</div>
                  <div className="font-medium text-sm">{label}</div>
                  <div className="text-xs mt-0.5 opacity-70">{desc}</div>
                </button>
              )
            })}
          </div>
        </div>

        {/* Language picker */}
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-1">Which languages?</h2>
          <p className="text-[#7d8590] text-sm mb-4">Select all you prefer — or let NEXUS decide</p>
          <div className="flex flex-wrap gap-2">
            {LANGUAGES.map((lang) => {
              const selected = selectedLanguages.includes(lang)
              return (
                <button
                  key={lang}
                  onClick={() => toggleLanguage(lang)}
                  className={`px-4 py-2 rounded-full border text-sm transition-all
                    ${selected
                      ? 'border-[#58a6ff] bg-[#58a6ff1a] text-[#58a6ff]'
                      : 'border-[#30363d] bg-[#161b22] text-[#7d8590] hover:border-[#58a6ff] hover:text-[#e6edf3]'
                    }`}
                >
                  {lang}
                </button>
              )
            })}
          </div>
        </div>

        {/* Deploy target */}
        <div className="mb-8">
          <h2 className="text-lg font-semibold mb-1">Where will it run?</h2>
          <p className="text-[#7d8590] text-sm mb-4">Choose your deployment target</p>
          <div className="grid grid-cols-3 gap-3">
            {DEPLOY_TARGETS.map(({ id, label, icon, desc }) => (
              <button
                key={id}
                onClick={() => setDeployTarget(id)}
                className={`p-4 rounded-xl border text-left transition-all
                  ${deployTarget === id
                    ? 'border-[#58a6ff] bg-[#58a6ff1a] text-[#e6edf3]'
                    : 'border-[#30363d] bg-[#161b22] text-[#7d8590] hover:border-[#58a6ff] hover:text-[#e6edf3]'
                  }`}
              >
                <div className={`mb-2 ${deployTarget === id ? 'text-[#58a6ff]' : ''}`}>{icon}</div>
                <div className="font-medium text-sm">{label}</div>
                <div className="text-xs mt-0.5 opacity-70">{desc}</div>
              </button>
            ))}
          </div>
        </div>

        {/* Actions */}
        <div className="flex gap-3">
          <button
            onClick={() => setPage('home')}
            className="px-5 py-2.5 rounded-lg border border-[#30363d]
                       text-[#7d8590] hover:text-[#e6edf3] hover:border-[#58a6ff] transition-colors"
          >
            Back
          </button>
          <button
            onClick={handleContinue}
            disabled={!canContinue}
            className="flex-1 py-2.5 rounded-lg bg-[#58a6ff] text-white font-medium
                       disabled:opacity-30 disabled:cursor-not-allowed
                       hover:bg-[#79b8ff] transition-colors"
          >
            Start Building →
          </button>
        </div>
      </div>
    </div>
  )
}
