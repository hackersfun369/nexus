import { useEffect, useRef } from 'react'
import { useAppStore } from '@/store/app'
import { Code2, Eye, Copy, Check } from 'lucide-react'
import { useState } from 'react'
import hljs from 'highlight.js/lib/core'

// Register only the languages we need (keeps bundle small)
import go from 'highlight.js/lib/languages/go'
import python from 'highlight.js/lib/languages/python'
import typescript from 'highlight.js/lib/languages/typescript'
import javascript from 'highlight.js/lib/languages/javascript'
import kotlin from 'highlight.js/lib/languages/kotlin'
import dart from 'highlight.js/lib/languages/dart'
import xml from 'highlight.js/lib/languages/xml'
import json from 'highlight.js/lib/languages/json'
import yaml from 'highlight.js/lib/languages/yaml'
import bash from 'highlight.js/lib/languages/bash'
import sql from 'highlight.js/lib/languages/sql'
import markdown from 'highlight.js/lib/languages/markdown'
import plaintext from 'highlight.js/lib/languages/plaintext'

hljs.registerLanguage('go', go)
hljs.registerLanguage('python', python)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('kotlin', kotlin)
hljs.registerLanguage('dart', dart)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('html', xml)
hljs.registerLanguage('json', json)
hljs.registerLanguage('yaml', yaml)
hljs.registerLanguage('bash', bash)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('markdown', markdown)
hljs.registerLanguage('plaintext', plaintext)
hljs.registerLanguage('text', plaintext)
hljs.registerLanguage('dockerfile', bash)
hljs.registerLanguage('makefile', bash)

export default function CodeViewer() {
  const { activeFile, previewMode, setPreviewMode, generatedFiles } = useAppStore()
  const [copied, setCopied] = useState(false)
  const codeRef = useRef<HTMLElement>(null)

  const file = generatedFiles.find(f => f.path === activeFile)
  const code = file?.content ?? ''
  const lang = file?.language ?? 'text'

  useEffect(() => {
    if (codeRef.current && code) {
      codeRef.current.removeAttribute('data-highlighted')
      codeRef.current.textContent = code
      hljs.highlightElement(codeRef.current)
    }
  }, [code, lang])

  const handleCopy = () => {
    if (!code) return
    navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

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
                : 'text-[#7d8590] hover:text-[#e6edf3]'}`}
          >
            <Code2 className="w-3 h-3" />
            Code
          </button>
          <button
            onClick={() => setPreviewMode('preview')}
            className={`flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs transition-colors
              ${previewMode === 'preview'
                ? 'bg-[#161b22] text-[#e6edf3]'
                : 'text-[#7d8590] hover:text-[#e6edf3]'}`}
          >
            <Eye className="w-3 h-3" />
            Preview
          </button>
        </div>

        {activeFile && (
          <span className="text-xs text-[#7d8590] truncate flex-1">{activeFile}</span>
        )}

        {lang && lang !== 'text' && activeFile && (
          <span className="text-xs text-[#58a6ff] opacity-60 shrink-0">{lang}</span>
        )}

        {code && (
          <button
            onClick={handleCopy}
            className="shrink-0 p-1.5 rounded-md text-[#7d8590] hover:text-[#e6edf3]
                       hover:bg-[#30363d] transition-colors"
            title="Copy to clipboard"
          >
            {copied
              ? <Check className="w-3.5 h-3.5 text-[#3fb950]" />
              : <Copy className="w-3.5 h-3.5" />
            }
          </button>
        )}
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {previewMode === 'code' ? (
          activeFile ? (
            <pre className="p-4 text-xs font-mono leading-relaxed m-0 bg-transparent min-h-full">
              <code
                ref={codeRef}
                className={`language-${lang} hljs`}
                style={{ background: 'transparent', padding: 0 }}
              >
                {code}
              </code>
            </pre>
          ) : (
            <div className="h-full flex items-center justify-center">
              <p className="text-[#7d8590] text-sm">Select a file to view its contents</p>
            </div>
          )
        ) : (
          <div className="h-full flex items-center justify-center">
            <div className="text-center">
              <Eye className="w-8 h-8 text-[#30363d] mx-auto mb-3" />
              <p className="text-[#7d8590] text-sm">Live preview available after build</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
