import { afterEach, describe, expect, it } from 'vitest'

import { initializeAnalytics } from '@/utils/analytics'

describe('analytics', () => {
  afterEach(() => {
    document.head.innerHTML = ''
    delete window.plausible
    delete window.clarity
  })

  it('does not inject scripts when analytics is not configured', () => {
    initializeAnalytics({})

    expect(document.head.querySelectorAll('script')).toHaveLength(0)
  })

  it('injects Plausible once with SPA tracking enabled', () => {
    initializeAnalytics({ plausible_domain: 'sub2api.ai-baby-dance.com' })
    initializeAnalytics({ plausible_domain: 'sub2api.ai-baby-dance.com' })

    const script = document.head.querySelector<HTMLScriptElement>('script[data-sub2api-analytics="plausible"]')
    expect(script).not.toBeNull()
    expect(script?.src).toBe('https://plausible.ai-baby-dance.com/js/script.js')
    expect(script?.dataset.domain).toBe('sub2api.ai-baby-dance.com')
    expect(script?.dataset.spa).toBe('auto')
    expect(document.head.querySelectorAll('script[data-sub2api-analytics="plausible"]')).toHaveLength(1)
  })

  it('queues Clarity calls and injects a valid project once', () => {
    initializeAnalytics({ microsoft_clarity_project_id: 'clarity-test' })
    window.clarity?.('consent')
    initializeAnalytics({ microsoft_clarity_project_id: 'clarity-test' })

    const script = document.head.querySelector<HTMLScriptElement>('script[data-sub2api-analytics="clarity"]')
    expect(script?.src).toBe('https://www.clarity.ms/tag/clarity-test')
    expect(window.clarity?.q).toEqual([['consent']])
    expect(document.head.querySelectorAll('script[data-sub2api-analytics="clarity"]')).toHaveLength(1)
  })

  it('ignores invalid script URLs and Clarity project IDs', () => {
    initializeAnalytics({
      plausible_domain: 'sub2api.ai-baby-dance.com',
      plausible_script_url: 'javascript:alert(1)',
      microsoft_clarity_project_id: 'bad project id'
    })

    expect(document.head.querySelectorAll('script')).toHaveLength(0)
  })
})
