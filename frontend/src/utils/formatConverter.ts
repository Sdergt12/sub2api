import { buildSub2APIDataFromCPA, type CPABuildOptions, type CPAFileInput } from './cpaImport'

export type ConverterInputFormat = 'gpt-session' | 'cpa' | 'sub2api'
export type ConverterOutputFormat = 'cpa' | 'sub2api'

export interface ConversionInput {
  filename: string
  text: string
}

export interface ConversionOptions extends CPABuildOptions {
  inputFormat: ConverterInputFormat
  outputFormat: ConverterOutputFormat
}

export interface ConversionIssue {
  filename: string
  path: string
  reason: string
}

export interface ConversionPreviewAccount {
  name: string
  email?: string
  account_id?: string
  plan_type?: string
  expires_at?: string
  source: string
}

export interface ConversionResult {
  output: unknown
  accountCount: number
  fileCount: number
  sensitive: boolean
  detectedFormat: string
  previewAccounts: ConversionPreviewAccount[]
  issues: ConversionIssue[]
}

interface SessionSource {
  value: Record<string, unknown>
  filename: string
  path: string
}

interface ConvertedSession {
  name: string
  email?: string
  accountId?: string
  planType?: string
  expiresAt?: string
  cpa: Record<string, unknown>
  sub2apiAccount: Record<string, unknown>
}

const sensitiveKeys = new Set([
  'access_token',
  'accesstoken',
  'refresh_token',
  'refreshtoken',
  'id_token',
  'idtoken',
  'session_token',
  'sessiontoken',
  'api_key',
  'apikey',
  'key',
  'token',
  'authorization',
  'cookie',
  'password'
])

function parseJSON(text: string, filename: string): unknown {
  try {
    return JSON.parse(text)
  } catch {
    throw new Error(`${filename || '输入内容'} JSON 解析失败`)
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function normalizeArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [value]
}

function getPathValue(source: unknown, path: string): unknown {
  return path.split('.').reduce<unknown>((current, key) => {
    if (!isPlainObject(current)) return undefined
    return current[key]
  }, source)
}

function firstNonEmpty(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value === 'string' && value.trim() !== '') return value.trim()
    if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  }
  return undefined
}

function pickString(record: Record<string, unknown>, paths: string[]): string | undefined {
  return firstNonEmpty(...paths.map((path) => getPathValue(record, path)))
}

function decodeBase64Url(value: string): string {
  const normalized = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, '=')
  if (typeof atob === 'function') {
    const binary = atob(padded)
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    return new TextDecoder().decode(bytes)
  }
  const bufferCtor = (globalThis as unknown as { Buffer?: { from(input: string, encoding: string): { toString(encoding: string): string } } }).Buffer
  if (bufferCtor) return bufferCtor.from(padded, 'base64').toString('utf8')
  throw new Error('当前运行环境不支持 base64url 解码')
}

function encodeBase64UrlJson(value: unknown): string {
  const json = JSON.stringify(value)
  if (typeof btoa === 'function') {
    const bytes = new TextEncoder().encode(json)
    let binary = ''
    for (let index = 0; index < bytes.length; index += 0x8000) {
      binary += String.fromCharCode(...bytes.subarray(index, index + 0x8000))
    }
    return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '')
  }
  const bufferCtor = (globalThis as unknown as { Buffer?: { from(input: string): { toString(encoding: string): string } } }).Buffer
  if (bufferCtor) return bufferCtor.from(json).toString('base64url')
  throw new Error('当前运行环境不支持 base64url 编码')
}

function parseJwtPayload(token?: string): Record<string, unknown> | undefined {
  if (!token) return undefined
  const parts = token.split('.')
  if (parts.length < 2 || !parts[1]) return undefined
  try {
    const parsed = JSON.parse(decodeBase64Url(parts[1]))
    return isPlainObject(parsed) ? parsed : undefined
  } catch {
    return undefined
  }
}

function getOpenAIAuthSection(payload?: Record<string, unknown>): Record<string, unknown> {
  const auth = payload?.['https://api.openai.com/auth']
  return isPlainObject(auth) ? auth : {}
}

