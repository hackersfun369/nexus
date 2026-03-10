import { useState, useRef, useEffect } from 'react'
import { useAppStore } from '@/store/app'
import { Send, Cpu, User } from 'lucide-react'

function renderMessage(content: string) {
  // Simple markdown-like rendering
  return content
    .split('\n')
    .map((line, i) => {
      // Bold
      line = line.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>')
      // Bullet
      if (line.startsWith('• ')) {
        return `<div key=${i} class="flex gap-2 text-sm"><span class="text-[#58a6ff] mt-0.5">•</span><span>${line.slice(2)}</span></div>`
      }
      return `<p key=${i} class="text-sm leading-relaxed">${line}</p>`
    })
    .join('')
}

export default function ChatPanel() {
  const { messages, addMessage, isGenerating, setIsGenerating } = useAppStore()
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const handleSend = () => {
    if (!input.trim() || isGenerating) return
    const text = input.trim()
    setInput('')
    addMessage('user', text)
    setIsGenerating(true)

    // Simulate NEXUS response
    setTimeout(() => {
      addMessage('nexus',
        `I understand you want: **${text}**\n\n` +
        `I'm resolving the required plugins and generating the code structure. ` +
        `This will appear in the file tree on the left and code viewer on the right.\n\n` +
        `• Plugin resolution: ✓\n` +
        `• Architecture: ✓\n` +
        `• Code generation: in progress...`
      )
      setIsGenerating(false)
    }, 1200)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header */}
      <div className="px-4 py-3 border-b border-[#30363d] shrink-0">
        <h2 className="text-sm font-medium text-[#7d8590]">Chat</h2>
      </div>

      {/* Messages */}
      <div className="flex-1 overflow-y-auto px-4 py-4 space-y-4">
        {messages.map((msg) => (
          <div key={msg.id}
            className={`flex gap-3 ${msg.role === 'user' ? 'flex-row-reverse' : ''}`}>
            {/* Avatar */}
            <div className={`w-7 h-7 rounded-full flex items-center justify-center shrink-0
              ${msg.role === 'nexus'
                ? 'bg-[#58a6ff1a] border border-[#58a6ff33]'
                : 'bg-[#161b22] border border-[#30363d]'
              }`}>
              {msg.role === 'nexus'
                ? <Cpu className="w-3.5 h-3.5 text-[#58a6ff]" />
                : <User className="w-3.5 h-3.5 text-[#7d8590]" />
              }
            </div>

            {/* Bubble */}
            <div className={`max-w-[80%] rounded-xl px-4 py-2.5 space-y-1
              ${msg.role === 'nexus'
                ? 'bg-[#161b22] border border-[#30363d] text-[#e6edf3]'
                : 'bg-[#58a6ff1a] border border-[#58a6ff33] text-[#e6edf3]'
              }`}
              dangerouslySetInnerHTML={{ __html: renderMessage(msg.content) }}
            />
          </div>
        ))}

        {isGenerating && (
          <div className="flex gap-3">
            <div className="w-7 h-7 rounded-full flex items-center justify-center
                            bg-[#58a6ff1a] border border-[#58a6ff33] shrink-0">
              <Cpu className="w-3.5 h-3.5 text-[#58a6ff]" />
            </div>
            <div className="bg-[#161b22] border border-[#30363d] rounded-xl px-4 py-3">
              <div className="flex gap-1">
                {[0,1,2].map(i => (
                  <span key={i}
                    className="w-1.5 h-1.5 rounded-full bg-[#58a6ff] animate-bounce"
                    style={{ animationDelay: `${i * 150}ms` }}
                  />
                ))}
              </div>
            </div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="p-4 border-t border-[#30363d] shrink-0">
        <div className="flex gap-2">
          <textarea
            className="flex-1 bg-[#161b22] border border-[#30363d] rounded-lg px-3 py-2
                       text-sm text-[#e6edf3] placeholder-[#7d8590] resize-none outline-none
                       focus:border-[#58a6ff] transition-colors"
            placeholder="Describe a feature, ask a question..."
            rows={2}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isGenerating}
            className="px-3 rounded-lg bg-[#58a6ff] disabled:opacity-30
                       disabled:cursor-not-allowed hover:bg-[#79b8ff] transition-colors"
          >
            <Send className="w-4 h-4 text-white" />
          </button>
        </div>
      </div>
    </div>
  )
}
