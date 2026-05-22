<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 class="page-title text-2xl font-bold">Token 风险审查</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              从 token 审计日志中提炼滥用风险，展示风险分类、评分、命中规则和管理员处置动作。页面只展示 hash 与脱敏摘要，不展示完整 token/API key。
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary" type="button" :disabled="loading" @click="backfillEvents">
              回填审计日志
            </button>
            <button class="btn btn-primary" type="button" :disabled="loading" @click="fetchAll">
              刷新
            </button>
          </div>
        </div>

        <div class="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div>
            <label class="input-label">时间范围</label>
            <Select v-model="filters.time_range" :options="timeRangeOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">风险等级</label>
            <Select v-model="filters.risk_level" :options="riskLevelOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">风险分类</label>
            <Select v-model="filters.risk_category" :options="riskCategoryOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">处理状态</label>
            <Select v-model="filters.status" :options="statusOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">Token 类型</label>
            <Select v-model="filters.token_type" :options="tokenTypeOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">搜索</label>
            <input v-model.trim="filters.q" class="input" placeholder="hash / path / IP / 原因" @keyup.enter="resetAndFetch" />
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
        <div v-for="item in summaryCards" :key="item.key" class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">{{ item.value }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.meta }}</p>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div class="card p-4 xl:col-span-2">
          <div class="mb-3 flex items-center justify-between">
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">风险类型分布</h2>
            <span class="text-xs text-gray-500 dark:text-gray-400">近 {{ filters.time_range }}</span>
          </div>
          <div v-if="categoryDistribution.length === 0" class="py-8 text-center text-sm text-gray-500">暂无分类数据</div>
          <div v-else class="space-y-3">
            <div v-for="item in categoryDistribution" :key="item.name">
              <div class="mb-1 flex items-center justify-between text-sm">
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ item.name }}</span>
                <span class="text-gray-500">{{ item.count }}</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-blue-500" :style="{ width: `${item.percent}%` }"></div>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">高风险主体</h2>
          <div class="space-y-3">
            <div v-for="item in topSubjects" :key="item.label" class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ item.label }}</span>
                <span class="text-xs text-gray-500">{{ item.count }} 次</span>
              </div>
              <div class="mt-1 truncate font-mono text-sm text-gray-900 dark:text-white" :title="item.subject">{{ item.subject || '-' }}</div>
              <div class="mt-1 text-xs text-gray-500">累计风险分 {{ item.score }}</div>
            </div>
            <div v-if="topSubjects.length === 0" class="py-8 text-center text-sm text-gray-500">暂无高风险主体</div>
          </div>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div v-if="loading" class="flex items-center justify-center py-16">
          <LoadingSpinner size="lg" />
        </div>

        <EmptyState
          v-else-if="events.length === 0"
          title="暂无 Token 风险事件"
          description="无效鉴权、越权、高频、embedded 绕过、API key 多 IP 使用等风险会在这里展示。"
        />

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/60">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">时间</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">风险</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">分类</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">主体</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">请求</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">频率</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">状态</th>
                <th class="px-4 py-3 text-right text-xs font-semibold uppercase tracking-wide text-gray-500">操作</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in events" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatTime(row.created_at) }}</td>
                <td class="px-4 py-3">
                  <span :class="riskBadgeClass(row.risk_level)" class="inline-flex rounded-full px-2 py-1 text-xs font-semibold">
                    {{ row.risk_level }} · {{ row.risk_score }}
                  </span>
                </td>
                <td class="px-4 py-3">
                  <div class="flex max-w-[260px] flex-wrap gap-1">
                    <span v-for="category in row.risk_categories" :key="category" class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">
                      {{ category }}
                    </span>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div>user={{ row.user_id ?? '-' }} / key={{ row.api_key_id ?? '-' }}</div>
                  <div class="font-mono text-xs text-gray-500">
                    {{ tokenSummary(row) }}
                  </div>
                  <div class="font-mono text-xs text-gray-500">{{ row.client_ip || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div class="max-w-[360px] truncate" :title="`${row.method} ${row.path}`">{{ row.method || '-' }} {{ row.path || '-' }}</div>
                  <div class="text-xs text-gray-500">HTTP {{ row.status_code || '-' }} · {{ row.failure_reason || row.result || '-' }}</div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-xs text-gray-600 dark:text-gray-300">
                  <div>5m {{ row.count_5m ?? 0 }} / 1h {{ row.count_1h ?? 0 }}</div>
                  <div>24h {{ row.count_24h ?? 0 }} · IP {{ row.distinct_ip_24h ?? 0 }}</div>
                </td>
                <td class="px-4 py-3">
                  <span class="rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ row.status }}</span>
                </td>
                <td class="px-4 py-3 text-right">
                  <button class="btn btn-secondary btn-sm" type="button" @click="openDetail(row)">详情/处置</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="total > 0"
          :page="page"
          :total="total"
          :page-size="pageSize"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </div>
    </div>

    <BaseDialog :show="detailOpen" title="风险详情与处置" width="wide" @close="detailOpen = false">
      <div v-if="selectedEvent" class="space-y-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs text-gray-500">风险解释</p>
            <p class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedEvent.explanation || '-' }}</p>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs text-gray-500">脱敏凭证</p>
            <p class="mt-1 font-mono text-sm text-gray-800 dark:text-gray-100">{{ tokenSummary(selectedEvent) }}</p>
            <p class="mt-1 truncate font-mono text-xs text-gray-500" :title="selectedEvent.token_hash">hash={{ selectedEvent.token_hash || '-' }}</p>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">命中规则</p>
          <div class="mt-2 flex flex-wrap gap-2">
            <span v-for="rule in selectedEvent.matched_rules" :key="rule" class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">{{ rule }}</span>
            <span v-if="selectedEvent.matched_rules.length === 0" class="text-sm text-gray-500">暂无</span>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">建议处置动作</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-for="action in selectedEvent.recommended_actions"
              :key="action"
              class="btn btn-secondary btn-sm"
              type="button"
              :disabled="actionLoading"
              @click="applyAction(action)"
            >
              {{ actionLabel(action) }}
            </button>
          </div>
          <textarea v-model.trim="actionNote" class="input mt-3 min-h-[80px]" placeholder="处置备注，不要填写完整 token 或隐私原文"></textarea>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">处置记录</p>
          <div v-if="actions.length === 0" class="mt-2 text-sm text-gray-500">暂无处置记录</div>
          <div v-else class="mt-2 space-y-2">
            <div v-for="item in actions" :key="item.id" class="rounded-lg bg-gray-50 p-2 text-sm dark:bg-dark-700/50">
              <div class="flex justify-between gap-2">
                <span class="font-medium">{{ actionLabel(item.action) }} · {{ item.result }}</span>
                <span class="text-xs text-gray-500">{{ formatTime(item.created_at) }}</span>
              </div>
              <p v-if="item.note" class="mt-1 text-xs text-gray-500">{{ item.note }}</p>
            </div>
          </div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import tokenRiskAPI, { type TokenRiskAction, type TokenRiskEvent, type TokenRiskSummary } from '@/api/admin/tokenRisk'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const loading = ref(false)