function getOpenAIProfileSection(payload?: Record<string, unknown>): Record<string, unknown> {
  const profile = payload?.['https://api.openai.com/profile']
  return isPlainObject(profile) ? profile : {}
}

function normalizeTimestamp(value: unknown): string | undefined {
  if (value === undefined || value === null || value === '') return undefined
  if (typeof value === 'number' && Number.isFinite(value)) {
    const milliseconds = value > 1e11 ? value : value * 1000
    const date = new Date(milliseconds)
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString()
  }
  if (typeof value === 'string') {
    const numeric = Number(value)
    if (Number.isFinite(numeric) && value.trim() !== '') return normalizeTimestamp(numeric)
    const date = new Date(value)
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString()
  }
  return undefined
}

function timestampFromUnixSeconds(value: unknown): string | undefined {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return undefined
  return new Date(Math.trunc(numeric) * 1000).toISOString()
}

function unixSecondsFromJwtExp(value: unknown): number | undefined {
  const numeric = Number(value)
  if (!Number.isFinite(numeric) || numeric <= 0) return undefined
  return Math.trunc(numeric)
}

function epochSecondsFromValue(value?: string): number {
  if (!value) return 0
  const parsed = Date.parse(value)
  if (Number.isNaN(parsed)) return 0
  return Math.trunc(parsed / 1000)
}

function getExpiresIn(expiresAt?: string, now = new Date()): number | undefined {
  if (!expiresAt) return undefined
  const expiresMs = Date.parse(expiresAt)
  if (Number.isNaN(expiresMs)) return undefined
  return Math.max(0, Math.trunc((expiresMs - now.getTime()) / 1000))
}

function toEmailKey(email?: string): string | undefined {
  return email?.trim().toLowerCase().replace(/[^a-z0-9._-]+/g, '_') || undefined
}

function stripUnavailable<T>(value: T): T {
  if (Array.isArray(value)) {
    return value.map(stripUnavailable).filter((item) => item !== undefined) as T
  }
  if (!isPlainObject(value)) return value
  return Object.fromEntries(
    Object.entries(value)
      .map(([key, entry]) => [key, stripUnavailable(entry)])
      .filter(([, entry]) => entry !== undefined && entry !== null && entry !== '')
  ) as T
}

function buildSyntheticCodexIdToken(
  email: string | undefined,
  accountId: string | undefined,
  planType: string | undefined,
  userId: string | undefined,
  expiresAt: string | undefined
): string | undefined {
  if (!accountId) return undefined
  const now = Math.trunc(Date.now() / 1000)
  const authInfo: Record<string, unknown> = { chatgpt_account_id: accountId }
  if (planType) authInfo.chatgpt_plan_type = planType
  if (userId) {
    authInfo.chatgpt_user_id = userId
    authInfo.user_id = userId
  }
  const payload = stripUnavailable({
    iat: now,
    exp: epochSecondsFromValue(expiresAt) || now + 90 * 24 * 60 * 60,
    email,
    'https://api.openai.com/auth': authInfo,
    'https://api.openai.com/profile': { email }
  })
  return `${encodeBase64UrlJson({ alg: 'none', typ: 'JWT', cpa_synthetic: true })}.${encodeBase64UrlJson(payload)}.synthetic`
}

function tokenFromRecord(record: Record<string, unknown>): string | undefined {
  return pickString(record, [
    'accessToken',
    'access_token',
    'tokens.accessToken',
    'tokens.access_token',
    'token.accessToken',
    'token.access_token',
    'credentials.accessToken',
    'credentials.access_token'
  ])
}

function collectSessionLikeObjects(value: unknown, filename: string, basePath = '$'): SessionSource[] {
  const found: SessionSource[] = []
  const visited = new WeakSet<object>()

  function visit(item: unknown, path: string) {
    if (Array.isArray(item)) {
      item.forEach((child, index) => visit(child, `${path}[${index}]`))
      return
    }
    if (!isPlainObject(item)) return
    if (visited.has(item)) return
    visited.add(item)

    if (tokenFromRecord(item)) {
      found.push({ value: item, filename, path })
      return
    }

    for (const [key, child] of Object.entries(item)) {
      if (['accessToken', 'access_token', 'sessionToken', 'session_token'].includes(key)) continue
      visit(child, `${path}.${key}`)
    }
  }

  visit(value, basePath)
  return found
}

