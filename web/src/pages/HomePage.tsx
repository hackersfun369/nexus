import { useState } from 'react'
import { useAppStore } from '@/store/app'
import { ArrowRight, Zap, Shield, Globe, Terminal, Cpu } from 'lucide-react'

const EXAMPLE_PROMPTS = [
  "Build me a food delivery app for Android",
  "Create a React dashboard with authentication",
  "Make a REST API backend with Python and PostgreSQL",
  "Build a cross-platform desktop app with file management",
  "Create a Flutter mobile app with real-time chat",
]

export default function HomePage() {
  const { setPage, addMessage } = useAppStore()
  const [input, setInput] = useState('')

  const handleStart = (prompt: string) => {
    if (!prompt.trim()) return
    addMessage('user', prompt)
    setPage('wizard')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleStart(input)
    }
  }

  return (
    <div className="h-full flex flex-col items-center justify-center px-4">
      {/* Logo */}
      <div className="mb-8 text-center">
        <div className="flex items-center justify-center gap-3 mb-3">
          <Cpu className="w-10 h-10 text-[#58a6ff]" />
          <h1 className="text-4xl font-bold tracking-tight">NEXUS</h1>
        </div>
        <p className="text-[#7d8590] text-lg">
          Intelligent Development System — Symbolic AI Edition
        </p>
        <p className="text-[#7d8590] text-sm mt-1">
          No LLM · No Cloud · Runs entirely on your machine
        </p>
      </div>

      {/* Main input */}
      <div className="w-full max-w-2xl mb-6">
        <div className="relative">
          <textarea
            className="w-full bg-[#161b22] border border-[#30363d] rounded-xl px-5 py-4 pr-14
                       text-[#e6edf3] placeholder-[#7d8590] resize-none outline-none
                       focus:border-[#58a6ff] transition-colors text-base leading-relaxed"
            placeholder="Describe what you want to build..."
            rows={3}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
          />
          <button
            onClick={() => handleStart(input)}
            disabled={!input.trim()}
            className="absolute right-3 bottom-3 p-2 rounded-lg bg-[#58a6ff]
                       disabled:opacity-30 disabled:cursor-not-allowed
                       hover:bg-[#79b8ff] transition-colors"
          >
            <ArrowRight className="w-5 h-5 text-white" />
          </button>
        </div>
        <p className="text-[#7d8590] text-xs mt-2 text-center">
          Press Enter to continue · Shift+Enter for new line
        </p>
      </div>

      {/* Example prompts */}
      <div className="w-full max-w-2xl mb-10">
        <p className="text-[#7d8590] text-xs uppercase tracking-wider mb-3 text-center">
          Try an example
        </p>
        <div className="flex flex-wrap gap-2 justify-center">
          {EXAMPLE_PROMPTS.map((prompt) => (
            <button
              key={prompt}
              onClick={() => handleStart(prompt)}
              className="px-3 py-1.5 text-sm bg-[#161b22] border border-[#30363d]
                         rounded-full text-[#7d8590] hover:text-[#e6edf3]
                         hover:border-[#58a6ff] transition-colors"
            >
              {prompt}
            </button>
          ))}
        </div>
      </div>

      {/* Feature pills */}
      <div className="flex flex-wrap gap-4 justify-center">
        {[
          { icon: <Zap className="w-4 h-4" />,     label: '33 quality rules' },
          { icon: <Shield className="w-4 h-4" />,   label: 'Security analysis' },
          { icon: <Globe className="w-4 h-4" />,    label: 'Multi-platform' },
          { icon: <Terminal className="w-4 h-4" />, label: 'Plugin system' },
        ].map(({ icon, label }) => (
          <div key={label}
            className="flex items-center gap-2 px-3 py-1.5 rounded-full
                       bg-[#161b22] border border-[#30363d] text-[#7d8590] text-sm">
            <span className="text-[#58a6ff]">{icon}</span>
            {label}
          </div>
        ))}
      </div>
    </div>
  )
}
