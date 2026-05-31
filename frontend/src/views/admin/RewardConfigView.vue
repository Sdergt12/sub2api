<template>
  <div class="space-y-6">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 dark:text-white">运营额度配置</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          动态调整签到奖励、连续签到加成、游戏中心次数、成本、每日上限和付费局赔率。Worker 和签到服务最多 30 秒生效。
        </p>
      </div>
      <button class="btn-primary" :disabled="loading || saving || !config" @click="save">
        {{ saving ? '保存中...' : '保存配置' }}
      </button>
    </div>

    <div v-if="error" class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-200">
      {{ error }}
    </div>

    <div v-if="loading" class="card p-6 text-sm text-gray-500 dark:text-gray-400">正在加载配置...</div>

    <template v-else-if="config">
      <section class="card space-y-4 p-6">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">签到奖励</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">基础奖励按权重随机，连签加成只在指定天数额外发放。</p>
          </div>
          <button class="btn-secondary" @click="addSignTier">新增档位</button>
        </div>

        <div class="overflow-x-auto">
          <table class="table">
            <thead>
              <tr>
                <th>最小金额</th>
                <th>最大金额</th>
                <th>权重</th>
                <th class="w-20">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(tier, index) in config.sign.reward_tiers" :key="index">
                <td><input v-model.number="tier.min" class="input w-28" type="number" min="0" step="0.01" /></td>
                <td><input v-model.number="tier.max" class="input w-28" type="number" min="0" step="0.01" /></td>
                <td><input v-model.number="tier.weight" class="input w-24" type="number" min="1" step="1" /></td>
                <td><button class="text-sm text-red-600 hover:text-red-700" @click="removeSignTier(index)">删除</button></td>
              </tr>
            </tbody>
          </table>
        </div>

        <div class="grid gap-4 md:grid-cols-4">
          <label class="block text-sm">
            <span class="label">3 天加成</span>
            <input v-model.number="config.sign.bonus_day3" class="input mt-1" type="number" min="0" step="0.01" />
          </label>
          <label class="block text-sm">
            <span class="label">7 天加成</span>
            <input v-model.number="config.sign.bonus_day7" class="input mt-1" type="number" min="0" step="0.01" />
          </label>
          <label class="block text-sm">
            <span class="label">15 天加成</span>
            <input v-model.number="config.sign.bonus_day15" class="input mt-1" type="number" min="0" step="0.01" />
          </label>
          <label class="block text-sm">
            <span class="label">30 天加成</span>
            <input v-model.number="config.sign.bonus_day30" class="input mt-1" type="number" min="0" step="0.01" />
          </label>
        </div>
      </section>

      <section class="card space-y-4 p-6">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">游戏中心额度</h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">次数、付费成本和每日净收益上限只影响新开局，不回算历史局。</p>
        </div>

        <div class="grid gap-4 md:grid-cols-4">
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

        <div class="grid gap-3 md:grid-cols-3">
          <label v-for="game in games" :key="game.id" class="flex items-center gap-2 rounded-lg border border-gray-200 p-3 text-sm dark:border-dark-700">
            <input
              type="checkbox"
              :checked="config.game_center.disabled_game_ids.includes(game.id)"
              @change="toggleDisabledGame(game.id, ($event.target as HTMLInputElement).checked)"
            />
            <span>禁用 {{ game.label }}</span>
          </label>
        </div>
      </section>

      <section class="card space-y-5 p-6">
        <div>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">付费局赔率桶</h2>
          <p class="text-sm text-gray-500 dark:text-gray-400">每个玩法和风险模式至少保留一个正权重桶；金额可以为负数，用于扣除型结果。</p>
        </div>

        <div v-for="game in games" :key="game.id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <h3 class="font-semibold text-gray-900 dark:text-white">{{ game.label }}</h3>
          <div class="mt-4 grid gap-4 lg:grid-cols-2">
            <div v-for="mode in riskModes" :key="mode.id" class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ mode.label }}</span>
                <button class="btn-secondary py-1 text-xs" @click="addBucket(game.id, mode.id)">新增桶</button>
              </div>
              <div class="space-y-2">
                <div
                  v-for="(bucket, index) in config.game_center.reward_profiles[game.id][mode.id]"
                  :key="index"
                  class="grid grid-cols-[1fr_1fr_auto] items-center gap-2"
                >
                  <input v-model.number="bucket.bucket" class="input" type="number" step="0.01" placeholder="金额" />
                  <input v-model.number="bucket.weight" class="input" type="number" min="1" step="1" placeholder="权重" />
                  <button class="text-sm text-red-600" @click="removeBucket(game.id, mode.id, index)">删除</button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import {
  getRewardRuntimeConfig,
  updateRewardRuntimeConfig,
  type RewardGameId,
  type RewardRiskMode,
  type RewardRuntimeConfig,
} from '@/api/admin/settings';

