import { afterEach, describe, expect, it, vi } from 'vitest'
import { generateImages, listImageModels, normalizeImages } from '../images'

afterEach(() => vi.unstubAllGlobals())
describe('image gateway client', () => {
  it('uses API key rather than the console token and does not retry a paid POST', async () => {
    localStorage.setItem('auth_token', 'console-token')
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { message: 'quota exceeded' } }), { status: 429 }))
    vi.stubGlobal('fetch', fetcher)
    await expect(generateImages('user-key', { model: 'gpt-image-1', prompt: 'cat', size: 'auto', quality: 'auto' })).rejects.toThrow('quota exceeded')
    expect(fetcher).toHaveBeenCalledTimes(1)
    expect(fetcher.mock.calls[0][0]).toBe('/v1/images/generations')
    expect(fetcher.mock.calls[0][1].headers.Authorization).toBe('Bearer user-key')
  })
  it('submits editing files as multipart without a manually set content type', async () => {
    const fetcher = vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [{ url: 'https://example.com/image.png' }] })))
    vi.stubGlobal('fetch', fetcher)
    const image = new File(['image'], 'input.png', { type: 'image/png' })
    const mask = new File(['mask'], 'mask.png', { type: 'image/png' })
    await generateImages('key', { model: 'gpt-image-1', prompt: 'edit', size: 'auto', quality: 'auto', image, mask })
    const [url, request] = fetcher.mock.calls[0]
    expect(url).toBe('/v1/images/edits')
    expect(request.headers['Content-Type']).toBeUndefined()
    expect(request.body.get('image')).toBe(image)
    expect(request.body.get('mask')).toBe(mask)
  })
  it('normalizes base64 and HTTPS images and rejects active URL schemes', () => {
    expect(normalizeImages({ data: [{ b64_json: 'aGVsbG8=' }] })[0].url).toBe('data:image/png;base64,aGVsbG8=')
    expect(normalizeImages({ data: [{ url: 'https://example.com/a' }] })[0].url).toBe('https://example.com/a')
    expect(() => normalizeImages({ data: [{ url: 'javascript:alert(1)' }] })).toThrow()
    expect(() => normalizeImages({ data: [] })).toThrow()
  })
  it('keeps custom model aliases from the group model list', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: [{ id: 'studio-image' }] }))))
    expect(await listImageModels('key')).toEqual(['studio-image'])
  })
})