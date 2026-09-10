<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import keysAPI from '@/api/keys'
import { generateImages, listImageModels, type GeneratedImage } from '@/api/images'
import type { ApiKey } from '@/types'

const keys = ref<ApiKey[]>([])
const keyId = ref<number>()
const selectedKey = computed(() => keys.value.find(key => key.id === keyId.value))
const models = ref<string[]>([])
const model = ref('')
const mode = ref<'generate' | 'edit'>('generate')
const prompt = ref('')
const size = ref('auto')
const quality = ref('auto')
const source = ref<File>()
const mask = ref<File>()
const sourcePreview = ref('')
const maskPreview = ref('')
const loadingKeys = ref(true)
const loadingModels = ref(false)
const modelsReady = ref(false)
const busy = ref(false)
const error = ref('')
const elapsed = ref(0)
const history = ref<{ id: number; prompt: string; model: string; images: GeneratedImage[] }[]>([])
let modelController: AbortController | undefined
let generationController: AbortController | undefined
let timer: ReturnType<typeof setInterval> | undefined
let disposed = false
const canSubmit = computed(() => !!selectedKey.value && modelsReady.value && !!prompt.value.trim() && !busy.value && (mode.value === 'generate' || !!source.value))

function message(cause: unknown) { return cause instanceof Error ? cause.message : '请求失败，请稍后重试。' }
async function loadKeys() {
  loadingKeys.value = true
  error.value = ''
  try {
    const all: ApiKey[] = []
    let page = 1
    while (true) {
      const response = await keysAPI.list(page, 100, { status: 'active' })
      all.push(...response.items)
      if (page >= response.pages) break
      page++
    }
    if (disposed) return
    keys.value = all.filter(key => key.status === 'active' && (!key.expires_at || Date.parse(key.expires_at) > Date.now()) && key.group?.platform === 'openai' && key.group.status === 'active' && key.group.allow_image_generation)
    keyId.value = keys.value[0]?.id
  } catch (cause) { if (!disposed) error.value = message(cause) }
  finally { loadingKeys.value = false }
}
watch(keyId, async () => {
  modelController?.abort()
  const controller = new AbortController()
  modelController = controller
  model.value = ''
  models.value = []
  modelsReady.value = false
  loadingModels.value = false
  if (!selectedKey.value) return
  loadingModels.value = true
  error.value = ''
  try {
    const available = await listImageModels(selectedKey.value.key, controller.signal)
    if (controller.signal.aborted || disposed) return
    models.value = available
    modelsReady.value = true
  } catch (cause) { if (!controller.signal.aborted && !disposed) error.value = message(cause) }
  finally { if (!controller.signal.aborted) loadingModels.value = false }
})
function selectFile(event: Event, kind: 'source' | 'mask') {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!(kind === 'mask' ? file.type === 'image/png' : ['image/png', 'image/jpeg', 'image/webp'].includes(file.type)) || file.size > 20 * 1024 * 1024) {
    error.value = kind === 'mask' ? '请上传不超过 20 MB 的 PNG 蒙版。' : '请上传不超过 20 MB 的 PNG、JPEG 或 WebP 图片。'
    return
  }
  error.value = ''
  clearFile(kind)
  if (kind === 'source') { source.value = file; sourcePreview.value = URL.createObjectURL(file) }
  else { mask.value = file; maskPreview.value = URL.createObjectURL(file) }
}
function clearFile(kind: 'source' | 'mask') {
  const preview = kind === 'source' ? sourcePreview : maskPreview
  if (preview.value) URL.revokeObjectURL(preview.value)
  preview.value = ''
  if (kind === 'source') { source.value = undefined; clearFile('mask') }
  else mask.value = undefined
}
async function submit() {
  if (!canSubmit.value || !selectedKey.value) return
  busy.value = true
  error.value = ''
  elapsed.value = 0
  const request = { prompt: prompt.value, model: model.value, size: size.value, quality: quality.value, image: mode.value === 'edit' ? source.value : undefined, mask: mode.value === 'edit' ? mask.value : undefined }
  generationController = new AbortController()
  timer = setInterval(() => { elapsed.value++ }, 1000)
  try {
    const images = await generateImages(selectedKey.value.key, request, generationController.signal)
    if (disposed) return
    history.value.unshift({ id: Date.now(), prompt: request.prompt, model: request.model || '自动', images })
    history.value = history.value.slice(0, 10)
  } catch (cause) {
    if (!disposed) error.value = `${message(cause)} 若请求超时或连接中断，请先查看用量记录；重新提交可能再次计费。`
  } finally {
    busy.value = false
    if (timer) clearInterval(timer)
  }
}
async function download(image: GeneratedImage, id: number) {
  try {
    const response = await fetch(image.url, { credentials: 'omit', referrerPolicy: 'no-referrer' })
    if (!response.ok) throw new Error(`下载失败（HTTP ${response.status}）`)
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `sub2api-${id}.${blob.type.includes('jpeg') ? 'jpg' : blob.type.includes('webp') ? 'webp' : 'png'}`
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (cause) { error.value = `${message(cause)} 可点击“打开原图”后保存。` }
}
onMounted(loadKeys)
onBeforeUnmount(() => {
  disposed = true
  modelController?.abort()
  generationController?.abort()
  if (timer) clearInterval(timer)
  clearFile('source')
})
</script>

<template>
  <AppLayout>
    <div class="image-workspace">
      <header class="mb-6">
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">图片工作室</h1>
        <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">用文字创造图片，或上传图片进行编辑。使用所选密钥的分组权限与计费规则。</p>
      </header>
      <div v-if="error" role="alert" class="mb-4 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-200">{{ error }}</div>
      <div v-if="loadingKeys" role="status" class="mb-4">正在加载可用密钥…</div>
      <div v-else-if="!keys.length" class="mb-4 rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900">
        没有可用的生图密钥。请在 <RouterLink to="/keys" class="underline">API 密钥</RouterLink> 中创建或启用属于允许生图的 OpenAI 分组的密钥。
        <button type="button" class="ml-2 underline" @click="loadKeys">重新加载</button>
      </div>
      <div class="grid gap-6 lg:grid-cols-[minmax(300px,380px)_1fr]">
        <form class="studio-panel self-start" @submit.prevent="submit">
          <fieldset :disabled="busy" class="space-y-5">
            <legend class="sr-only">图片生成参数</legend>
            <div class="flex rounded-lg bg-gray-100 p-1 dark:bg-gray-800" role="group" aria-label="生成模式">
              <button type="button" :aria-pressed="mode === 'generate'" class="mode-button" :class="{ selected: mode === 'generate' }" @click="mode = 'generate'">文字生图</button>
              <button type="button" data-testid="edit-mode" :aria-pressed="mode === 'edit'" class="mode-button" :class="{ selected: mode === 'edit' }" @click="mode = 'edit'">图片编辑</button>
            </div>
            <label class="studio-label">使用密钥
              <select v-model="keyId" class="studio-input" :disabled="loadingKeys || !keys.length">
                <option v-for="key in keys" :key="key.id" :value="key.id">{{ key.name }} · {{ key.group?.name }}</option>
              </select>
            </label>
            <label class="studio-label">模型
              <select v-model="model" class="studio-input" :disabled="loadingModels || !modelsReady">
                <option value="">自动（由网关选择）</option>
                <option v-for="name in models" :key="name" :value="name">{{ name }}</option>
              </select>
              <span v-if="loadingModels" class="text-xs text-gray-500" role="status">正在读取分组模型…</span>
            </label>
            <div v-if="mode === 'edit'" class="space-y-3">
              <label class="studio-label">原图 <span class="text-xs font-normal text-gray-500">PNG / JPEG / WebP，最多 20 MB</span>
                <input type="file" accept="image/png,image/jpeg,image/webp" class="studio-input" @change="selectFile($event, 'source')" />
              </label>
              <div v-if="sourcePreview" class="space-y-2">
                <img :src="sourcePreview" alt="待编辑原图" class="max-h-48 rounded-lg object-contain" />
                <button type="button" class="text-sm text-red-600" @click="clearFile('source')">移除原图</button>
              </div>
              <label class="studio-label">蒙版（可选）
                <input type="file" accept="image/png" :disabled="!source" class="studio-input" @change="selectFile($event, 'mask')" />
                <span class="text-xs font-normal text-gray-500">上传与原图同尺寸的 PNG，透明区域表示需要编辑的位置。蒙版需要原生图片账号支持。</span>
              </label>
              <div v-if="maskPreview">
                <img :src="maskPreview" alt="编辑蒙版" class="max-h-24 rounded-lg" />
                <button type="button" class="text-sm text-red-600" @click="clearFile('mask')">移除蒙版</button>
              </div>
            </div>
            <label class="studio-label">{{ mode === 'edit' ? '描述你想修改的内容' : '描述你想生成的图片' }}
              <textarea v-model="prompt" required rows="5" class="studio-input resize-y" placeholder="例如：一只坐在窗边的橘猫，温暖的午后阳光，水彩风格" />
            </label>
            <div class="grid grid-cols-2 gap-3">
              <label class="studio-label">尺寸
                <select v-model="size" class="studio-input">
                  <option value="auto">自动</option><option value="1024x1024">正方形 1024 × 1024</option><option value="1536x1024">横向 1536 × 1024</option><option value="1024x1536">竖向 1024 × 1536</option>
                </select>
              </label>
              <label class="studio-label">质量
                <select v-model="quality" class="studio-input"><option value="auto">自动</option><option value="low">低</option><option value="medium">中</option><option value="high">高</option></select>
              </label>
            </div>
            <p class="text-xs leading-relaxed text-gray-500">指定模型、尺寸或质量需要原生图片账号。默认“自动”兼容基础图片桥接。每次提交生成 1 张图片。</p>
            <button data-testid="submit" type="submit" :disabled="!canSubmit" class="w-full rounded-xl bg-primary-600 px-4 py-3 font-medium text-white hover:bg-primary-700 disabled:cursor-not-allowed disabled:opacity-50">{{ busy ? `正在生成 · ${elapsed} 秒` : mode === 'edit' ? '编辑图片' : '生成图片' }}</button>
          </fieldset>
          <p v-if="busy" role="status" aria-live="polite" class="mt-3 text-sm text-gray-500">正在等待图片服务返回，请保持页面打开。离开页面不会保证取消上游生成或计费。</p>
        </form>
        <section class="min-w-0" aria-label="图片结果">
          <div class="mb-4 flex items-center justify-between">
            <h2 class="font-semibold text-gray-900 dark:text-white">本次会话的作品</h2>
            <RouterLink to="/usage" class="text-sm text-primary-600">查看用量记录 →</RouterLink>
          </div>
          <p class="mb-4 text-xs text-gray-500">最多保留最近 10 次结果，离开或刷新页面后清空。请及时下载需要保留的图片。</p>
          <div v-if="!history.length" class="studio-panel flex min-h-[420px] flex-col items-center justify-center border-dashed text-center">
            <span class="mb-4 text-5xl text-gray-300" aria-hidden="true">▧</span>
            <h3 class="font-medium text-gray-800 dark:text-gray-200">{{ busy ? '图片正在创作中' : '从一个想法开始' }}</h3>
            <p class="mt-2 text-sm text-gray-500">{{ busy ? '生成完成后会在这里显示。' : '输入描述，或上传图片开启编辑。' }}</p>
          </div>
          <article v-for="entry in history" :key="entry.id" class="studio-panel mb-5">
            <div class="mb-4 flex items-start justify-between gap-3">
              <div class="min-w-0"><p class="whitespace-pre-wrap break-words text-sm text-gray-800 dark:text-gray-200">{{ entry.prompt }}</p><p class="mt-1 text-xs text-gray-500">{{ entry.model }} · {{ new Date(entry.id).toLocaleTimeString() }}</p></div>
              <button type="button" :disabled="busy" class="shrink-0 text-sm text-primary-600" @click="prompt = entry.prompt">复用描述</button>
            </div>
            <div v-for="(image, index) in entry.images" :key="index">
              <img data-testid="result-image" :src="image.url" :alt="entry.prompt" referrerpolicy="no-referrer" class="mx-auto max-h-[640px] w-full rounded-lg object-contain" />
              <p v-if="image.revisedPrompt" class="mt-2 text-xs text-gray-500">{{ image.revisedPrompt }}</p>
              <div class="mt-4 flex gap-4 text-sm text-primary-600">
                <button type="button" @click="download(image, entry.id)">下载图片</button>
                <a :href="image.url" target="_blank" rel="noopener noreferrer">打开原图</a>
              </div>
            </div>
          </article>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
.studio-panel { @apply rounded-2xl border border-gray-200 bg-white p-5 dark:border-gray-700 dark:bg-gray-900; }
.studio-label { @apply flex flex-col gap-2 text-sm font-medium text-gray-700 dark:text-gray-200; }
.studio-input { @apply w-full rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm font-normal text-gray-900 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/20 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800 dark:text-white; }
.mode-button { @apply flex-1 rounded-md px-3 py-2 text-sm text-gray-500; }
.mode-button.selected { @apply bg-white font-medium text-primary-600 shadow-sm dark:bg-gray-700 dark:text-primary-400; }
</style>