<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col gap-3">
          <div class="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
            <div class="grid w-full grid-cols-1 gap-3 sm:grid-cols-2 xl:w-auto xl:grid-cols-[260px_180px_180px_180px]">
              <SearchInput
                v-model="filters.model"
                :placeholder="t('grokVideoConsole.filters.modelPlaceholder')"
                class="w-full"
                @search="applyFilters"
              />
              <Select
                v-model="filters.status"
                :options="statusOptions"
                class="w-full"
                @change="applyFilters"
              />
              <Select
                v-model="filters.apiKeyId"
                :options="apiKeyOptions"
                class="w-full"
                @change="applyFilters"
              />
              <Select
                v-model="filters.activeOnly"
                :options="activeOnlyOptions"
                class="w-full"
                @change="applyFilters"
              />
            </div>
            <div class="flex flex-wrap items-center justify-end gap-2">
              <AutoRefreshButton
                :enabled="autoRefresh.enabled.value"
                :interval-seconds="autoRefresh.intervalSeconds.value"
                :countdown="autoRefresh.countdown.value"
                :intervals="autoRefresh.intervals"
                @update:enabled="handleAutoRefreshEnabledChange"
                @update:interval="autoRefresh.setInterval"
              />
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="loading"
                :title="t('common.refresh')"
                @click="refreshPage"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="loading"
                @click="resetFilters"
              >
                {{ t('grokVideoConsole.filters.reset') }}
              </button>
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="jobs"
          :loading="loading"
          row-key="request_id"
          :sticky-first-column="false"
          :sticky-actions-column="true"
          :expandable-actions="false"
        >
          <template #cell-request_id="{ row }">
            <button
              type="button"
              class="flex max-w-[220px] flex-col text-left"
              @click="openDetails(row)"
            >
              <span class="truncate font-medium text-gray-900 dark:text-white">{{ row.request_id }}</span>
              <span class="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{{ row.prompt_preview || '-' }}</span>
            </button>
          </template>

          <template #cell-model="{ value }">
            <span class="block max-w-[180px] truncate text-sm text-gray-700 dark:text-gray-300" :title="value">{{ value || '-' }}</span>
          </template>

          <template #cell-api_key_id="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ apiKeyNameById.get(value) || '-' }}</span>
          </template>

          <template #cell-status="{ row }">
            <span :class="statusBadgeClass(row.status)" class="badge">
              {{ statusLabel(row.status) }}
            </span>
          </template>

          <template #cell-progress="{ row }">
            <div class="min-w-[160px]">
              <div class="flex items-center justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
                <span class="truncate">{{ progressText(row) }}</span>
                <span class="tabular-nums">{{ normalizedProgress(row.progress) }}%</span>
              </div>
              <div class="mt-1 h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
                <div
                  class="h-full rounded-full bg-primary-500 transition-all"
                  :style="{ width: `${normalizedProgress(row.progress)}%` }"
                />
              </div>
            </div>
          </template>

          <template #cell-submitted_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center justify-center gap-1">
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                @click="openDetails(row)"
              >
                <Icon name="eye" size="sm" />
                <span class="text-xs">{{ t('grokVideoConsole.actions.details') }}</span>
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                :disabled="refreshingRowIds.has(row.request_id)"
                @click="refreshSingle(row)"
              >
                <Icon name="refresh" size="sm" :class="refreshingRowIds.has(row.request_id) ? 'animate-spin' : ''" />
                <span class="text-xs">{{ t('grokVideoConsole.actions.refresh') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <div class="flex min-h-[260px] flex-col items-center justify-center py-6 md:min-h-[300px]">
              <Icon name="clock" size="xl" class="mb-4 h-12 w-12 text-gray-400 dark:text-dark-500" />
              <p class="text-lg font-medium text-gray-900 dark:text-gray-100">{{ t('grokVideoConsole.empty.title') }}</p>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.empty.description') }}</p>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog
      :show="detailsOpen"
      :title="t('grokVideoConsole.details.title')"
      width="wide"
      :close-on-click-outside="true"
      @close="closeDetails"
    >
      <div v-if="selectedJob" class="space-y-5">
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.requestId') }}</p>
            <p class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-white">{{ selectedJob.request_id }}</p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.status') }}</p>
            <span :class="statusBadgeClass(selectedJob.status)" class="badge mt-1">{{ statusLabel(selectedJob.status) }}</span>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.model') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ selectedJob.model || '-' }}</p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.apiKey') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ apiKeyNameById.get(selectedJob.api_key_id ?? -1) || '-' }}</p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.submittedAt') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatDateTime(selectedJob.submitted_at) }}</p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.lastUpdatedAt') }}</p>
            <p class="mt-1 text-sm text-gray-900 dark:text-white">{{ formatDateTime(selectedJob.updated_at) }}</p>
          </div>
        </div>

        <div>
          <div class="flex items-center justify-between gap-3">
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.progress') }}</p>
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ normalizedProgress(selectedJob.progress) }}%</span>
          </div>
          <div class="mt-2 h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-700">
            <div class="h-full rounded-full bg-primary-500 transition-all" :style="{ width: `${normalizedProgress(selectedJob.progress)}%` }" />
          </div>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ selectedJob.progress_text || t('grokVideoConsole.progress.defaultText') }}</p>
        </div>

        <div>
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.prompt') }}</p>
          <p class="mt-1 whitespace-pre-wrap break-words rounded-lg bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:bg-dark-800 dark:text-gray-300">{{ selectedJob.prompt_preview || '-' }}</p>
        </div>

        <div v-if="resultLinks(selectedJob).length > 0" class="space-y-2">
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.results') }}</p>
          <div class="space-y-2">
            <a
              v-for="(url, index) in resultLinks(selectedJob)"
              :key="`${selectedJob.request_id}-${index}`"
              :href="url"
              target="_blank"
              rel="noopener noreferrer"
              class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-sm text-primary-600 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:text-primary-400 dark:hover:bg-dark-800"
            >
              <span class="truncate">{{ url }}</span>
              <Icon name="chevronRight" size="sm" class="ml-2 flex-shrink-0" />
            </a>
          </div>
        </div>

        <div v-if="selectedJob.cover_image_url" class="space-y-2">
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.coverImage') }}</p>
          <a
            :href="selectedJob.cover_image_url"
            target="_blank"
            rel="noopener noreferrer"
            class="flex items-center justify-between rounded-lg border border-gray-200 px-3 py-2 text-sm text-primary-600 transition-colors hover:bg-gray-50 dark:border-dark-700 dark:text-primary-400 dark:hover:bg-dark-800"
          >
            <span class="truncate">{{ selectedJob.cover_image_url }}</span>
            <Icon name="chevronRight" size="sm" class="ml-2 flex-shrink-0" />
          </a>
        </div>

        <div v-if="selectedJob.last_error_message || selectedJob.last_error_code" class="space-y-2">
          <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('grokVideoConsole.details.error') }}</p>
          <div class="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700 dark:border-red-900/40 dark:bg-red-900/10 dark:text-red-300">
            <p v-if="selectedJob.last_error_code" class="font-medium">{{ selectedJob.last_error_code }}</p>
            <p class="mt-1 break-words">{{ selectedJob.last_error_message || '-' }}</p>
          </div>
        </div>
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import type { Column } from '@/components/common/types'
import SearchInput from '@/components/common/SearchInput.vue'
import Select, { type SelectOption } from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import grokVideoAPI, { type GrokVideoJob } from '@/api/grokVideo'
import { keysAPI } from '@/api'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const jobs = ref<GrokVideoJob[]>([])
const apiKeys = ref<ApiKey[]>([])
const selectedJob = ref<GrokVideoJob | null>(null)
const detailsOpen = ref(false)
const refreshingRowIds = ref(new Set<string>())
const autoRefreshInitialized = ref(false)
let listAbortController: AbortController | null = null

