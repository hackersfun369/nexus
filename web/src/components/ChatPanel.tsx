import { useState, useRef, useEffect } from 'react'
import { useAppStore } from '@/store/app'
import { generatePreview } from '@/store/api'
import { Send } from 'lucide-react'

function MessageBubble({ role, content }: { role: string; content: string }) {
  const isNexus = role === 'nexus'

  // Simple markdown-ish rendering: **bold**, bullet points
  const renderContent = (text: string) => {
    return text.split('\n').map((line, i) => {
      // Bold
      const parts = line.split(/\*\*(.*?)\*\*/g)
      const rendered = parts.map((part, j) =>
        j % 2 === 1 ? <strong key={j} className="text-[#e6edf3] font-semibold">{part}</strong> : part
      )
      // Bullet
      if (line.startsWith('• ') || line.startsWith('- ')) {
        return (
          <div key={i} className="flex gap-2 my-0.5">
            <span className="text-[#58a6ff] shrink-0">•</span>
            <span>{rendered}</span>
          </div>
        )
      }
      if (line === '') return <div key={i} className="h-2" />
      return <div key={i}>{rendered}</div>
    })
  }

  return (
    <div className={`flex gap-3 ${isNexus ? '' : 'flex-row-reverse'}`}>
      <div className={`w-7 h-7 rounded-full shrink-0 flex items-center justify-center text-xs font-bold
        ${isNexus
          ? 'bg-[#58a6ff1a] border border-[#58a6ff33] text-[#58a6ff]'
          : 'bg-[#3fb9501a] border border-[#3fb95033] text-[#3fb950]'
        }`}>
        {isNexus ? 'N' : 'U'}
      </div>
      <div className={`max-w-[85%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed
        ${isNexus
          ? 'bg-[#161b22] border border-[#30363d] text-[#7d8590] rounded-tl-sm'
          : 'bg-[#58a6ff1a] border border-[#58a6ff33] text-[#e6edf3] rounded-tr-sm'
        }`}>
        {renderContent(content)}
      </div>
    </div>
  )
}

export default function ChatPanel() {
  const {
    messages, addMessage,
    selectedPlatforms, selectedLanguages,
    setGeneratedFiles, setActiveFile,
    setIsGenerating, isGenerating,
    setQuality, generatedFiles,
  } = useAppStore()

  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = async () => {
    const text = input.trim()
    if (!text || isGenerating) return
    setInput('')

    addMessage('user', text)

    // Build a refined prompt combining original + follow-up
    const originalPrompt = messages[0]?.content ?? ''
    const platform = selectedPlatforms[0] ?? ''
    const existingFiles = generatedFiles.map(f => f.path).join(', ')

    const refinedPrompt = `${originalPrompt}. Additional requirement: ${text}`

    setIsGenerating(true)
    addMessage('nexus', `Refining project based on your request...\n\nUpdating: ${existingFiles || 'all files'}`)

    try {
      const result = await generatePreview(refinedPrompt, platform, selectedLanguages)

      setGeneratedFiles(result.files.map(f => ({
        path: f.path,
        content: f.content,
        language: f.lang,
      })))

      if (result.files.length > 0) {
        setActiveFile(result.files[0].path)
      }

      if (result.quality) {
        setQuality(result.quality)
      }

      addMessage('nexus',
        `✓ Project updated — **${result.file_count} files** regenerated\n\n` +
        `**Quality score: ${result.quality?.score ?? '–'}/100** · ${result.quality?.issue_count ?? 0} issues\n\n` +
        `Files updated:\n${result.files.map(f => `• ${f.path}`).join('\n')}`
      )
    } catch (err: any) {
      addMessage('nexus', `✗ Failed to refine: ${err.message}`)
    } finally {
      setIsGenerating(false)
    }
  }

  const handleKey = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Messages */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.map((msg, i) => (
          <MessageBubble key={i} role={msg.role} content={msg.content} />
        ))}
        {isGenerating && (
          <div className="flex gap-3">
            <div className="w-7 h-7 rounded-full shrink-0 flex items-center justify-center text-xs font-bold
                            bg-[#58a6ff1a] border border-[#58a6ff33] text-[#58a6ff]">N</div>
            <div className="bg-[#161b22] border border-[#30363d] rounded-2xl rounded-tl-sm px-4 py-3">
              <div className="flex gap-1.5 items-center">
                <span className="w-1.5 h-1.5 rounded-full bg-[#58a6ff] animate-bounce" style={{animationDelay:'0ms'}} />
                <span className="w-1.5 h-1.5 rounded-full bg-[#58a6ff] animate-bounce" style={{animationDelay:'150ms'}} />
                <span className="w-1.5 h-1.5 rounded-full bg-[#58a6ff] animate-bounce" style={{animationDelay:'300ms'}} />
              </div>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="border-t border-[#30363d] p-3 shrink-0">
        <div className="flex gap-2 items-end bg-[#161b22] border border-[#30363d]
                        rounded-xl px-3 py-2 focus-within:border-[#58a6ff] transition-colors">
          <textarea
            value={input}
            onChange={e => setInput(e.target.value)}
            onKeyDown={handleKey}
            placeholder="Describe a feature, ask a question..."
            rows={1}
            className="flex-1 bg-transparent text-sm text-[#e6edf3] placeholder-[#7d8590]
                       resize-none outline-none leading-relaxed max-h-32"
            style={{ fieldSizing: 'content' } as any}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isGenerating}
            className="shrink-0 w-7 h-7 rounded-lg bg-[#58a6ff] text-white flex items-center justify-center
                       hover:bg-[#79c0ff] transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            <Send className="w-3.5 h-3.5" />
          </button>
        </div>
        <p className="text-[#7d8590] text-xs mt-1.5 px-1">Enter to send · Shift+Enter for new line</p>
      </div>
    </div>
  )
}
