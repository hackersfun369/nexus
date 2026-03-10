const BASE = '/api/v1'

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

export interface GenerateResponse {
  app_name: string
  platform: string
  language: string
  framework: string
  plugin_id: string
  features: string[]
  entities: string[]
  files: { path: string; content: string; size: number; lang: string }[]
  file_count: number
  total_bytes: number
  duration_ms: number
  output_dir: string
  errors: string[]
  quality?: QualityReport
}

export async function generatePreview(
  prompt: string,
  platform?: string,
  languages?: string[]
): Promise<GenerateResponse> {
  const res = await fetch(`${BASE}/generate`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ prompt, platform, languages, preview: true }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'unknown error' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export function downloadZipUrl(prompt: string, platform?: string, language?: string): string {
  const params = new URLSearchParams({ prompt })
  if (platform) params.set('platform', platform)
  if (language) params.set('language', language)
  return `${BASE}/generate/zip?${params.toString()}`
}
