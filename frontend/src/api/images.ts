/** Gateway image requests use the user's API key, never the console JWT. */
export interface ImageRequest {
  model: string
  prompt: string
  size: string
  quality: string
  image?: File
  mask?: File
}

export interface GeneratedImage { url: string; revisedPrompt?: string }

async function gatewayRequest(path: string, key: string, init: RequestInit = {}) {
  const response = await fetch(`/v1/${path}`, {
    ...init,
    credentials: 'omit',
    redirect: 'error',
    headers: { ...init.headers, Authorization: `Bearer ${key}` },
  })
  const text = await response.text()
  let payload: any
  try { payload = JSON.parse(text) } catch { throw new Error(`图片服务返回了非 JSON 响应（HTTP ${response.status}），请检查网关或代理超时。`) }
  if (!response.ok) throw new Error(payload?.error?.message || `图片请求失败（HTTP ${response.status}）`)
  return payload
}

export function normalizeImages(payload: unknown): GeneratedImage[] {
  const data = (payload as { data?: unknown[] })?.data
  if (!Array.isArray(data) || !data.length) throw new Error('服务未返回图片。')
  return data.map(entry => {
    const item = entry as { url?: string; b64_json?: string; revised_prompt?: string }
    let url: string
    if (typeof item.b64_json === 'string' && /^[A-Za-z0-9+/=\r\n]+$/.test(item.b64_json)) {
      url = `data:image/png;base64,${item.b64_json}`
    } else if (typeof item.url === 'string' && /^https?:\/\//i.test(item.url)) {
      url = new URL(item.url).href
    } else throw new Error('服务返回了无效的图片地址。')
    return { url, revisedPrompt: item.revised_prompt }
  })
}

export async function listImageModels(key: string, signal?: AbortSignal): Promise<string[]> {
  const payload = await gatewayRequest('models', key, { signal })
  if (!Array.isArray(payload?.data)) throw new Error('无法读取分组模型列表。')
  return [...new Set<string>(payload.data.map((item: { id: string }) => item.id).filter((id: unknown) => typeof id === 'string'))]
}

export async function generateImages(key: string, request: ImageRequest, signal?: AbortSignal): Promise<GeneratedImage[]> {
  if (!key || !request.model || !request.prompt.trim()) throw new Error('请选择密钥、模型并填写提示词。')
  if (request.mask && !request.image) throw new Error('使用蒙版时必须上传原图。')
  const fields = { model: request.model, prompt: request.prompt.trim(), size: request.size, quality: request.quality, n: '1', output_format: 'png' }
  let body: BodyInit
  let headers: Record<string, string> = {}
  if (request.image) {
    const form = new FormData()
    Object.entries(fields).forEach(([field, value]) => form.append(field, value))
    form.append('image', request.image)
    if (request.mask) form.append('mask', request.mask)
    body = form
  } else {
    body = JSON.stringify({ ...fields, n: 1 })
    headers = { 'Content-Type': 'application/json' }
  }
  return normalizeImages(await gatewayRequest(`images/${request.image ? 'edits' : 'generations'}`, key, { method: 'POST', headers, body, signal }))
}
