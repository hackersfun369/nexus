import { useAppStore } from '@/store/app'
import { Cpu, Settings, Download, Play, Home } from 'lucide-react'

export default function TopBar() {
  const { project, selectedPlatforms, isGenerating, setPage } = useAppStore()

  return (
    <div className="h-12 border-b border-[#30363d] bg-[#161b22] flex items-center px-4 gap-4 shrink-0">
      {/* Logo */}
      <div className="flex items-center gap-2">
        <Cpu className="w-5 h-5 text-[#58a6ff]" />
        <span className="font-semibold text-sm">NEXUS</span>
      </div>

      <div className="w-px h-5 bg-[#30363d]" />

      {/* Project name */}
      <span className="text-[#7d8590] text-sm">
        {project?.name ?? 'New Project'}
      </span>

      {/* Platforms */}
      <div className="flex gap-1.5">
        {selectedPlatforms.map(p => (
          <span key={p}
            className="px-2 py-0.5 rounded-full bg-[#58a6ff1a] text-[#58a6ff]
                       border border-[#58a6ff33] text-xs capitalize">
            {p}
          </span>
        ))}
      </div>

      <div className="flex-1" />

      {/* Actions */}
      <div className="flex items-center gap-2">
        {isGenerating && (
          <span className="text-[#7d8590] text-xs flex items-center gap-1.5">
            <span className="w-1.5 h-1.5 rounded-full bg-[#58a6ff] animate-pulse" />
            Generating...
          </span>
        )}

        <button
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm
                     border border-[#30363d] text-[#7d8590]
                     hover:text-[#e6edf3] hover:border-[#58a6ff] transition-colors"
        >
          <Play className="w-3.5 h-3.5" />
          Preview
        </button>

        <button
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm
                     bg-[#238636] text-white hover:bg-[#2ea043] transition-colors"
        >
          <Download className="w-3.5 h-3.5" />
          Download
        </button>

        <button
          onClick={() => setPage('home')}
          className="p-1.5 rounded-lg text-[#7d8590] hover:text-[#e6edf3] transition-colors"
        >
          <Home className="w-4 h-4" />
        </button>
      </div>
    </div>
  )
}
