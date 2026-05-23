<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div>
            <h1 class="page-title text-2xl font-bold">格式转换工具</h1>
            <p class="mt-1 max-w-4xl text-sm text-gray-500 dark:text-gray-400">
              参考 GPTSession2CPAandSub2API 的本地转换思路，支持 ChatGPT Web Session、9router/Codex OAuth、AxonHub/Codex-Manager、CPA 与 Sub2API JSON。
              转换只在浏览器内完成；只有点击“导入到分组”时，完整凭据才会发送到当前 sub2api 后端。
            </p>
          </div>
          <div class="rounded-xl border border-blue-200 bg-blue-50 px-4 py-3 text-xs text-blue-800 dark:border-blue-900 dark:bg-blue-950/30 dark:text-blue-200">
            预览默认脱敏，复制、下载和导入前请确认敏感凭据安全。
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(420px,0.9fr)]">
        <div class="card space-y-5 p-5">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div>
              <label class="input-label">输入格式</label>
              <select v-model="options.inputFormat" class="input">
                <option value="gpt-session">GPT Session / Codex OAuth</option>
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

          <div v-if="options.inputFormat === 'cpa'" class="grid grid-cols-1 gap-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-2">
            <div>
              <label class="input-label">账号平台</label>
              <input v-model.trim="options.platform" class="input" placeholder="openai" />
            </div>
            <div>
              <label class="input-label">账号类型</label>
              <input v-model.trim="options.accountType" class="input" placeholder="oauth" />
            </div>
            <div>
              <label class="input-label">默认并发</label>
              <input v-model.number="options.concurrency" type="number" min="0" class="input" />
            </div>
            <div>
              <label class="input-label">默认优先级</label>
              <input v-model.number="options.priority" type="number" min="0" class="input" />
            </div>
            <div>
              <label class="input-label">命名方式</label>
              <select v-model="options.nameSource" class="input">
                <option value="email">邮箱优先</option>
                <option value="filename">文件名</option>
                <option value="index">序号</option>
              </select>
            </div>
            <div>
              <label class="input-label">序号前缀</label>
              <input v-model.trim="options.namePrefix" class="input" placeholder="acc" />
            </div>
          </div>

          <div class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <label class="input-label mb-0">目标分组</label>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  选择后会把 group_ids 写入转换后的每个账号，复制、下载和快捷导入都会带上分组。
                </p>
              </div>
              <button type="button" class="btn btn-secondary btn-sm" :disabled="groupsLoading" @click="loadGroups">
                {{ groupsLoading ? '加载中...' : '刷新分组' }}
              </button>
            </div>
            <div v-if="groupsError" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ groupsError }}</div>
            <div v-else-if="groups.length === 0" class="mt-3 text-sm text-gray-500 dark:text-gray-400">
              {{ groupsLoading ? '正在加载分组...' : '暂无可选分组，导入时将不绑定额外分组。' }}
            </div>
            <div v-else class="mt-3 grid max-h-44 grid-cols-1 gap-2 overflow-auto pr-1 md:grid-cols-2">
              <label
                v-for="group in groups"
                :key="group.id"
                class="flex cursor-pointer items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 transition-colors hover:border-primary-300 hover:bg-primary-50 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-700 dark:hover:bg-primary-900/20"
              >
                <input v-model="selectedGroupIds" type="checkbox" class="rounded border-gray-300" :value="group.id" />
                <span class="min-w-0 flex-1 truncate">{{ group.name }}</span>
                <span class="text-xs text-gray-400">{{ group.platform }}</span>
              </label>
            </div>
          </div>

          <div class="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            输入和输出可能包含 access token、session token 或 refresh token。不要把完整结果粘贴到公开日志、Issue、聊天窗口或前端错误截图。
          </div>

          <div>
            <label class="input-label">粘贴 JSON</label>
            <textarea
              v-model="inputText"
              class="input min-h-[260px] font-mono text-xs"
              placeholder='{"user":{"email":"user@example.com"},"account":{"id":"...","planType":"plus"},"accessToken":"...","sessionToken":"..."}'
            ></textarea>
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
                {{ result ? `${result.fileCount} 个输入，${result.accountCount} 个账号，识别为 ${result.detectedFormat}` : '尚未转换' }}
              </p>
              <p v-if="selectedGroupIds.length" class="mt-1 text-xs text-blue-600 dark:text-blue-300">
                将绑定到 {{ selectedGroupIds.length }} 个目标分组。
              </p>
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
              <input v-model="maskedPreview" type="checkbox" class="rounded border-gray-300" />
              预览脱敏
            </label>
          </div>

          <div v-if="result?.previewAccounts.length" class="grid grid-cols-1 gap-2 md:grid-cols-2">
            <div
              v-for="account in result.previewAccounts.slice(0, 6)"
              :key="`${account.source}-${account.name}-${account.account_id}`"
              class="rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-700/50"
            >
              <div class="font-medium text-gray-900 dark:text-white">{{ account.name }}</div>
              <div class="mt-1 font-mono text-xs text-gray-500">{{ account.email || account.account_id || '-' }}</div>
              <div class="mt-1 text-xs text-gray-500">{{ account.plan_type || 'unknown' }} · {{ account.source }}</div>
            </div>
          </div>

          <div v-if="result?.issues.length" class="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
            <div class="font-medium">部分记录已跳过</div>
            <div v-for="issue in result.issues.slice(0, 4)" :key="`${issue.filename}-${issue.path}`" class="mt-1">
              {{ issue.filename }} {{ issue.path }}：{{ issue.reason }}
            </div>
          </div>

          <div v-if="result?.sensitive" class="rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-200">
            输出包含敏感凭据。复制、下载或导入使用的是完整结果，预览脱敏不会修改实际输出。
          </div>

          <div v-if="importResult" class="rounded-xl border border-emerald-200 bg-emerald-50 p-3 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-200">
            导入完成：账号创建 {{ importResult.account_created }}，账号失败 {{ importResult.account_failed }}，代理创建 {{ importResult.proxy_created }}，代理复用 {{ importResult.proxy_reused }}。
          </div>

          <pre class="max-h-[520px] overflow-auto rounded-xl bg-gray-950 p-4 text-xs text-gray-100">{{ previewText }}</pre>

          <div class="flex flex-wrap gap-2">
            <button type="button" class="btn btn-primary" :disabled="!canDirectImport || importing" @click="importConverted">
              {{ importing ? '导入中...' : '导入到分组' }}
            </button>
            <button type="button" class="btn btn-secondary" :disabled="!result" @click="copyOutput">复制完整结果</button>
            <button type="button" class="btn btn-secondary" :disabled="!result" @click="downloadOutput">下载 JSON</button>
          </div>
          <p v-if="result && options.outputFormat !== 'sub2api'" class="text-xs text-gray-500 dark:text-gray-400">
            只有 Sub2API 输出可以直接导入；CPA 输出请下载后用于外部工具。
          </p>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import {
  attachGroupIdsToPayload,
  convertPayload,
  isSub2APIDataPayload,
  maskSensitive,
  type ConversionInput,
  type ConversionOptions,
  type ConversionResult
} from '@/utils/formatConverter'
import type { AdminDataImportResult, AdminDataPayload, AdminGroup } from '@/types'