export function convertSession(record: Record<string, unknown>, options: { sourceName?: string; now?: Date } = {}): ConvertedSession {
  const accessToken = tokenFromRecord(record)
  if (!accessToken) throw new Error('缺少 accessToken')

  const sessionToken = pickString(record, [
    'sessionToken',
    'session_token',
    'tokens.sessionToken',
    'tokens.session_token',
    'token.sessionToken',
    'token.session_token',
    'credentials.session_token'
  ])
  const refreshToken = pickString(record, [
    'refreshToken',
    'refresh_token',
    'tokens.refreshToken',
    'tokens.refresh_token',
    'token.refreshToken',
    'token.refresh_token',
    'credentials.refresh_token'
  ])
  const inputIdToken = pickString(record, [
    'idToken',
    'id_token',
    'tokens.idToken',
    'tokens.id_token',
    'token.idToken',
    'token.id_token',
    'credentials.id_token'
  ])

  const payload = parseJwtPayload(accessToken)
  const idPayload = parseJwtPayload(inputIdToken)
  const auth = getOpenAIAuthSection(payload)
  const idAuth = getOpenAIAuthSection(idPayload)
  const profile = getOpenAIProfileSection(payload)
  const accessTokenExpiresAt = unixSecondsFromJwtExp(payload?.exp)
  const expiresAt =
    timestampFromUnixSeconds(payload?.exp) ||
    normalizeTimestamp(record.expires) ||
    normalizeTimestamp(record.expiresAt) ||
    normalizeTimestamp(record.expired) ||
    normalizeTimestamp(record.expires_at)

  const email = firstNonEmpty(
    getPathValue(record, 'user.email'),
    record.email,
    getPathValue(record, 'account.email'),
    getPathValue(record, 'meta.email'),
    getPathValue(record, 'credentials.email'),
    profile.email,
    idPayload?.email,
    payload?.email
  )
  const accountId = firstNonEmpty(
    getPathValue(record, 'account.id'),
    record.accountId,
    record.account_id,
    record.id,
    record.chatgptAccountId,
    record.chatgpt_account_id,
    getPathValue(record, 'meta.chatgptAccountId'),
    getPathValue(record, 'meta.chatgpt_account_id'),
    getPathValue(record, 'tokens.accountId'),
    getPathValue(record, 'tokens.account_id'),
    getPathValue(record, 'tokens.chatgptAccountId'),
    getPathValue(record, 'tokens.chatgpt_account_id'),
    getPathValue(record, 'providerSpecificData.chatgptAccountId'),
    getPathValue(record, 'providerSpecificData.chatgpt_account_id'),
    getPathValue(record, 'credentials.chatgpt_account_id'),
    auth.chatgpt_account_id,
    idAuth.chatgpt_account_id
  )
  const chatgptAccountId = firstNonEmpty(
    record.chatgptAccountId,
    record.chatgpt_account_id,
    getPathValue(record, 'providerSpecificData.chatgptAccountId'),
    getPathValue(record, 'providerSpecificData.chatgpt_account_id'),
    getPathValue(record, 'credentials.chatgpt_account_id'),
    auth.chatgpt_account_id,
    idAuth.chatgpt_account_id,
    accountId
  )
  const userId = firstNonEmpty(
    getPathValue(record, 'user.id'),
    record.userId,
    record.user_id,
    record.chatgptUserId,
    record.chatgpt_user_id,
    getPathValue(record, 'providerSpecificData.chatgptUserId'),
    getPathValue(record, 'providerSpecificData.chatgpt_user_id'),
    auth.chatgpt_user_id,
    auth.user_id,
    idAuth.chatgpt_user_id,
    idAuth.user_id
  )
  const planType = firstNonEmpty(
    getPathValue(record, 'account.planType'),
    getPathValue(record, 'account.plan_type'),
    record.planType,
    record.plan_type,
    getPathValue(record, 'providerSpecificData.chatgptPlanType'),
    getPathValue(record, 'providerSpecificData.chatgpt_plan_type'),
    getPathValue(record, 'credentials.plan_type'),
    auth.chatgpt_plan_type,
    idAuth.chatgpt_plan_type
  )
  const sourceName = firstNonEmpty(options.sourceName, 'pasted-json') || 'pasted-json'
  const name = firstNonEmpty(record.name, getPathValue(record, 'meta.label'), email, sourceName, 'ChatGPT Account') || 'ChatGPT Account'
  const syntheticIdToken = inputIdToken ? undefined : buildSyntheticCodexIdToken(email, accountId, planType, userId, expiresAt)
  const idToken = inputIdToken || syntheticIdToken
  const now = options.now || new Date()
  const exportedAt = now.toISOString()
  const expiresIn = getExpiresIn(expiresAt, now)

  const cpa = stripUnavailable({
    type: 'codex',
    account_id: accountId,
    chatgpt_account_id: chatgptAccountId || accountId,
    email,
    name,
    plan_type: planType,
    chatgpt_plan_type: planType,
    id_token: idToken,
    id_token_synthetic: Boolean(syntheticIdToken) || undefined,
    access_token: accessToken,
    refresh_token: refreshToken || '',
    session_token: sessionToken,
    last_refresh: exportedAt,
    expired: expiresAt,
    disabled: record.disabled === true || undefined
  })

  const sub2apiAccount = stripUnavailable({
    name,
    platform: 'openai',
    type: 'oauth',
    expires_at: accessTokenExpiresAt,
    auto_pause_on_expired: true,
    concurrency: 10,
    priority: 1,
    credentials: {
      access_token: accessToken,
      refresh_token: refreshToken,
      session_token: sessionToken,
      chatgpt_account_id: chatgptAccountId || accountId,
      chatgpt_user_id: userId,
      email,
      expires_at: expiresAt,
      expires_in: expiresIn,
      plan_type: planType
    },
    extra: {
      email_key: toEmailKey(email),
      source: 'gpt_session_converter',
      source_name: sourceName,
      last_refresh: exportedAt
    }
  })

  return { name, email, accountId: chatgptAccountId || accountId, planType, expiresAt, cpa, sub2apiAccount }
}

