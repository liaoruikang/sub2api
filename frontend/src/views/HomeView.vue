<template>
  <div v-if="hasHomeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
              <path d="M4 4.5A2.5 2.5 0 0 1 6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5z" />
            </svg>
          </a>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <svg v-if="isDark" width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
            </svg>
            <svg v-else width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <div v-else class="home-prototype-shell">
    <header class="relative z-20 px-6 py-5">
      <nav class="mx-auto flex max-w-7xl items-center justify-between gap-5">
        <router-link class="flex min-w-0 items-center gap-3 text-gray-900 no-underline dark:text-white" to="/">
          <span class="brand-logo">
            <img :src="siteLogo || '/logo.png'" :alt="siteName" class="h-full w-full rounded-xl object-cover" />
          </span>
          <span class="hidden min-w-0 sm:block">
            <strong class="block truncate text-[17px] font-bold tracking-[-0.03em]">{{ siteName }}</strong>
            <span class="mt-0.5 block truncate text-xs text-gray-500 dark:text-dark-400">{{ siteSubtitle }}</span>
          </span>
        </router-link>

        <nav class="flex items-center gap-2.5" aria-label="Actions">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            class="nav-icon-btn"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            :aria-label="t('home.viewDocs')"
            :title="t('home.viewDocs')"
          >
            <svg width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
              <path d="M4 4.5A2.5 2.5 0 0 1 6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5z" />
            </svg>
          </a>
          <button
            class="nav-icon-btn"
            type="button"
            :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <svg v-if="isDark" width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M4.93 19.07l1.41-1.41M17.66 6.34l1.41-1.41" />
            </svg>
            <svg v-else width="19" height="19" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
          <router-link
            class="inline-flex min-h-10 items-center justify-center gap-2 rounded-full bg-gradient-to-r from-dark-900 via-primary-700 to-blue-600 px-4 text-xs font-semibold text-white shadow-lg shadow-primary-500/25 transition-all hover:-translate-y-0.5 hover:shadow-primary-500/30 dark:from-dark-800 dark:via-primary-700 dark:to-blue-600"
            :to="isAuthenticated ? dashboardPath : '/login'"
          >
            <span>{{ isAuthenticated ? t('home.dashboard') : t('home.login') }}</span>
            <span aria-hidden="true">↗</span>
          </router-link>
        </nav>
      </nav>
    </header>

    <main class="relative z-10">
      <section class="mx-auto grid min-h-[calc(100vh-88px)] max-w-7xl items-center gap-14 px-6 py-16 lg:grid-cols-[minmax(0,1fr)_minmax(520px,1.08fr)] lg:gap-20 lg:py-24 xl:px-8" aria-labelledby="hero-title">
        <div class="text-center lg:text-left">
          <div class="glass-chip mx-auto lg:mx-0">
            <span class="signal-dot" aria-hidden="true"></span>
            <span>{{ t('home.heroSubtitle') }}</span>
          </div>
          <h1 id="hero-title" class="mt-6 text-6xl font-black leading-[0.86] tracking-[-0.085em] text-gray-950 dark:text-white md:text-7xl lg:text-8xl xl:text-[7.25rem]">
            <span>{{ t('home.features.unifiedGateway') }}</span>
            <span class="block bg-gradient-to-r from-[#73f5a8] via-[#59e8f0] to-blue-600 bg-clip-text text-transparent drop-shadow-lg">{{ siteName }}</span>
          </h1>
          <p class="mx-auto mt-8 max-w-3xl text-xl leading-9 text-gray-600 dark:text-dark-300 lg:mx-0 md:text-2xl">{{ siteSubtitle || t('home.heroDescription') }}</p>
          <div class="mt-10 flex flex-wrap items-center justify-center gap-4 lg:justify-start">
            <router-link
              class="btn btn-primary rounded-full px-8 py-4 text-base shadow-xl shadow-primary-500/30"
              :to="isAuthenticated ? dashboardPath : '/login'"
            >
              <span>{{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}</span>
              <span class="grid h-7 w-7 place-items-center rounded-full bg-white/15" aria-hidden="true">→</span>
            </router-link>
            <a
              v-if="docUrl"
              class="glass-button hidden sm:inline-flex"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
            >
              <span>{{ t('home.viewDocs') }}</span>
            </a>
          </div>
          <div class="mt-12 grid max-w-2xl gap-4 sm:grid-cols-3 lg:mx-0 mx-auto">
            <div class="metric-card"><b>1 API</b><span>{{ t('home.tags.subscriptionToApi') }}</span></div>
            <div class="metric-card"><b>Sticky</b><span>{{ t('home.tags.stickySession') }}</span></div>
            <div class="metric-card"><b>Realtime</b><span>{{ t('home.tags.realtimeBilling') }}</span></div>
          </div>
        </div>

        <div class="flex min-h-[560px] justify-center lg:min-h-[680px] lg:justify-end" aria-label="Gateway visualization">
          <div class="orbital-card">
            <div class="route-line" aria-hidden="true"></div>
            <div class="route-line route-line-small" aria-hidden="true"></div>
            <div class="center-logo"><img :src="siteLogo || '/logo.png'" alt="" class="h-full w-full rounded-[30px] object-cover" /></div>

            <div class="node node-a">
              <small>Claude</small>
              <strong>{{ t('home.features.unifiedGateway') }}</strong>
              <span>200 OK</span>
            </div>
            <div class="node node-b">
              <small>Pool</small>
              <strong>{{ t('home.features.multiAccount') }}</strong>
              <span>8 active</span>
            </div>
            <div class="node node-c">
              <small>Quota</small>
              <strong>{{ t('home.features.balanceQuota') }}</strong>
              <span>$0.024</span>
            </div>

            <div class="terminal" aria-label="Terminal preview">
              <div class="flex h-9 items-center gap-2 border-b border-white/10 px-3.5 text-xs text-slate-400"><span class="dot bg-red-500"></span><span class="dot bg-yellow-400"></span><span class="dot bg-green-500 mr-2"></span><span>sub2api.gateway</span></div>
              <div class="space-y-1.5 p-4 font-mono text-xs leading-6 text-cyan-50 sm:text-[13px]">
                <div class="terminal-line terminal-line-1"><span class="text-[#73f5a8]">$</span> <span class="text-[#59e8f0]">curl</span> <span class="text-indigo-300">-X POST</span> /v1/messages</div>
                <div class="terminal-line terminal-line-2"><span class="text-amber-300">route</span> → claude / sticky-session / account#08</div>
                <div class="terminal-line terminal-line-3"><span class="rounded bg-emerald-400/15 px-2 py-0.5 font-semibold text-[#73f5a8]">200 OK</span> <span class="text-amber-200">{ "content": "Hello!" }</span></div>
                <div class="terminal-line terminal-line-4"><span class="text-[#73f5a8]">$</span><span class="cursor"></span></div>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl px-6 pb-20 xl:px-8" aria-label="Product details">
        <div class="mb-6 flex flex-wrap items-center justify-center gap-3">
          <div class="tag-pill">
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M7 7h11l-3-3"/><path d="M17 17H6l3 3"/><path d="M18 7 6 19"/></svg>
            <span>{{ t('home.tags.subscriptionToApi') }}</span>
          </div>
          <div class="tag-pill">
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
            <span>{{ t('home.tags.stickySession') }}</span>
          </div>
          <div class="tag-pill">
            <svg width="17" height="17" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 3v18h18"/><path d="m7 15 4-4 3 3 5-7"/></svg>
            <span>{{ t('home.tags.realtimeBilling') }}</span>
          </div>
        </div>

        <div class="grid gap-5 md:grid-cols-3">
          <article class="feature-card">
            <div class="feature-icon">
              <svg width="23" height="23" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="6" rx="2"/><rect x="3" y="14" width="18" height="6" rx="2"/><path d="M7 7h.01M7 17h.01"/></svg>
            </div>
            <h3>{{ t('home.features.unifiedGateway') }}</h3>
            <p>{{ t('home.features.unifiedGatewayDesc') }}</p>
          </article>
          <article class="feature-card">
            <div class="feature-icon">
              <svg width="23" height="23" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
            </div>
            <h3>{{ t('home.features.multiAccount') }}</h3>
            <p>{{ t('home.features.multiAccountDesc') }}</p>
          </article>
          <article class="feature-card">
            <div class="feature-icon">
              <svg width="23" height="23" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 1v22"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7H14a3.5 3.5 0 0 1 0 7H6"/></svg>
            </div>
            <h3>{{ t('home.features.balanceQuota') }}</h3>
            <p>{{ t('home.features.balanceQuotaDesc') }}</p>
          </article>
        </div>

        <div class="mt-8 rounded-4xl border border-gray-200/60 bg-white/60 p-8 shadow-card backdrop-blur-xl dark:border-dark-700/60 dark:bg-dark-800/60">
          <div class="mb-5">
            <h2 class="text-3xl font-black tracking-[-0.055em] text-gray-950 dark:text-white md:text-4xl">{{ t('home.providers.title') }}</h2>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-400">{{ t('home.providers.description') }}</p>
          </div>
          <div class="grid gap-2.5 sm:grid-cols-2 lg:grid-cols-5">
            <div class="provider-card">
              <div class="flex items-center justify-between gap-2.5"><span class="provider-icon provider-icon--claude" aria-hidden="true"><ClaudeMark /></span><span class="badge">{{ t('home.providers.supported') }}</span></div>
              <div class="provider-name">{{ t('home.providers.claude') }}</div>
            </div>
            <div class="provider-card">
              <div class="flex items-center justify-between gap-2.5"><span class="provider-icon provider-icon--gpt" aria-hidden="true"><OpenAiMark /></span><span class="badge">{{ t('home.providers.supported') }}</span></div>
              <div class="provider-name">GPT</div>
            </div>
            <div class="provider-card">
              <div class="flex items-center justify-between gap-2.5"><span class="provider-icon provider-icon--gemini" aria-hidden="true"><GeminiMark /></span><span class="badge">{{ t('home.providers.supported') }}</span></div>
              <div class="provider-name">{{ t('home.providers.gemini') }}</div>
            </div>
            <div class="provider-card">
              <div class="flex items-center justify-between gap-2.5"><span class="provider-icon provider-icon--antigravity" aria-hidden="true"><AntigravityMark /></span><span class="badge">{{ t('home.providers.supported') }}</span></div>
              <div class="provider-name">{{ t('home.providers.antigravity') }}</div>
            </div>
            <div class="provider-card opacity-60">
              <div class="flex items-center justify-between gap-2.5"><span class="provider-icon provider-icon--more" aria-hidden="true">+</span><span class="badge badge-muted">{{ t('home.providers.soon') }}</span></div>
              <div class="provider-name">{{ t('home.providers.more') }}</div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <footer class="relative z-10 mx-auto w-full max-w-7xl border-t border-gray-200/60 px-6 py-8 text-center text-sm text-gray-500 dark:border-dark-800/60 dark:text-dark-400">
      <span>&copy; {{ currentYear }} {{ siteName }}. </span>
      <span>{{ t('home.footer.allRightsReserved') }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { useAppStore, useAuthStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API'
)
const siteLogo = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', {
    allowRelative: true,
    allowDataUrl: true,
  })
)
const siteSubtitle = computed(
  () => appStore.cachedPublicSettings?.site_subtitle || t('home.heroDescription')
)
const docUrl = computed(() =>
  sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || '')
)
const homeContent = computed(
  () => appStore.cachedPublicSettings?.home_content || ''
)
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

