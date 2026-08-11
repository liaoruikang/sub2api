<template>
  <BaseDialog
    :show="show"
    :title="mode === 'members' ? t('admin.users.tagUsersTitle', { name: tag?.name }) : t('admin.users.addUsersToTagTitle', { name: tag?.name })"
    width="extra-wide"
    :z-index="60"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
          <span class="rounded-full bg-primary-50 px-2.5 py-1 font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
            {{ tag?.name }}
          </span>
          <span v-if="mode === 'add'">{{ t('admin.users.selectedUsers', { count: selectedCount }) }}</span>
        </div>
        <div class="flex gap-2">
          <button
            v-if="mode === 'members'"
            type="button"
            class="btn btn-primary"
            @click="startAdding"
          >
            {{ t('admin.users.addUsers') }}
          </button>
          <button v-else type="button" class="btn btn-secondary" @click="showMembers">
            {{ t('admin.users.viewTagUsers') }}
          </button>
        </div>
      </div>

      <div class="flex flex-col gap-3 sm:flex-row">
        <input
          v-model="search"
          type="search"
          class="input min-w-0 flex-1"
          :placeholder="t('admin.users.searchTagUsers')"
          @input="handleSearchInput"
        />
        <div class="w-full sm:w-36">
          <Select
            :model-value="status"
            :options="statusOptions"
            :aria-label="t('admin.users.tagUserStatus')"
            @update:model-value="handleStatusChange"
          />
        </div>
      </div>

      <div v-if="errorMessage" class="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMessage }}
      </div>

      <DataTable
        :columns="columns"
        :data="rows"
        :loading="loading"
        :selectable="mode === 'add'"
        :selected-keys="selectedIds"
        row-key="id"
        :selection-label="selectionLabel"
        @update:selected-keys="handleSelectedKeysUpdate"
      >
        <template #cell-username="{ value }">
          <span class="text-gray-600 dark:text-gray-300">{{ value || t('admin.users.noUsername') }}</span>
        </template>
        <template #cell-status="{ value }">
          <span
            class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
            :class="value === 'active'
              ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300'
              : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'"
          >
            {{ value === 'active' ? t('common.active') : t('admin.users.disabled') }}
          </span>
        </template>
        <template #empty>
          <div class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ mode === 'members' ? t('admin.users.noTagUsers') : t('admin.users.noUsersToAdd') }}
          </div>
        </template>
      </DataTable>

      <Pagination
        v-if="total > 0"
        :total="total"
        :page="page"
        :page-size="pageSize"
        :show-jump="true"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />

      <div v-if="mode === 'add'" class="flex flex-col gap-3 border-t border-gray-200 pt-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-sm text-gray-500 dark:text-dark-400">
          {{ t('admin.users.tagUsersLimit', { count: selectedCount }) }}
        </p>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="saving || selectedCount === 0 || selectedCount > maxUsers"
          @click="addSelectedUsers"
        >
          {{ saving ? t('admin.users.saving') : t('admin.users.addSelectedUsers') }}
        </button>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { AdminUser, UserTag } from '@/types'
import type { TagUser } from '@/api/admin/tags'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import { useTableSelection } from '@/composables/useTableSelection'

const props = defineProps<{
  show: boolean
  tag: UserTag | null
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'success'): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const maxUsers = 500
const mode = ref<'members' | 'add'>('members')
const rows = ref<Array<TagUser | AdminUser>>([])
const search = ref('')
const status = ref<'' | 'active' | 'disabled'>('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')

const {
  selectedIds,
  selectedCount,
  setSelectedIds,
  clear: clearSelection
} = useTableSelection({
  rows,
  getId: (row) => row.id
})

const columns = computed(() => [
  { key: 'email', label: t('admin.users.columns.email') },
  { key: 'username', label: t('admin.users.columns.username') },
  { key: 'status', label: t('admin.users.columns.status') }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.users.allStatus') },
  { value: 'active', label: t('common.active') },
  { value: 'disabled', label: t('admin.users.disabled') }
])

const selectionLabel = (row: TagUser | AdminUser) =>
  t('admin.users.bulkLimits.selectUser', { email: row.email })

const getErrorMessage = (error: any, fallback: string) =>
  error?.response?.data?.detail || error?.message || fallback

const loadRows = async () => {
  if (!props.tag || !props.show) return
  loading.value = true
  errorMessage.value = ''
  try {
    if (mode.value === 'members') {
      const result = await adminAPI.tags.getTagUsers(props.tag.id, page.value, pageSize.value, {
        search: search.value || undefined,
        status: status.value || undefined
      })
      rows.value = result.items
      total.value = result.total
      page.value = result.page
      pageSize.value = result.page_size
    } else {
      const result = await adminAPI.users.list(page.value, pageSize.value, {
        search: search.value || undefined,
        status: status.value || undefined
      })
      rows.value = result.items
      total.value = result.total
      page.value = result.page
      pageSize.value = result.page_size
    }
  } catch (error: any) {
    rows.value = []
    total.value = 0
    errorMessage.value = getErrorMessage(error, t('admin.users.failedToLoadTagUsers'))
  } finally {
    loading.value = false
  }
}

const resetState = () => {
  mode.value = 'members'
  rows.value = []
  search.value = ''
  status.value = ''
  page.value = 1
  pageSize.value = 20
  total.value = 0
  errorMessage.value = ''
  saving.value = false
  clearSelection()
}

const startAdding = () => {
  mode.value = 'add'
  page.value = 1
  search.value = ''
  status.value = 'active'
  errorMessage.value = ''
  clearSelection()
  void loadRows()
}

const showMembers = () => {
  mode.value = 'members'
  page.value = 1
  search.value = ''
  status.value = ''
  errorMessage.value = ''
  clearSelection()
  void loadRows()
}

let searchTimer: ReturnType<typeof setTimeout> | null = null
const handleSearchInput = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    page.value = 1
    void loadRows()
  }, 250)
}

const handleStatusChange = (value: string | number | boolean | null) => {
  status.value = value === 'active' || value === 'disabled' ? value : ''
  page.value = 1
  void loadRows()
}

const handlePageChange = (nextPage: number) => {
  page.value = nextPage
  void loadRows()
}

const handlePageSizeChange = (nextPageSize: number) => {
  pageSize.value = nextPageSize
  page.value = 1
  void loadRows()
}

const handleSelectedKeysUpdate = (keys: Array<string | number>) => {
  setSelectedIds(keys.filter((key): key is number => typeof key === 'number'))
}

const addSelectedUsers = async () => {
  if (!props.tag || saving.value || selectedCount.value === 0 || selectedCount.value > maxUsers) return
  saving.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.tags.addUsersToTag(props.tag.id, selectedIds.value)
    appStore.showSuccess(t('admin.users.tagUsersAdded', { count: result.affected }))
    emit('success')
    clearSelection()
    mode.value = 'members'
    page.value = 1
    search.value = ''
    status.value = ''
    await loadRows()
    if (result.affected > 0) {
      errorMessage.value = ''
    }
  } catch (error: any) {
    errorMessage.value = getErrorMessage(error, t('admin.users.failedToAddTagUsers'))
  } finally {
    saving.value = false
  }
}

const handleClose = () => {
  if (searchTimer) clearTimeout(searchTimer)
  resetState()
  emit('close')
}

watch(
  () => [props.show, props.tag?.id] as const,
  ([show]) => {
    if (show) {
      resetState()
      void loadRows()
    }
  }
)
</script>