const actionLoading = ref(false)
const events = ref<TokenRiskEvent[]>([])
const summary = ref<TokenRiskSummary | null>(null)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const detailOpen = ref(false)
const selectedEvent = ref<TokenRiskEvent | null>(null)
const actions = ref<TokenRiskAction[]>([])
const actionNote = ref('')

const filters = reactive({
  time_range: '24h',
  risk_level: '',
  risk_category: '',
  status: 'open',
  token_type: '',
  q: ''
})

const timeRangeOptions = [
  { value: '1h', label: '1h' },
  { value: '6h', label: '6h' },
  { value: '24h', label: '24h' },
  { value: '7d', label: '7d' },
  { value: '30d', label: '30d' }
]

const riskLevelOptions = [
  { value: '', label: '全部' },
  { value: 'low', label: 'low' },
  { value: 'medium', label: 'medium' },
  { value: 'high', label: 'high' },
  { value: 'critical', label: 'critical' }
]

const riskCategoryOptions = [
  { value: '', label: '全部分类' },
  ...[
    'auth_invalid',
    'auth_expired',
    'auth_forged',
    'permission_violation',
    'admin_api_probe',
    'high_frequency',
    'batch_register',
    'registrar_abuse',
    'config_tamper',
    'balance_or_reward_abuse',
    'game_abuse',
    'embedded_bypass',
    'api_key_sharing',
    'adult_content',
    'grey_industry',
    'abnormal_geo_or_ua',
    'insufficient_balance_abuse',
    'suspicious_path_scan'
  ].map((value) => ({ value, label: value }))
]

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'open', label: '待处理' },
  { value: 'handled', label: '已处理' },
  { value: 'false_positive', label: '误报' },
  { value: 'watching', label: '观察中' }
]

const tokenTypeOptions = [
  { value: '', label: '全部' },
  { value: 'jwt', label: 'jwt' },
  { value: 'admin_jwt', label: 'admin_jwt' },
  { value: 'api_key', label: 'api_key' },
  { value: 'admin_api_key', label: 'admin_api_key' },
  { value: 'embedded', label: 'embedded' }
]