function mark(path: string | string[], viewBox = '0 0 24 24') {
  const paths = Array.isArray(path) ? path : [path]
  return defineComponent({
    render() {
      return h(
        'svg',
        { viewBox, fill: 'currentColor', 'aria-hidden': 'true' },
        paths.map((d) => h('path', { d }))
      )
    }
  })
}

const ClaudeMark = mark(
  'm3.127 10.604 3.135-1.76.053-.153-.053-.085H6.11l-.525-.032-1.791-.048-1.554-.065-1.505-.08-.38-.081L0 7.832l.036-.234.32-.214.455.04 1.009.069 1.513.105 1.097.064 1.626.17h.259l.036-.105-.089-.065-.068-.064-1.566-1.062-1.695-1.121-.887-.646-.48-.327-.243-.306-.104-.67.435-.48.585.04.15.04.593.456 1.267.981 1.654 1.218.242.202.097-.068.012-.049-.109-.181-.9-1.626-.96-1.655-.428-.686-.113-.411a2 2 0 0 1-.068-.484l.496-.674L4.446 0l.662.089.279.242.411.94.666 1.48 1.033 2.014.302.597.162.553.06.17h.105v-.097l.085-1.134.157-1.392.154-1.792.052-.504.25-.605.497-.327.387.186.319.456-.045.294-.19 1.23-.37 1.93-.243 1.29h.142l.161-.16.654-.868 1.097-1.372.484-.545.565-.601.363-.287h.686l.505.751-.226.775-.707.895-.585.759-.839 1.13-.524.904.048.072.125-.012 1.897-.403 1.024-.186 1.223-.21.553.258.06.263-.218.536-1.307.323-1.533.307-2.284.54-.028.02.032.04 1.029.098.44.024h1.077l2.005.15.525.346.315.424-.053.323-.807.411-3.631-.863-.872-.218h-.12v.073l.726.71 1.331 1.202 1.667 1.55.084.383-.214.302-.226-.032-1.464-1.101-.565-.497-1.28-1.077h-.084v.113l.295.432 1.557 2.34.08.718-.112.234-.404.141-.444-.08-.911-1.28-.94-1.44-.759-1.291-.093.053-.448 4.821-.21.246-.484.186-.403-.307-.214-.496.214-.98.258-1.28.21-1.016.19-1.263.112-.42-.008-.028-.092.012-.953 1.307-1.448 1.957-1.146 1.227-.274.109-.477-.247.045-.44.266-.39 1.586-2.018.956-1.25.617-.723-.004-.105h-.036l-4.212 2.736-.75.096-.324-.302.04-.496.154-.162 1.267-.871z',
  '0 0 16 16'
)
const OpenAiMark = mark('M22.282 9.821a5.985 5.985 0 0 0-.516-4.91 6.046 6.046 0 0 0-6.51-2.9A6.065 6.065 0 0 0 4.981 4.18a5.985 5.985 0 0 0-3.998 2.9 6.046 6.046 0 0 0 .743 7.097 5.98 5.98 0 0 0 .51 4.911 6.051 6.051 0 0 0 6.515 2.9A5.985 5.985 0 0 0 13.26 24a6.056 6.056 0 0 0 5.772-4.206 5.99 5.99 0 0 0 3.997-2.9 6.056 6.056 0 0 0-.747-7.073zM13.26 22.43a4.476 4.476 0 0 1-2.876-1.04l.141-.081 4.779-2.758a.795.795 0 0 0 .392-.681v-6.737l2.02 1.168a.071.071 0 0 1 .038.052v5.583a4.504 4.504 0 0 1-4.494 4.494zM3.6 18.304a4.47 4.47 0 0 1-.535-3.014l.142.085 4.783 2.759a.771.771 0 0 0 .78 0l5.843-3.369v2.332a.08.08 0 0 1-.033.062L9.74 19.95a4.5 4.5 0 0 1-6.14-1.646zM2.34 7.896a4.485 4.485 0 0 1 2.366-1.973V11.6a.766.766 0 0 0 .388.676l5.815 3.355-2.02 1.168a.076.076 0 0 1-.071 0l-4.83-2.786A4.504 4.504 0 0 1 2.34 7.872zm16.597 3.855l-5.833-3.387L15.119 7.2a.076.076 0 0 1 .071 0l4.83 2.791a4.494 4.494 0 0 1-.676 8.105v-5.678a.79.79 0 0 0-.407-.667zm2.01-3.023l-.141-.085-4.774-2.782a.776.776 0 0 0-.785 0L9.409 9.23V6.897a.066.066 0 0 1 .028-.061l4.83-2.787a4.5 4.5 0 0 1 6.68 4.66zm-12.64 4.135l-2.02-1.164a.08.08 0 0 1-.038-.057V6.075a4.5 4.5 0 0 1 7.375-3.453l-.142.08L8.704 5.46a.795.795 0 0 0-.393.681z')
const GeminiMark = mark('M20.616 10.835a14.147 14.147 0 0 1-4.45-3.001 14.111 14.111 0 0 1-3.678-6.452.503.503 0 0 0-.975 0 14.134 14.134 0 0 1-3.679 6.452 14.155 14.155 0 0 1-4.45 3.001c-.65.28-1.318.505-2.002.678a.502.502 0 0 0 0 .975c.684.172 1.35.397 2.002.677a14.147 14.147 0 0 1 4.45 3.001 14.112 14.112 0 0 1 3.679 6.453.502.502 0 0 0 .975 0c.172-.685.397-1.351.677-2.003a14.145 14.145 0 0 1 3.001-4.45 14.113 14.113 0 0 1 6.453-3.678.503.503 0 0 0 0-.975 13.245 13.245 0 0 1-2.003-.678z')
const AntigravityMark = mark('M19.35 10.04C18.67 6.59 15.64 4 12 4 9.11 4 6.6 5.64 5.35 8.04 2.34 8.36 0 10.91 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z')

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped lang="postcss">
.home-prototype-shell {
  @apply relative flex min-h-screen flex-col overflow-hidden bg-gradient-to-br from-slate-50 via-primary-50/40 to-slate-100 text-gray-950 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950 dark:text-white;
}

