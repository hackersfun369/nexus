import { useAppStore } from '@/store/app'
import { File, Folder, FolderOpen } from 'lucide-react'

const DEMO_FILES = [
  { path: 'src/', type: 'folder', depth: 0 },
  { path: 'src/main.py', type: 'file', depth: 1 },
  { path: 'src/api/', type: 'folder', depth: 1 },
  { path: 'src/api/routes.py', type: 'file', depth: 2 },
  { path: 'src/api/models.py', type: 'file', depth: 2 },
  { path: 'src/db/', type: 'folder', depth: 1 },
  { path: 'src/db/schema.sql', type: 'file', depth: 2 },
  { path: 'tests/', type: 'folder', depth: 0 },
  { path: 'tests/test_api.py', type: 'file', depth: 1 },
  { path: 'requirements.txt', type: 'file', depth: 0 },
  { path: 'README.md', type: 'file', depth: 0 },
]

function fileIcon(path: string) {
  if (path.endsWith('.py'))  return '🐍'
  if (path.endsWith('.ts') || path.endsWith('.tsx')) return '🔷'
  if (path.endsWith('.js') || path.endsWith('.jsx')) return '🟨'
  if (path.endsWith('.sql')) return '🗄️'
  if (path.endsWith('.md'))  return '📝'
  if (path.endsWith('.txt') || path.endsWith('.toml') || path.endsWith('.yaml')) return '📄'
  if (path.endsWith('.kt'))  return '🟣'
  if (path.endsWith('.go'))  return '🔵'
  return '📄'
}

export default function FileTree() {
  const { activeFile, setActiveFile, generatedFiles } = useAppStore()
  const files = generatedFiles.length > 0
    ? generatedFiles.map(f => ({ path: f.path, type: 'file' as const, depth: f.path.split('/').length - 1 }))
    : DEMO_FILES

  return (
    <div className="flex flex-col h-full">
      <div className="px-3 py-2.5 border-b border-[#30363d] shrink-0">
        <p className="text-xs text-[#7d8590] uppercase tracking-wider">Files</p>
      </div>
      <div className="flex-1 overflow-y-auto py-1">
        {files.map((f) => (
          <button
            key={f.path}
            onClick={() => f.type === 'file' && setActiveFile(f.path)}
            className={`w-full flex items-center gap-1.5 px-3 py-1 text-xs text-left
              transition-colors hover:bg-[#30363d33]
              ${activeFile === f.path ? 'bg-[#58a6ff1a] text-[#58a6ff]' : 'text-[#7d8590]'}
            `}
            style={{ paddingLeft: `${12 + f.depth * 12}px` }}
          >
            {f.type === 'folder'
              ? <FolderOpen className="w-3.5 h-3.5 text-[#58a6ff] shrink-0" />
              : <span className="shrink-0">{fileIcon(f.path)}</span>
            }
            <span className="truncate">
              {f.path.split('/').filter(Boolean).pop()}
            </span>
          </button>
        ))}
      </div>
    </div>
  )
}
