import { useEffect } from 'react'
import { useAppStore } from '@/store/app'
import { generatePreview } from '@/store/api'
import ChatPanel from '@/components/ChatPanel'
import FileTree from '@/components/FileTree'
import CodeViewer from '@/components/CodeViewer'
import TopBar from '@/components/TopBar'

export default function BuilderPage() {
  const {
    addMessage, messages,
    selectedPlatforms, selectedLanguages,
    setGeneratedFiles, setActiveFile,
    setIsGenerating,
  } = useAppStore()

  useEffect(() => {
    if (messages.length === 1) {
      const prompt = messages[0].content
      const platform = selectedPlatforms[0] ?? ''
      const languages = selectedLanguages

      setIsGenerating(true)
      addMessage('nexus',
        `Analyzing your requirements and selecting plugins...\n\n` +
        `• Platform: **${platform || 'web'}**\n` +
        `• Languages: **${languages.length > 0 ? languages.join(', ') : 'auto-detect'}**\n\n` +
        `Generating project structure...`
      )

      generatePreview(prompt, platform, languages)
        .then((result) => {
          // Store generated files
          setGeneratedFiles(result.files.map(f => ({
            path: f.path,
            content: f.content,
            language: f.lang,
          })))

          // Set first file active
          if (result.files.length > 0) {
            setActiveFile(result.files[0].path)
          }

          const featureList = result.features.length > 0
            ? result.features.map(f => `• ${f}`).join('\n')
            : '• Core functionality'

          addMessage('nexus',
            `✓ Generated **${result.app_name}** using **${result.plugin_id}**\n\n` +
            `**${result.file_count} files** · ${result.total_bytes} bytes · ${result.duration_ms}ms\n\n` +
            `**Detected features:**\n${featureList}\n\n` +
            `Select any file in the tree to view its code. Click **Download** to get the full project as a ZIP.`
          )
        })
        .catch((err) => {
          addMessage('nexus',
            `✗ Generation failed: ${err.message}\n\nMake sure the NEXUS server is running.`
          )
        })
        .finally(() => setIsGenerating(false))
    }
  }, [])

  return (
    <div className="h-full flex flex-col">
      <TopBar />
      <div className="flex-1 flex overflow-hidden">
        <div className="w-56 border-r border-[#30363d] flex flex-col bg-[#161b22]">
          <FileTree />
        </div>
        <div className="flex-1 flex flex-col border-r border-[#30363d]">
          <ChatPanel />
        </div>
        <div className="w-[45%] flex flex-col">
          <CodeViewer />
        </div>
      </div>
    </div>
  )
}
