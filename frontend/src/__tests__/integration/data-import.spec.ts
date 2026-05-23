import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import ImportDataModal from '@/components/admin/account/ImportDataModal.vue'
import { adminAPI } from '@/api/admin'

const showError = vi.fn()
const showSuccess = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      getAll: vi.fn()
    },
    accounts: {
      importData: vi.fn()
    }
  }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

describe('ImportDataModal', () => {
  beforeEach(() => {
    showError.mockReset()
    showSuccess.mockReset()
    vi.mocked(adminAPI.accounts.importData).mockReset()
    vi.mocked(adminAPI.groups.getAll).mockResolvedValue([])
  })

  it('未选择文件时提示错误', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.find('form').trigger('submit')
    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportSelectFile')
  })

  it('无效 JSON 时提示解析失败', async () => {
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const input = wrapper.find('input[type="file"]')
    const file = new File(['invalid json'], 'data.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve('invalid json')
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()

    expect(showError).toHaveBeenCalledWith('admin.accounts.dataImportParseFailed')
  })

  it('sub2api JSON 模式保持原导入路径', async () => {
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
    })
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    const payload = {
      type: 'sub2api-data',
      version: 1,
      exported_at: '2026-05-22T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'a', platform: 'openai', type: 'oauth', credentials: { token: 'x' }, concurrency: 3, priority: 50 }],
    }
    const input = wrapper.find('input[type="file"]')
    const file = new File([JSON.stringify(payload)], 'sub2api.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify(payload))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    await Promise.resolve()

    expect(adminAPI.accounts.importData).toHaveBeenCalledWith({
      data: payload,
      skip_default_group_bind: true,
    })
    expect(showSuccess).toHaveBeenCalled()
  })

  it('CPA JSON 模式会转换后复用账号导入接口', async () => {
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 2,
      account_failed: 0,
    })
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await wrapper.findAll('button').find((button) => button.text().includes('admin.accounts.dataImportModeCpa'))?.trigger('click')

    const input = wrapper.find('input[type="file"]')
    const file = new File([JSON.stringify([{ email: 'a@example.com' }, { email: 'b@example.com' }])], 'tokens.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify([{ email: 'a@example.com' }, { email: 'b@example.com' }]))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    await Promise.resolve()

    const call = vi.mocked(adminAPI.accounts.importData).mock.calls[0][0]
    expect(call.data.type).toBe('sub2api-data')
    expect(call.data.proxies).toEqual([])
    expect(call.data.accounts.map((account) => account.name)).toEqual(['a@example.com', 'b@example.com'])
    expect(call.data.accounts.every((account) => account.group_ids === undefined)).toBe(true)
    expect(call.skip_default_group_bind).toBe(true)
  })

  it('导入时会将目标分组写入每个账号', async () => {
    vi.mocked(adminAPI.groups.getAll).mockResolvedValue([
      { id: 7, name: 'OpenAI 分组', platform: 'openai', status: 'active' } as any
    ])
    vi.mocked(adminAPI.accounts.importData).mockResolvedValue({
      proxy_created: 0,
      proxy_reused: 0,
      proxy_failed: 0,
      account_created: 1,
      account_failed: 0,
    })
    const wrapper = mount(ImportDataModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' }
        }
      }
    })

    await Promise.resolve()
    await Promise.resolve()
    await wrapper.find('input[type="checkbox"]').setValue(true)

    const payload = {
      type: 'sub2api-data',
      version: 1,
      exported_at: '2026-05-22T00:00:00Z',
      proxies: [],
      accounts: [{ name: 'a', platform: 'openai', type: 'oauth', credentials: { token: 'x' }, concurrency: 3, priority: 50 }],
    }
    const input = wrapper.find('input[type="file"]')
    const file = new File([JSON.stringify(payload)], 'sub2api.json', { type: 'application/json' })
    Object.defineProperty(file, 'text', {
      value: () => Promise.resolve(JSON.stringify(payload))
    })
    Object.defineProperty(input.element, 'files', {
      value: [file]
    })

    await input.trigger('change')
    await wrapper.find('form').trigger('submit')
    await Promise.resolve()
    await Promise.resolve()

    const call = vi.mocked(adminAPI.accounts.importData).mock.calls[0][0]
    expect(call.data.accounts[0].group_ids).toEqual([7])
    expect(call.skip_default_group_bind).toBe(true)
  })
})
