<template>
  <AppLayout>
  <div class="space-y-5 pb-20">
    <section class="card p-5">
      <div class="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.22em] text-primary-600 dark:text-primary-300">
            Operations
          </p>
          <h1 class="mt-2 text-2xl font-bold text-gray-900 dark:text-white">运营配置</h1>
          <p class="mt-1 max-w-3xl text-sm text-gray-500 dark:text-gray-400">
            统一调整签到奖励和游戏中心额度。配置保存后由主站提供运行时配置，Worker 和签到服务最多 30 秒内读取生效。
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="btn btn-secondary" :disabled="loading || saving" type="button" @click="load">刷新</button>
          <button class="btn btn-primary" :disabled="loading || saving || !config" type="button" @click="save">
            {{ saving ? '保存中...' : '保存配置' }}
          </button>
        </div>
      </div>
    </section>

    <div
      v-if="error"
      class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-200"
    >
      {{ error }}
    </div>
    <div
      v-if="success"
      class="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-200"
    >
      {{ success }}
    </div>

    <div v-if="loading" class="card p-8 text-center text-sm text-gray-500 dark:text-gray-400">正在加载运营配置...</div>

    <template v-else-if="config">
      <section class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-5">
        <div v-for="item in summaryCards" :key="item.label" class="card p-4">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ item.label }}</p>
          <p class="mt-2 text-xl font-bold text-gray-900 dark:text-white">{{ item.value }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ item.hint }}</p>
        </div>
      </section>

      <section class="grid grid-cols-1 gap-5 xl:grid-cols-[minmax(0,1.1fr)_minmax(0,0.9fr)]">
        <div class="card space-y-4 p-5">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">签到奖励</h2>
              <p class="text-sm text-gray-500 dark:text-gray-400">基础奖励按权重随机；连签加成只在对应天数额外发放。</p>
            </div>
            <button class="btn btn-secondary btn-sm" type="button" @click="addSignTier">新增档位</button>
          </div>

          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
              <thead class="bg-gray-50 dark:bg-dark-900/50">
                <tr>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500">最小金额</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500">最大金额</th>
                  <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500">权重</th>
                  <th class="px-3 py-2 text-right text-xs font-semibold text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="(tier, index) in config.sign.reward_tiers" :key="index">
                  <td class="px-3 py-2"><input v-model.number="tier.min" class="input w-28" type="number" min="0" step="0.01" /></td>
                  <td class="px-3 py-2"><input v-model.number="tier.max" class="input w-28" type="number" min="0" step="0.01" /></td>
                  <td class="px-3 py-2"><input v-model.number="tier.weight" class="input w-24" type="number" min="1" step="1" /></td>
                  <td class="px-3 py-2 text-right">
                    <button class="text-sm text-red-600 hover:text-red-700" type="button" @click="removeSignTier(index)">删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <label v-for="bonus in bonusFields" :key="bonus.key" class="block rounded-xl bg-gray-50 p-3 text-sm dark:bg-dark-700/50">
              <span class="label">{{ bonus.label }}</span>
              <input v-model.number="config.sign[bonus.key]" class="input mt-1" type="number" min="0" step="0.01" />
            </label>
          </div>
        </div>

        <div class="card space-y-4 p-5">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">游戏额度</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">次数、成本和每日净收益上限只影响新开局，不回算历史局。</p>
          </div>

          <div class="grid gap-3 sm:grid-cols-2">
            <label class="block text-sm">
              <span class="label">免费次数</span>
              <input v-model.number="config.game_center.free_play_limit" class="input mt-1" type="number" min="1" max="50" step="1" />
            </label>
            <label class="block text-sm">
              <span class="label">付费次数</span>
              <input v-model.number="config.game_center.paid_play_limit" class="input mt-1" type="number" min="1" max="50" step="1" />
            </label>
            <label class="block text-sm">
              <span class="label">付费成本</span>
              <input v-model="config.game_center.paid_play_cost" class="input mt-1" type="text" placeholder="2.00" />
            </label>
            <label class="block text-sm">
              <span class="label">每日净收益上限</span>
              <input v-model="config.game_center.daily_net_reward_hard_cap" class="input mt-1" type="text" placeholder="25.00" />
            </label>
          </div>

          <label class="flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-100">
            <input v-model="config.game_center.disable_play_cap_for_testing" class="mt-1" type="checkbox" />
            <span>
              <span class="block font-medium">临时关闭次数上限</span>
              <span class="block text-xs opacity-80">仅用于排查或活动测试，线上常规运营不建议开启。</span>
            </span>
          </label>

          <div>
            <p class="mb-2 text-sm font-medium text-gray-900 dark:text-white">禁用玩法</p>
            <div class="grid gap-2">
              <label v-for="game in games" :key="game.id" class="flex items-center justify-between rounded-xl border border-gray-200 p-3 text-sm dark:border-dark-700">
                <span>{{ game.label }}</span>
                <input
                  type="checkbox"
                  :checked="config.game_center.disabled_game_ids.includes(game.id)"
                  @change="toggleDisabledGame(game.id, ($event.target as HTMLInputElement).checked)"
                />
              </label>
            </div>
          </div>
        </div>
      </section>

      <section class="card space-y-5 p-5">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">赔率配置</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">每个玩法和风险模式至少保留一个正权重桶；金额可为负数，用于扣除型结果。</p>
          </div>
          <span class="text-xs text-gray-500">权重越高，命中概率越高。</span>
        </div>

        <div class="grid gap-4 xl:grid-cols-3">
          <div v-for="game in games" :key="game.id" class="rounded-2xl border border-gray-200 p-4 dark:border-dark-700">
            <h3 class="font-semibold text-gray-900 dark:text-white">{{ game.label }}</h3>
            <div class="mt-4 space-y-4">
              <div v-for="mode in riskModes" :key="mode.id" class="space-y-2">
                <div class="flex items-center justify-between">
                  <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ mode.label }}</span>
                  <button class="btn btn-secondary btn-sm py-1 text-xs" type="button" @click="addBucket(game.id, mode.id)">新增桶</button>
                </div>
                <div class="space-y-2">
                  <div
                    v-for="(bucket, index) in config.game_center.reward_profiles[game.id][mode.id]"
                    :key="index"
                    class="grid grid-cols-[1fr_1fr_auto] items-center gap-2"
                  >
                    <input v-model.number="bucket.bucket" class="input" type="number" step="0.01" placeholder="金额" />
                    <input v-model.number="bucket.weight" class="input" type="number" min="1" step="1" placeholder="权重" />
                    <button class="text-sm text-red-600" type="button" @click="removeBucket(game.id, mode.id, index)">删除</button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </template>

    <div class="sticky bottom-0 z-20 border-t border-gray-200 bg-white/90 px-4 py-3 backdrop-blur dark:border-dark-700 dark:bg-dark-900/90">
      <div class="mx-auto flex max-w-7xl items-center justify-between gap-3">
        <p class="text-xs text-gray-500 dark:text-gray-400">保存后最多 30 秒生效。请避免在高峰期大幅调整赔率。</p>
        <button class="btn btn-primary" :disabled="loading || saving || !config" type="button" @click="save">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>
  </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import {
  getRewardRuntimeConfig,
  updateRewardRuntimeConfig,
  type RewardGameId,
  type RewardRiskMode,
  type RewardRuntimeConfig,
} from '@/api/admin/settings'

