import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAppStore } from '@/stores/app'

vi.mock('@/api/admin/system', () => ({
  checkUpdates: vi.fn(),
}))

vi.mock('@/api/auth', () => ({
  getPublicSettings: vi.fn(),
}))

describe('useAppStore UI 模式与主题联动', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    document.documentElement.className = ''
    delete document.documentElement.dataset.uiMode
  })

  it('切换到 Gundam 时强制深色并保存原主题', () => {
    localStorage.setItem('theme', 'light')
    const store = useAppStore()

    store.setUIMode('gundam')

    expect(store.uiMode).toBe('gundam')
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(localStorage.getItem('sub2api_theme_before_gundam')).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('从 Gundam 切回官方模式时恢复进入前主题', () => {
    localStorage.setItem('theme', 'light')
    const store = useAppStore()

    store.setUIMode('gundam')
    store.setUIMode('official')

    expect(store.uiMode).toBe('official')
    expect(localStorage.getItem('theme')).toBe('light')
    expect(localStorage.getItem('sub2api_theme_before_gundam')).toBeNull()
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })

  it('历史 Gundam 缓存启动时也保持深色', () => {
    localStorage.setItem('theme', 'light')
    localStorage.setItem('sub2api_ui_mode', 'gundam')
    const store = useAppStore()

    store.initUIMode()

    expect(store.uiMode).toBe('gundam')
    expect(localStorage.getItem('theme')).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })
})
