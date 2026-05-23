<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.24em] text-blue-600 dark:text-blue-300">Token Risk Console</p>
            <h1 class="page-title mt-2 text-2xl font-bold">Token 审查告警看板</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              先看告警、主体和建议动作，再下钻事件详情。页面只展示 token hash、摘要和脱敏主体，不展示完整 token、JWT 或 API key。
            </p>
          </div>
          <div class="flex flex-wrap gap-2">
            <button class="btn btn-secondary" type="button" :disabled="loading" @click="backfillEvents">回填审计日志</button>
            <button class="btn btn-primary" type="button" :disabled="loading" @click="fetchAll">刷新</button>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
        <button
          v-for="item in summaryCards"
          :key="item.key"
          type="button"
          class="card p-4 text-left transition hover:-translate-y-0.5"
          @click="item.apply?.()"
        >
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-2xl font-bold" :class="item.tone">{{ item.value }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.meta }}</p>
        </button>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div class="card p-4 xl:col-span-2">
          <div class="mb-4 flex items-center justify-between">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">需要优先处理</h2>
              <p class="text-xs text-gray-500 dark:text-gray-400">默认聚焦 open 状态中的 high / critical 事件。</p>
            </div>
            <button class="btn btn-secondary btn-sm" type="button" @click="showAdvancedFilters = !showAdvancedFilters">
              {{ showAdvancedFilters ? '收起筛选' : '高级筛选' }}
            </button>
          </div>

          <div v-if="priorityEvents.length === 0" class="rounded-xl bg-gray-50 py-10 text-center text-sm text-gray-500 dark:bg-dark-700/50">
            当前没有待处理的高危 Token 事件。
          </div>
          <div v-else class="space-y-3">
            <div v-for="event in priorityEvents" :key="event.id" class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800/70">
              <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span :class="riskBadgeClass(event.risk_level)" class="rounded-full px-2 py-1 text-xs font-semibold">{{ event.risk_level }} · {{ event.risk_score }}</span>
                    <span class="rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ event.status }}</span>
                    <span class="text-xs text-gray-500">{{ formatTime(event.last_seen_at || event.created_at) }}</span>
                  </div>
                  <p class="mt-2 truncate text-sm font-medium text-gray-900 dark:text-white" :title="`${event.method} ${event.path}`">
                    {{ event.method || '-' }} {{ event.path || '-' }}
                  </p>
                  <p class="mt-1 text-xs text-gray-500">
                    {{ simpleExplanation(event) }}
                  </p>
                  <div class="mt-2 flex flex-wrap gap-1">
                    <span v-for="category in event.risk_categories.slice(0, 4)" :key="category" class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">
                      {{ category }}
                    </span>
                  </div>
                </div>
                <button class="btn btn-primary btn-sm shrink-0" type="button" @click="openDetail(event)">查看并处理</button>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">高风险主体</h2>
          <p class="mb-3 mt-1 text-xs text-gray-500 dark:text-gray-400">按用户、token hash 和 API key 聚合。</p>
          <div class="space-y-3">
            <button
              v-for="item in topSubjects"
              :key="`${item.label}-${item.subject}`"
              type="button"
              class="w-full rounded-xl bg-gray-50 p-3 text-left dark:bg-dark-700/50"
              @click="filters.q = item.subject; resetAndFetch()"
            >
              <div class="flex items-center justify-between gap-2">
                <span class="text-xs font-semibold text-gray-500 dark:text-gray-400">{{ item.label }}</span>
                <span class="text-xs text-gray-500">{{ item.count }} 次</span>
              </div>
              <div class="mt-1 truncate font-mono text-sm text-gray-900 dark:text-white" :title="item.subject">{{ item.subject || '-' }}</div>
              <div class="mt-1 text-xs text-gray-500">累计风险分 {{ item.score }}</div>
            </button>
            <div v-if="topSubjects.length === 0" class="py-8 text-center text-sm text-gray-500">暂无高风险主体</div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div class="card p-4 xl:col-span-2">
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">风险类型分布</h2>
          <div v-if="categoryDistribution.length === 0" class="py-8 text-center text-sm text-gray-500">暂无分类数据</div>
          <div v-else class="space-y-3">
            <button
              v-for="item in categoryDistribution"
              :key="item.name"
              type="button"
              class="w-full text-left"
              @click="filters.risk_category = item.name; resetAndFetch()"
            >
              <div class="mb-1 flex items-center justify-between text-sm">
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ item.name }}</span>
                <span class="text-gray-500">{{ item.count }}</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-blue-500" :style="{ width: `${item.percent}%` }"></div>
              </div>
            </button>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="mb-3 text-base font-semibold text-gray-900 dark:text-white">风险等级概览</h2>
          <div class="space-y-3">
            <div v-for="level in levelBars" :key="level.name">
              <div class="mb-1 flex items-center justify-between text-sm">
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ level.name }}</span>
                <span class="text-gray-500">{{ level.count }}</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full" :class="level.color" :style="{ width: `${level.percent}%` }"></div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="showAdvancedFilters" class="card p-5">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
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

      <div class="card overflow-hidden">
        <div class="flex flex-col gap-2 border-b border-gray-100 p-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">事件明细</h2>
            <p class="text-xs text-gray-500">用于下钻排查，默认只看 open 事件，可在高级筛选中调整。</p>
          </div>
          <span class="text-xs text-gray-500">共 {{ total }} 条</span>
        </div>

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
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">主体</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">请求</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500">频率</th>
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
                  <div class="mt-1 flex max-w-[220px] flex-wrap gap-1">
                    <span v-for="category in row.risk_categories.slice(0, 3)" :key="category" class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">{{ category }}</span>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div>user={{ row.user_id ?? '-' }} / key={{ row.api_key_id ?? '-' }}</div>
                  <div class="font-mono text-xs text-gray-500">{{ tokenSummary(row) }}</div>
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
        <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">发生了什么</p>
            <p class="mt-1 text-sm text-gray-800 dark:text-gray-100">
              {{ selectedEvent.method || '-' }} {{ selectedEvent.path || '-' }} 返回 HTTP {{ selectedEvent.status_code || '-' }}，主体 {{ tokenSummary(selectedEvent) }}。
            </p>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">为什么判定</p>
            <p class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ selectedEvent.explanation || simpleExplanation(selectedEvent) }}</p>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">建议怎么处理</p>
            <p class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ recommendedText(selectedEvent) }}</p>
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
          <p class="text-sm font-semibold text-gray-900 dark:text-white">处置动作</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <button
              v-for="action in actionButtons"
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
const showAdvancedFilters = ref(false)

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

