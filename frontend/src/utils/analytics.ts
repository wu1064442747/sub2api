export interface AnalyticsSettings {
  plausible_domain?: string
  plausible_script_url?: string
  microsoft_clarity_project_id?: string
}

const DEFAULT_PLAUSIBLE_SCRIPT_URL = 'https://plausible.ai-baby-dance.com/js/script.js'
const CLARITY_PROJECT_ID_PATTERN = /^[A-Za-z0-9_-]+$/
const ANALYTICS_MARKER = 'data-sub2api-analytics'

function isHTTPSURL(value: string): boolean {
  try {
    return new URL(value).protocol === 'https:'
  } catch {
    return false
  }
}

function hasAnalyticsScript(kind: string): boolean {
  return Boolean(document.querySelector(`script[${ANALYTICS_MARKER}="${kind}"]`))
}

function initializePlausible(settings: AnalyticsSettings): void {
  const domain = settings.plausible_domain?.trim()
  const scriptURL = settings.plausible_script_url?.trim() || DEFAULT_PLAUSIBLE_SCRIPT_URL
  if (!domain || !isHTTPSURL(scriptURL) || hasAnalyticsScript('plausible')) {
    return
  }

  window.plausible = window.plausible || ((...args: unknown[]) => {
    window.plausible!.q = window.plausible!.q || []
    window.plausible!.q.push(args)
  })

  const script = document.createElement('script')
  script.defer = true
  script.src = scriptURL
  script.dataset.domain = domain
  script.dataset.spa = 'auto'
  script.dataset.sub2apiAnalytics = 'plausible'
  document.head.appendChild(script)
}

function initializeClarity(settings: AnalyticsSettings): void {
  const projectID = settings.microsoft_clarity_project_id?.trim()
  if (!projectID || !CLARITY_PROJECT_ID_PATTERN.test(projectID) || hasAnalyticsScript('clarity')) {
    return
  }

  window.clarity = window.clarity || ((...args: unknown[]) => {
    window.clarity!.q = window.clarity!.q || []
    window.clarity!.q.push(args)
  })

  const script = document.createElement('script')
  script.async = true
  script.src = `https://www.clarity.ms/tag/${encodeURIComponent(projectID)}`
  script.dataset.sub2apiAnalytics = 'clarity'
  document.head.appendChild(script)
}

export function initializeAnalytics(settings: AnalyticsSettings | null | undefined): void {
  if (typeof window === 'undefined' || typeof document === 'undefined' || !settings) {
    return
  }

  initializePlausible(settings)
  initializeClarity(settings)
}
