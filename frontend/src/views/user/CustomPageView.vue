<template>
  <AppLayout>
    <div class="custom-page-layout">
      <div class="card flex-1 min-h-0 overflow-hidden">
        <div v-if="loading" class="flex h-full items-center justify-center py-12">
          <div
            class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
          ></div>
        </div>

        <div
          v-else-if="!menuItem"
          class="flex h-full items-center justify-center p-10 text-center"
        >
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notFoundTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notFoundDesc') }}
            </p>
          </div>
        </div>

        <!-- Markdown mode with TOC -->
        <div v-else-if="isMarkdownMode" class="flex h-full overflow-hidden">
          <!-- TOC Sidebar -->
          <aside
            v-show="tocVisible"
            class="toc-sidebar"
          >
            <div class="toc-header">
              <span class="toc-title">目录</span>
              <button class="toc-close-btn" @click="tocVisible = false">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 18l-6-6 6-6"/></svg>
              </button>
            </div>
            <nav class="toc-nav">
              <a
                v-for="item in tocItems"
                :key="item.id"
                :href="'#' + item.id"
                class="toc-item"
                :class="[
                  `toc-level-${item.level}`,
                  { 'toc-active': activeHeadingId === item.id }
                ]"
                @click.prevent="scrollToHeading(item.id)"
              >
                {{ item.text }}
              </a>
            </nav>
          </aside>

          <!-- TOC Toggle Button (when collapsed) -->
          <button
            v-show="!tocVisible && tocItems.length > 0"
            class="toc-toggle-btn"
            @click="tocVisible = true"
          >
            <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12h18M3 6h18M3 18h18"/></svg>
            <span class="ml-1 text-xs">目录</span>
          </button>

          <!-- Content -->
          <div
            ref="markdownContainer"
            class="markdown-page-content flex-1 h-full overflow-auto p-6 md:p-10"
            v-html="renderedHtml"
            @scroll="onContentScroll"
          ></div>
        </div>

        <!-- URL not configured -->
        <div v-else-if="!isValidUrl" class="flex h-full items-center justify-center p-10 text-center">
          <div class="max-w-md">
            <div
              class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700"
            >
              <Icon name="link" size="lg" class="text-gray-400" />
            </div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('customPage.notConfiguredTitle') }}
            </h3>
            <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
              {{ t('customPage.notConfiguredDesc') }}
            </p>
          </div>
        </div>

        <!-- Iframe embed mode -->
        <div v-else class="custom-embed-shell">
          <section v-if="showGameCenterLeaderboard" class="game-leaderboard-panel">
            <div class="game-leaderboard-header">
              <div>
                <p class="game-leaderboard-kicker">Game Center</p>
                <h3 class="game-leaderboard-title">排行榜</h3>
              </div>
              <div class="game-leaderboard-tabs" role="group" aria-label="排行榜时间范围">
                <button
                  v-for="range in leaderboardRanges"
                  :key="range.value"
                  type="button"
                  class="game-leaderboard-tab"
                  :class="{ 'game-leaderboard-tab-active': leaderboardRange === range.value }"
                  @click="setLeaderboardRange(range.value)"
                >
                  {{ range.label }}
                </button>
              </div>
            </div>
            <div v-if="leaderboardLoading" class="game-leaderboard-empty">正在加载排行榜...</div>
            <div v-else-if="leaderboardError" class="game-leaderboard-empty text-red-500">{{ leaderboardError }}</div>
            <div v-else-if="leaderboardItems.length === 0" class="game-leaderboard-empty">暂无战绩，完成一局后将出现在这里。</div>
            <div v-else class="game-leaderboard-list">
              <div v-for="item in leaderboardItems" :key="item.user_id" class="game-leaderboard-row">
                <span class="game-leaderboard-rank">#{{ item.rank }}</span>
                <img
                  v-if="item.avatar_url"
                  :src="item.avatar_url"
                  alt=""
                  class="game-leaderboard-avatar"
                />
                <span v-else class="game-leaderboard-avatar game-leaderboard-avatar-fallback">
                  {{ item.username.slice(0, 1).toUpperCase() }}
                </span>
                <span class="game-leaderboard-name">{{ item.username }}</span>
                <span class="game-leaderboard-meta">{{ item.play_count }} 局 · 胜率 {{ formatWinRate(item.win_rate) }}</span>
                <span class="game-leaderboard-score">{{ formatAmount(item.net_amount) }}</span>
              </div>
            </div>
          </section>
          <a
            :href="embeddedUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="btn btn-secondary btn-sm custom-open-fab"
          >
            <Icon name="externalLink" size="sm" class="mr-1.5" :stroke-width="2" />
            {{ t('customPage.openInNewTab') }}
          </a>
          <iframe
            :src="embeddedUrl"
            class="custom-embed-frame"
            allowfullscreen
          ></iframe>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { getGameCenterLeaderboard, type GameCenterLeaderboardItem, type GameCenterRange } from '@/api/gameCenter'