const pagination = reactive({
  page: 1,
  page_size: 20,
  total: 0,
  pages: 0,
})

const filters = reactive({
  model: '',
  status: '',
  apiKeyId: null as number | null,
  activeOnly: 'all' as 'all' | 'active_only',
})

const columns = computed<Column[]>(() => [
  { key: 'request_id', label: t('grokVideoConsole.columns.requestId') },
  { key: 'model', label: t('grokVideoConsole.columns.model') },
  { key: 'api_key_id', label: t('grokVideoConsole.columns.apiKey') },
  { key: 'status', label: t('grokVideoConsole.columns.status') },
  { key: 'progress', label: t('grokVideoConsole.columns.progress') },
  { key: 'submitted_at', label: t('grokVideoConsole.columns.submittedAt') },
  { key: 'updated_at', label: t('grokVideoConsole.columns.updatedAt') },
  { key: 'actions', label: t('grokVideoConsole.columns.actions'), class: 'text-center' },
])

const statusOptions = computed<SelectOption[]>(() => [
  { value: '', label: t('grokVideoConsole.filters.allStatuses') },
  { value: 'pending', label: t('grokVideoConsole.status.pending') },
  { value: 'running', label: t('grokVideoConsole.status.running') },
  { value: 'completed', label: t('grokVideoConsole.status.completed') },
  { value: 'failed', label: t('grokVideoConsole.status.failed') },
  { value: 'cancelled', label: t('grokVideoConsole.status.cancelled') },
])

