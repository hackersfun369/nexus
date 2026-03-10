import { create } from 'zustand'

export type Page = 'home' | 'wizard' | 'builder'
export type Platform = 'android' | 'web' | 'windows' | 'mac' | 'cli' | 'backend' | 'all'
export type DeployTarget = 'local' | 'cloud' | 'both'

export interface Message {
  id: string
  role: 'user' | 'nexus'
  content: string
  timestamp: Date
}

export interface QualityIssue {
  rule_id: string
  severity: string
  category: string
  file: string
  line: number
  message: string
  fix: string
}

export interface QualityReport {
  issues: QualityIssue[]
  issue_count: number
  score: number
  passed: boolean
}

export interface Project {
  id: string
  name: string
  description: string
  platforms: Platform[]
  languages: string[]
  deployTarget: DeployTarget
  createdAt: Date
}

export interface GeneratedFile {
  path: string
  content: string
  language: string
}

interface AppState {
  // Navigation
  page: Page
  setPage: (page: Page) => void

  // Current project
  project: Project | null
  setProject: (project: Project) => void

  // Chat messages
  messages: Message[]
  addMessage: (role: 'user' | 'nexus', content: string) => void
  clearMessages: () => void

  // Wizard state
  selectedPlatforms: Platform[]
  togglePlatform: (platform: Platform) => void
  selectedLanguages: string[]
  toggleLanguage: (lang: string) => void
  deployTarget: DeployTarget
  setDeployTarget: (target: DeployTarget) => void

  // Generated files
  generatedFiles: GeneratedFile[]
  setGeneratedFiles: (files: GeneratedFile[]) => void
  activeFile: string | null
  setActiveFile: (path: string | null) => void

  // Quality
  quality: QualityReport | null
  setQuality: (q: QualityReport | null) => void

  // UI state
  sidebarOpen: boolean
  setSidebarOpen: (open: boolean) => void
  previewMode: 'code' | 'preview'
  setPreviewMode: (mode: 'code' | 'preview') => void
  isAnalyzing: boolean
  setIsAnalyzing: (v: boolean) => void
  isGenerating: boolean
  setIsGenerating: (v: boolean) => void
}

let msgCounter = 0

export const useAppStore = create<AppState>((set) => ({
  // Navigation
  page: 'home',
  setPage: (page) => set({ page }),

  // Project
  project: null,
  setProject: (project) => set({ project }),

  // Messages
  messages: [],
  addMessage: (role, content) => set((state) => ({
    messages: [...state.messages, {
      id: String(++msgCounter),
      role,
      content,
      timestamp: new Date(),
    }]
  })),
  clearMessages: () => set({ messages: [] }),

  // Wizard
  selectedPlatforms: [],
  togglePlatform: (platform) => set((state) => ({
    selectedPlatforms: state.selectedPlatforms.includes(platform)
      ? state.selectedPlatforms.filter(p => p !== platform)
      : [...state.selectedPlatforms, platform]
  })),
  selectedLanguages: [],
  toggleLanguage: (lang) => set((state) => ({
    selectedLanguages: state.selectedLanguages.includes(lang)
      ? state.selectedLanguages.filter(l => l !== lang)
      : [...state.selectedLanguages, lang]
  })),
  deployTarget: 'local',
  setDeployTarget: (deployTarget) => set({ deployTarget }),

  // Files
  generatedFiles: [],
  setGeneratedFiles: (generatedFiles) => set({ generatedFiles }),
  activeFile: null,
  setActiveFile: (activeFile) => set({ activeFile }),

  // UI
  sidebarOpen: true,
  setSidebarOpen: (sidebarOpen) => set({ sidebarOpen }),
  previewMode: 'code',
  setPreviewMode: (previewMode) => set({ previewMode }),
  isAnalyzing: false,
  setIsAnalyzing: (isAnalyzing) => set({ isAnalyzing }),
  isGenerating: false,
  setIsGenerating: (isGenerating) => set({ isGenerating }),

  quality: null,
  setQuality: (quality) => set({ quality }),
}))
