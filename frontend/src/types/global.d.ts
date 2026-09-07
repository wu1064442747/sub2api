import type { PublicSettings } from '@/types'

declare global {
  interface Window {
    __APP_CONFIG__?: PublicSettings
    plausible?: ((...args: unknown[]) => void) & { q?: unknown[][] }
    clarity?: ((...args: unknown[]) => void) & { q?: unknown[][] }
  }
}

export {}
