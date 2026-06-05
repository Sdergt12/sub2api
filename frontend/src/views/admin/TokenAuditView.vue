<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="card p-5">
        <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <p class="text-xs font-semibold uppercase tracking-[0.24em] text-blue-600 dark:text-blue-300">
              Token Risk Console
            </p>
            <h1 class="page-title mt-2 text-2xl font-bold">Token 审查告警看板</h1>
            <p class="mt-1 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              聚合同一用户、Token、API key、IP 和接口行为，优先展示多 IP、RPM 高频、越权探测和 embedded 绕过。
              页面只展示脱敏摘要、hash 和内容审核摘要，不展示完整 token、JWT、API key 或原始请求正文。
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
      </section>

      <section class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
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
      </section>

      <section class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div class="card p-4 xl:col-span-2">
          <div class="mb-4 flex items-center justify-between">
            <div>
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">需要优先处理</h2>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                默认聚焦 open 状态的 high / critical 事件，以及多 IP、高 RPM、越权和 embedded 异常。
              </p>
            </div>
            <button class="btn btn-secondary btn-sm" type="button" @click="showAdvancedFilters = !showAdvancedFilters">
              {{ showAdvancedFilters ? '收起筛选' : '高级筛选' }}
            </button>
          </div>
          <div v-if="priorityEvents.length === 0" class="rounded-xl bg-gray-50 py-10 text-center text-sm text-gray-500 dark:bg-dark-700/50">
            当前没有待处理的高风险 Token 事件。
          </div>
          <div v-else class="space-y-3">
            <article
              v-for="event in priorityEvents"
              :key="event.id"
              class="rounded-xl border border-gray-200 bg-white p-3 dark:border-dark-700 dark:bg-dark-800/70"
            >
              <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <span :class="riskBadgeClass(event.risk_level)" class="rounded-full px-2 py-1 text-xs font-semibold">
                      {{ riskLevelText(event.risk_level) }} · {{ event.risk_score }}
                    </span>
                    <span class="rounded-full bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                      {{ statusText(event.status) }}
                    </span>
                    <span v-if="isMultiIp(event)" class="rounded-full bg-red-50 px-2 py-1 text-xs text-red-700 dark:bg-red-900/30 dark:text-red-200">
                      多 IP {{ event.distinct_ip_24h }}
                    </span>
                    <span v-if="rpm(event) >= 20" class="rounded-full bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">
                      RPM {{ rpm(event) }}
                    </span>
                    <span class="text-xs text-gray-500">{{ formatTime(event.last_seen_at || event.created_at) }}</span>
                  </div>
                  <p class="mt-2 truncate text-sm font-medium text-gray-900 dark:text-white" :title="`${event.method} ${event.path}`">
                    {{ event.method || '-' }} {{ event.path || '-' }}
                  </p>
                  <p class="mt-1 text-xs text-gray-500">{{ eventConclusion(event) }}</p>
                  <div class="mt-2 flex flex-wrap gap-1">
                    <span
                      v-for="category in event.risk_categories.slice(0, 4)"
                      :key="category"
                      class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-200"
                    >
                      {{ categoryLabel(category) }}
                    </span>
                  </div>
                </div>
                <button class="btn btn-primary btn-sm shrink-0" type="button" @click="openDetail(event)">
                  查看并处理
                </button>
              </div>
            </article>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">异常主体聚合</h2>
          <p class="mb-3 mt-1 text-xs text-gray-500 dark:text-gray-400">按用户、Token hash 和 API key 汇总风险分。</p>
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
              <div class="mt-1 truncate font-mono text-sm text-gray-900 dark:text-white" :title="item.subject">
                {{ item.subject || '-' }}
              </div>
              <div class="mt-1 text-xs text-gray-500">累计风险分 {{ item.score }}</div>
            </button>
            <div v-if="topSubjects.length === 0" class="py-8 text-center text-sm text-gray-500">暂无高风险主体</div>
          </div>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-4 xl:grid-cols-3">
        <div class="card p-4 xl:col-span-2">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">API key 多 IP / RPM 聚合</h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            这里不是按单条日志看，而是把同一 API key 的 24 小时 IP 数、5 分钟请求数和近似 RPM 放到第一屏。
          </p>
          <div class="mt-3 space-y-2">
            <button
              v-for="row in suspiciousApiKeyRows"
              :key="row.id"
              type="button"
              class="grid w-full grid-cols-1 gap-2 rounded-xl border border-gray-100 bg-gray-50 p-3 text-left dark:border-dark-700 dark:bg-dark-700/50 md:grid-cols-[minmax(0,1fr)_auto]"
              @click="openDetail(row)"
            >
              <div class="min-w-0">
                <div class="truncate font-mono text-sm font-semibold text-gray-900 dark:text-white">
                  {{ tokenSummary(row) }}
                </div>
                <p class="mt-1 truncate text-xs text-gray-500">{{ row.method || '-' }} {{ row.path || '-' }}</p>
              </div>
              <div class="flex flex-wrap items-center gap-2 text-xs">
                <span class="rounded bg-red-50 px-2 py-1 text-red-700 dark:bg-red-900/30 dark:text-red-200">
                  IP {{ row.distinct_ip_24h ?? 0 }}
                </span>
                <span class="rounded bg-amber-50 px-2 py-1 text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">
                  5m {{ row.count_5m ?? 0 }} / RPM {{ rpm(row) }}
                </span>
                <span class="rounded bg-gray-100 px-2 py-1 text-gray-700 dark:bg-dark-800 dark:text-gray-200">
                  1h {{ row.count_1h ?? 0 }}
                </span>
              </div>
            </button>
            <div v-if="suspiciousApiKeyRows.length === 0" class="rounded-xl bg-gray-50 py-8 text-center text-sm text-gray-500 dark:bg-dark-700/50">
              当前列表中没有明显的 API key 多 IP / 高频聚合异常。
            </div>
          </div>
        </div>

        <div class="card p-4">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">风险类型分布</h2>
          <div class="mt-4 space-y-3">
            <div v-for="item in categoryDistribution" :key="item.key">
              <div class="mb-1 flex justify-between text-sm">
                <span class="font-medium text-gray-700 dark:text-gray-200">{{ categoryLabel(item.key) }}</span>
                <span class="text-gray-500">{{ item.count }}</span>
              </div>
              <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                <div class="h-full rounded-full bg-blue-500" :style="{ width: `${item.percent}%` }"></div>
              </div>
            </div>
            <div v-if="categoryDistribution.length === 0" class="py-8 text-center text-sm text-gray-500">暂无分类数据</div>
          </div>
        </div>
      </section>

      <section v-if="showAdvancedFilters" class="card p-5">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-6">
          <div><label class="input-label">时间范围</label><Select v-model="filters.time_range" :options="timeRangeOptions" @change="resetAndFetch" /></div>
          <div><label class="input-label">风险等级</label><Select v-model="filters.risk_level" :options="riskLevelOptions" @change="resetAndFetch" /></div>
          <div><label class="input-label">风险分类</label><Select v-model="filters.risk_category" :options="riskCategoryOptions" @change="resetAndFetch" /></div>
          <div><label class="input-label">处理状态</label><Select v-model="filters.status" :options="statusOptions" @change="resetAndFetch" /></div>
          <div><label class="input-label">Token 类型</label><Select v-model="filters.token_type" :options="tokenTypeOptions" @change="resetAndFetch" /></div>
          <div><label class="input-label">搜索</label><input v-model.trim="filters.q" class="input" placeholder="hash / path / IP / 原因" @keyup.enter="resetAndFetch" /></div>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="flex flex-col gap-2 border-b border-gray-100 p-4 dark:border-dark-700 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">事件明细</h2>
            <p class="text-xs text-gray-500">用于下钻排查，默认只看 open 事件，可在高级筛选中调整。</p>
          </div>
          <span class="text-xs text-gray-500">共 {{ total }} 条</span>
        </div>
        <div v-if="loading" class="flex items-center justify-center py-16"><LoadingSpinner size="lg" /></div>
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
                    {{ riskLevelText(row.risk_level) }} · {{ row.risk_score }}
                  </span>
                  <div class="mt-1 flex max-w-[260px] flex-wrap gap-1">
                    <span v-for="category in row.risk_categories.slice(0, 3)" :key="category" class="rounded bg-blue-50 px-2 py-0.5 text-xs text-blue-700 dark:bg-blue-900/30 dark:text-blue-200">{{ categoryLabel(category) }}</span>
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
                  <div>5m {{ row.count_5m ?? 0 }} / RPM {{ rpm(row) }}</div>
                  <div>1h {{ row.count_1h ?? 0 }} / IP {{ row.distinct_ip_24h ?? 0 }}</div>
                </td>
                <td class="px-4 py-3 text-right"><button class="btn btn-secondary btn-sm" type="button" @click="openDetail(row)">详情/处置</button></td>
              </tr>
            </tbody>
          </table>
        </div>
        <Pagination v-if="total > 0" :page="page" :total="total" :page-size="pageSize" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" />
      </section>
    </div>

    <BaseDialog :show="detailOpen" title="风险详情与处置" width="wide" @close="detailOpen = false">
      <div v-if="selectedEvent" class="space-y-4">
        <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">发生了什么</p>
            <p class="mt-1 text-sm text-gray-800 dark:text-gray-100">{{ detailExplanation?.summary || eventConclusion(selectedEvent) }}</p>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">为什么判定</p>
            <ul class="mt-1 list-disc space-y-1 pl-4 text-sm text-gray-800 dark:text-gray-100">
              <li v-for="reason in detailReasons" :key="reason">{{ reason }}</li>
            </ul>
          </div>
          <div class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700/50">
            <p class="text-xs font-semibold text-gray-500">建议怎么处理</p>
            <ul class="mt-1 list-disc space-y-1 pl-4 text-sm text-gray-800 dark:text-gray-100">
              <li v-for="step in detailSteps" :key="step">{{ step }}</li>
            </ul>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 xl:grid-cols-4">
          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">主体信息</p>
            <dl class="mt-3 space-y-2 text-sm">
              <div class="flex justify-between gap-3">
                <dt class="text-gray-500">用户名</dt>
                <dd class="truncate font-medium text-gray-900 dark:text-white" :title="subjectProfile?.username || ''">{{ subjectProfile?.username || '-' }}</dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-gray-500">用户 ID</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ subjectProfile?.user_id ?? selectedEvent.user_id ?? '-' }}</dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-gray-500">API key</dt>
                <dd class="truncate font-mono text-gray-900 dark:text-white" :title="subjectProfile?.api_key_name || subjectProfile?.api_key_summary || ''">
                  {{ subjectProfile?.api_key_name || subjectProfile?.api_key_summary || '-' }}
                </dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-gray-500">Key ID</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ subjectProfile?.api_key_id ?? selectedEvent.api_key_id ?? '-' }}</dd>
              </div>
              <div class="flex justify-between gap-3">
                <dt class="text-gray-500">Token 类型</dt>
                <dd class="font-mono text-gray-900 dark:text-white">{{ subjectProfile?.token_type || selectedEvent.token_type || '-' }}</dd>
              </div>
            </dl>
          </div>

          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">RPM 快照</p>
            <div class="mt-3 grid grid-cols-2 gap-2 text-sm">
              <div class="rounded-lg bg-gray-50 p-2 dark:bg-dark-700/50">
                <p class="text-xs text-gray-500">5 分钟</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ rpmSnapshot?.count_5m ?? selectedEvent.count_5m ?? 0 }} 次 / {{ rpmSnapshot?.rpm_5m ?? rpm(selectedEvent) }} RPM</p>
              </div>
              <div class="rounded-lg bg-gray-50 p-2 dark:bg-dark-700/50">
                <p class="text-xs text-gray-500">1 小时</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ rpmSnapshot?.count_1h ?? selectedEvent.count_1h ?? 0 }} 次 / {{ rpmSnapshot?.rpm_1h ?? '-' }} RPM</p>
              </div>
              <div class="rounded-lg bg-gray-50 p-2 dark:bg-dark-700/50">
                <p class="text-xs text-gray-500">24 小时</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ rpmSnapshot?.count_24h ?? selectedEvent.count_24h ?? 0 }} 次</p>
              </div>
              <div class="rounded-lg bg-gray-50 p-2 dark:bg-dark-700/50">
                <p class="text-xs text-gray-500">来源 IP</p>
                <p class="mt-1 font-semibold" :class="(rpmSnapshot?.distinct_ip_24h ?? selectedEvent.distinct_ip_24h ?? 0) >= 4 ? 'text-red-600' : 'text-gray-900 dark:text-white'">
                  {{ rpmSnapshot?.distinct_ip_24h ?? selectedEvent.distinct_ip_24h ?? 0 }} 个
                </p>
              </div>
            </div>
          </div>

          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700 xl:col-span-2">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">来源 IP</p>
            <div v-if="ipBreakdown.length === 0" class="mt-3 text-sm text-gray-500">暂无 IP 聚合数据</div>
            <div v-else class="mt-3 max-h-52 overflow-auto">
              <table class="min-w-full text-sm">
                <thead class="text-xs text-gray-500">
                  <tr>
                    <th class="py-1 text-left">IP</th>
                    <th class="py-1 text-right">次数</th>
                    <th class="py-1 text-left">状态码</th>
                    <th class="py-1 text-right">最近</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in ipBreakdown" :key="item.value" class="border-t border-gray-100 dark:border-dark-700">
                    <td class="py-1.5 font-mono text-gray-900 dark:text-white">{{ item.value }}</td>
                    <td class="py-1.5 text-right">{{ item.count }}</td>
                    <td class="py-1.5 text-xs text-gray-500">{{ statusCodeSummary(item) }}</td>
                    <td class="py-1.5 text-right text-xs text-gray-500">{{ formatTime(item.last_seen_at || '') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-3 xl:grid-cols-3">
          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">User-Agent</p>
            <div class="mt-3 space-y-2">
              <div v-for="item in uaBreakdown.slice(0, 5)" :key="item.value" class="rounded-lg bg-gray-50 p-2 text-sm dark:bg-dark-700/50">
                <div class="flex justify-between gap-2">
                  <span class="truncate" :title="item.value">{{ item.value }}</span>
                  <span class="font-semibold">{{ item.count }}</span>
                </div>
              </div>
              <div v-if="uaBreakdown.length === 0" class="text-sm text-gray-500">暂无 UA 数据</div>
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">请求路径</p>
            <div class="mt-3 space-y-2">
              <div v-for="item in pathBreakdown.slice(0, 6)" :key="item.value" class="rounded-lg bg-gray-50 p-2 text-sm dark:bg-dark-700/50">
                <div class="flex justify-between gap-2">
                  <span class="truncate font-mono" :title="item.value">{{ item.value }}</span>
                  <span class="font-semibold">{{ item.count }}</span>
                </div>
              </div>
              <div v-if="pathBreakdown.length === 0" class="text-sm text-gray-500">暂无路径数据</div>
            </div>
          </div>
          <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
            <p class="text-sm font-semibold text-gray-900 dark:text-white">失败原因</p>
            <div class="mt-3 space-y-2">
              <div v-for="item in failureBreakdown.slice(0, 6)" :key="item.value" class="rounded-lg bg-gray-50 p-2 text-sm dark:bg-dark-700/50">
                <div class="flex justify-between gap-2">
                  <span class="truncate" :title="item.value">{{ item.value }}</span>
                  <span class="font-semibold">{{ item.count }}</span>
                </div>
              </div>
              <div v-if="failureBreakdown.length === 0" class="text-sm text-gray-500">暂无失败原因数据</div>
            </div>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">最近事件</p>
          <div v-if="recentEvents.length === 0" class="mt-3 text-sm text-gray-500">暂无最近事件</div>
          <div v-else class="mt-3 overflow-x-auto">
            <table class="min-w-full text-sm">
              <thead class="text-xs text-gray-500">
                <tr>
                  <th class="py-2 text-left">时间</th>
                  <th class="py-2 text-left">IP</th>
                  <th class="py-2 text-left">请求</th>
                  <th class="py-2 text-left">状态/原因</th>
                  <th class="py-2 text-right">风险</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in recentEvents" :key="item.id" class="border-t border-gray-100 dark:border-dark-700">
                  <td class="whitespace-nowrap py-2 text-gray-500">{{ formatTime(item.created_at) }}</td>
                  <td class="py-2 font-mono">{{ item.client_ip || '-' }}</td>
                  <td class="max-w-[360px] truncate py-2 font-mono" :title="`${item.method} ${item.path}`">{{ item.method || '-' }} {{ item.path || '-' }}</td>
                  <td class="max-w-[260px] truncate py-2 text-gray-500" :title="item.failure_reason">HTTP {{ item.status_code || '-' }} · {{ item.failure_reason || '-' }}</td>
                  <td class="py-2 text-right">{{ riskLevelText(item.risk_level) }} · {{ item.risk_score }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">相关内容审核记录</p>
          <p class="mt-1 text-xs text-gray-500">{{ detailExplanation?.content_availability || '无可用内容摘要。' }}</p>
          <div v-if="relatedContentLogs.length === 0" class="mt-3 rounded-lg bg-gray-50 p-3 text-sm text-gray-500 dark:bg-dark-700/50">
            暂无可关联的脱敏内容摘要。/v1/models 这类无正文接口不会产生 prompt 内容；历史未记录的请求内容无法恢复。
          </div>
          <div v-else class="mt-3 space-y-2">
            <div v-for="log in relatedContentLogs" :key="log.id" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-700/50">
              <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500">
                <span>{{ formatTime(log.created_at) }}</span>
                <span>{{ log.endpoint || '-' }}</span>
                <span>{{ log.model || '-' }}</span>
                <span :class="log.flagged ? 'text-red-600' : 'text-gray-500'">
                  {{ log.action }} / {{ log.highest_category || 'none' }} / {{ Number(log.highest_score || 0).toFixed(3) }}
                </span>
              </div>
              <p class="mt-2 whitespace-pre-wrap text-gray-800 dark:text-gray-100">{{ log.input_excerpt || '无摘要' }}</p>
            </div>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">命中规则</p>
          <div class="mt-2 flex flex-wrap gap-2">
            <span v-for="rule in selectedEvent.matched_rules" :key="rule" class="rounded bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-200">{{ ruleLabel(rule) }}</span>
            <span v-if="selectedEvent.matched_rules.length === 0" class="text-sm text-gray-500">暂无</span>
          </div>
        </div>

        <div class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">处置动作</p>
          <div class="mt-3 flex flex-wrap gap-2">
            <button v-for="action in actionButtons" :key="action" class="btn btn-secondary btn-sm" type="button" :disabled="actionLoading" @click="applyAction(action)">
              {{ actionLabel(action) }}
            </button>
          </div>
          <textarea v-model.trim="actionNote" class="input mt-3 min-h-[80px]" placeholder="处置备注，不要填写完整 token、API key 或隐私原文"></textarea>
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
import tokenRiskAPI, {
  type TokenRiskAction,
  type TokenRiskBreakdownItem,
  type TokenRiskEvent,
  type TokenRiskHumanExplanation,
  type TokenRiskRecentEvent,
  type TokenRiskRelatedContentLog,
  type TokenRiskRPMSnapshot,
  type TokenRiskSubjectProfile,
  type TokenRiskSummary,
} from '@/api/admin/tokenRisk'
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
const relatedContentLogs = ref<TokenRiskRelatedContentLog[]>([])
const detailExplanation = ref<TokenRiskHumanExplanation | null>(null)
const subjectProfile = ref<TokenRiskSubjectProfile | null>(null)
const ipBreakdown = ref<TokenRiskBreakdownItem[]>([])
const uaBreakdown = ref<TokenRiskBreakdownItem[]>([])
const pathBreakdown = ref<TokenRiskBreakdownItem[]>([])
const failureBreakdown = ref<TokenRiskBreakdownItem[]>([])
const recentEvents = ref<TokenRiskRecentEvent[]>([])
const rpmSnapshot = ref<TokenRiskRPMSnapshot | null>(null)
const actionNote = ref('')
const showAdvancedFilters = ref(false)
const filters = reactive({ time_range: '24h', risk_level: '', risk_category: '', status: 'open', token_type: '', q: '' })

const timeRangeOptions = ['1h', '6h', '24h', '7d', '30d'].map((value) => ({ value, label: value }))
const riskLevelOptions = [{ value: '', label: '全部' }, { value: 'low', label: '低' }, { value: 'medium', label: '中' }, { value: 'high', label: '高' }, { value: 'critical', label: '严重' }]
const riskCategoryKeys = ['auth_invalid', 'auth_expired', 'auth_forged', 'permission_violation', 'admin_api_probe', 'high_frequency', 'batch_register', 'registrar_abuse', 'config_tamper', 'balance_or_reward_abuse', 'game_abuse', 'embedded_bypass', 'api_key_sharing', 'adult_content', 'grey_industry', 'abnormal_geo_or_ua', 'insufficient_balance_abuse', 'suspicious_path_scan']
const riskCategoryOptions = [{ value: '', label: '全部分类' }, ...riskCategoryKeys.map((value) => ({ value, label: categoryLabel(value) }))]
const statusOptions = [{ value: '', label: '全部状态' }, { value: 'open', label: '待处理' }, { value: 'handled', label: '已处理' }, { value: 'false_positive', label: '误报' }, { value: 'watching', label: '观察中' }]
const tokenTypeOptions = [{ value: '', label: '全部' }, 'jwt', 'admin_jwt', 'api_key', 'admin_api_key', 'embedded'].map((item) => typeof item === 'string' ? { value: item, label: item } : item)
const actionButtons = ['mark_handled', 'mark_false_positive', 'watch_user', 'watch_token', 'force_relogin', 'send_warning', 'send_reminder']

const summaryCards = computed(() => {
  const data = summary.value
  return [
    { key: 'open', label: '待处理', value: data?.open ?? 0, meta: '需要管理员确认', tone: 'text-amber-600', apply: () => setStatusFilter('open') },
    { key: 'critical', label: '严重', value: data?.critical ?? 0, meta: '立即排查', tone: 'text-red-600', apply: () => setRiskLevelFilter('critical') },
    { key: 'high', label: '高风险', value: data?.high ?? 0, meta: '优先处理', tone: 'text-orange-600', apply: () => setRiskLevelFilter('high') },
    { key: 'users', label: '异常用户', value: data?.distinct_users ?? 0, meta: '去重 user_id', tone: 'text-gray-900 dark:text-white' },
    { key: 'keys', label: '异常 API Key', value: data?.distinct_api_keys ?? 0, meta: '去重 key id', tone: 'text-gray-900 dark:text-white' },
    { key: 'total', label: '风险事件', value: data?.total ?? 0, meta: '当前时间范围', tone: 'text-gray-900 dark:text-white' },
  ]
})

const priorityEvents = computed(() => {
  const source = summary.value?.recent_high_risk?.length ? summary.value.recent_high_risk : events.value
  return source
    .filter((item) => item.status === 'open')
    .sort((a, b) => priorityScore(b) - priorityScore(a))
    .slice(0, 6)
})

const suspiciousApiKeyRows = computed(() => {
  return events.value
    .filter((item) => item.api_key_id && (isMultiIp(item) || rpm(item) >= 20 || (item.count_1h ?? 0) >= 120 || item.risk_categories.includes('api_key_sharing') || item.risk_categories.includes('high_frequency')))
    .sort((a, b) => priorityScore(b) - priorityScore(a))
    .slice(0, 8)
})

const topSubjects = computed(() => {
  const data = summary.value
  if (!data) return []
  return [
    ...(data.top_api_keys || []).slice(0, 4).map((item) => ({ ...item, label: 'API Key' })),
    ...(data.top_users || []).slice(0, 3).map((item) => ({ ...item, label: '用户' })),
    ...(data.top_tokens || []).slice(0, 3).map((item) => ({ ...item, label: 'Token Hash' })),
  ].slice(0, 8)
})

const categoryDistribution = computed(() => {
  const data = summary.value?.by_category || {}
  const max = Math.max(1, ...Object.values(data))
  return Object.entries(data)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 8)
    .map(([key, count]) => ({ key, count, percent: Math.max(4, Math.round((count / max) * 100)) }))
})

const detailReasons = computed(() => detailExplanation.value?.reasons?.length ? detailExplanation.value.reasons : [selectedEvent.value ? eventConclusion(selectedEvent.value) : '-'])
const detailSteps = computed(() => detailExplanation.value?.recommended_next_steps?.length ? detailExplanation.value.recommended_next_steps : [selectedEvent.value ? recommendedText(selectedEvent.value) : '-'])

function rpm(row: TokenRiskEvent): number {
  return Math.round(((row.count_5m ?? 0) / 5) * 10) / 10
}

function isMultiIp(row: TokenRiskEvent): boolean {
  return (row.distinct_ip_24h ?? 0) >= 4
}

function priorityScore(row: TokenRiskEvent): number {
  let score = row.risk_score || 0
  if (isMultiIp(row)) score += 30
  if (rpm(row) >= 20) score += 25
  if (row.risk_categories.includes('admin_api_probe') || row.risk_categories.includes('embedded_bypass')) score += 20
  return score
}

function formatTime(value: string): string {
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? (value || '-') : d.toLocaleString()
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

function statusCodeSummary(item: TokenRiskBreakdownItem): string {
  const codes = item.status_codes || {}
  const pairs = Object.entries(codes)
    .sort((a, b) => Number(a[0]) - Number(b[0]))
    .slice(0, 4)
    .map(([code, count]) => `${code}:${count}`)
  return pairs.length ? pairs.join(' / ') : '-'
}

function eventConclusion(row: TokenRiskEvent): string {
  if (isMultiIp(row)) return `同一 API key 近 24 小时出现 ${row.distinct_ip_24h} 个来源 IP，疑似共享、泄露或跨环境异常使用。`
  if (rpm(row) >= 20) return `同主体近 5 分钟约 ${rpm(row)} RPM，疑似客户端循环重试、脚本调用或共享 key。`
  if (row.risk_categories.includes('insufficient_balance_abuse')) return `余额不足后仍持续重试，建议检查客户端重试策略或临时暂停该 key。`
  if (row.risk_categories.includes('admin_api_probe')) return `普通主体访问后台或管理员接口，存在越权探测风险。`
  if (row.risk_categories.includes('embedded_bypass')) return `embedded 鉴权参数异常，需检查入口 k、token、来源域名和嵌入链路。`
  if (row.failure_reason) return `${categoryLabel(row.risk_categories[0] || '')}：${row.failure_reason}`
  return row.explanation || `${row.risk_categories.map(categoryLabel).join('、') || '未知风险'}，命中 ${row.matched_rules.length || 0} 条规则`
}

function recommendedText(row: TokenRiskEvent): string {
  if (!row?.recommended_actions?.length) return '先查看同用户、同 IP、同 token hash 的历史行为，再决定标记已处理、误报或观察。'
  return row.recommended_actions.map(actionLabel).join('、')
}

function actionLabel(action: string): string {
  return ({
    mark_handled: '标记已处理',
    mark_false_positive: '标记误报',
    watch_user: '观察用户',
    watch_token: '观察 Token',
    force_relogin: '强制重新登录',
    send_warning: '发送警告',
    send_reminder: '发送提醒',
  } as Record<string, string>)[action] || action
}

function statusText(status: string): string {
  return ({ open: '待处理', handled: '已处理', false_positive: '误报', watching: '观察中' } as Record<string, string>)[status] || status
}

function riskLevelText(level: string): string {
  return ({ low: '低', medium: '中', high: '高', critical: '严重' } as Record<string, string>)[level] || level
}

function categoryLabel(value: string): string {
  return ({
    balance_or_reward_abuse: '余额/权限异常',
    insufficient_balance_abuse: '余额不足持续重试',
    permission_violation: '权限不足',
    high_frequency: '高频请求',
    api_key_sharing: 'API key 多 IP',
    admin_api_probe: '管理员接口探测',
    embedded_bypass: 'embedded 绕过',
    grey_industry: '疑似灰产',
    adult_content: '疑似色情',
    auth_invalid: '鉴权无效',
    auth_expired: 'token 过期',
    auth_forged: '疑似伪造',
    config_tamper: '配置篡改',
    batch_register: '批量注册',
    registrar_abuse: '注册机异常',
    game_abuse: '游戏套利',
    suspicious_path_scan: '路径扫描',
    abnormal_geo_or_ua: '来源异常',
  } as Record<string, string>)[value] || value || '未知风险'
}

function ruleLabel(value: string): string {
  return ({
    insufficient_balance_single: '单次余额不足',
    insufficient_balance_repeated: '余额不足持续重试',
    permission_denied: '权限被拒绝',
    high_frequency_window: '短时间高频',
    rpm_anomaly_window: 'RPM 异常',
    multi_ip_api_key_usage: '多 IP 使用同一 API key',
    non_admin_token_admin_path: '非管理员主体访问管理路径',
    embedded_auth_bypass: 'embedded 鉴权绕过',
  } as Record<string, string>)[value] || value
}

function setStatusFilter(value: string) { filters.status = value; resetAndFetch() }
function setRiskLevelFilter(value: string) { filters.risk_level = value; resetAndFetch() }
function buildQuery() {
  return {
    page: page.value,
    page_size: pageSize.value,
    time_range: filters.time_range,
    risk_level: filters.risk_level || undefined,
    risk_category: filters.risk_category || undefined,
    token_type: filters.token_type || undefined,
    status: filters.status || undefined,
    q: filters.q || undefined,
  }
}

async function fetchSummary() { summary.value = await tokenRiskAPI.getSummary(filters.time_range) }
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
function resetAndFetch() { page.value = 1; fetchAll() }
function handlePageChange(nextPage: number) { page.value = nextPage; fetchAll() }
function handlePageSizeChange(nextPageSize: number) { pageSize.value = nextPageSize; page.value = 1; fetchAll() }

async function openDetail(row: TokenRiskEvent) {
  detailOpen.value = true
  selectedEvent.value = row
  actions.value = []
  relatedContentLogs.value = []
  detailExplanation.value = null
  subjectProfile.value = null
  ipBreakdown.value = []
  uaBreakdown.value = []
  pathBreakdown.value = []
  failureBreakdown.value = []
  recentEvents.value = []
  rpmSnapshot.value = null
  actionNote.value = ''
  try {
    const detail = await tokenRiskAPI.getEvent(row.id)
    selectedEvent.value = detail.event
    actions.value = detail.actions || []
    relatedContentLogs.value = detail.related_content_logs || []
    detailExplanation.value = detail.human_explanation || null
    subjectProfile.value = detail.subject_profile || null
    ipBreakdown.value = detail.ip_breakdown || []
    uaBreakdown.value = detail.ua_breakdown || []
    pathBreakdown.value = detail.path_breakdown || []
    failureBreakdown.value = detail.failure_breakdown || []
    recentEvents.value = detail.recent_events || []
    rpmSnapshot.value = detail.rpm_snapshot || null
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
    await tokenRiskAPI.createAction(selectedEvent.value.id, { action, note: actionNote.value, confirm: confirmRequired })
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