const actionButtons = ['mark_handled', 'mark_false_positive', 'watch_user', 'watch_token', 'force_relogin', 'send_warning', 'send_reminder']

const summaryCards = computed(() => {
  const data = summary.value
  return [
    { key: 'open', label: '待处理', value: data?.open ?? 0, meta: '需要管理员确认', tone: 'text-amber-600', apply: () => setStatusFilter('open') },
    { key: 'critical', label: 'Critical', value: data?.critical ?? 0, meta: '立即排查', tone: 'text-red-600', apply: () => setRiskLevelFilter('critical') },
    { key: 'high', label: 'High', value: data?.high ?? 0, meta: '优先处理', tone: 'text-orange-600', apply: () => setRiskLevelFilter('high') },
    { key: 'users', label: '异常用户', value: data?.distinct_users ?? 0, meta: '去重 user_id', tone: 'text-gray-900 dark:text-white' },
    { key: 'keys', label: '异常 API Key', value: data?.distinct_api_keys ?? 0, meta: '去重 key id', tone: 'text-gray-900 dark:text-white' },
    { key: 'total', label: '风险事件', value: data?.total ?? 0, meta: '当前时间范围', tone: 'text-gray-900 dark:text-white' }
  ]
})

const priorityEvents = computed(() => {
  const highRisk = (summary.value?.recent_high_risk || []).filter((item) => item.status === 'open')
  if (highRisk.length > 0) return highRisk.slice(0, 6)
  return events.value.filter((item) => item.status === 'open' && ['high', 'critical'].includes(item.risk_level)).slice(0, 6)
})

const categoryDistribution = computed(() => {
  const data = summary.value?.by_category || {}
  const max = Math.max(1, ...Object.values(data))
  return Object.entries(data)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 10)
    .map(([name, count]) => ({ name, count, percent: Math.max(4, Math.round((count / max) * 100)) }))
})

const levelBars = computed(() => {
  const levels = summary.value?.by_level || {}
  const max = Math.max(1, ...Object.values(levels))
  return [
    { name: 'critical', count: levels.critical || 0, color: 'bg-red-500' },
    { name: 'high', count: levels.high || 0, color: 'bg-orange-500' },
    { name: 'medium', count: levels.medium || 0, color: 'bg-blue-500' },
    { name: 'low', count: levels.low || 0, color: 'bg-gray-400' }
  ].map((item) => ({ ...item, percent: Math.max(4, Math.round((item.count / max) * 100)) }))
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

function simpleExplanation(row: TokenRiskEvent): string {
  return row.explanation || `${row.risk_categories.join(', ') || 'unknown'} 命中 ${row.matched_rules.length || 0} 条规则`
}

function recommendedText(row: TokenRiskEvent): string {
  if (row.recommended_actions.length === 0) return '先查看同用户、同 IP、同 token hash 的历史行为，再决定是否标记或观察。'
  return row.recommended_actions.map(actionLabel).join('、')
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

function setStatusFilter(value: string) {
  filters.status = value
  resetAndFetch()
}

function setRiskLevelFilter(value: string) {
  filters.risk_level = value
  resetAndFetch()
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
  if (confirmRequired && !window.confirm('确认执行高危处置动作？该动作会写入审计记录。')) return
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