const summaryCards = computed(() => {
  const data = summary.value
  return [
    { key: 'total', label: '风险事件', value: data?.total ?? 0, meta: '当前范围内全部事件' },
    { key: 'open', label: '待处理', value: data?.open ?? 0, meta: '需要管理员确认' },
    { key: 'critical', label: 'Critical', value: data?.critical ?? 0, meta: '建议优先处置' },
    { key: 'high', label: 'High', value: data?.high ?? 0, meta: '高风险事件' },
    { key: 'users', label: '异常用户', value: data?.distinct_users ?? 0, meta: '去重 user_id' },
    { key: 'keys', label: '异常 API Key', value: data?.distinct_api_keys ?? 0, meta: '去重 key id' }
  ]
})

const categoryDistribution = computed(() => {
  const data = summary.value?.by_category || {}
  const max = Math.max(1, ...Object.values(data))
  return Object.entries(data)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .map(([name, count]) => ({ name, count, percent: Math.max(4, Math.round((count / max) * 100)) }))
})

const topSubjects = computed(() => {
  const data = summary.value
  if (!data) return []
  return [
    ...(data.top_users || []).slice(0, 3).map((item) => ({ ...item, label: '用户' })),
    ...(data.top_tokens || []).slice(0, 3).map((item) => ({ ...item, label: 'Token Hash' })),
    ...(data.top_api_keys || []).slice(0, 3).map((item) => ({ ...item, label: 'API Key' }))
  ].slice(0, 6)
})

function formatTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value || '-'
  return d.toLocaleString()
}

function riskBadgeClass(level: string): string {
  if (level === 'critical') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (level === 'high') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (level === 'medium') return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
  return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
}

function tokenSummary(row: TokenRiskEvent): string {
  if (row.api_key_summary) return `api_key=${row.api_key_summary}`
  if (row.token_prefix || row.token_suffix) return `${row.token_prefix || '***'}...${row.token_suffix || '***'}`
  return row.token_hash ? `hash=${row.token_hash.slice(0, 12)}...` : '-'
}

function actionLabel(action: string): string {
  const labels: Record<string, string> = {
    mark_handled: '标记已处理',
    mark_false_positive: '标记误报',
    watch_user: '观察用户',
    watch_token: '观察 Token',
    force_relogin: '强制重新登录',
    send_warning: '发送警告',
    send_reminder: '发送提醒'
  }
  return labels[action] || action
}

function buildQuery() {
  return {
    page: page.value,
    page_size: pageSize.value,
    time_range: filters.time_range,
    risk_level: filters.risk_level || undefined,
    risk_category: filters.risk_category || undefined,
    token_type: filters.token_type || undefined,
    status: filters.status || undefined,
    q: filters.q || undefined
  }
}

async function fetchSummary() {
  summary.value = await tokenRiskAPI.getSummary(filters.time_range)
}

async function fetchEvents() {
  const res = await tokenRiskAPI.listEvents(buildQuery())
  events.value = res.items || []
  total.value = res.total || 0
}

async function fetchAll() {
  loading.value = true
  try {
    await Promise.all([fetchSummary(), fetchEvents()])
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || 'Token 风险加载失败')
  } finally {
    loading.value = false
  }
}

async function backfillEvents() {
  loading.value = true
  try {
    const res = await tokenRiskAPI.backfill(filters.time_range)
    appStore.showSuccess(`已回填 ${res.ingested || 0} 条审计日志`)
    await fetchAll()
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || '回填失败')
  } finally {
    loading.value = false
  }
}

function resetAndFetch() {
  page.value = 1
  fetchAll()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  fetchAll()
}

function handlePageSizeChange(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  fetchAll()
}

async function openDetail(row: TokenRiskEvent) {
  detailOpen.value = true
  selectedEvent.value = row
  actions.value = []
  actionNote.value = ''
  try {
    const detail = await tokenRiskAPI.getEvent(row.id)
    selectedEvent.value = detail.event
    actions.value = detail.actions || []
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || '详情加载失败')
  }
}

async function applyAction(action: string) {
  if (!selectedEvent.value) return
  const confirmRequired = action === 'force_relogin'
  if (confirmRequired && !window.confirm('确认执行高危处置动作？该动作会写入审计记录。')) {
    return
  }
  actionLoading.value = true
  try {
    await tokenRiskAPI.createAction(selectedEvent.value.id, {
      action,
      note: actionNote.value,
      confirm: confirmRequired
    })
    appStore.showSuccess('处置动作已记录')
    await openDetail(selectedEvent.value)
    await fetchAll()
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || '处置失败')
  } finally {
    actionLoading.value = false
  }
}

onMounted(fetchAll)
</script>