const activeOnlyOptions = computed<SelectOption[]>(() => [
  { value: 'all', label: t('grokVideoConsole.filters.allJobs') },
  { value: 'active_only', label: t('grokVideoConsole.filters.activeOnly') },
])

const apiKeyOptions = computed<SelectOption[]>(() => [
  { value: null, label: t('grokVideoConsole.filters.allApiKeys') },
  ...apiKeys.value.map((key) => ({
    value: key.id,
    label: key.name,
  })),
])

const apiKeyNameById = computed(() => {
  const map = new Map<number, string>()
  for (const key of apiKeys.value) {
    map.set(key.id, key.name)
  }
  return map
})

const hasActiveJobs = computed(() => jobs.value.some((job) => !isTerminalStatus(job.status)))

const autoRefresh = useAutoRefresh({
  storageKey: 'grok-video-console-auto-refresh',
  intervals: [5, 10, 15, 30] as const,
  defaultInterval: 10,
  onRefresh: async () => {
    await refreshVisibleActiveJobs()
  },
  shouldPause: () => document.hidden || loading.value,
})

function normalizedProgress(progress: number): number {
  if (!Number.isFinite(progress)) return 0
  return Math.max(0, Math.min(100, Math.round(progress)))
}

function normalizeStatus(status: string): string {
  const normalized = (status || '').trim().toLowerCase()
  switch (normalized) {
    case 'queued':
    case 'created':
    case 'submitted':
      return 'pending'
    case 'in_progress':
    case 'processing':
      return 'running'
    case 'done':
    case 'succeeded':
    case 'success':
    case 'complete':
    case 'finished':
      return 'completed'
    case 'error':
      return 'failed'
    case 'canceled':
      return 'cancelled'
    default:
      return normalized
  }
}

function isTerminalStatus(status: string): boolean {
  return ['completed', 'failed', 'cancelled'].includes(normalizeStatus(status))
}

function statusLabel(status: string): string {
  const normalized = normalizeStatus(status)
  if (!normalized) return '-'
  const key = `grokVideoConsole.status.${normalized}`
  const translated = t(key)
  return translated === key ? normalized : translated
}

function statusBadgeClass(status: string): string {
  switch (normalizeStatus(status)) {
    case 'completed':
      return 'badge-success'
    case 'failed':
      return 'badge-danger'
    case 'cancelled':
      return 'badge-gray'
    case 'running':
      return 'badge-primary'
    default:
      return 'badge-warning'
  }
}

function progressText(job: GrokVideoJob): string {
  const text = job.progress_text?.trim()
  if (text) return text
  if (isTerminalStatus(job.status)) return statusLabel(job.status)
  return t('grokVideoConsole.progress.defaultText')
}

function formatDateTime(value?: string | null): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function resultLinks(job: GrokVideoJob): string[] {
  const urls = new Set<string>()
  if (job.result_url) urls.add(job.result_url)
  for (const url of job.result_urls || []) {
    if (url) urls.add(url)
  }
  return Array.from(urls)
}

async function loadApiKeys() {
  try {
    const response = await keysAPI.list(1, 100)
    apiKeys.value = response.items || []
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('grokVideoConsole.loadApiKeysFailed')))
  }
}

function buildListFilters() {
  return {
    status: filters.status || undefined,
    api_key_id: filters.apiKeyId,
    model: filters.model.trim() || undefined,
    active_only: filters.activeOnly === 'active_only',
  }
}

