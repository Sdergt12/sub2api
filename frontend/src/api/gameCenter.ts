import { apiClient } from './client'

export type GameCenterRange = 'today' | '7d' | '30d' | 'all'
export type GameCenterStakeType = 'free' | 'paid'

export interface GameCenterLeaderboardItem {
  rank: number
  user_id: number
  username: string
  avatar_url: string
  net_amount: number
  play_count: number
  win_rate: number
  last_played_at?: string
}

export interface GameCenterLeaderboardResponse {
  range: GameCenterRange
  game_key?: string
  limit: number
  items: GameCenterLeaderboardItem[]
}

export interface GameCenterRecordPlayPayload {
  game_key: string
  round_id: string
  stake_type: GameCenterStakeType
  cost_amount: number
  reward_amount: number
  net_amount: number
  metadata?: Record<string, unknown>
}

export interface GameCenterPlay {
  id: number
  user_id: number
  game_key: string
  round_id: string
  stake_type: GameCenterStakeType
  cost_amount: number
  reward_amount: number
  net_amount: number
  played_at: string
  duplicate?: boolean
}

export interface GameCenterMeResponse {
  stats: {
    user_id: number
    today_play_count: number
    today_net_amount: number
    today_free_count: Record<string, number>
    today_paid_count: Record<string, number>
    remaining: Record<string, { free: number; paid: number }>
  }
  today_rank: number
}

export async function getGameCenterLeaderboard(params: {
  game_key?: string
  range?: GameCenterRange
  limit?: number
} = {}) {
  const { data } = await apiClient.get<GameCenterLeaderboardResponse>('/game-center/leaderboard', { params })
  return data
}

export async function recordGameCenterPlay(payload: GameCenterRecordPlayPayload) {
  const { data } = await apiClient.post<GameCenterPlay>('/game-center/plays', payload)
  return data
}

export async function getMyGameCenterStats() {
  const { data } = await apiClient.get<GameCenterMeResponse>('/game-center/me')
  return data
}