.home-prototype-shell::before {
  content: "";
  @apply pointer-events-none absolute inset-0 opacity-70;
  background:
    radial-gradient(circle at 15% 8%, rgba(115, 245, 168, 0.34), transparent 28rem),
    radial-gradient(circle at 85% 12%, rgba(38, 108, 255, 0.24), transparent 30rem),
    radial-gradient(circle at 58% 82%, rgba(89, 232, 240, 0.2), transparent 32rem),
    linear-gradient(rgba(20,184,166,0.045) 1px, transparent 1px),
    linear-gradient(90deg, rgba(38,108,255,0.04) 1px, transparent 1px);
  background-size: auto, auto, auto, 72px 72px, 72px 72px;
  mask-image: radial-gradient(circle at 50% 25%, black, transparent 76%);
}

.home-prototype-shell::after {
  content: "";
  @apply pointer-events-none absolute inset-0 opacity-[0.08] mix-blend-overlay;
  background-image: url("data:image/svg+xml,%3Csvg viewBox='0 0 160 160' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='.82' numOctaves='3' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E");
}

.brand-logo {
  @apply h-11 w-11 flex-shrink-0 rounded-2xl bg-gradient-to-br from-[#73f5a8] via-[#59e8f0] to-blue-600 p-[3px] shadow-lg shadow-blue-500/25 ring-1 ring-white/30;
}

.nav-icon-btn {
  @apply inline-flex h-10 w-10 items-center justify-center rounded-xl border border-gray-200/70 bg-white/70 text-gray-600 shadow-sm backdrop-blur-xl transition-all hover:-translate-y-0.5 hover:border-primary-300 hover:text-gray-900 hover:shadow-md dark:border-dark-700/70 dark:bg-dark-800/70 dark:text-dark-300 dark:hover:border-primary-700 dark:hover:text-white;
}

.glass-chip {
  @apply inline-flex max-w-full items-center gap-2.5 rounded-full border border-gray-200/70 bg-white/70 px-3 py-2 text-sm font-semibold text-gray-600 shadow-sm backdrop-blur-xl dark:border-dark-700/70 dark:bg-dark-800/70 dark:text-dark-300;
}

.signal-dot {
  @apply relative h-6 w-6 rounded-full bg-gradient-to-br from-[#73f5a8] via-[#59e8f0] to-blue-600 shadow-glow;
}

.signal-dot::after {
  content: "";
  @apply absolute -inset-1.5 rounded-full border border-primary-300/60 dark:border-primary-400/50;
  animation: pulse-ring 2.4s ease-out infinite;
}

.glass-button {
  @apply min-h-10 items-center justify-center gap-2 rounded-full border border-gray-200/70 bg-white/70 px-4 text-sm font-semibold text-gray-700 shadow-sm backdrop-blur-xl transition-all hover:-translate-y-0.5 hover:border-primary-300 hover:text-gray-950 hover:shadow-md dark:border-dark-700/70 dark:bg-dark-800/70 dark:text-dark-200 dark:hover:border-primary-700 dark:hover:text-white;
}

.metric-card {
  @apply rounded-3xl border border-gray-200/60 bg-white/70 p-5 text-left shadow-md backdrop-blur-xl dark:border-dark-700/60 dark:bg-dark-800/70;
}

.metric-card b {
  @apply block text-2xl font-black tracking-[-0.04em] text-gray-950 dark:text-white;
}

.metric-card span {
  @apply mt-1.5 block text-sm leading-6 text-gray-500 dark:text-dark-400;
}

.orbital-card {
  @apply relative inline-block w-full max-w-[620px] overflow-hidden rounded-[56px] border border-gray-200/70 bg-white/60 shadow-card-hover backdrop-blur-2xl transition-transform duration-300 dark:border-dark-700/70 dark:bg-dark-800/60;
  aspect-ratio: 1 / 1.04;
  transform: perspective(1100px) rotateY(-7deg) rotateX(4deg);
}

.orbital-card:hover {
  transform: perspective(1100px) rotateY(-2deg) rotateX(1deg) translateY(-6px);
}

.orbital-card::before {
  content: "";
  @apply absolute inset-5 rounded-[34px] border border-primary-300/30 dark:border-primary-400/20;
  background:
    linear-gradient(rgba(20,184,166,.08) 1px, transparent 1px),
    linear-gradient(90deg, rgba(38,108,255,.07) 1px, transparent 1px);
  background-size: 34px 34px;
  mask-image: radial-gradient(circle at 50% 46%, black, transparent 72%);
}

.center-logo {
  @apply absolute left-1/2 top-[42%] z-[2] h-36 w-36 -translate-x-1/2 -translate-y-1/2 rounded-[34px] bg-gradient-to-br from-[#73f5a8] via-[#59e8f0] to-blue-600 p-2 shadow-glow-lg sm:h-[188px] sm:w-[188px] sm:rounded-[46px];
}

.route-line {
  @apply absolute left-1/2 top-[42%] h-[78%] w-[78%] -translate-x-1/2 -translate-y-1/2 rounded-full border border-dashed border-primary-300/60 dark:border-primary-400/40;
  animation: orbit-spin 28s linear infinite;
}

.route-line-small {
  @apply h-[56%] w-[56%] border-[#73f5a8]/50;
  animation-duration: 19s;
  animation-direction: reverse;
}

.node {
  @apply absolute z-[3] w-[132px] rounded-3xl border border-white/25 bg-dark-900/80 p-4 text-white shadow-2xl backdrop-blur-xl sm:w-[154px] sm:p-4;
}

.node small {
  @apply block text-[10px] font-extrabold uppercase tracking-[0.12em] text-[#73f5a8] sm:text-[11px];
}

.node strong {
  @apply mt-2 block text-[15px] font-bold tracking-[-0.02em] sm:text-base;
}

.node span {
  @apply mt-1 block text-xs text-slate-400;
}

.node-a { @apply left-3 top-12 sm:left-8 sm:top-[74px]; }
.node-b { @apply right-3 top-24 sm:right-6 sm:top-[155px]; }
.node-c { @apply bottom-24 left-5 sm:bottom-[90px] sm:left-[60px]; }

.terminal {
  @apply absolute bottom-5 left-5 right-5 z-[4] overflow-hidden rounded-[2rem] border border-primary-300/20 bg-dark-950/90 shadow-2xl sm:bottom-9 sm:left-9 sm:right-9;
}

.dot {
  @apply h-2.5 w-2.5 rounded-full;
}

.terminal-line {
  @apply opacity-0;
  transform: translateY(6px);
  animation: line-reveal .5s ease forwards;
}

.terminal-line-1 { animation-delay: .2s; }
.terminal-line-2 { animation-delay: .85s; }
.terminal-line-3 { animation-delay: 1.45s; }
.terminal-line-4 { animation-delay: 2.1s; }

.cursor {
  @apply ml-1 inline-block h-[15px] w-2 bg-[#73f5a8] align-[-2px];
  animation: cursor-blink 1s step-end infinite;
}

.tag-pill {
  @apply inline-flex items-center gap-2 rounded-full border border-gray-200/70 bg-white/70 px-4 py-2.5 text-sm font-semibold text-gray-600 shadow-sm backdrop-blur-xl dark:border-dark-700/70 dark:bg-dark-800/70 dark:text-dark-300;
}

.tag-pill svg {
  @apply text-primary-500;
}

.feature-card {
  @apply relative min-h-[270px] overflow-hidden rounded-[2rem] border border-gray-200/60 bg-white/70 p-8 shadow-card backdrop-blur-xl transition-all hover:-translate-y-1 hover:border-primary-300/70 hover:shadow-card-hover dark:border-dark-700/60 dark:bg-dark-800/70 dark:hover:border-primary-700/70;
}

.feature-card::after {
  content: "";
  @apply absolute -bottom-11 -right-7 h-40 w-40 rounded-full bg-primary-300/20 blur-sm dark:bg-primary-500/10;
}

.feature-icon {
  @apply relative z-10 grid h-14 w-14 place-items-center rounded-2xl bg-gradient-to-br from-[#73f5a8] via-[#59e8f0] to-blue-600 text-white shadow-lg shadow-primary-500/25;
}

.feature-card h3 {
  @apply relative z-10 mt-6 text-xl font-bold tracking-[-0.04em] text-gray-950 dark:text-white;
}

.feature-card p {
  @apply relative z-10 mt-3 text-base leading-8 text-gray-600 dark:text-dark-400;
}

.provider-card {
  @apply flex min-h-[112px] flex-col justify-between rounded-3xl border border-gray-200/60 bg-white/70 p-5 shadow-sm backdrop-blur-xl dark:border-dark-700/60 dark:bg-dark-900/50;
}

.provider-icon {
  @apply grid h-10 w-10 flex-shrink-0 place-items-center rounded-2xl shadow-md shadow-blue-500/20;
}

.provider-icon svg {
  @apply h-[18px] w-[18px];
}

.provider-icon--claude { @apply bg-gradient-to-br from-amber-400 to-amber-600 text-white; }
.provider-icon--gpt { @apply bg-gradient-to-br from-slate-900 to-black text-white; }
.provider-icon--gemini { @apply bg-gradient-to-br from-blue-400 to-blue-600 text-white; }
.provider-icon--antigravity { @apply bg-gradient-to-br from-[#73f5a8] to-[#59e8f0] text-dark-950; }
.provider-icon--more { @apply bg-gradient-to-br from-slate-500 to-slate-700 text-base font-black text-white; }

.badge {
  @apply rounded-full bg-primary-100 px-2 py-1 text-[10px] font-black text-primary-700 dark:bg-primary-900/30 dark:text-primary-300;
}

.badge-muted {
  @apply bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400;
}

.provider-name {
  @apply mt-3 text-sm font-black tracking-[-0.02em] text-gray-950 dark:text-white;
}

@keyframes pulse-ring {
  0% { transform: scale(.72); opacity: .9; }
  100% { transform: scale(1.55); opacity: 0; }
}

@keyframes orbit-spin {
  to { transform: translate(-50%, -50%) rotate(360deg); }
}

@keyframes line-reveal {
  to { opacity: 1; transform: translateY(0); }
}

@keyframes cursor-blink {
  50% { opacity: 0; }
}

@media (max-width: 640px) {
  .orbital-card {
    @apply rounded-[34px];
    transform: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: .01ms !important;
    animation-iteration-count: 1 !important;
    scroll-behavior: auto !important;
    transition-duration: .01ms !important;
  }
}
</style>
