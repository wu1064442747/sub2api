import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ImagesView from '../ImagesView.vue'
const { list, listImageModels, generateImages } = vi.hoisted(() => ({ list: vi.fn(), listImageModels: vi.fn(), generateImages: vi.fn() }))
vi.mock('@/api/keys', () => ({ default: { list } }))
vi.mock('@/api/images', () => ({ listImageModels, generateImages }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
const key = { id: 1, name: 'Studio', key: 'secret', status: 'active', group: { name: 'Images', platform: 'openai', status: 'active', allow_image_generation: true } }
function render() { return mount(ImagesView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, RouterLink: { template: '<a><slot /></a>' } } } }) }
beforeEach(() => {
  vi.clearAllMocks()
  list.mockResolvedValue({ items: [key], pages: 1 })
  listImageModels.mockResolvedValue(['gpt-image-1'])
  generateImages.mockResolvedValue([{ url: 'data:image/png;base64,aGVsbG8=' }])
})
describe('image workspace', () => {
  it('blocks users without a usable image group key', async () => {
    list.mockResolvedValue({ items: [{ ...key, group: { ...key.group, allow_image_generation: false } }], pages: 1 })
    const wrapper = render(); await flushPromises()
    expect(wrapper.text()).toContain('没有可用的生图密钥')
    expect(wrapper.get('[data-testid="submit"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('uses the selected key, prevents duplicate submission, and displays results', async () => {
    let finish!: (images: unknown) => void
    generateImages.mockReturnValue(new Promise(resolve => { finish = resolve }))
    const wrapper = render(); await flushPromises()
    await wrapper.get('textarea').setValue('A cat')
    await wrapper.get('form').trigger('submit')
    expect(generateImages).toHaveBeenCalledTimes(1)
    expect(generateImages.mock.calls[0][0]).toBe('secret')
    expect(wrapper.get('[data-testid="submit"]').attributes('disabled')).toBeDefined()
    finish([{ url: 'data:image/png;base64,aGVsbG8=' }]); await flushPromises()
    expect(wrapper.find('[data-testid="result-image"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('secret')
    wrapper.unmount()
  })
  it('requires a source image in edit mode', async () => {
    const wrapper = render(); await flushPromises()
    await wrapper.get('[data-testid="edit-mode"]').trigger('click')
    await wrapper.get('textarea').setValue('Change the background')
    expect(wrapper.get('[data-testid="submit"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('shows a gateway failure without automatically reissuing the request', async () => {
    generateImages.mockRejectedValue(new Error('quota exceeded'))
    const wrapper = render(); await flushPromises()
    await wrapper.get('textarea').setValue('A cat')
    await wrapper.get('form').trigger('submit'); await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('quota exceeded')
    expect(generateImages).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})