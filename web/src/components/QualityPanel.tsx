import { useState } from 'react'
import { useAppStore } from '@/store/app'
import { ShieldCheck, ShieldAlert, ChevronDown, ChevronUp } from 'lucide-react'

const SEVERITY_COLOR: Record<string, string> = {
  critical: 'text-[#f85149]',
  high:     'text-[#f85149]',
  medium:   'text-[#d29922]',
  low:      'text-[#7d8590]',
}

const SEVERITY_BG: Record<string, string> = {
  critical: 'bg-[#f851491a] border-[#f8514933]',
  high:     'bg-[#f851491a] border-[#f8514933]',
  medium:   'bg-[#d299221a] border-[#d2992233]',
  low:      'bg-[#7d85901a] border-[#7d859033]',
}

export default function QualityPanel() {
  const { quality, setActiveFile } = useAppStore()
  const [expanded, setExpanded] = useState(true)
  const [activeCategory, setActiveCategory] = useState<string | null>(null)

  if (!quality) return null

  const categories = Array.from(new Set(quality.issues.map(i => i.category)))
  const filtered = activeCategory
    ? quality.issues.filter(i => i.category === activeCategory)
    : quality.issues

  const scoreColor = quality.score >= 80
    ? 'text-[#3fb950]'
    : quality.score >= 60
    ? 'text-[#d29922]'
    : 'text-[#f85149]'

  return (
    <div className="border-t border-[#30363d] shrink-0">
      {/* Header */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-3 py-2
                   hover:bg-[#30363d33] transition-colors"
      >
        <div className="flex items-center gap-2">
          {quality.passed
            ? <ShieldCheck className="w-3.5 h-3.5 text-[#3fb950]" />
            : <ShieldAlert className="w-3.5 h-3.5 text-[#d29922]" />
          }
          <span className="text-xs font-medium text-[#e6edf3]">Quality</span>
          <span className={`text-xs font-bold ${scoreColor}`}>{quality.score}/100</span>
        </div>
        <div className="flex items-center gap-1.5">
          {quality.issue_count > 0 && (
            <span className="text-xs bg-[#d299221a] text-[#d29922] border border-[#d2992233]
                             px-1.5 py-0.5 rounded-full">
              {quality.issue_count}
            </span>
          )}
          {expanded
            ? <ChevronDown className="w-3 h-3 text-[#7d8590]" />
            : <ChevronUp className="w-3 h-3 text-[#7d8590]" />
          }
        </div>
      </button>

      {expanded && (
        <div className="max-h-64 overflow-y-auto">
          {/* Category filter */}
          {categories.length > 1 && (
            <div className="flex flex-wrap gap-1 px-2 pb-2">
              <button
                onClick={() => setActiveCategory(null)}
                className={`px-2 py-0.5 rounded-full text-xs transition-colors
                  ${!activeCategory
                    ? 'bg-[#58a6ff1a] text-[#58a6ff] border border-[#58a6ff33]'
                    : 'text-[#7d8590] hover:text-[#e6edf3]'
                  }`}
              >
                all
              </button>
              {categories.map(cat => (
                <button
                  key={cat}
                  onClick={() => setActiveCategory(cat === activeCategory ? null : cat)}
                  className={`px-2 py-0.5 rounded-full text-xs transition-colors
                    ${activeCategory === cat
                      ? 'bg-[#58a6ff1a] text-[#58a6ff] border border-[#58a6ff33]'
                      : 'text-[#7d8590] hover:text-[#e6edf3]'
                    }`}
                >
                  {cat}
                </button>
              ))}
            </div>
          )}

          {/* Issues list */}
          {filtered.length === 0 ? (
            <div className="px-3 py-3 text-center">
              <ShieldCheck className="w-5 h-5 text-[#3fb950] mx-auto mb-1" />
              <p className="text-xs text-[#7d8590]">No issues found</p>
            </div>
          ) : (
            <div className="space-y-1 px-2 pb-2">
              {filtered.map((issue, idx) => (
                <button
                  key={idx}
                  onClick={() => setActiveFile(issue.file)}
                  className={`w-full text-left p-2 rounded-lg border text-xs
                             transition-colors hover:opacity-90
                             ${SEVERITY_BG[issue.severity] ?? 'bg-[#161b22] border-[#30363d]'}`}
                >
                  <div className="flex items-center justify-between mb-0.5">
                    <span className={`font-mono font-bold ${SEVERITY_COLOR[issue.severity] ?? 'text-[#7d8590]'}`}>
                      {issue.rule_id}
                    </span>
                    <span className="text-[#7d8590] truncate ml-1">
                      {issue.file.split('/').pop()}:{issue.line}
                    </span>
                  </div>
                  <p className="text-[#e6edf3] leading-relaxed">{issue.message}</p>
                  {issue.fix && (
                    <p className="text-[#7d8590] mt-0.5 italic">Fix: {issue.fix}</p>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
