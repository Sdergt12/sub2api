import { buildSub2APIDataFromCPA, type CPABuildOptions, type CPAFileInput, type CPASub2APIDataPayload } from './cpaImport'

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

export interface ConversionResult {
  output: unknown
  accountCount: number
  fileCount: number
  sensitive: boolean
}

const sensitiveKeys = new Set([
  'access_token',
  'refresh_token',
  'id_token',
  'session_token',
  'api_key',
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
    throw new Error(`${filename || 'input'} JSON 解析失败`)
  }
}

function normalizeArray(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [value]
}

function normalizeRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === 'object' && !Array.isArray(value)) return value as Record<string, unknown>
  return { raw_value: value }
}

export function convertGPTSessionToCPA(input: unknown): unknown[] {
  const records = normalizeArray(input)
  return records.map((item) => {
    const record = normalizeRecord(item)
    const nestedUser = normalizeRecord(record.user)
    const email = String(record.email ?? nestedUser.email ?? '').trim()
    const accountID = String(record.account_id ?? record.user_id ?? nestedUser.id ?? '').trim()
    // GPT Session 原始字段直接作为 credentials 保留，避免误删兼容字段。
    return {
      ...record,
      email: email || undefined,
      account_id: accountID || undefined
    }
  })
}

export function convertCPAInputsToSub2API(files: CPAFileInput[], options: CPABuildOptions): CPASub2APIDataPayload {
  return buildSub2APIDataFromCPA(files, options)
}

export function convertPayload(inputs: ConversionInput[], options: ConversionOptions): ConversionResult {
  if (inputs.length === 0) {
    throw new Error('请先提供输入内容或上传文件')
  }
  if (options.outputFormat === 'cpa' && options.inputFormat === 'sub2api') {
    throw new Error('Sub2API 到 CPA 不是安全的无损转换，本工具不支持')
  }

  const parsed = inputs.map((item) => ({
    filename: item.filename || 'input.json',
    data: parseJSON(item.text, item.filename || 'input.json')
  }))

  if (options.inputFormat === 'sub2api') {
    const payload = normalizeRecord(parsed[0].data)
    const accounts = Array.isArray(payload.accounts) ? payload.accounts.length : 0
    return { output: payload, accountCount: accounts, fileCount: parsed.length, sensitive: containsSensitiveValue(payload) }
  }

  const cpaFiles =
    options.inputFormat === 'gpt-session'
      ? parsed.map((item) => ({ filename: item.filename, data: convertGPTSessionToCPA(item.data) }))
      : parsed

  if (options.outputFormat === 'cpa') {
    const output = cpaFiles.flatMap((item) => normalizeArray(item.data))
    return { output, accountCount: output.length, fileCount: cpaFiles.length, sensitive: containsSensitiveValue(output) }
  }

  const output = convertCPAInputsToSub2API(cpaFiles, options)
  return { output, accountCount: output.accounts.length, fileCount: cpaFiles.length, sensitive: containsSensitiveValue(output) }
}

export function maskSensitive(input: unknown): unknown {
  if (Array.isArray(input)) return input.map(maskSensitive)
  if (!input || typeof input !== 'object') return input
  const out: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(input as Record<string, unknown>)) {
    if (sensitiveKeys.has(key.toLowerCase())) {
      out[key] = maskValue(value)
    } else {
      out[key] = maskSensitive(value)
    }
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