function buildSub2apiDocument(converted: ConvertedSession[], now = new Date()): Record<string, unknown> {
  return {
    type: 'sub2api-data',
    version: 1,
    exported_at: now.toISOString().replace(/\.\d{3}Z$/, 'Z'),
    proxies: [],
    accounts: converted.map((item) => item.sub2apiAccount)
  }
}

function convertSessionSources(sources: SessionSource[], now = new Date()): { converted: ConvertedSession[]; issues: ConversionIssue[] } {
  const converted: ConvertedSession[] = []
  const issues: ConversionIssue[] = []
  for (const source of sources) {
    try {
      converted.push(convertSession(source.value, { sourceName: source.filename, now }))
    } catch (error) {
      issues.push({
        filename: source.filename,
        path: source.path,
        reason: error instanceof Error ? error.message : '转换失败'
      })
    }
  }
  return { converted, issues }
}

function previewFromConverted(converted: ConvertedSession[]): ConversionPreviewAccount[] {
  return converted.map((item) => ({
    name: item.name,
    email: item.email,
    account_id: item.accountId,
    plan_type: item.planType,
    expires_at: item.expiresAt,
    source: 'GPT Session'
  }))
}

export function convertPayload(inputs: ConversionInput[], options: ConversionOptions): ConversionResult {
  if (inputs.length === 0) throw new Error('请先提供输入内容或上传文件')
  if (options.outputFormat === 'cpa' && options.inputFormat === 'sub2api') {
    throw new Error('Sub2API 到 CPA 不是安全的无损转换，本工具不支持')
  }
  if (options.concurrency < 0) throw new Error('concurrency must be >= 0')
  if (options.priority < 0) throw new Error('priority must be >= 0')

  const parsed = inputs.map((item) => ({
    filename: item.filename || 'input.json',
    data: parseJSON(item.text, item.filename || 'input.json')
  }))

  if (options.inputFormat === 'sub2api') {
    const payload = parsed[0].data
    const record = isPlainObject(payload) ? payload : {}
    const accounts = Array.isArray(record.accounts) ? record.accounts.length : 0
    return {
      output: record,
      accountCount: accounts,
      fileCount: parsed.length,
      sensitive: containsSensitiveValue(record),
      detectedFormat: 'Sub2API',
      previewAccounts: [],
      issues: []
    }
  }

  if (options.inputFormat === 'cpa') {
    if (options.outputFormat === 'cpa') {
      const output = parsed.flatMap((item) => normalizeArray(item.data))
      return {
        output: output.length === 1 ? output[0] : output,
        accountCount: output.length,
        fileCount: parsed.length,
        sensitive: containsSensitiveValue(output),
        detectedFormat: 'CPA',
        previewAccounts: output.map((item) => {
          const record = isPlainObject(item) ? item : {}
          return {
            name: firstNonEmpty(record.name, record.email, 'CPA Account') || 'CPA Account',
            email: firstNonEmpty(record.email),
            account_id: firstNonEmpty(record.account_id, record.chatgpt_account_id),
            plan_type: firstNonEmpty(record.plan_type, record.chatgpt_plan_type),
            expires_at: firstNonEmpty(record.expired),
            source: 'CPA'
          }
        }),
        issues: []
      }
    }
    const output = buildSub2APIDataFromCPA(
      parsed.map((item) => ({ filename: item.filename, data: item.data })) as CPAFileInput[],
      options
    )
    return {
      output,
      accountCount: output.accounts.length,
      fileCount: parsed.length,
      sensitive: containsSensitiveValue(output),
      detectedFormat: 'CPA',
      previewAccounts: output.accounts.map((account) => ({
        name: account.name,
        email: firstNonEmpty(account.credentials.email),
        account_id: firstNonEmpty(account.credentials.account_id, account.credentials.chatgpt_account_id),
        source: 'CPA'
      })),
      issues: []
    }
  }

  const sources = parsed.flatMap((item) => collectSessionLikeObjects(item.data, item.filename))
  if (sources.length === 0) throw new Error('未找到包含 accessToken/access_token 的 GPT Session 或 Codex OAuth 记录')

  const now = new Date()
  const { converted, issues } = convertSessionSources(sources, now)
  if (converted.length === 0) {
    throw new Error(issues[0]?.reason || '没有可转换的账号')
  }

  const output =
    options.outputFormat === 'cpa'
      ? converted.length === 1
        ? converted[0].cpa
        : converted.map((item) => item.cpa)
      : buildSub2apiDocument(converted, now)

  return {
    output,
    accountCount: converted.length,
    fileCount: parsed.length,
    sensitive: containsSensitiveValue(output),
    detectedFormat: 'GPT Session / Codex OAuth',
    previewAccounts: previewFromConverted(converted),
    issues
  }
}

export function maskSensitive(input: unknown): unknown {
  if (Array.isArray(input)) return input.map(maskSensitive)
  if (!input || typeof input !== 'object') return input
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
    if (sensitiveKeys.has(key.toLowerCase())) out[key] = maskValue(value)
    else out[key] = maskSensitive(value)
  }
  return out
}

export function containsSensitiveValue(input: unknown): boolean {
  if (Array.isArray(input)) return input.some(containsSensitiveValue)
  if (!input || typeof input !== 'object') return false
  return Object.entries(input as Record<string, unknown>).some(([key, value]) => sensitiveKeys.has(key.toLowerCase()) || containsSensitiveValue(value))
}

function maskValue(value: unknown): string {
  const raw = String(value ?? '')
  if (!raw) return ''
  if (raw.length <= 10) return '***'
  return `${raw.slice(0, 4)}...${raw.slice(-4)}`
}
