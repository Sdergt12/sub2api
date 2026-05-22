<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.dataImportTitle')"
    width="normal"
    close-on-click-outside
    @close="handleClose"
  >
    <form id="import-data-form" class="space-y-4" @submit.prevent="handleImport">
      <div class="text-sm text-gray-600 dark:text-dark-300">
        {{ t('admin.accounts.dataImportHint') }}
      </div>
      <div
        class="rounded-lg border border-amber-200 bg-amber-50 p-3 text-xs text-amber-600 dark:border-amber-800 dark:bg-amber-900/20 dark:text-amber-400"
      >
        {{ t('admin.accounts.dataImportWarning') }}
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportMode') }}</label>
        <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-colors"
            :class="importMode === 'sub2api'
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700'"
            @click="setImportMode('sub2api')"
          >
            <div class="text-sm font-semibold">{{ t('admin.accounts.dataImportModeSub2api') }}</div>
            <div class="mt-1 text-xs opacity-75">{{ t('admin.accounts.dataImportModeSub2apiHint') }}</div>
          </button>
          <button
            type="button"
            class="rounded-xl border px-4 py-3 text-left transition-colors"
            :class="importMode === 'cpa'
              ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-200'
              : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700'"
            @click="setImportMode('cpa')"
          >
            <div class="text-sm font-semibold">{{ t('admin.accounts.dataImportModeCpa') }}</div>
            <div class="mt-1 text-xs opacity-75">{{ t('admin.accounts.dataImportModeCpaHint') }}</div>
          </button>
        </div>
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.dataImportFile') }}</label>
        <div
          class="flex items-center justify-between gap-3 rounded-lg border border-dashed border-gray-300 bg-gray-50 px-4 py-3 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="min-w-0">
            <div class="truncate text-sm text-gray-700 dark:text-dark-200">
              {{ fileLabel || t('admin.accounts.dataImportSelectFile') }}
            </div>
            <div class="text-xs text-gray-500 dark:text-dark-400">
              {{ importMode === 'cpa' ? t('admin.accounts.dataImportMultiJsonHint') : 'JSON (.json)' }}
            </div>
          </div>
          <button type="button" class="btn btn-secondary shrink-0" @click="openFilePicker">
            {{ t('common.chooseFile') }}
          </button>
        </div>
        <input
          ref="fileInput"
          type="file"
          class="hidden"
          accept="application/json,.json"
          :multiple="importMode === 'cpa'"
          @change="handleFileChange"
        />
      </div>

      <div v-if="importMode === 'cpa'" class="grid grid-cols-1 gap-4 rounded-xl border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-2">
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaPlatform') }}</label>
          <input v-model.trim="cpaOptions.platform" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaAccountType') }}</label>
          <input v-model.trim="cpaOptions.accountType" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaConcurrency') }}</label>
          <input v-model.number="cpaOptions.concurrency" type="number" min="0" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaPriority') }}</label>
          <input v-model.number="cpaOptions.priority" type="number" min="0" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaNameSource') }}</label>
          <select v-model="cpaOptions.nameSource" class="input">
            <option value="email">{{ t('admin.accounts.cpaNameSourceEmail') }}</option>
            <option value="filename">{{ t('admin.accounts.cpaNameSourceFilename') }}</option>
            <option value="index">{{ t('admin.accounts.cpaNameSourceIndex') }}</option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.cpaNamePrefix') }}</label>
          <input v-model.trim="cpaOptions.namePrefix" class="input" />
        </div>
      </div>

      <div
        v-if="cpaPreview"
        class="rounded-xl border border-blue-200 bg-blue-50 p-4 text-sm text-blue-800 dark:border-blue-800 dark:bg-blue-900/20 dark:text-blue-200"
      >
        {{ t('admin.accounts.cpaPreview', cpaPreview) }}
      </div>

      <div
        v-if="result"
        class="space-y-2 rounded-xl border border-gray-200 p-4 dark:border-dark-700"
      >
        <div class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accounts.dataImportResult') }}
        </div>
        <div class="text-sm text-gray-700 dark:text-dark-300">
          {{ t('admin.accounts.dataImportResultSummary', result) }}
        </div>

        <div v-if="errorItems.length" class="mt-2">
          <div class="text-sm font-medium text-red-600 dark:text-red-400">
            {{ t('admin.accounts.dataImportErrors') }}
          </div>
          <div
            class="mt-2 max-h-48 overflow-auto rounded-lg bg-gray-50 p-3 font-mono text-xs dark:bg-dark-800"
          >
            <div v-for="(item, idx) in errorItems" :key="idx" class="whitespace-pre-wrap">
              {{ item.kind }} {{ item.name || item.proxy_key || '-' }} — {{ item.message }}
            </div>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button class="btn btn-secondary" type="button" :disabled="importing" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          class="btn btn-primary"
          type="submit"
          form="import-data-form"
          :disabled="importing"
        >
          {{ importing ? t('admin.accounts.dataImporting') : t('admin.accounts.dataImportButton') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { buildSub2APIDataFromCPA, type CPANameSource } from '@/utils/cpaImport'
import type { AdminDataImportResult, AdminDataPayload } from '@/types'

interface Props {
  show: boolean
}

interface Emits {
  (e: 'close'): void
  (e: 'imported'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const { t } = useI18n()
const appStore = useAppStore()

type ImportMode = 'sub2api' | 'cpa'

const importing = ref(false)
const importMode = ref<ImportMode>('sub2api')
const files = ref<File[]>([])
const result = ref<AdminDataImportResult | null>(null)
const cpaPreview = ref<{ files: number; accounts: number; name_source: string } | null>(null)
const cpaOptions = reactive({
  platform: 'openai',
  accountType: 'oauth',
  concurrency: 3,
  priority: 50,
  nameSource: 'email' as CPANameSource,
  namePrefix: 'acc',
})

const fileInput = ref<HTMLInputElement | null>(null)
const fileLabel = computed(() => {
  if (files.value.length === 0) return ''
  if (files.value.length === 1) return files.value[0].name
  return t('admin.accounts.dataImportSelectedFiles', { count: files.value.length })
})

const errorItems = computed(() => result.value?.errors || [])

watch(
  () => props.show,
  (open) => {
    if (open) {
      files.value = []
      result.value = null
      cpaPreview.value = null
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  }
)

const openFilePicker = () => {
  fileInput.value?.click()
}

const setImportMode = (mode: ImportMode) => {
  importMode.value = mode
  files.value = []
  result.value = null
  cpaPreview.value = null
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

const handleFileChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  files.value = Array.from(target.files || [])
  result.value = null
  cpaPreview.value = null
}

const handleClose = () => {
  if (importing.value) return
  emit('close')
}

const readFileAsText = async (sourceFile: File): Promise<string> => {
  if (typeof sourceFile.text === 'function') {
    return sourceFile.text()
  }

  if (typeof sourceFile.arrayBuffer === 'function') {
    const buffer = await sourceFile.arrayBuffer()
    return new TextDecoder().decode(buffer)
  }

  return await new Promise<string>((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error || new Error('Failed to read file'))
    reader.readAsText(sourceFile)
  })
}

const parseCPAFiles = async (): Promise<AdminDataPayload> => {
  const parsedFiles = []
  for (const item of files.value) {
    const text = await readFileAsText(item)
    parsedFiles.push({
      filename: item.name,
      data: JSON.parse(text),
    })
  }

  const payload = buildSub2APIDataFromCPA(parsedFiles, {
    platform: cpaOptions.platform,
    accountType: cpaOptions.accountType,
    concurrency: Number(cpaOptions.concurrency),
    priority: Number(cpaOptions.priority),
    nameSource: cpaOptions.nameSource,
    namePrefix: cpaOptions.namePrefix,
  })

  // CPA token 文件结构不属于系统导出格式，转换后只走现有导入契约。
  cpaPreview.value = {
    files: files.value.length,
    accounts: payload.accounts.length,
    name_source: cpaOptions.nameSource,
  }
  return payload as AdminDataPayload
}

const handleImport = async () => {
  if (files.value.length === 0) {
    appStore.showError(t('admin.accounts.dataImportSelectFile'))
    return
  }

  importing.value = true
  try {
    let dataPayload: AdminDataPayload
    if (importMode.value === 'cpa') {
      dataPayload = await parseCPAFiles()
    } else {
      const text = await readFileAsText(files.value[0])
      dataPayload = JSON.parse(text)
    }

    const res = await adminAPI.accounts.importData({
      data: dataPayload,
      skip_default_group_bind: true
    })

    result.value = res

    const msgParams: Record<string, unknown> = {
      account_created: res.account_created,
      account_failed: res.account_failed,
      proxy_created: res.proxy_created,
      proxy_reused: res.proxy_reused,
      proxy_failed: res.proxy_failed,
    }
    if (res.account_failed > 0 || res.proxy_failed > 0) {
      appStore.showError(t('admin.accounts.dataImportCompletedWithErrors', msgParams))
    } else {
      appStore.showSuccess(t('admin.accounts.dataImportSuccess', msgParams))
      emit('imported')
    }
  } catch (error: any) {
    if (error instanceof SyntaxError) {
      appStore.showError(t('admin.accounts.dataImportParseFailed'))
    } else {
      appStore.showError(error?.message || t('admin.accounts.dataImportFailed'))
    }
  } finally {
    importing.value = false
  }
}
</script>
