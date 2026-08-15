<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <div class="relative w-full md:w-72">
            <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              v-model="filters.search"
              type="text"
              class="input pl-10"
              :placeholder="t('admin.affiliates.withdrawals.searchPlaceholder')"
              @input="debounceLoad"
            />
          </div>
          <div class="w-full sm:w-40">
            <Select
              v-model="filters.status"
              :options="statusOptions"
              :aria-label="t('admin.affiliates.withdrawals.statusFilter')"
              @change="reloadFromFirstPage"
            />
          </div>
          <input v-model="filters.start_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.startAt')" @change="reloadFromFirstPage" />
          <input v-model="filters.end_at" type="date" class="input w-full sm:w-44" :title="t('admin.affiliates.records.endAt')" @change="reloadFromFirstPage" />
          <button class="btn btn-secondary px-2 md:px-3" :disabled="loading" :title="t('common.refresh')" @click="loadRecords">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>
      </template>

      <template #table>
        <DataTable
          :columns="columns"
          :data="records"
          :loading="loading"
          :server-side-sort="true"
          default-sort-key="created_at"
          default-sort-order="desc"
          sort-storage-key="admin-affiliate-withdrawals-table-sort"
          @sort="handleSort"
        >
          <template #cell-request_no="{ row }">
            <span class="font-mono text-xs text-gray-700 dark:text-gray-300">{{ row.request_no }}</span>
          </template>
          <template #cell-user="{ row }">
            <div class="space-y-0.5">
              <div class="font-mono text-xs text-gray-500 dark:text-dark-400">#{{ row.user_id }}</div>
              <div class="max-w-56 truncate font-medium text-gray-900 dark:text-white">{{ row.user_email || '-' }}</div>
              <div class="max-w-56 truncate text-xs text-gray-500 dark:text-dark-400">{{ row.username || '-' }}</div>
            </div>
          </template>
          <template #cell-amount="{ row }">
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">{{ formatMoney(row.amount) }}</p>
              <p class="text-xs text-gray-400">{{ t('admin.affiliates.withdrawals.feeDetail', { rate: formatNumber(row.fee_rate), fee: formatMoney(row.fee_amount) }) }}</p>
            </div>
          </template>
          <template #cell-payout_amount="{ row }">
            <span class="font-semibold text-emerald-600 dark:text-emerald-400">{{ formatMoney(row.payout_amount) }}</span>
          </template>
          <template #cell-alipay_account="{ row }">
            <span class="font-mono text-sm text-gray-700 dark:text-gray-300">{{ row.alipay_account || row.alipay_account_masked || '-' }}</span>
          </template>
          <template #cell-status="{ row }">
            <div class="space-y-1">
              <span :class="statusClass(row.status)" class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium">
                {{ t(`admin.affiliates.withdrawals.status.${row.status}`) }}
              </span>
              <p v-if="row.reject_reason" class="max-w-52 whitespace-normal text-xs text-red-500">{{ row.reject_reason }}</p>
            </div>
          </template>
          <template #cell-operator="{ row }">
            <div class="space-y-0.5 text-sm text-gray-700 dark:text-gray-300">
              <p>{{ row.operator_email || '-' }}</p>
              <p class="text-xs text-gray-400">{{ formatDateTime(row.processed_at) }}</p>
            </div>
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ formatDateTime(row.created_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div v-if="row.status === 'pending'" class="flex flex-wrap justify-end gap-2">
              <button type="button" class="btn btn-primary btn-sm" :disabled="processingID === row.id" @click="openComplete(row)">
                <Icon name="check" size="sm" />
                <span>{{ t('admin.affiliates.withdrawals.complete') }}</span>
              </button>
              <button type="button" class="btn btn-secondary btn-sm text-red-600 dark:text-red-400" :disabled="processingID === row.id" @click="openReject(row)">
                <Icon name="x" size="sm" />
                <span>{{ t('admin.affiliates.withdrawals.reject') }}</span>
              </button>
            </div>
            <span v-else class="text-xs text-gray-400">-</span>
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

    <ConfirmDialog
      :show="Boolean(completingRecord)"
      :title="t('admin.affiliates.withdrawals.completeTitle')"
      :message="completeMessage"
      :confirm-text="t('admin.affiliates.withdrawals.confirmPaid')"
      @confirm="confirmComplete"
      @cancel="completingRecord = null"
    />

    <BaseDialog
      :show="Boolean(rejectingRecord)"
      :title="t('admin.affiliates.withdrawals.rejectTitle')"
      width="normal"
      @close="closeReject"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600 dark:text-gray-400">
          {{ t('admin.affiliates.withdrawals.rejectMessage', { amount: formatMoney(rejectingRecord?.amount || 0) }) }}
        </p>
        <div>
          <label class="input-label" for="withdrawal-reject-reason">{{ t('admin.affiliates.withdrawals.rejectReason') }}</label>
          <textarea
            id="withdrawal-reject-reason"
            v-model="rejectReason"
            rows="4"
            maxlength="500"
            class="input resize-none"
            :placeholder="t('admin.affiliates.withdrawals.rejectReasonPlaceholder')"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" :disabled="processingID !== null" @click="closeReject">{{ t('common.cancel') }}</button>
          <button type="button" class="btn btn-danger" :disabled="!rejectReason.trim() || processingID !== null" @click="confirmReject">
            {{ t('admin.affiliates.withdrawals.confirmReject') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'
import { affiliatesAPI, type AffiliateWithdrawalRecord, type AffiliateWithdrawalStatus, type ListAffiliateWithdrawalsParams } from '@/api/admin/affiliates'
import { useAppStore } from '@/stores/app'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime as formatDisplayDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const records = ref<AffiliateWithdrawalRecord[]>([])
const filters = reactive<{ search: string; status: AffiliateWithdrawalStatus | ''; start_at: string; end_at: string }>({
  search: '',
  status: '',
  start_at: '',
  end_at: '',
})
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const sortState = reactive({ sort_by: 'created_at', sort_order: 'desc' as 'asc' | 'desc' })
const completingRecord = ref<AffiliateWithdrawalRecord | null>(null)
const rejectingRecord = ref<AffiliateWithdrawalRecord | null>(null)
const rejectReason = ref('')
const processingID = ref<number | null>(null)
let debounceTimer: ReturnType<typeof setTimeout> | null = null

const statusOptions = computed(() => [
  { value: '', label: t('admin.affiliates.withdrawals.status.all') },
  { value: 'pending', label: t('admin.affiliates.withdrawals.status.pending') },
  { value: 'paid', label: t('admin.affiliates.withdrawals.status.paid') },
  { value: 'rejected', label: t('admin.affiliates.withdrawals.status.rejected') },
])

const columns = computed<Column[]>(() => [
  { key: 'request_no', label: t('admin.affiliates.withdrawals.columns.requestNo'), sortable: true },
  { key: 'user', label: t('admin.affiliates.withdrawals.columns.user'), sortable: true },
  { key: 'amount', label: t('admin.affiliates.withdrawals.columns.amount'), sortable: true },
  { key: 'payout_amount', label: t('admin.affiliates.withdrawals.columns.payout'), sortable: true },
  { key: 'alipay_account', label: t('admin.affiliates.withdrawals.columns.alipay') },
  { key: 'status', label: t('admin.affiliates.withdrawals.columns.status'), sortable: true },
  { key: 'operator', label: t('admin.affiliates.withdrawals.columns.operator') },
  { key: 'created_at', label: t('admin.affiliates.withdrawals.columns.createdAt'), sortable: true },
  { key: 'actions', label: t('admin.affiliates.withdrawals.columns.actions'), class: 'text-right' },
])

const completeMessage = computed(() => {
  const record = completingRecord.value
  if (!record) return ''
  return t('admin.affiliates.withdrawals.completeMessage', {
    account: record.alipay_account || record.alipay_account_masked,
    payout: formatMoney(record.payout_amount),
    amount: formatMoney(record.amount),
  })
})

function userTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone
  } catch {
    return 'UTC'
  }
}

function buildParams(): ListAffiliateWithdrawalsParams {
  return {
    page: pagination.page,
    page_size: pagination.page_size,
    search: filters.search.trim() || undefined,
    status: filters.status,
    start_at: filters.start_at || undefined,
    end_at: filters.end_at || undefined,
    sort_by: sortState.sort_by,
    sort_order: sortState.sort_order,
    timezone: userTimezone(),
  }
}

async function loadRecords(): Promise<void> {
  loading.value = true
  try {
    const result = await affiliatesAPI.listWithdrawalRecords(buildParams())
    records.value = result.items || []
    pagination.total = result.total || 0
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    loading.value = false
  }
}

function debounceLoad(): void {
  if (debounceTimer) clearTimeout(debounceTimer)
  debounceTimer = setTimeout(reloadFromFirstPage, 300)
}

function reloadFromFirstPage(): void {
  pagination.page = 1
  void loadRecords()
}

function handlePageChange(page: number): void {
  pagination.page = page
  void loadRecords()
}

function handlePageSizeChange(pageSize: number): void {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadRecords()
}

function handleSort(key: string, order: 'asc' | 'desc'): void {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  void loadRecords()
}

function openComplete(record: AffiliateWithdrawalRecord): void {
  completingRecord.value = record
}

function openReject(record: AffiliateWithdrawalRecord): void {
  rejectingRecord.value = record
  rejectReason.value = ''
}

function closeReject(): void {
  if (processingID.value !== null) return
  rejectingRecord.value = null
  rejectReason.value = ''
}

async function confirmComplete(): Promise<void> {
  const record = completingRecord.value
  if (!record || processingID.value !== null) return
  processingID.value = record.id
  try {
    await affiliatesAPI.completeWithdrawal(record.id)
    completingRecord.value = null
    appStore.showSuccess(t('admin.affiliates.withdrawals.completeSuccess'))
    await loadRecords()
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    processingID.value = null
  }
}

async function confirmReject(): Promise<void> {
  const record = rejectingRecord.value
  const reason = rejectReason.value.trim()
  if (!record || !reason || processingID.value !== null) return
  processingID.value = record.id
  try {
    await affiliatesAPI.rejectWithdrawal(record.id, reason)
    rejectingRecord.value = null
    rejectReason.value = ''
    appStore.showSuccess(t('admin.affiliates.withdrawals.rejectSuccess'))
    await loadRecords()
  } catch (error) {
    appStore.showError(extractI18nErrorMessage(error, t, 'admin.affiliates.errors', t('common.error')))
  } finally {
    processingID.value = null
  }
}

function formatMoney(value: number): string {
  return `$${Number(value || 0).toFixed(2)}`
}

function formatNumber(value: number): string {
  return Number(value || 0).toFixed(2).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1')
}

function formatDateTime(value?: string): string {
  return value ? formatDisplayDateTime(value) : '-'
}

function statusClass(status: AffiliateWithdrawalStatus): string {
  if (status === 'paid') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'rejected') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

onMounted(() => {
  void loadRecords()
})
</script>