function isAbortLikeError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const abortError = error as { name?: string; code?: string }
  return abortError.name === 'AbortError' || abortError.name === 'CanceledError' || abortError.code === 'ERR_CANCELED'
}

async function loadJobs() {
  listAbortController?.abort()
  listAbortController = new AbortController()
  loading.value = true
  try {
    const response = await grokVideoAPI.list(pagination.page, pagination.page_size, buildListFilters(), {
      signal: listAbortController.signal,
    })
    jobs.value = response.items || []
    pagination.total = response.total || 0
    pagination.page = response.page || pagination.page
    pagination.page_size = response.page_size || pagination.page_size
    pagination.pages = response.pages || 0
    syncSelectedJobFromList()
    syncAutoRefreshState()
  } catch (error: unknown) {
    if (isAbortLikeError(error)) return
    appStore.showError(extractApiErrorMessage(error, t('grokVideoConsole.loadJobsFailed')))
  } finally {
    loading.value = false
  }
}

function syncSelectedJobFromList() {
  if (!selectedJob.value) return
  const next = jobs.value.find((job) => job.request_id === selectedJob.value?.request_id)
  if (next) {
    selectedJob.value = next
  }
}

function handleAutoRefreshEnabledChange(enabled: boolean) {
  autoRefreshInitialized.value = true
  autoRefresh.setEnabled(enabled)
}

function syncAutoRefreshState() {
  if (hasActiveJobs.value) {
    if (!autoRefreshInitialized.value) {
      autoRefreshInitialized.value = true
      autoRefresh.setEnabled(true)
      return
    }
    if (autoRefresh.enabled.value) {
      autoRefresh.resetCountdown()
      autoRefresh.start()
    }
    return
  }
  if (autoRefresh.enabled.value) {
    autoRefresh.setEnabled(false)
  } else {
    autoRefresh.stop()
  }
}

async function refreshVisibleActiveJobs() {
  const requestIds = jobs.value
    .filter((job) => !isTerminalStatus(job.status))
    .map((job) => job.request_id)

  if (requestIds.length === 0) {
    syncAutoRefreshState()
    return
  }

  try {
    const response = await grokVideoAPI.refresh({ request_ids: requestIds })
    mergeJobs(response.items || [])
    syncAutoRefreshState()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('grokVideoConsole.refreshFailed')))
  }
}

function mergeJobs(updatedJobs: GrokVideoJob[]) {
  if (updatedJobs.length === 0) return
  const byRequestId = new Map(updatedJobs.map((job) => [job.request_id, job]))
  jobs.value = jobs.value.map((job) => byRequestId.get(job.request_id) || job)
  syncSelectedJobFromList()
}

function applyFilters() {
  pagination.page = 1
  loadJobs()
}

function resetFilters() {
  filters.model = ''
  filters.status = ''
  filters.apiKeyId = null
  filters.activeOnly = 'all'
  applyFilters()
}

async function refreshPage() {
  await loadJobs()
}

async function refreshSingle(job: GrokVideoJob) {
  const next = new Set(refreshingRowIds.value)
  next.add(job.request_id)
  refreshingRowIds.value = next
  try {
    const response = await grokVideoAPI.refresh({ request_ids: [job.request_id] })
    mergeJobs(response.items || [])
    if (selectedJob.value?.request_id === job.request_id) {
      const refreshed = (response.items || []).find((item) => item.request_id === job.request_id)
      if (refreshed) {
        selectedJob.value = refreshed
      }
    }
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('grokVideoConsole.refreshFailed')))
  } finally {
    const done = new Set(refreshingRowIds.value)
    done.delete(job.request_id)
    refreshingRowIds.value = done
    syncAutoRefreshState()
  }
}

function openDetails(job: GrokVideoJob) {
  selectedJob.value = job
  detailsOpen.value = true
}

function closeDetails() {
  detailsOpen.value = false
}

function handlePageChange(page: number) {
  pagination.page = page
  loadJobs()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page = 1
  pagination.page_size = pageSize
  loadJobs()
}

onMounted(async () => {
  await loadApiKeys()
  await loadJobs()
})

onUnmounted(() => {
  listAbortController?.abort()
  autoRefresh.stop()
})
</script>
