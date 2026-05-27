import { describe, expect, it } from 'vitest'
import { buildSub2APIDataFromCPA } from '@/utils/cpaImport'

const baseOptions = {
  platform: 'openai' as const,
  accountType: 'oauth' as const,
  concurrency: 3,
  priority: 50,
  nameSource: 'email' as const,
  namePrefix: 'acc',
}

describe('buildSub2APIDataFromCPA', () => {
  it('converts a single object into one sub2api account', () => {
    const payload = buildSub2APIDataFromCPA(
      [{ filename: 'token_a.json', data: { email: 'a@example.com', access_token: 'redacted' } }],
      baseOptions
    )

    expect(payload.type).toBe('sub2api-data')
    expect(payload.version).toBe(1)
    expect(payload.proxies).toEqual([])
    expect(payload.accounts).toHaveLength(1)
    expect(payload.accounts[0]).toMatchObject({
      name: 'a@example.com',
      platform: 'openai',
      type: 'oauth',
      concurrency: 3,
      priority: 50,
      credentials: { email: 'a@example.com', access_token: 'redacted' },
    })
  })

  it('converts arrays and deduplicates names', () => {
    const payload = buildSub2APIDataFromCPA(
      [
        {
          filename: 'batch.json',
          data: [
            { email: 'same@example.com' },
            { email: 'same@example.com' },
          ],
        },
      ],
      baseOptions
    )

    expect(payload.accounts.map((account) => account.name)).toEqual([
      'same@example.com',
      'same@example.com-2',
    ])
  })

  it('supports filename and index name sources', () => {
    const filenamePayload = buildSub2APIDataFromCPA(
      [{ filename: 'token_file.json', data: { token: 'a' } }],
      { ...baseOptions, nameSource: 'filename' }
    )
    const indexPayload = buildSub2APIDataFromCPA(
      [{ filename: 'token_file.json', data: [{ token: 'a' }, { token: 'b' }] }],
      { ...baseOptions, nameSource: 'index', namePrefix: 'cpa' }
    )

    expect(filenamePayload.accounts[0].name).toBe('token_file')
    expect(indexPayload.accounts.map((account) => account.name)).toEqual(['cpa-001', 'cpa-002'])
  })

  it('wraps non-object records as raw_value', () => {
    const payload = buildSub2APIDataFromCPA(
      [{ filename: 'raw.json', data: ['plain-token'] }],
      baseOptions
    )

    expect(payload.accounts[0].credentials).toEqual({ raw_value: 'plain-token' })
  })

  it('rejects negative concurrency and priority', () => {
    expect(() =>
      buildSub2APIDataFromCPA([{ filename: 'a.json', data: {} }], {
        ...baseOptions,
        concurrency: -1,
      })
    ).toThrow('concurrency must be >= 0')

    expect(() =>
      buildSub2APIDataFromCPA([{ filename: 'a.json', data: {} }], {
        ...baseOptions,
        priority: -1,
      })
    ).toThrow('priority must be >= 0')
  })
})