import { buildEmbeddedUrl, detectTheme } from '@/utils/embedded-url'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

interface TocItem {
  id: string
  text: string
  level: number
}

const { t, locale } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const adminSettingsStore = useAdminSettingsStore()

const loading = ref(false)
const pageTheme = ref<'light' | 'dark'>('light')
const renderedHtml = ref('')
const markdownContainer = ref<HTMLElement | null>(null)
const tocItems = ref<TocItem[]>([])
const tocVisible = ref(typeof window !== 'undefined' ? window.innerWidth > 768 : true)
const activeHeadingId = ref('')
const leaderboardRange = ref<GameCenterRange>('today')
const leaderboardLoading = ref(false)
const leaderboardError = ref('')
const leaderboardItems = ref<GameCenterLeaderboardItem[]>([])
let themeObserver: MutationObserver | null = null

const leaderboardRanges: Array<{ value: GameCenterRange; label: string }> = [
  { value: 'today', label: '今日' },
  { value: '7d', label: '7日' },
  { value: '30d', label: '30日' },
  { value: 'all', label: '全部' },
]

const menuItemId = computed(() => route.params.id as string)

const menuItem = computed(() => {
  const id = menuItemId.value
  const publicItems = appStore.cachedPublicSettings?.custom_menu_items ?? []
  const found = publicItems.find((item) => item.id === id) ?? null
  if (found) return found
  if (authStore.isAdmin) {
    return adminSettingsStore.customMenuItems.find((item) => item.id === id) ?? null
  }
  return null
})

const markdownSlug = computed(() => {
  const item = menuItem.value
  if (!item) return ''
  if (item.page_slug) return item.page_slug
  if (item.url?.startsWith('md:')) return item.url.slice(3)
  return ''
})

const isMarkdownMode = computed(() => !!markdownSlug.value)

const embeddedUrl = computed(() => {
  if (!menuItem.value || isMarkdownMode.value) return ''
  return buildEmbeddedUrl(
    menuItem.value.url,
    authStore.user?.id,
    authStore.token,
    pageTheme.value,
    locale.value,
  )
})

const isValidUrl = computed(() => {
  if (isMarkdownMode.value) return false
  const url = embeddedUrl.value
  return url.startsWith('http://') || url.startsWith('https://')
})

const showGameCenterLeaderboard = computed(() => {
  const item = menuItem.value
  if (!item || isMarkdownMode.value) return false
  const text = `${item.label || ''} ${item.url || ''}`.toLowerCase()
  return text.includes('game') || text.includes('游戏') || text.includes('签到') || text.includes('/external/sign/')
})

