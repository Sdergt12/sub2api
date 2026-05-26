export type CPANameSource = 'email' | 'filename' | 'index'

export interface CPAFileInput {
  filename: string
  data: unknown
}

export interface CPASub2APIDataAccount {
  name: string
  platform: string
  type: string
  credentials: Record<string, unknown>
  concurrency: number
  priority: number
  group_ids?: number[]
}

export interface CPASub2APIDataPayload {
  type: 'sub2api-data'
  version: 1
  exported_at: string
  proxies: []
  accounts: CPASub2APIDataAccount[]
}

export interface CPABuildOptions {
  platform: string
  accountType: string
  concurrency: number
  priority: number
  nameSource: CPANameSource
  namePrefix: string
}

const defaultNamePrefix = 'acc'

function normalizeRecords(data: unknown): unknown[] {
  return Array.isArray(data) ? data : [data]
}

function stripExtension(filename: string): string {
  const trimmed = filename.trim()
  const dot = trimmed.lastIndexOf('.')
  return dot > 0 ? trimmed.slice(0, dot) : trimmed || 'account'
}

function chooseName(
  credentials: Record<string, unknown>,
  filename: string,
  index: number,
  nameSource: CPANameSource,
  namePrefix: string
): string {
  if (nameSource === 'index') {
    return `${namePrefix || defaultNamePrefix}-${String(index).padStart(3, '0')}`
  }

  if (nameSource === 'email') {
    const email = String(credentials.email ?? '').trim()
    if (email) return email

    const accountID = String(credentials.account_id ?? '').trim()
    if (accountID) return accountID
  }

  return stripExtension(filename)
}

function dedupeName(name: string, used: Map<string, number>): string {
  const current = (used.get(name) ?? 0) + 1
  used.set(name, current)
  return current === 1 ? name : `${name}-${current}`
}

function normalizeCredentials(item: unknown): Record<string, unknown> {
  if (item && typeof item === 'object' && !Array.isArray(item)) {
    return item as Record<string, unknown>
  }
  return { raw_value: item }
}

export function buildSub2APIDataFromCPA(files: CPAFileInput[], options: CPABuildOptions): CPASub2APIDataPayload {
  if (options.concurrency < 0) {
    throw new Error('concurrency must be >= 0')
  }
  if (options.priority < 0) {
    throw new Error('priority must be >= 0')
  }

  const accounts: CPASub2APIDataAccount[] = []
  const usedNames = new Map<string, number>()
  let counter = 1

  for (const file of files) {
    for (const item of normalizeRecords(file.data)) {
      const credentials = normalizeCredentials(item)
      const baseName = chooseName(
        credentials,
        file.filename,
        counter,
        options.nameSource,
        options.namePrefix
      )
      const name = dedupeName(baseName, usedNames)

      accounts.push({
        name,
        platform: options.platform,
        type: options.accountType,
        credentials,
        concurrency: options.concurrency,
        priority: options.priority,
      })
      counter += 1
    }
  }

  return {
    type: 'sub2api-data',
    version: 1,
    exported_at: new Date().toISOString().replace(/\.\d{3}Z$/, 'Z'),
    proxies: [],
    accounts,
  }
}
