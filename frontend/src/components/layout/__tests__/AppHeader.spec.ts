import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import zh from '@/i18n/locales/zh/common'
import AppHeader from '../AppHeader.vue'

const { appStore, authStore } = vi.hoisted(() => ({
  appStore: { cachedPublicSettings: { model_plaza_enabled: true }, docUrl: '' },
  authStore: { user: null as { username: string } | null },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
  useOnboardingStore: () => ({}),
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/adminSettings', () => ({ useAdminSettingsStore: () => ({}) }))
vi.mock('vue-router', () => ({ useRoute: () => ({ meta: {} }), useRouter: () => ({}) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...(await importOriginal<typeof import('vue-i18n')>()),
  useI18n: () => ({ t: (key: string) => ({
    'nav.modelPlaza': zh.nav.modelPlaza,
    'nav.chatgptClaudePurchase': zh.nav.chatgptClaudePurchase,
  })[key] ?? key }),
}))

const shopUrl = 'https://www.16688.com.cn/shop/S447242'

function mountHeader({ signedIn, plazaEnabled }: { signedIn: boolean; plazaEnabled: boolean }) {
  appStore.cachedPublicSettings.model_plaza_enabled = plazaEnabled
  authStore.user = signedIn ? { username: 'test' } : null
  return shallowMount(AppHeader, { global: { stubs: { RouterLink: true } } })
}

describe('AppHeader purchase link', () => {
  it('renders immediately before Model Plaza and opens the shop in a new tab', () => {
    const wrapper = mountHeader({ signedIn: true, plazaEnabled: true })
    try {
      const link = wrapper.get(`a[href="${shopUrl}"]`)
      expect(link.text()).toBe('ChatGPT和Claude购买')
      expect(link.attributes('aria-label')).toBe('ChatGPT和Claude购买')
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toBe('noopener noreferrer')
      expect(link.element.nextElementSibling?.getAttribute('aria-label')).toBe('模型广场')
    } finally {
      wrapper.unmount()
    }
  })

  it.each([
    { signedIn: false, plazaEnabled: true },
    { signedIn: true, plazaEnabled: false },
  ])('does not render without the Model Plaza entry (%j)', (settings) => {
    const wrapper = mountHeader(settings)
    try {
      expect(wrapper.find(`a[href="${shopUrl}"]`).exists()).toBe(false)
    } finally {
      wrapper.unmount()
    }
  })
})
