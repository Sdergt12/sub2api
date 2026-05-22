<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <h1 class="page-title text-2xl font-bold">格式转换工具</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          参考 GPTSession2CPAandSub2API 的本地转换思路，支持 GPT Session、CPA、Sub2API JSON 转换。转换只在浏览器内完成，不上传第三方；预览默认脱敏。
        </p>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <div class="card space-y-4 p-5">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div>
              <label class="input-label">输入格式</label>
              <select v-model="options.inputFormat" class="input">
                <option value="gpt-session">GPT Session JSON</option>
                <option value="cpa">CPA JSON</option>
                <option value="sub2api">Sub2API JSON</option>
              </select>
            </div>
            <div>
              <label class="input-label">输出格式</label>
              <select v-model="options.outputFormat" class="input">
                <option value="sub2api">Sub2API</option>
                <option value="cpa">CPA</option>
              </select>
            </div>
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <input v-model.trim="options.platform" class="input" placeholder="platform=openai" />
            <input v-model.trim="options.accountType" class="input" placeholder="type=oauth" />
            <input v-model.number="options.concurrency" type="number" min="0" class="input" placeholder="concurrency" />
            <input v-model.number="options.priority" type="number" min="0" class="input" placeholder="priority" />
          </div>

          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <select v-model="options.nameSource" class="input">
              <option value="email">邮箱优先</option>
              <option value="filename">文件名</option>
              <option value="index">序号</option>
            </select>
            <input v-model.trim="options.namePrefix" class="input" placeholder="序号前缀 acc" />
          </div>

          <div>
            <label class="input-label">粘贴 JSON</label>
            <textarea v-model="inputText" class="input min-h-[220px] font-mono text-xs" placeholder='{"email":"user@example.com","access_token":"..."}'></textarea>
          </div>

          <div class="flex flex-wrap gap-2">
            <input ref="fileInput" type="file" class="hidden" accept="application/json,.json,.txt" multiple @change="handleFiles" />
            <button type="button" class="btn btn-secondary" @click="fileInput?.click()">上传文件</button>
            <button type="button" class="btn btn-primary" @click="runConvert">转换</button>
            <button type="button" class="btn btn-secondary" @click="clearAll">清空</button>
          </div>

          <div v-if="files.length" class="rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-700/50">
            已选择 {{ files.length }} 个文件：{{ files.map((item) => item.name).join(', ') }}
          </div>
        </div>

        <div class="card space-y-4 p-5">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">转换结果</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">
                {{ result ? `${result.fileCount} 个输入，${result.accountCount} 个账号` : '尚未转换' }}
              </p>
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="maskedPreview" type="checkbox" class="rounded border-gray-300" />
              预览脱敏
            </label>
          </div>

          <div v-if="result?.sensitive" class="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-200">
            输出包含 session/token/API key 等敏感字段。复制或下载前请确认存储位置安全，不要粘贴到公开日志、Issue 或聊天窗口。
          </div>

          <pre class="max-h-[520px] overflow-auto rounded-xl bg-gray-950 p-4 text-xs text-gray-100">{{ previewText }}</pre>

          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-secondary" :disabled="!result" @click="copyOutput">复制完整结果</button>
            <button type="button" class="btn btn-secondary" :disabled="!result" @click="downloadOutput">下载 JSON</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores/app'
import { convertPayload, maskSensitive, type ConversionInput, type ConversionResult } from '@/utils/formatConverter'

const appStore = useAppStore()
const inputText = ref('')
const files = ref<File[]>([])
const result = ref<ConversionResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const maskedPreview = ref(true)

const options = reactive({
  inputFormat: 'gpt-session' as const,
  outputFormat: 'sub2api' as const,
  platform: 'openai',
  accountType: 'oauth',
  concurrency: 3,
  priority: 50,
  nameSource: 'email' as const,
  namePrefix: 'acc'
})

const previewText = computed(() => {
  if (!result.value) return '等待转换...'
  const output = maskedPreview.value ? maskSensitive(result.value.output) : result.value.output
  return JSON.stringify(output, null, 2)
})

async function readFile(file: File): Promise<ConversionInput> {
  const text = typeof file.text === 'function' ? await file.text() : await new Response(file).text()
  if (text.length > 5 * 1024 * 1024) {
    throw new Error(`${file.name} 超过 5MB 限制`)
  }
  return { filename: file.name, text }
}

function handleFiles(event: Event) {
  const target = event.target as HTMLInputElement
  files.value = Array.from(target.files || [])
}

async function buildInputs(): Promise<ConversionInput[]> {
  const inputs: ConversionInput[] = []
  if (inputText.value.trim()) inputs.push({ filename: 'pasted.json', text: inputText.value })
  for (const file of files.value) inputs.push(await readFile(file))
  return inputs
}

async function runConvert() {
  try {
    const inputs = await buildInputs()
    result.value = convertPayload(inputs, options)
    appStore.showSuccess('转换完成')
  } catch (err: any) {
    appStore.showError(err?.message || '转换失败')
  }
}

function clearAll() {
  inputText.value = ''
  files.value = []
  result.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function copyOutput() {
  if (!result.value) return
  await navigator.clipboard.writeText(JSON.stringify(result.value.output, null, 2))
  appStore.showSuccess('已复制完整结果，请注意敏感信息安全')
}

function downloadOutput() {
  if (!result.value) return
  const blob = new Blob([JSON.stringify(result.value.output, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sub2api-format-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}
</script>
