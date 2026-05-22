import { describe, expect, it } from 'vitest'
import { convertPayload, maskSensitive } from '@/utils/formatConverter'

const options = {
  inputFormat: 'gpt-session' as const,
  outputFormat: 'sub2api' as const,
  platform: 'openai',
  accountType: 'oauth',
  concurrency: 3,
  priority: 50,
  nameSource: 'email' as const,
  namePrefix: 'acc'
}

describe('formatConverter', () => {
  it('converts GPT Session JSON to Sub2API payload', () => {
    const result = convertPayload([{ filename: 'session.json', text: '{"email":"a@example.com","access_token":"secret-token"}' }], options)

    expect(result.accountCount).toBe(1)
    expect((result.output as any).type).toBe('sub2api-data')
    expect((result.output as any).accounts[0].name).toBe('a@example.com')
  })

  it('converts GPT Session JSON to CPA output', () => {
    const result = convertPayload(
      [{ filename: 'session.json', text: '[{"user":{"email":"b@example.com","id":"u1"},"refresh_token":"r"}]' }],
      { ...options, outputFormat: 'cpa' }
    )

    expect(result.accountCount).toBe(1)
    expect((result.output as any[])[0].email).toBe('b@example.com')
    expect((result.output as any[])[0].account_id).toBe('u1')
  })

  it('rejects invalid JSON', () => {
    expect(() => convertPayload([{ filename: 'bad.json', text: '{' }], options)).toThrow('JSON 解析失败')
  })

  it('masks sensitive fields recursively', () => {
    expect(maskSensitive({ credentials: { access_token: 'abcdefghijklmnop' } })).toEqual({
      credentials: { access_token: 'abcd...mnop' }
    })
  })
})