const appStore = useAppStore()
const inputText = ref('')
const files = ref<File[]>([])
const result = ref<ConversionResult | null>(null)
const importResult = ref<AdminDataImportResult | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const maskedPreview = ref(true)
const groups = ref<AdminGroup[]>([])
const groupsLoading = ref(false)
const groupsError = ref('')
const selectedGroupIds = ref<number[]>([])
const importing = ref(false)

const options = reactive<ConversionOptions>({
  inputFormat: 'gpt-session',
  outputFormat: 'sub2api',
  platform: 'openai',
  accountType: 'oauth',
  concurrency: 3,
  priority: 50,
  nameSource: 'email',
  namePrefix: 'acc',
  targetGroupIds: []
})

watch(
  selectedGroupIds,
  (value) => {
    options.targetGroupIds = value
    if (result.value && options.outputFormat === 'sub2api') {
      result.value = {
        ...result.value,
        output: attachGroupIdsToPayload(result.value.output as AdminDataPayload, value)
      }
    }
  },
  { deep: true }
)

const previewText = computed(() => {
  if (!result.value) return '等待转换...'
  const output = maskedPreview.value ? maskSensitive(result.value.output) : result.value.output
  return JSON.stringify(output, null, 2)
})

const canDirectImport = computed(() => {
  return Boolean(result.value && options.outputFormat === 'sub2api' && isSub2APIDataPayload(result.value.output))
})

onMounted(() => {
  void loadGroups()
})

async function loadGroups() {
  groupsLoading.value = true
  groupsError.value = ''
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (error: any) {
    groupsError.value = error?.message || '分组加载失败'
  } finally {
    groupsLoading.value = false
  }
}

async function readFile(file: File): Promise<ConversionInput> {
  const text = typeof file.text === 'function' ? await file.text() : await new Response(file).text()
  if (text.length > 5 * 1024 * 1024) throw new Error(`${file.name} 超过 5MB 限制`)
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
    importResult.value = null
    result.value = convertPayload(inputs, { ...options, targetGroupIds: selectedGroupIds.value })
    appStore.showSuccess('转换完成')
  } catch (err: any) {
    appStore.showError(err?.message || '转换失败')
  }
}

function clearAll() {
  inputText.value = ''
  files.value = []
  result.value = null
  importResult.value = null
  if (fileInput.value) fileInput.value.value = ''
}

async function importConverted() {
  if (!result.value || !isSub2APIDataPayload(result.value.output)) return
  if (result.value.sensitive && !window.confirm('导入会把完整敏感凭据发送到当前 sub2api 后端，确认继续？')) return
  importing.value = true
  try {
    const payload = attachGroupIdsToPayload(result.value.output, selectedGroupIds.value)
    importResult.value = await adminAPI.accounts.importData({
      data: payload,
      skip_default_group_bind: true
    })
    appStore.showSuccess(`导入完成：账号 ${importResult.value.account_created}，失败 ${importResult.value.account_failed}`)
  } catch (error: any) {
    appStore.showError(error?.message || '导入失败')
  } finally {
    importing.value = false
  }
}

async function copyOutput() {
  if (!result.value) return
  if (result.value.sensitive && !window.confirm('完整结果包含敏感凭据，确认复制到剪贴板？')) return
  await navigator.clipboard.writeText(JSON.stringify(result.value.output, null, 2))
  appStore.showSuccess('已复制完整结果，请注意敏感信息安全')
}

function downloadOutput() {
  if (!result.value) return
  if (result.value.sensitive && !window.confirm('完整结果包含敏感凭据，确认下载到本机？')) return
  const blob = new Blob([JSON.stringify(result.value.output, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `sub2api-format-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}
</script>
