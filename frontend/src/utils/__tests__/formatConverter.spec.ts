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

function jwtWithPayload(payload: Record<string, unknown>): string {
  const encode = (value: unknown) => Buffer.from(JSON.stringify(value)).toString('base64url')
  return `${encode({ alg: 'none', typ: 'JWT' })}.${encode(payload)}.sig`
}

describe('formatConverter', () => {
  it('converts ChatGPT Web session to Sub2API with access token expiry', () => {
    const accessToken = jwtWithPayload({
      exp: 1780473960,
      'https://api.openai.com/auth': {
        chatgpt_account_id: 'chatgpt-account-1',
        chatgpt_plan_type: 'plus',
        chatgpt_user_id: 'user-1'
      }
    })

    const result = convertPayload(
      [{
        filename: 'session.json',
        text: JSON.stringify({
          user: { email: 'mark@example.com' },
          account: { id: 'fallback-account', planType: 'plus' },
          accessToken,
          sessionToken: 'session-token'
        })
      }],
      options
    )

    const output = result.output as any
    expect(result.accountCount).toBe(1)
    expect(output.type).toBe('sub2api-data')
    expect(output.accounts[0].expires_at).toBe(1780473960)
    expect(output.accounts[0].auto_pause_on_expired).toBe(true)
    expect(output.accounts[0].credentials.access_token).toBe(accessToken)
    expect(output.accounts[0].credentials.chatgpt_account_id).toBe('chatgpt-account-1')
    expect(output.accounts[0].credentials.chatgpt_user_id).toBe('user-1')
    expect(output.accounts[0].credentials.email).toBe('mark@example.com')
  })

  it('converts GPT Session JSON to CPA and generates a parseable synthetic id_token', () => {
    const result = convertPayload(
      [{
        filename: 'session.json',
        text: JSON.stringify({
          user: { id: 'user-test', email: 'mark@example.com' },
          expires: '2026-08-06T14:29:36.155Z',
          account: { id: '00000000-0000-4000-9000-000000000000', planType: 'plus' },
          accessToken: 'access-token',
          sessionToken: 'session-token'
        })
      }],
      { ...options, outputFormat: 'cpa' }
    )

    const cpa = result.output as any
    expect(cpa.type).toBe('codex')
    expect(cpa.account_id).toBe('00000000-0000-4000-9000-000000000000')
    expect(cpa.chatgpt_account_id).toBe('00000000-0000-4000-9000-000000000000')
    expect(cpa.email).toBe('mark@example.com')
    expect(cpa.access_token).toBe('access-token')
    expect(cpa.session_token).toBe('session-token')
    expect(cpa.id_token_synthetic).toBe(true)
    expect(String(cpa.id_token).split('.')).toHaveLength(3)
  })

  it('converts 9router/Codex OAuth JSON to CPA', () => {
    const result = convertPayload(
      [{
        filename: '9router.json',
        text: JSON.stringify({
          provider: 'codex',
          authType: 'oauth',
          accessToken: 'access-token',
          refreshToken: 'refresh-token',
          expiresAt: '2026-08-06T14:29:36.155Z',
          providerSpecificData: {
            chatgptAccountId: 'chatgpt-account-9r',
            chatgptPlanType: 'plus'
          },
          email: 'nine@example.com'
        })
      }],
      { ...options, outputFormat: 'cpa' }
    )

    const cpa = result.output as any
    expect(cpa.account_id).toBe('chatgpt-account-9r')
    expect(cpa.chatgpt_plan_type).toBe('plus')
    expect(cpa.refresh_token).toBe('refresh-token')
  })

  it('collects nested session-like objects in batch JSON', () => {
    const result = convertPayload(
      [{
        filename: 'batch.json',
        text: JSON.stringify({
          accounts: [
            { user: { email: 'a@example.com' }, accessToken: 'token-a' },
            { meta: { label: 'B' }, tokens: { access_token: 'token-b', chatgpt_account_id: 'acct-b' } }
          ]
        })
      }],
      options
    )

    expect(result.accountCount).toBe(2)
    expect((result.output as any).accounts[1].credentials.chatgpt_account_id).toBe('acct-b')
  })

  it('rejects invalid JSON and missing accessToken', () => {
    expect(() => convertPayload([{ filename: 'bad.json', text: '{' }], options)).toThrow('JSON 解析失败')
    expect(() => convertPayload([{ filename: 'empty.json', text: '{"user":{"email":"a@example.com"}}' }], options)).toThrow('accessToken')
  })

  it('masks sensitive fields recursively including camelCase token keys', () => {
    expect(maskSensitive({ credentials: { accessToken: 'abcdefghijklmnop', access_token: 'qrstuvwxyz123456' } })).toEqual({
      credentials: { accessToken: 'abcd...mnop', access_token: 'qrst...3456' }
    })
  })
})
