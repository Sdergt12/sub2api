<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <h1 class="text-2xl font-bold text-gray-900 dark:text-white">Token 审查</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              展示登录 token、管理员 token 和 API key 的异常使用审计。页面只展示 hash 与前后缀摘要，不展示完整 token。
            </p>
          </div>
          <button class="btn btn-primary" type="button" :disabled="loading" @click="fetchAudits">
            刷新
          </button>
        </div>

        <div class="mt-5 grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
          <div>
            <label class="input-label">时间范围</label>
            <Select v-model="filters.time_range" :options="timeRangeOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">风险等级</label>
            <Select v-model="filters.risk_level" :options="riskLevelOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">Token 类型</label>
            <Select v-model="filters.token_type" :options="tokenTypeOptions" @change="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">用户 ID</label>
            <input v-model="filters.user_id" class="input" inputmode="numeric" placeholder="例如 1" @keyup.enter="resetAndFetch" />
          </div>
          <div>
            <label class="input-label">搜索</label>
            <input v-model="filters.q" class="input" placeholder="rule / hash / path / IP" @keyup.enter="resetAndFetch" />
          </div>
        </div>
      </div>

      <div class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div v-if="loading" class="flex items-center justify-center py-16">
          <LoadingSpinner size="lg" />
        </div>

        <EmptyState
          v-else-if="audits.length === 0"
          title="暂无 token 审计记录"
          description="出现过期、伪造、越权、禁用、IP 限制或管理员敏感写操作后会在这里显示。"
        />

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-900/60">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">时间</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">风险</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">类型</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">Token 摘要</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">规则 / 结果</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">用户</th>
                <th class="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">来源</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="row in audits" :key="row.id" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatTime(row.created_at) }}</td>
                <td class="px-4 py-3">
                  <span :class="riskBadgeClass(extraString(row, 'risk_level'))" class="inline-flex rounded-full px-2 py-1 text-xs font-semibold">
                    {{ extraString(row, 'risk_level') || '-' }}
                  </span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ extraString(row, 'token_type') || '-' }}</td>
                <td class="px-4 py-3">
                  <div class="space-y-1 font-mono text-xs text-gray-700 dark:text-gray-300">
                    <div>{{ extraString(row, 'token_prefix') || '-' }}...{{ extraString(row, 'token_suffix') || '-' }}</div>
                    <div class="max-w-[280px] truncate text-gray-500 dark:text-gray-400" :title="extraString(row, 'token_hash')">
                      {{ extraString(row, 'token_hash') || '-' }}
                    </div>
                  </div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div class="font-medium text-gray-900 dark:text-white">{{ extraString(row, 'rule') || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">
                    {{ extraString(row, 'result') || '-' }} / {{ extraString(row, 'failure_reason') || '-' }} / HTTP {{ extraString(row, 'status_code') || '-' }}
                  </div>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div>user={{ row.user_id ?? (extraString(row, 'user_id') || '-') }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">key={{ extraString(row, 'api_key_id') || '-' }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div>{{ extraString(row, 'method') || '-' }} {{ extraString(row, 'path') || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ extraString(row, 'client_ip') || '-' }}</div>
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
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import { opsAPI, type OpsSystemLog } from '@/api/admin/ops'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const loading = ref(false)
const audits = ref<OpsSystemLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

const filters = reactive({
  time_range: '24h' as '5m' | '30m' | '1h' | '6h' | '24h' | '7d' | '30d',
  risk_level: '',
  token_type: '',
  user_id: '',
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
  { value: 'medium', label: 'medium' },
  { value: 'high', label: 'high' },
  { value: 'critical', label: 'critical' }
]

const tokenTypeOptions = [
  { value: '', label: '全部' },
  { value: 'jwt', label: 'jwt' },
  { value: 'admin_jwt', label: 'admin_jwt' },
  { value: 'admin_api_key', label: 'admin_api_key' },
  { value: 'api_key', label: 'api_key' }
]

function extraString(row: OpsSystemLog, key: string): string {
  const value = row.extra?.[key]
  if (value == null) return ''
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return ''
}

function formatTime(value: string): string {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value || '-'
  return d.toLocaleString()
}

function riskBadgeClass(level: string): string {
  if (level === 'critical') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  if (level === 'high') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300'
}

function buildQuery() {
  const query: Record<string, any> = {
    page: page.value,
    page_size: pageSize.value,
    time_range: filters.time_range,
    component: 'audit.token',
    q: filters.q.trim() || undefined,
    risk_level: filters.risk_level || undefined,
    token_type: filters.token_type || undefined
  }

  const userID = Number.parseInt(filters.user_id.trim(), 10)
  if (Number.isFinite(userID) && userID > 0) {
    query.user_id = userID
  }
  return query
}

async function fetchAudits() {
  loading.value = true
  try {
    const res = await opsAPI.listSystemLogs(buildQuery())
    audits.value = res.items || []
    total.value = res.total || 0
  } catch (err: any) {
    appStore.showError(err?.response?.data?.detail || err?.message || 'Token 审计加载失败')
  } finally {
    loading.value = false
  }
}

function resetAndFetch() {
  page.value = 1
  fetchAudits()
}

function handlePageChange(nextPage: number) {
  page.value = nextPage
  fetchAudits()
}

function handlePageSizeChange(nextPageSize: number) {
  pageSize.value = nextPageSize
  page.value = 1
  fetchAudits()
}

onMounted(fetchAudits)
</script>
