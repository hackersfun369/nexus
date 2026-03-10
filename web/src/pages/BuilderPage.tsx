import { useEffect } from 'react'
import { useAppStore } from '@/store/app'
import ChatPanel from '@/components/ChatPanel'
import FileTree from '@/components/FileTree'
import CodeViewer from '@/components/CodeViewer'
import TopBar from '@/components/TopBar'

export default function BuilderPage() {
  const { addMessage, messages, selectedPlatforms, selectedLanguages } = useAppStore()

  useEffect(() => {
    if (messages.length === 1) {
      // NEXUS responds to the initial prompt
      const platforms = selectedPlatforms.join(', ') || 'web'
      const languages = selectedLanguages.length > 0
        ? selectedLanguages.join(', ')
        : 'best fit for your platform'

      setTimeout(() => {
        addMessage('nexus',
          `Got it! I'll build your **${platforms}** application using **${languages}**.\n\n` +
          `Here's what I'm planning:\n` +
          `• Resolving required language plugins...\n` +
          `• Setting up project architecture...\n` +
          `• Generating code structure...\n\n` +
          `Tell me more about the features you need — authentication, database, payments, etc.`
        )
      }, 600)
    }
  }, [])

  return (
    <div className="h-full flex flex-col">
      <TopBar />
      <div className="flex-1 flex overflow-hidden">
        {/* Left: File tree */}
        <div className="w-56 border-r border-[#30363d] flex flex-col bg-[#161b22]">
          <FileTree />
        </div>

        {/* Middle: Chat */}
        <div className="flex-1 flex flex-col border-r border-[#30363d]">
          <ChatPanel />
        </div>

        {/* Right: Code viewer */}
        <div className="w-[45%] flex flex-col">
          <CodeViewer />
        </div>
      </div>
    </div>
  )
}
