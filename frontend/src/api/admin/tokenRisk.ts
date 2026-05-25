import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export type TokenRiskLevel = 'low' | 'medium' | 'high' | 'critical'
export type TokenRiskStatus = 'open' | 'handled' | 'false_positive' | 'watching'

export interface TokenRiskEvent {
  id: number
  created_at: string
  updated_at: string
  last_seen_at: string
  source_log_id?: number
  user_id?: number
  api_key_id?: number
  token_type: string
  token_hash: string
  token_prefix: string
  token_suffix: string
  api_key_summary: string
  client_ip: string
  user_agent: string
  method: string
  path: string
  status_code: number
  result: string
  failure_reason: string
  risk_score: number
  risk_level: TokenRiskLevel
  risk_categories: string[]
  matched_rules: string[]
  recommended_actions: string[]
  explanation: string
  status: TokenRiskStatus
  false_positive: boolean
  handled_by_user_id?: number
  handled_at?: string
  count_5m?: number
  count_1h?: number
  count_24h?: number
  distinct_ip_24h?: number
}

export interface TokenRiskAction {
  id: number
  created_at: string
  event_id: number
  actor_user_id: number
  action: string
  note: string
  result: string
  metadata?: Record<string, unknown>
}

export interface TokenRiskRelatedContentLog {
  id: number
  created_at: string
  request_id: string
  user_id?: number
  api_key_id?: number
  endpoint: string
  provider: string
  model: string
  action: string
  flagged: boolean
  highest_category: string
  highest_score: number
  input_excerpt: string
  violation_count: number
  auto_banned: boolean
}

export interface TokenRiskRelatedActivity {
  count_5m: number
  count_1h: number
  count_24h: number
  distinct_ip_24h: number
}

export interface TokenRiskHumanExplanation {
  summary: string
  reasons: string[]
  recommended_next_steps: string[]
  content_availability: string
}

export interface TokenRiskSubjectStat {
  subject: string
  user_id?: number
  count: number
  score: number
}

export interface TokenRiskSummary {
  total: number
  open: number
  handled: number
  false_positive: number
  high: number
  critical: number
  distinct_users: number
  distinct_tokens: number
  distinct_api_keys: number
  by_level: Record<string, number>
  by_category: Record<string, number>
  top_users: TokenRiskSubjectStat[]
  top_tokens: TokenRiskSubjectStat[]
  top_api_keys: TokenRiskSubjectStat[]
  recent_high_risk: TokenRiskEvent[]
}

export interface TokenRiskWatchlistItem {
  id: number
  created_at: string
  updated_at: string
  subject_type: string
  subject_value: string
  reason: string
  actor_user_id: number
  active: boolean
}

export interface TokenRiskEventQuery {
  page?: number
  page_size?: number
  time_range?: string
  risk_level?: string
  risk_category?: string
  token_type?: string
  status?: string
  user_id?: number
  api_key_id?: number
  q?: string
}

export interface TokenRiskEventDetail {
  event: TokenRiskEvent
  actions: TokenRiskAction[]
  related_content_logs: TokenRiskRelatedContentLog[]
  related_activity: TokenRiskRelatedActivity
  human_explanation: TokenRiskHumanExplanation
}

export interface TokenRiskActionRequest {
  action: string
  note?: string
  confirm?: boolean
}

export async function getSummary(timeRange = '24h'): Promise<TokenRiskSummary> {
  const { data } = await apiClient.get<TokenRiskSummary>('/admin/token-risks/summary', {
    params: { time_range: timeRange }
  })
  return data
}

export async function listEvents(params: TokenRiskEventQuery): Promise<PaginatedResponse<TokenRiskEvent>> {
  const { data } = await apiClient.get<PaginatedResponse<TokenRiskEvent>>('/admin/token-risks/events', { params })
  return data
}

export async function getEvent(id: number): Promise<TokenRiskEventDetail> {
  const { data } = await apiClient.get<TokenRiskEventDetail>(`/admin/token-risks/events/${id}`)
  return data
}

export async function createAction(id: number, payload: TokenRiskActionRequest): Promise<TokenRiskAction> {
  const { data } = await apiClient.post<TokenRiskAction>(`/admin/token-risks/events/${id}/actions`, payload)
  return data
}

export async function backfill(timeRange = '24h'): Promise<{ ingested: number }> {
  const { data } = await apiClient.post<{ ingested: number }>('/admin/token-risks/backfill', null, {
    params: { time_range: timeRange }
  })
  return data
}

export async function listWatchlist(active = true): Promise<{ items: TokenRiskWatchlistItem[] }> {
  const { data } = await apiClient.get<{ items: TokenRiskWatchlistItem[] }>('/admin/token-risks/watchlist', {
    params: { active }
  })
  return data
}

export async function addWatchlist(payload: {
  subject_type: string
  subject_value: string
  reason?: string
}): Promise<TokenRiskWatchlistItem> {
  const { data } = await apiClient.post<TokenRiskWatchlistItem>('/admin/token-risks/watchlist', payload)
  return data
}

export async function removeWatchlist(id: number): Promise<{ ok: boolean }> {
  const { data } = await apiClient.delete<{ ok: boolean }>(`/admin/token-risks/watchlist/${id}`)
  return data
}

export const tokenRiskAPI = {
  getSummary,
  listEvents,
  getEvent,
  createAction,
  backfill,
  listWatchlist,
  addWatchlist,
  removeWatchlist
}

export default tokenRiskAPI