const games: Array<{ id: RewardGameId; label: string }> = [
  { id: 'flip_card', label: '翻牌' },
  { id: 'lucky_wheel', label: '转盘' },
  { id: 'smash_egg', label: '砸蛋' },
];

const riskModes: Array<{ id: RewardRiskMode; label: string }> = [
  { id: 'steady', label: '稳健模式' },
  { id: 'high_multiplier', label: '高倍模式' },
];

const loading = ref(false);
const saving = ref(false);
const error = ref('');
const config = ref<RewardRuntimeConfig | null>(null);

function validateConfig(value: RewardRuntimeConfig): string {
  if (!value.sign.reward_tiers.length) return '签到奖励至少需要一个档位';
  for (const tier of value.sign.reward_tiers) {
    if (tier.min < 0 || tier.max < tier.min || tier.weight <= 0) return '签到奖励档位金额或权重不合法';
  }
  if (value.game_center.free_play_limit < 1 || value.game_center.free_play_limit > 50) return '免费次数必须在 1-50 之间';
  if (value.game_center.paid_play_limit < 1 || value.game_center.paid_play_limit > 50) return '付费次数必须在 1-50 之间';
  for (const game of games) {
    for (const mode of riskModes) {
      const buckets = value.game_center.reward_profiles[game.id][mode.id];
      if (!buckets?.length || buckets.some((item) => item.weight <= 0)) return `${game.label} ${mode.label} 赔率桶不合法`;
    }
  }
  return '';
}

async function load() {
  loading.value = true;
  error.value = '';
  try {
    config.value = await getRewardRuntimeConfig();
  } catch (err: any) {
    error.value = err?.message || '加载运营额度配置失败';
  } finally {
    loading.value = false;
  }
}

async function save() {
  if (!config.value) return;
  const validation = validateConfig(config.value);
  if (validation) {
    error.value = validation;
    return;
  }
  saving.value = true;
  error.value = '';
  try {
    config.value = await updateRewardRuntimeConfig(config.value);
  } catch (err: any) {
    error.value = err?.message || '保存运营额度配置失败';
  } finally {
    saving.value = false;
  }
}

function addSignTier() {
  config.value?.sign.reward_tiers.push({ min: 0.5, max: 2, weight: 10 });
}

function removeSignTier(index: number) {
  if (!config.value || config.value.sign.reward_tiers.length <= 1) return;
  config.value.sign.reward_tiers.splice(index, 1);
}

function toggleDisabledGame(gameId: RewardGameId, checked: boolean) {
  if (!config.value) return;
  const ids = new Set(config.value.game_center.disabled_game_ids);
  if (checked) ids.add(gameId);
  else ids.delete(gameId);
  config.value.game_center.disabled_game_ids = Array.from(ids) as RewardGameId[];
}

function addBucket(gameId: RewardGameId, mode: RewardRiskMode) {
  config.value?.game_center.reward_profiles[gameId][mode].push({ bucket: 1, weight: 10 });
}

function removeBucket(gameId: RewardGameId, mode: RewardRiskMode, index: number) {
  const buckets = config.value?.game_center.reward_profiles[gameId][mode];
  if (!buckets || buckets.length <= 1) return;
  buckets.splice(index, 1);
}

onMounted(load);
</script>