type SignBonusKey = 'bonus_day3' | 'bonus_day7' | 'bonus_day15' | 'bonus_day30'

const games: Array<{ id: RewardGameId; label: string }> = [
  { id: 'flip_card', label: '翻牌' },
  { id: 'lucky_wheel', label: '转盘' },
  { id: 'smash_egg', label: '砸蛋' },
]

const riskModes: Array<{ id: RewardRiskMode; label: string }> = [
  { id: 'steady', label: '稳健模式' },
  { id: 'high_multiplier', label: '高倍模式' },
]

const bonusFields: Array<{ key: SignBonusKey; label: string }> = [
  { key: 'bonus_day3', label: '3 天加成' },
  { key: 'bonus_day7', label: '7 天加成' },
  { key: 'bonus_day15', label: '15 天加成' },
  { key: 'bonus_day30', label: '30 天加成' },
]

const loading = ref(false)
const saving = ref(false)
const error = ref('')
const success = ref('')
const config = ref<RewardRuntimeConfig | null>(null)

const summaryCards = computed(() => {
  if (!config.value) return []
  const tiers = config.value.sign.reward_tiers
  const min = Math.min(...tiers.map((item) => Number(item.min || 0)))
  const max = Math.max(...tiers.map((item) => Number(item.max || 0)))
  const disabled = config.value.game_center.disabled_game_ids.length
  return [
    { label: '签到范围', value: `${formatMoney(min)} - ${formatMoney(max)}`, hint: `${tiers.length} 个奖励档位` },
    { label: '连签加成', value: `3/7/15/30 天`, hint: `${formatMoney(config.value.sign.bonus_day3)} / ${formatMoney(config.value.sign.bonus_day7)} / ${formatMoney(config.value.sign.bonus_day15)} / ${formatMoney(config.value.sign.bonus_day30)}` },
    { label: '游戏次数', value: `${config.value.game_center.free_play_limit}+${config.value.game_center.paid_play_limit}`, hint: '免费 + 付费' },
    { label: '付费成本', value: formatMoney(config.value.game_center.paid_play_cost), hint: '付费局入场成本' },
    { label: '日净收益上限', value: formatMoney(config.value.game_center.daily_net_reward_hard_cap), hint: disabled ? `已禁用 ${disabled} 个玩法` : '所有玩法可用' },
  ]
})

