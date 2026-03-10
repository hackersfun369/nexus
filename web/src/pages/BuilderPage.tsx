import { useEffect } from 'react'
import { useAppStore } from '@/store/app'
import { generatePreview } from '@/store/api'
import ChatPanel from '@/components/ChatPanel'
import FileTree from '@/components/FileTree'
import CodeViewer from '@/components/CodeViewer'
import TopBar from '@/components/TopBar'
import QualityPanel from '@/components/QualityPanel'

export default function BuilderPage() {
  const {
    addMessage, messages,
    selectedPlatforms, selectedLanguages,
    setGeneratedFiles, setActiveFile,
    setIsGenerating, setQuality,
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

          const featureList = result.features.length > 0
            ? result.features.map(f => `• ${f}`).join('\n')
            : '• Core functionality'

          const qualityLine = result.quality
            ? `\n\n**Quality score: ${result.quality.score}/100** ${result.quality.passed ? '✓ Passed' : '⚠ Issues found'} · ${result.quality.issue_count} issues`
            : ''

          addMessage('nexus',
            `✓ Generated **${result.app_name}** using **${result.plugin_id}**\n\n` +
            `**${result.file_count} files** · ${result.total_bytes} bytes · ${result.duration_ms}ms\n\n` +
            `**Detected features:**\n${featureList}` +
            qualityLine + `\n\nSelect any file in the tree to view its code.`
          )
        })
        .catch((err) => {
          addMessage('nexus', `✗ Generation failed: ${err.message}`)
        })
        .finally(() => setIsGenerating(false))
    }
  }, [])

  return (
    <div className="h-full flex flex-col">
      <TopBar />
      <div className="flex-1 flex overflow-hidden">
        {/* Left: file tree + quality */}
        <div className="w-56 border-r border-[#30363d] flex flex-col bg-[#161b22]">
          <FileTree />
          <QualityPanel />
        </div>

        {/* Middle: chat */}
        <div className="flex-1 flex flex-col border-r border-[#30363d]">
          <ChatPanel />
        </div>

        {/* Right: code viewer */}
        <div className="w-[45%] flex flex-col">
          <CodeViewer />
        </div>
      </div>
    </div>
  )
}
