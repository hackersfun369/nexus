import { useAppStore } from '@/store/app'
import { Code2, Eye } from 'lucide-react'

export default function CodeViewer() {
  const { activeFile, previewMode, setPreviewMode, generatedFiles } = useAppStore()

  const file = generatedFiles.find(f => f.path === activeFile)
  const code = file?.content ?? ''
  const lang = file?.language ?? 'text'

  return (
    <div className="flex flex-col h-full bg-[#0d1117]">
      {/* Header */}
      <div className="h-10 border-b border-[#30363d] flex items-center px-4 gap-3 shrink-0 bg-[#161b22]">
        <div className="flex gap-1 bg-[#0d1117] rounded-lg p-0.5">
          <button
            onClick={() => setPreviewMode('code')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs transition-colors
              ${previewMode === 'code'
                ? 'bg-[#161b22] text-[#e6edf3]'
                : 'text-[#7d8590] hover:text-[#e6edf3]'
              }`}
          >
            <Code2 className="w-3 h-3" />
            Code
          </button>
          <button
            onClick={() => setPreviewMode('preview')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs transition-colors
              ${previewMode === 'preview'
                ? 'bg-[#161b22] text-[#e6edf3]'
                : 'text-[#7d8590] hover:text-[#e6edf3]'
              }`}
          >
            <Eye className="w-3 h-3" />
            Preview
          </button>
        </div>

        {activeFile && (
          <span className="text-xs text-[#7d8590] truncate">{activeFile}</span>
        )}
        {lang && lang !== 'text' && activeFile && (
          <span className="ml-auto text-xs text-[#58a6ff] opacity-60">{lang}</span>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {previewMode === 'code' ? (
          <pre className="p-4 text-xs text-[#e6edf3] font-mono leading-relaxed min-h-full">
            {activeFile
              ? code
              : <span className="text-[#7d8590]">Select a file from the tree to view its contents</span>
            }
          </pre>
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <Eye className="w-8 h-8 text-[#30363d] mx-auto mb-3" />
              <p className="text-[#7d8590] text-sm">Live preview</p>
              <p className="text-[#7d8590] text-xs mt-1">Available after code generation</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