function formatMoney(value: number | string): string {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed.toFixed(2) : '0.00'
}

function validateConfig(value: RewardRuntimeConfig): string {
  if (!value.sign.reward_tiers.length) return '签到奖励至少需要一个档位'
  for (const tier of value.sign.reward_tiers) {
    if (!Number.isFinite(Number(tier.min)) || !Number.isFinite(Number(tier.max)) || !Number.isFinite(Number(tier.weight))) {
      return '签到奖励档位必须是有效数字'
    }
    if (tier.min < 0 || tier.max < tier.min || tier.weight <= 0) return '签到奖励档位金额或权重不合法'
  }
  if (value.game_center.free_play_limit < 1 || value.game_center.free_play_limit > 50) return '免费次数必须在 1-50 之间'
  if (value.game_center.paid_play_limit < 1 || value.game_center.paid_play_limit > 50) return '付费次数必须在 1-50 之间'
  if (!Number.isFinite(Number(value.game_center.paid_play_cost)) || Number(value.game_center.paid_play_cost) < 0) return '付费成本不合法'
  if (!Number.isFinite(Number(value.game_center.daily_net_reward_hard_cap)) || Number(value.game_center.daily_net_reward_hard_cap) < 0) return '每日净收益上限不合法'
  for (const game of games) {
    for (const mode of riskModes) {
      const buckets = value.game_center.reward_profiles[game.id]?.[mode.id]
      if (!buckets?.length || buckets.some((item) => !Number.isFinite(Number(item.bucket)) || !Number.isFinite(Number(item.weight)) || item.weight <= 0)) {
        return `${game.label} ${mode.label} 赔率桶不合法`
      }
    }
  }
  return ''
}

async function load() {
  loading.value = true
  error.value = ''
  success.value = ''
  try {
    config.value = await getRewardRuntimeConfig()
  } catch (err: any) {
    error.value = err?.response?.data?.detail || err?.message || '加载运营配置失败'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!config.value) return
  const validation = validateConfig(config.value)
  if (validation) {
    error.value = validation
    success.value = ''
    return
  }
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    config.value = await updateRewardRuntimeConfig(config.value)
    success.value = '运营配置已保存，运行时配置将在 30 秒内同步。'
  } catch (err: any) {
    error.value = err?.response?.data?.detail || err?.message || '保存运营配置失败'
  } finally {
    saving.value = false
  }
}

function addSignTier() {
  config.value?.sign.reward_tiers.push({ min: 0.5, max: 2, weight: 10 })
}

function removeSignTier(index: number) {
  if (!config.value || config.value.sign.reward_tiers.length <= 1) return
  config.value.sign.reward_tiers.splice(index, 1)
}

function toggleDisabledGame(gameId: RewardGameId, checked: boolean) {
  if (!config.value) return
  const ids = new Set(config.value.game_center.disabled_game_ids)
  if (checked) ids.add(gameId)
  else ids.delete(gameId)
  config.value.game_center.disabled_game_ids = Array.from(ids) as RewardGameId[]
}

function addBucket(gameId: RewardGameId, mode: RewardRiskMode) {
  config.value?.game_center.reward_profiles[gameId][mode].push({ bucket: 1, weight: 10 })
}

function removeBucket(gameId: RewardGameId, mode: RewardRiskMode, index: number) {
  const buckets = config.value?.game_center.reward_profiles[gameId][mode]
  if (!buckets || buckets.length <= 1) return
  buckets.splice(index, 1)
}

onMounted(load)
</script>