function generateHeadingId(text: string, index: number): string {
  const base = text
    .toLowerCase()
    .replace(/[^\w一-鿿]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base ? `${base}-${index}` : `heading-${index}`
}

function isRelativeMarkdownAsset(src: string): boolean {
  const trimmed = src.trim()
  if (!trimmed || /^[a-z][a-z0-9+.-]*:/i.test(trimmed) || trimmed.startsWith('//') || trimmed.startsWith('/')) {
    return false
  }
  const [pathPart] = trimmed.split(/([?#].*)/, 2)
  return pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .every((part) => part !== '..' && !part.includes('\\'))
}

function buildPageImageUrl(slug: string, src: string): string {
  const trimmed = src.trim()
  const [pathPart, suffix = ''] = trimmed.split(/([?#].*)/, 2)
  const encodedPath = pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .map((part) => encodeURIComponent(part))
    .join('/')
  return `/api/v1/pages/${encodeURIComponent(slug)}/images/${encodedPath}${suffix}`
}

async function fetchAndRenderMarkdown(slug: string) {
  loading.value = true
  tocItems.value = []
  activeHeadingId.value = ''
  try {
    const resp = await fetch(`/api/v1/pages/${encodeURIComponent(slug)}`, {
      headers: authStore.token ? { Authorization: `Bearer ${authStore.token}` } : {},
    })
    if (!resp.ok) {
      renderedHtml.value = '<p class="text-red-500">Page not found</p>'
      return
    }
    let raw = await resp.text()

    raw = raw.replace(
      /!\[([^\]]*)\]\(([^)]+)\)/g,
      (match, alt, src) => isRelativeMarkdownAsset(src) ? `![${alt}](${buildPageImageUrl(slug, src)})` : match
    )

    const html = marked.parse(raw) as string
    const sanitized = DOMPurify.sanitize(html, {
      ADD_TAGS: ['iframe'],
      ADD_ATTR: ['allowfullscreen', 'frameborder', 'src'],
    })

    // Inject IDs into headings and build TOC
    const toc: TocItem[] = []
    let headingIndex = 0
    const withIds = sanitized.replace(
      /<(h[1-4])[^>]*>(.*?)<\/h[1-4]>/gi,
      (_, tag: string, content: string) => {
        const level = parseInt(tag[1])
        const text = content.replace(/<[^>]+>/g, '').trim()
        const id = generateHeadingId(text, headingIndex++)
        toc.push({ id, text, level })
        return `<${tag} id="${id}">${content}</${tag}>`
      }
    )

    renderedHtml.value = withIds
    tocItems.value = toc
  } catch {
    renderedHtml.value = '<p class="text-red-500">Failed to load page</p>'
  } finally {
    loading.value = false
    await nextTick()
    await nextTick()
    injectCopyButtons()
  }
}

function scrollToHeading(id: string) {
  const container = markdownContainer.value
  if (!container) return
  const el = container.querySelector(`#${CSS.escape(id)}`)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
    activeHeadingId.value = id
    if (window.innerWidth <= 640) {
      tocVisible.value = false
    }
  }
}

let scrollRafId = 0
function onContentScroll() {
  if (scrollRafId) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = 0
    const container = markdownContainer.value
    if (!container || tocItems.value.length === 0) return

    const containerRect = container.getBoundingClientRect()
    let current = ''

    for (const item of tocItems.value) {
      const el = container.querySelector(`#${CSS.escape(item.id)}`) as HTMLElement | null
      if (el) {
        const elRect = el.getBoundingClientRect()
        if (elRect.top - containerRect.top <= 100) {
          current = item.id
        }
      }
    }
    activeHeadingId.value = current
  })
}

function injectCopyButtons() {
  const container = markdownContainer.value
  if (!container) return

  container.querySelectorAll('pre').forEach((pre) => {
    if (pre.querySelector('.copy-btn')) return
    const btn = document.createElement('button')
    btn.className = 'copy-btn'
    btn.textContent = '复制'
    btn.addEventListener('click', async () => {
      const code = pre.querySelector('code')?.textContent ?? pre.textContent ?? ''
      try {
        await navigator.clipboard.writeText(code)
        btn.textContent = '已复制 ✓'
        setTimeout(() => { btn.textContent = '复制' }, 2000)
      } catch {
        btn.textContent = '失败'
        setTimeout(() => { btn.textContent = '复制' }, 2000)
      }
    })
    pre.style.position = 'relative'
    pre.appendChild(btn)
  })
}

function formatAmount(value: number): string {
  const prefix = value > 0 ? '+' : ''
  return `${prefix}${Number(value || 0).toFixed(2)}`
}

function formatWinRate(value: number): string {
  return `${Math.round(Number(value || 0) * 100)}%`
}

async function loadLeaderboard() {
  if (!showGameCenterLeaderboard.value) {
    leaderboardItems.value = []
    return
  }
  leaderboardLoading.value = true
  leaderboardError.value = ''
  try {
    const result = await getGameCenterLeaderboard({
      range: leaderboardRange.value,
      limit: 10,
    })
    leaderboardItems.value = result.items
  } catch (err: any) {
    leaderboardItems.value = []
    leaderboardError.value = err?.message || '排行榜加载失败'
  } finally {
    leaderboardLoading.value = false
  }
}

function setLeaderboardRange(range: GameCenterRange) {
  leaderboardRange.value = range
  loadLeaderboard()
}

watch(markdownSlug, (slug) => {
  if (slug) {
    fetchAndRenderMarkdown(slug)
  } else {
    renderedHtml.value = ''
    tocItems.value = []
  }
}, { immediate: true })

watch(showGameCenterLeaderboard, () => {
  loadLeaderboard()
}, { immediate: true })

onMounted(async () => {
  pageTheme.value = detectTheme()

  if (typeof document !== 'undefined') {
    themeObserver = new MutationObserver(() => {
      pageTheme.value = detectTheme()
    })
    themeObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class'],
    })
  }

  if (appStore.publicSettingsLoaded) return
  loading.value = true
  try {
    await appStore.fetchPublicSettings()
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  if (themeObserver) {
    themeObserver.disconnect()
    themeObserver = null
  }
})
</script>

<style scoped>
.custom-page-layout {
  @apply flex flex-col;
  height: calc(100vh - 64px - 4rem);
}

.toc-sidebar {
  @apply flex flex-col h-full border-r border-gray-200 dark:border-dark-600 bg-gray-50 dark:bg-dark-800;
  width: min(240px, 30%);
  min-width: 160px;
  max-width: 280px;
  overflow: hidden;
}

@media (max-width: 640px) {
  .toc-sidebar {
    position: absolute;
    left: 0;
    top: 0;
    z-index: 20;
    width: 70%;
    max-width: 240px;
    height: 100%;
    box-shadow: 2px 0 8px rgba(0, 0, 0, 0.1);
  }
}

.toc-header {
  @apply flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-dark-600;
}

.toc-title {
  @apply text-sm font-semibold text-gray-700 dark:text-dark-200;
}

.toc-close-btn {
  @apply p-1 rounded text-gray-400 hover:text-gray-600 dark:hover:text-dark-200 hover:bg-gray-200 dark:hover:bg-dark-600 transition-colors;
}

.toc-nav {
  @apply flex-1 overflow-y-auto py-2 px-2;
}

.toc-item {
  @apply block px-2 py-1.5 text-sm rounded transition-colors truncate;
  @apply text-gray-600 dark:text-dark-300 hover:text-gray-900 dark:hover:text-white hover:bg-gray-200 dark:hover:bg-dark-600;
}

.toc-item.toc-active {
  @apply text-primary-600 dark:text-primary-400 bg-primary-50 dark:bg-primary-900/20 font-medium;
}

.toc-level-1 { padding-left: 8px; }
.toc-level-2 { padding-left: 20px; }
.toc-level-3 { padding-left: 32px; }
.toc-level-4 { padding-left: 44px; }

.toc-toggle-btn {
  @apply absolute left-2 top-2 z-10 flex items-center px-2 py-1.5 rounded-md text-sm;
  @apply bg-white dark:bg-dark-700 border border-gray-200 dark:border-dark-500;
  @apply text-gray-600 dark:text-dark-300 hover:bg-gray-100 dark:hover:bg-dark-600;
  @apply shadow-sm transition-colors cursor-pointer;
}

.custom-embed-shell {
  @apply relative;
  @apply flex h-full w-full flex-col overflow-hidden rounded-2xl;
  @apply bg-gradient-to-b from-gray-50 to-white dark:from-dark-900 dark:to-dark-950;
  @apply p-0;
}

.custom-open-fab {
  @apply absolute right-3 top-3 z-10;
  @apply shadow-sm backdrop-blur supports-[backdrop-filter]:bg-white/80 dark:supports-[backdrop-filter]:bg-dark-800/80;
}

.custom-embed-frame {
  display: block;
  margin: 0;
  width: 100%;
  flex: 1 1 auto;
  min-height: 0;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}

.game-leaderboard-panel {
  @apply border-b border-gray-200 bg-white/90 px-4 py-3 dark:border-dark-700 dark:bg-dark-900/90;
}

.game-leaderboard-header {
  @apply mb-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between;
}

.game-leaderboard-kicker {
  @apply text-[11px] font-semibold uppercase tracking-[0.25em] text-primary-500;
}

.game-leaderboard-title {
  @apply text-base font-semibold text-gray-900 dark:text-white;
}

.game-leaderboard-tabs {
  @apply flex flex-wrap gap-1;
}

.game-leaderboard-tab {
  @apply rounded-md border border-gray-200 px-2.5 py-1 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-100 dark:border-dark-600 dark:text-dark-300 dark:hover:bg-dark-700;
}

.game-leaderboard-tab-active {
  @apply border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300;
}

.game-leaderboard-empty {
  @apply rounded-lg border border-dashed border-gray-200 px-3 py-4 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400;
}

.game-leaderboard-list {
  @apply grid gap-2 lg:grid-cols-2;
}

.game-leaderboard-row {
  @apply grid grid-cols-[3rem_2rem_minmax(0,1fr)_auto] items-center gap-2 rounded-lg border border-gray-200 bg-gray-50 px-3 py-2 text-sm dark:border-dark-700 dark:bg-dark-800;
}

.game-leaderboard-rank {
  @apply font-mono text-xs font-semibold text-gray-500 dark:text-dark-300;
}

.game-leaderboard-avatar {
  @apply h-8 w-8 rounded-full object-cover;
}

.game-leaderboard-avatar-fallback {
  @apply flex items-center justify-center bg-primary-100 text-xs font-bold text-primary-700 dark:bg-primary-500/20 dark:text-primary-200;
}

.game-leaderboard-name {
  @apply min-w-0 truncate font-semibold text-gray-900 dark:text-white;
}

.game-leaderboard-meta {
  @apply hidden text-xs text-gray-500 dark:text-dark-400 sm:inline;
}

.game-leaderboard-score {
  @apply font-mono text-sm font-bold text-emerald-600 dark:text-emerald-300;
}
</style>

<style>
.markdown-page-content {
  line-height: 1.7;
  color: inherit;
}
.markdown-page-content h1 { @apply text-3xl font-bold mt-8 mb-4 pb-2 border-b border-gray-200 dark:border-dark-600; }
.markdown-page-content h2 { @apply text-2xl font-bold mt-6 mb-3; }
.markdown-page-content h3 { @apply text-xl font-semibold mt-5 mb-2; }
.markdown-page-content h4 { @apply text-lg font-semibold mt-4 mb-2; }
.markdown-page-content p { @apply mb-4; }
.markdown-page-content ul { @apply list-disc pl-6 mb-4; }
.markdown-page-content ol { @apply list-decimal pl-6 mb-4; }
.markdown-page-content li { @apply mb-1; }
.markdown-page-content a { @apply text-primary-500 hover:text-primary-600 underline; }
.markdown-page-content blockquote { @apply border-l-4 border-gray-300 dark:border-dark-500 pl-4 italic text-gray-600 dark:text-dark-300 my-4; }
.markdown-page-content img { @apply max-w-full h-auto rounded-lg my-4; }
.markdown-page-content table { @apply w-full border-collapse my-4; }
.markdown-page-content th { @apply border border-gray-300 dark:border-dark-500 px-3 py-2 bg-gray-50 dark:bg-dark-700 font-semibold text-left; }
.markdown-page-content td { @apply border border-gray-300 dark:border-dark-500 px-3 py-2; }
.markdown-page-content code { @apply bg-gray-100 dark:bg-dark-700 px-1.5 py-0.5 rounded text-sm font-mono; }
.markdown-page-content pre { @apply bg-gray-900 dark:bg-dark-900 text-gray-100 p-4 rounded-lg overflow-x-auto my-4 relative; }
.markdown-page-content pre code { @apply bg-transparent p-0 text-inherit; }
.markdown-page-content hr { @apply my-6 border-gray-200 dark:border-dark-600; }

.copy-btn {
  position: absolute;
  top: 8px;
  right: 8px;
  padding: 4px 10px;
  font-size: 12px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.15);
  color: #e2e8f0;
  border: 1px solid rgba(255, 255, 255, 0.2);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.2s, background 0.2s;
  font-family: inherit;
}
.copy-btn:hover { background: rgba(255, 255, 255, 0.25); }
pre:hover .copy-btn { opacity: 1; }
</style>
