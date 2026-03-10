import { useEffect, useState } from 'react'
import { useAppStore } from '@/store/app'
import { FolderOpen, Clock, ChevronRight, X } from 'lucide-react'

interface SavedProject {
  id: string
  name: string
  platform: string
  language: string
  plugin_id: string
  prompt: string
  file_count: number
  score: number
  created_at: string
}

const SCORE_COLOR = (s: number) =>
  s >= 80 ? 'text-[#3fb950]' : s >= 60 ? 'text-[#d29922]' : 'text-[#f85149]'

export default function ProjectHistory({ onClose }: { onClose: () => void }) {
  const { setGeneratedFiles, setActiveFile, setQuality, setPage, addMessage, clearMessages } = useAppStore()
  const [projects, setProjects] = useState<SavedProject[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetch('/api/v1/generated')
      .then(r => r.json())
      .then(d => setProjects(d.projects ?? []))
      .catch(() => setProjects([]))
      .finally(() => setLoading(false))
  }, [])

  const loadProject = async (p: SavedProject) => {
    try {
      const res = await fetch(`/api/v1/generated/${p.id}`)
      const data = await res.json()
      if (!data.files?.length) return

      setGeneratedFiles(data.files.map((f: any) => ({
        path: f.path,
        content: f.content,
        language: f.language,
      })))
      setActiveFile(data.files[0].path)
      setQuality(null)

      clearMessages()
      addMessage('user', p.prompt)
      addMessage('nexus',
        `✓ Loaded **${p.name}** (${p.plugin_id})\n\n` +
        `**${p.file_count} files** · Quality score: **${p.score}/100**\n\n` +
        `Restored from history. You can continue refining this project.`
      )
      setPage('builder')
      onClose()
    } catch {
      // ignore
    }
  }

  const formatDate = (dt: string) => {
    const d = new Date(dt)
    return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }

  return (
    <div className="fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-[#161b22] border border-[#30363d] rounded-2xl w-full max-w-lg max-h-[80vh]
                      flex flex-col shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between px-5 py-4 border-b border-[#30363d]">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-[#58a6ff]" />
            <span className="font-semibold text-sm">Project History</span>
          </div>
          <button onClick={onClose}
            className="p-1.5 rounded-lg text-[#7d8590] hover:text-[#e6edf3] hover:bg-[#30363d] transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* List */}
        <div className="flex-1 overflow-y-auto p-3 space-y-2">
          {loading ? (
            <div className="text-center py-8 text-[#7d8590] text-sm">Loading...</div>
          ) : projects.length === 0 ? (
            <div className="text-center py-8">
              <FolderOpen className="w-8 h-8 text-[#30363d] mx-auto mb-2" />
              <p className="text-[#7d8590] text-sm">No saved projects yet</p>
              <p className="text-[#7d8590] text-xs mt-1">Generate a project to see it here</p>
            </div>
          ) : (
            projects.map(p => (
              <button key={p.id} onClick={() => loadProject(p)}
                className="w-full text-left p-3 rounded-xl border border-[#30363d]
                           hover:border-[#58a6ff] hover:bg-[#58a6ff08] transition-all group">
                <div className="flex items-start justify-between gap-3">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-semibold text-sm text-[#e6edf3] truncate">{p.name}</span>
                      <span className="text-xs bg-[#58a6ff1a] text-[#58a6ff] border border-[#58a6ff33]
                                       px-1.5 py-0.5 rounded-full shrink-0 capitalize">{p.platform}</span>
                    </div>
                    <p className="text-xs text-[#7d8590] truncate mb-1.5">{p.prompt}</p>
                    <div className="flex items-center gap-3 text-xs text-[#7d8590]">
                      <span>{p.file_count} files</span>
                      <span className={SCORE_COLOR(p.score)}>{p.score}/100</span>
                      <span>{formatDate(p.created_at)}</span>
                    </div>
                  </div>
                  <ChevronRight className="w-4 h-4 text-[#30363d] group-hover:text-[#58a6ff]
                                           transition-colors shrink-0 mt-1" />
                </div>
              </button>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
