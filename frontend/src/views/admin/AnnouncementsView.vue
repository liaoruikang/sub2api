<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Left: Search + Filters -->
          <div class="flex-1 sm:max-w-64">
            <input
              v-model="searchQuery"
              type="text"
              :placeholder="t('admin.announcements.searchAnnouncements')"
              class="input"
              @input="handleSearch"
            />
          </div>
          <Select
            v-model="filters.status"
            :options="statusFilterOptions"
            class="w-40"
            @change="handleStatusChange"
          />

          <!-- Right: Action buttons -->
          <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
            <button
              @click="loadAnnouncements"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
            <button
              @click="openPriceMonitorDialog"
              class="btn btn-secondary"
              title="分组价格监测设置"
            >
              <Icon name="cog" size="md" class="mr-1" />
              分组价格监测
              <span
                :class="[
                  'ml-1 rounded-full px-1.5 py-0.5 text-xs',
                  priceMonitor.enabled
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                    : 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
                ]"
              >{{ priceMonitor.enabled ? '已启用' : '已停用' }}</span>
              <span
                v-if="priceMonitor.enabled"
                class="ml-1 inline-flex min-w-20 items-center gap-1 whitespace-nowrap text-xs tabular-nums text-gray-500 dark:text-dark-400"
              >
                <Icon name="clock" size="xs" />
                {{ priceMonitorCountdownText }}
              </span>
            </button>
            <button @click="openCreateDialog" class="btn btn-primary">
              <Icon name="plus" size="md" class="mr-1" />
              {{ t('admin.announcements.createAnnouncement') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="announcements"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          @sort="handleSort"
        >
          <template #cell-title="{ value, row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-medium text-gray-900 dark:text-white">{{ value }}</span>
              </div>
              <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                <span>#{{ row.id }}</span>
                <span class="text-gray-300 dark:text-dark-700">·</span>
                <span>{{ formatDateTime(row.created_at) }}</span>
              </div>
            </div>
          </template>

          <template #cell-source="{ row }">
            <span :class="['badge', row.created_by == null ? 'badge-gray' : 'badge-primary']">
              {{ row.created_by == null ? '系统发布' : '管理员发布' }}
            </span>
          </template>

          <template #cell-status="{ value }">
            <span
              :class="[
                'badge',
                value === 'active'
                  ? 'badge-success'
                  : value === 'draft'
                    ? 'badge-gray'
                    : 'badge-warning'
              ]"
            >
              {{ statusLabel(value) }}
            </span>
          </template>

          <template #cell-notify_mode="{ row }">
            <span
              :class="[
                'badge',
                row.notify_mode === 'popup'
                  ? 'badge-warning'
                  : 'badge-gray'
              ]"
            >
              {{ row.notify_mode === 'popup' ? t('admin.announcements.notifyModeLabels.popup') : t('admin.announcements.notifyModeLabels.silent') }}
            </span>
          </template>

          <template #cell-targeting="{ row }">
            <span class="text-sm text-gray-600 dark:text-gray-300">
              {{ targetingSummary(row.targeting) }}
            </span>
          </template>

          <template #cell-timeRange="{ row }">
            <div class="text-sm text-gray-600 dark:text-gray-300">
              <div>
                <span class="font-medium">{{ t('admin.announcements.form.startsAt') }}:</span>
                <span class="ml-1">{{ row.starts_at ? formatDateTime(row.starts_at) : t('admin.announcements.timeImmediate') }}</span>
              </div>
              <div class="mt-0.5">
                <span class="font-medium">{{ t('admin.announcements.form.endsAt') }}:</span>
                <span class="ml-1">{{ row.ends_at ? formatDateTime(row.ends_at) : t('admin.announcements.timeNever') }}</span>
              </div>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center space-x-1">
              <button
                @click="openPreview(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.announcements.preview')"
              >
                <Icon name="eye" size="sm" />
              </button>
              <button
                @click="openReadStatus(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-blue-50 hover:text-blue-600 dark:hover:bg-blue-900/20 dark:hover:text-blue-400"
                :title="t('admin.announcements.readStatus')"
              >
                <Icon name="chartBar" size="sm" />
              </button>
              <button
                @click="openEditDialog(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-700 dark:hover:bg-dark-600 dark:hover:text-gray-300"
                :title="t('common.edit')"
              >
                <Icon name="edit" size="sm" />
              </button>
              <button
                @click="handleDelete(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :title="t('common.delete')"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('empty.noData')"
              :description="t('admin.announcements.failedToLoad')"
              :action-text="t('admin.announcements.createAnnouncement')"
              @action="openCreateDialog"
            />
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

    <!-- Create/Edit Dialog -->
    <BaseDialog
      :show="showEditDialog"
      :title="isEditing ? t('admin.announcements.editAnnouncement') : t('admin.announcements.createAnnouncement')"
      width="wide"
      @close="closeEdit"
    >
      <form id="announcement-form" @submit.prevent="handleSave" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.announcements.form.title') }}</label>
          <input v-model="form.title" type="text" class="input" required />
        </div>

        <div>
          <label class="input-label">{{ t('admin.announcements.form.content') }}</label>
          <textarea v-model="form.content" rows="6" class="input" required></textarea>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.status') }}</label>
            <Select v-model="form.status" :options="statusOptions" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.notifyMode') }}</label>
            <Select v-model="form.notify_mode" :options="notifyModeOptions" />
            <p class="input-hint">{{ t('admin.announcements.form.notifyModeHint') }}</p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.announcements.form.startsAt') }}</label>
            <input v-model="form.starts_at_str" type="datetime-local" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.startsAtHint') }}</p>
          </div>
          <div>
            <label class="input-label">{{ t('admin.announcements.form.endsAt') }}</label>
            <input v-model="form.ends_at_str" type="datetime-local" class="input" />
            <p class="input-hint">{{ t('admin.announcements.form.endsAtHint') }}</p>
          </div>
        </div>

        <AnnouncementTargetingEditor
          v-model="form.targeting"
          :groups="subscriptionGroups"
        />
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" @click="closeEdit" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" form="announcement-form" :disabled="saving" class="btn btn-primary">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.announcements.deleteAnnouncement')"
      :message="t('admin.announcements.deleteConfirm')"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      danger
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />

    <!-- Read Status Dialog -->
    <AnnouncementReadStatusDialog
      :show="showReadStatusDialog"
      :announcement-id="readStatusAnnouncementId"
      @close="showReadStatusDialog = false"
    />

    <AnnouncementPopup
      :announcement="previewAnnouncement"
      preview
      @close="previewAnnouncement = null"
    />
  </AppLayout>

  <BaseDialog
    :show="showPriceMonitorDialog"
    title="分组价格监测设置"
    width="wide"
    @close="closePriceMonitorDialog"
  >
    <div class="space-y-5">
      <div class="flex items-start justify-between gap-4 rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/50">
        <div>
          <h3 class="text-sm font-semibold text-gray-900 dark:text-white">自动检测分组倍率变化</h3>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">同一监测周期内的多个变化会合并为一条公告。</p>
          <p
            v-if="priceMonitor.enabled"
            class="mt-2 inline-flex items-center gap-1.5 text-sm font-medium tabular-nums text-primary-600 dark:text-primary-400"
          >
            <Icon name="clock" size="sm" />
            距离下一次监测：{{ priceMonitorCountdownText }}
          </p>
        </div>
        <label class="inline-flex shrink-0 cursor-pointer items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200">
          <input v-model="priceMonitor.enabled" type="checkbox" class="peer sr-only" />
          <span
            :class="[
              'flex h-5 w-9 items-center rounded-full p-0.5 transition-colors duration-300 ease-out dark:bg-dark-600',
              priceMonitor.enabled ? 'bg-primary-600' : 'bg-gray-300'
            ]"
          >
            <span
              :class="[
                'h-4 w-4 rounded-full bg-white shadow transition-transform duration-300 ease-out',
                priceMonitor.enabled ? 'translate-x-4' : 'translate-x-0'
              ]"
            />
          </span>
          {{ priceMonitor.enabled ? '已启用' : '已停用' }}
        </label>
      </div>

      <div class="grid grid-cols-1 gap-4 sm:grid-cols-4">
        <label class="input-label">监测周期（秒）
          <input v-model.number="priceMonitor.interval_seconds" type="number" min="5" class="input mt-1" />
        </label>
        <label class="input-label">公告有效天数
          <input v-model.number="priceMonitor.duration_days" type="number" min="1" step="1" class="input mt-1" />
          <span class="input-hint">保存后从当前时间开始计算结束时间</span>
        </label>
        <label class="input-label">公告状态
          <Select v-model="priceMonitor.status" :options="statusOptions" class="mt-1" />
        </label>
        <label class="input-label">通知方式
          <Select v-model="priceMonitor.notify_mode" :options="notifyModeOptions" class="mt-1" />
        </label>
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">监测分组</h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">选择全部分组，或指定需要监测的分组。</p>
          </div>
          <span class="text-xs text-gray-500 dark:text-dark-400">
            {{ monitorAllGroups ? '全部分组' : `已选 ${priceMonitor.group_ids.length} 个` }}
          </span>
        </div>
        <label
          :class="[
            'flex cursor-pointer items-center gap-3 rounded-xl border p-3 transition-colors',
            monitorAllGroups
              ? 'border-primary-300 bg-primary-50 dark:border-primary-700 dark:bg-primary-900/20'
              : 'border-gray-200 hover:border-gray-300 dark:border-dark-700 dark:hover:border-dark-600'
          ]"
        >
          <input v-model="monitorAllGroups" type="checkbox" class="peer sr-only" />
          <span class="flex h-5 w-5 shrink-0 items-center justify-center rounded-md border border-gray-300 bg-white text-white peer-checked:border-primary-600 peer-checked:bg-primary-600 dark:border-dark-500 dark:bg-dark-800">
            <Icon v-if="monitorAllGroups" name="check" size="sm" />
          </span>
          <span class="text-sm font-medium text-gray-800 dark:text-gray-100">全部分组</span>
        </label>
        <div v-if="!monitorAllGroups" class="mt-3">
          <GroupSelector
            v-model="priceMonitor.group_ids"
            :groups="allGroups"
            :searchable="true"
          />
        </div>
      </div>

      <AnnouncementTargetingEditor v-model="priceMonitor.targeting" :groups="subscriptionGroups" />
    </div>
    <template #footer>
      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary" @click="closePriceMonitorDialog">取消</button>
        <button type="button" class="btn btn-primary" :disabled="priceMonitorSaving" @click="savePriceMonitor">
          {{ priceMonitorSaving ? t('common.saving') : '保存监测设置' }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { adminAPI } from '@/api/admin'
import { formatDateTime, formatDateTimeLocalInput, parseDateTimeLocalInput } from '@/utils/format'
import type { AdminGroup, Announcement, AnnouncementTargeting, GroupPriceMonitorConfig } from '@/types'
import type { Column } from '@/components/common/types'

import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'

import AnnouncementTargetingEditor from '@/components/admin/announcements/AnnouncementTargetingEditor.vue'
import AnnouncementReadStatusDialog from '@/components/admin/announcements/AnnouncementReadStatusDialog.vue'
import AnnouncementPopup from '@/components/common/AnnouncementPopup.vue'

const { t } = useI18n()
const appStore = useAppStore()

const announcements = ref<Announcement[]>([])
const loading = ref(false)

const filters = reactive({
  status: '',
})
const searchQuery = ref('')

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const statusFilterOptions = computed(() => [
  { value: '', label: t('admin.announcements.allStatus') },
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const statusOptions = computed(() => [
  { value: 'draft', label: t('admin.announcements.statusLabels.draft') },
  { value: 'active', label: t('admin.announcements.statusLabels.active') },
  { value: 'archived', label: t('admin.announcements.statusLabels.archived') }
])

const notifyModeOptions = computed(() => [
  { value: 'silent', label: t('admin.announcements.notifyModeLabels.silent') },
  { value: 'popup', label: t('admin.announcements.notifyModeLabels.popup') }
])

const columns = computed<Column[]>(() => [
  { key: 'title', label: t('admin.announcements.columns.title'), sortable: true },
  { key: 'source', label: '发布者' },
  { key: 'status', label: t('admin.announcements.columns.status'), sortable: true },
  { key: 'notify_mode', label: t('admin.announcements.columns.notifyMode'), sortable: true },
  { key: 'targeting', label: t('admin.announcements.columns.targeting') },
  { key: 'timeRange', label: t('admin.announcements.columns.timeRange') },
  { key: 'created_at', label: t('admin.announcements.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.announcements.columns.actions') }
])

const statusLabel = (status: string) => {
  if (status === 'draft') return t('admin.announcements.statusLabels.draft')
  if (status === 'active') return t('admin.announcements.statusLabels.active')
  if (status === 'archived') return t('admin.announcements.statusLabels.archived')
  return status
}

const targetingSummary = (targeting: AnnouncementTargeting) => {
  const anyOf = targeting?.any_of ?? []
  if (!anyOf || anyOf.length === 0) return t('admin.announcements.targetingSummaryAll')
  return t('admin.announcements.targetingSummaryCustom', { groups: anyOf.length })
}

// ===== CRUD / list =====
let currentController: AbortController | null = null

async function loadAnnouncements() {
  currentController?.abort()
  const requestController = new AbortController()
  currentController = requestController
  const { signal } = requestController

  try {
    loading.value = true
    const res = await adminAPI.announcements.list(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      search: searchQuery.value || undefined,
      sort_by: sortState.sort_by,
      sort_order: sortState.sort_order
    }, { signal })

    if (signal.aborted || currentController !== requestController) return

    announcements.value = res.items
    pagination.total = res.total
    pagination.pages = res.pages
    pagination.page = res.page
    pagination.page_size = res.page_size
  } catch (error: any) {
    if (
      signal.aborted ||
      currentController !== requestController ||
      error?.name === 'AbortError' ||
      error?.code === 'ERR_CANCELED'
    ) {
      return
    }
    console.error('Error loading announcements:', error)
    appStore.showError(error.response?.data?.detail || t('admin.announcements.failedToLoad'))
  } finally {
    if (currentController === requestController) {
      loading.value = false
      currentController = null
    }
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadAnnouncements()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  loadAnnouncements()
}

function handleStatusChange() {
  pagination.page = 1
  loadAnnouncements()
}

function handleSort(key: string, order: 'asc' | 'desc') {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadAnnouncements()
}

let searchDebounceTimer: number | null = null
function handleSearch() {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  searchDebounceTimer = window.setTimeout(() => {
    pagination.page = 1
    loadAnnouncements()
  }, 300)
}

// ===== Create/Edit dialog =====
const showEditDialog = ref(false)
const saving = ref(false)
const editingAnnouncement = ref<Announcement | null>(null)

const isEditing = computed(() => !!editingAnnouncement.value)

const form = reactive({
  title: '',
  content: '',
  status: 'draft',
  notify_mode: 'silent',
  starts_at_str: '',
  ends_at_str: '',
  targeting: { any_of: [] } as AnnouncementTargeting
})

const subscriptionGroups = ref<AdminGroup[]>([])
const allGroups = ref<AdminGroup[]>([])
const priceMonitorSaving = ref(false)
const showPriceMonitorDialog = ref(false)
const monitorAllGroups = ref(true)
const priceMonitorNow = ref(Date.now())
const priceMonitorNextCheckDeadline = ref<number | null>(null)
let priceMonitorCountdownTimer: number | null = null
let priceMonitorScheduleSyncing = false
let lastPriceMonitorScheduleSyncAt = 0
const priceMonitor = reactive<{
  enabled: boolean
  group_ids: number[]
  interval_seconds: number
  status: 'draft' | 'active' | 'archived'
  notify_mode: 'silent' | 'popup'
  duration_days: number
  targeting: AnnouncementTargeting
}>({
  enabled: false,
  group_ids: [],
  interval_seconds: 60,
  status: 'active',
  notify_mode: 'popup',
  duration_days: 3,
  targeting: { any_of: [] }
})

const priceMonitorRemainingSeconds = computed(() => {
  if (!priceMonitor.enabled || priceMonitorNextCheckDeadline.value === null) return null
  return Math.max(0, Math.ceil((priceMonitorNextCheckDeadline.value - priceMonitorNow.value) / 1000))
})

const priceMonitorCountdownText = computed(() => {
  const seconds = priceMonitorRemainingSeconds.value
  if (seconds === null) return '正在安排'
  if (seconds <= 0) return '正在监测'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const rest = seconds % 60
  if (hours > 0) return `${hours}小时 ${minutes}分 ${rest}秒`
  if (minutes > 0) return `${minutes}分 ${rest}秒`
  return `${rest}秒`
})

function applyPriceMonitorSchedule(config: GroupPriceMonitorConfig) {
  if (!config.enabled || !config.next_check_at) {
    priceMonitorNextCheckDeadline.value = null
    return
  }
  const nextCheckAt = new Date(config.next_check_at).getTime()
  const serverTime = config.server_time ? new Date(config.server_time).getTime() : Date.now()
  if (!Number.isFinite(nextCheckAt) || !Number.isFinite(serverTime)) {
    priceMonitorNextCheckDeadline.value = null
    return
  }
  priceMonitorNextCheckDeadline.value = Date.now() + Math.max(0, nextCheckAt - serverTime)
}

async function loadSubscriptionGroups() {
  try {
    const all = await adminAPI.groups.getAll()
    allGroups.value = all || []
    subscriptionGroups.value = (all || []).filter((g) => g.subscription_type === 'subscription')
  } catch (error: any) {
    console.error('Error loading groups:', error)
    // not fatal
  }
}

function fillPriceMonitor(config: GroupPriceMonitorConfig) {
  priceMonitor.enabled = config.enabled
  priceMonitor.group_ids = [...(config.group_ids || [])]
  monitorAllGroups.value = priceMonitor.group_ids.length === 0
  priceMonitor.interval_seconds = config.interval_seconds || 60
  priceMonitor.status = config.status || 'active'
  priceMonitor.notify_mode = config.notify_mode || 'popup'
  const configuredDays = Number(config.duration_days)
  const legacyDays = config.starts_at && config.ends_at
    ? Math.round((new Date(config.ends_at).getTime() - new Date(config.starts_at).getTime()) / 86400000)
    : 0
  priceMonitor.duration_days = configuredDays > 0 ? configuredDays : Math.max(1, legacyDays || 3)
  priceMonitor.targeting = config.targeting || { any_of: [] }
  applyPriceMonitorSchedule(config)
}

function openPriceMonitorDialog() {
  showPriceMonitorDialog.value = true
}

function closePriceMonitorDialog() {
  if (!priceMonitorSaving.value) showPriceMonitorDialog.value = false
}

async function loadPriceMonitor() {
  try { fillPriceMonitor(await adminAPI.announcements.getPriceMonitor()) } catch (error) { console.error('Error loading price monitor:', error) }
}

async function syncPriceMonitorSchedule() {
  if (priceMonitorScheduleSyncing) return
  priceMonitorScheduleSyncing = true
  try {
    applyPriceMonitorSchedule(await adminAPI.announcements.getPriceMonitor())
  } catch (error) {
    console.error('Error syncing price monitor schedule:', error)
  } finally {
    priceMonitorScheduleSyncing = false
  }
}

function tickPriceMonitorCountdown() {
  priceMonitorNow.value = Date.now()
  if (!priceMonitor.enabled) return
  if (priceMonitorRemainingSeconds.value !== null && priceMonitorRemainingSeconds.value !== 0) return
  if (priceMonitorNow.value - lastPriceMonitorScheduleSyncAt < 2000) return
  lastPriceMonitorScheduleSyncAt = priceMonitorNow.value
  void syncPriceMonitorSchedule()
}

async function savePriceMonitor() {
  if (priceMonitor.interval_seconds < 5) { appStore.showError('监测周期不能小于 5 秒'); return }
  if (priceMonitor.enabled && priceMonitor.duration_days < 1) { appStore.showError('公告有效天数不能小于 1 天'); return }
  priceMonitorSaving.value = true
  try {
    const saved = await adminAPI.announcements.updatePriceMonitor({
      enabled: priceMonitor.enabled,
      group_ids: monitorAllGroups.value ? [] : [...priceMonitor.group_ids],
      interval_seconds: priceMonitor.interval_seconds,
      status: priceMonitor.status,
      notify_mode: priceMonitor.notify_mode,
      duration_days: priceMonitor.duration_days,
      targeting: priceMonitor.targeting
    })
    fillPriceMonitor(saved)
    appStore.showSuccess(t('common.success'))
    showPriceMonitorDialog.value = false
  } catch (error: any) { appStore.showError(error.response?.data?.detail || '保存监测设置失败') } finally { priceMonitorSaving.value = false }
}

function resetForm() {
  form.title = ''
  form.content = ''
  form.status = 'draft'
  form.notify_mode = 'silent'
  form.starts_at_str = ''
  form.ends_at_str = ''
  form.targeting = { any_of: [] }
}

function fillFormFromAnnouncement(a: Announcement) {
  form.title = a.title
  form.content = a.content
  form.status = a.status
  form.notify_mode = a.notify_mode || 'silent'

  // Backend returns RFC3339 strings
  form.starts_at_str = a.starts_at ? formatDateTimeLocalInput(Math.floor(new Date(a.starts_at).getTime() / 1000)) : ''
  form.ends_at_str = a.ends_at ? formatDateTimeLocalInput(Math.floor(new Date(a.ends_at).getTime() / 1000)) : ''

  form.targeting = a.targeting ?? { any_of: [] }
}

function openCreateDialog() {
  editingAnnouncement.value = null
  resetForm()
  showEditDialog.value = true
}

function openEditDialog(row: Announcement) {
  editingAnnouncement.value = row
  fillFormFromAnnouncement(row)
  showEditDialog.value = true
}

function closeEdit() {
  showEditDialog.value = false
  editingAnnouncement.value = null
}

function buildCreatePayload() {
  const startsAt = parseDateTimeLocalInput(form.starts_at_str)
  const endsAt = parseDateTimeLocalInput(form.ends_at_str)

  return {
    title: form.title,
    content: form.content,
    status: form.status as any,
    notify_mode: form.notify_mode as any,
    targeting: form.targeting,
    starts_at: startsAt ?? undefined,
    ends_at: endsAt ?? undefined
  }
}

function buildUpdatePayload(original: Announcement) {
  const payload: any = {}

  if (form.title !== original.title) payload.title = form.title
  if (form.content !== original.content) payload.content = form.content
  if (form.status !== original.status) payload.status = form.status
  if (form.notify_mode !== (original.notify_mode || 'silent')) payload.notify_mode = form.notify_mode

  // starts_at / ends_at: distinguish unchanged vs clear(0) vs set
  const originalStarts = original.starts_at ? Math.floor(new Date(original.starts_at).getTime() / 1000) : null
  const originalEnds = original.ends_at ? Math.floor(new Date(original.ends_at).getTime() / 1000) : null

  const newStarts = parseDateTimeLocalInput(form.starts_at_str)
  const newEnds = parseDateTimeLocalInput(form.ends_at_str)

  if (newStarts !== originalStarts) {
    payload.starts_at = newStarts === null ? 0 : newStarts
  }
  if (newEnds !== originalEnds) {
    payload.ends_at = newEnds === null ? 0 : newEnds
  }

  // targeting: do shallow compare by JSON
  if (JSON.stringify(form.targeting ?? {}) !== JSON.stringify(original.targeting ?? {})) {
    payload.targeting = form.targeting
  }

  return payload
}

async function handleSave() {
  // Frontend validation for targeting (to avoid ANNOUNCEMENT_INVALID_TARGET)
  const anyOf = form.targeting?.any_of ?? []
  if (anyOf.length > 50) {
    appStore.showError(t('admin.announcements.failedToCreate'))
    return
  }
  for (const g of anyOf) {
    const allOf = g?.all_of ?? []
    if (allOf.length > 50) {
      appStore.showError(t('admin.announcements.failedToCreate'))
      return
    }
  }

  saving.value = true
  try {
    if (!editingAnnouncement.value) {
      const payload = buildCreatePayload()
      await adminAPI.announcements.create(payload)
      appStore.showSuccess(t('common.success'))
      showEditDialog.value = false
      await loadAnnouncements()
      return
    }

    const original = editingAnnouncement.value
    const payload = buildUpdatePayload(original)
    await adminAPI.announcements.update(original.id, payload)
    appStore.showSuccess(t('common.success'))
    showEditDialog.value = false
    editingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to save announcement:', error)
    appStore.showError(error.response?.data?.detail || (editingAnnouncement.value ? t('admin.announcements.failedToUpdate') : t('admin.announcements.failedToCreate')))
  } finally {
    saving.value = false
  }
}

// ===== Delete =====
const showDeleteDialog = ref(false)
const deletingAnnouncement = ref<Announcement | null>(null)

function handleDelete(row: Announcement) {
  deletingAnnouncement.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deletingAnnouncement.value) return

  try {
    await adminAPI.announcements.delete(deletingAnnouncement.value.id)
    appStore.showSuccess(t('common.success'))
    showDeleteDialog.value = false
    deletingAnnouncement.value = null
    await loadAnnouncements()
  } catch (error: any) {
    console.error('Failed to delete announcement:', error)
    appStore.showError(error.response?.data?.detail || t('admin.announcements.failedToDelete'))
  }
}

// ===== Read status =====
const showReadStatusDialog = ref(false)
const readStatusAnnouncementId = ref<number | null>(null)
const previewAnnouncement = ref<Announcement | null>(null)

function openPreview(row: Announcement) {
  previewAnnouncement.value = row
}

function openReadStatus(row: Announcement) {
  readStatusAnnouncementId.value = row.id
  showReadStatusDialog.value = true
}

onMounted(async () => {
  await loadSubscriptionGroups()
  await loadPriceMonitor()
  await loadAnnouncements()
  priceMonitorCountdownTimer = window.setInterval(tickPriceMonitorCountdown, 1000)
})

onUnmounted(() => {
  if (searchDebounceTimer) window.clearTimeout(searchDebounceTimer)
  if (priceMonitorCountdownTimer) window.clearInterval(priceMonitorCountdownTimer)
  currentController?.abort()
})
</script>
