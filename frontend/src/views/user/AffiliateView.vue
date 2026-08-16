<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"
        ></div>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="flex items-center gap-1.5 text-sm text-gray-500 dark:text-dark-400">
              <Icon name="dollar" size="sm" class="text-primary-500" />
              {{ t('affiliate.stats.rebateRate') }}
            </p>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">
              {{ formattedRebateRate }}<span class="ml-0.5 text-base font-medium">%</span>
            </p>
            <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
              {{ t('affiliate.stats.rebateRateHint') }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.invitedUsers') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCount(detail.aff_count) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600 dark:text-emerald-400">
              {{ formatCurrency(detail.aff_quota) }}
            </p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.stats.totalQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">
              {{ formatCurrency(detail.aff_history_quota) }}
            </p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600 dark:text-amber-400">
              {{ t('affiliate.stats.frozenQuota') }}: {{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.title') }}</h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.description') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.yourCode') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm font-semibold text-gray-900 dark:text-white sm:flex-1 sm:truncate">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyCode">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyCode') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.inviteLink') }}</p>
              <div class="flex flex-col items-stretch gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900 sm:flex-row sm:items-center">
                <code class="min-w-0 break-all text-sm text-gray-700 dark:text-gray-300 sm:flex-1 sm:truncate">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm w-full sm:w-auto sm:shrink-0" @click="copyInviteLink">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copyLink') }}</span>
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 rounded-xl border border-primary-200 bg-primary-50 p-4 dark:border-primary-900/40 dark:bg-primary-900/20">
            <p class="text-sm font-medium text-primary-800 dark:text-primary-200">{{ t('affiliate.tips.title') }}</p>
            <ul class="mt-2 space-y-1 text-sm text-primary-700 dark:text-primary-300">
              <li>1. {{ t('affiliate.tips.line1') }}</li>
              <li>2. {{ t('affiliate.tips.line2', { rate: `${formattedRebateRate}%` }) }}</li>
              <li>3. {{ t('affiliate.tips.line3') }}</li>
              <li v-if="detail.aff_frozen_quota > 0">4. {{ t('affiliate.tips.line4') }}</li>
            </ul>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.transfer.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.transfer.description') }}</p>
            </div>
            <button
              class="btn btn-primary"
              :disabled="transferring || detail.aff_quota <= 0"
              @click="transferQuota"
            >
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.transfer.transferring') : t('affiliate.transfer.button') }}</span>
            </button>
          </div>
          <p v-if="detail.aff_quota <= 0" class="mt-3 text-sm text-amber-600 dark:text-amber-400">
            {{ t('affiliate.transfer.empty') }}
          </p>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.withdrawal.title') }}</h3>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('affiliate.withdrawal.description') }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="openWithdrawalAccounts">
                <Icon name="creditCard" size="sm" />
                <span>{{ t('affiliate.withdrawal.accounts.manage') }}</span>
              </button>
              <button type="button" class="btn btn-secondary btn-sm shrink-0" @click="openWithdrawalRecords">
                <Icon name="clock" size="sm" />
                <span>{{ t('affiliate.withdrawal.records') }}</span>
              </button>
            </div>
          </div>

          <div
            v-if="!detail.withdrawal_config?.enabled"
            class="mt-4 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-700 dark:border-amber-900/50 dark:bg-amber-900/20 dark:text-amber-300"
          >
            {{ t('affiliate.withdrawal.disabled') }}
          </div>

          <div v-else class="mt-5 space-y-4">
            <div class="grid gap-4 md:grid-cols-2">
              <div>
                <label class="input-label" for="affiliate-withdrawal-amount">{{ t('affiliate.withdrawal.amount') }}</label>
                <div class="flex gap-2">
                  <div class="relative min-w-0 flex-1">
                    <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">$</span>
                    <input
                      id="affiliate-withdrawal-amount"
                      v-model.number="withdrawalAmount"
                      type="number"
                      min="0"
                      step="0.01"
                      class="input pl-7"
                      :placeholder="formatPlainAmount(detail.withdrawal_config.min_amount)"
                    />
                  </div>
                  <button type="button" class="btn btn-secondary shrink-0" @click="fillAllWithdrawalQuota">
                    {{ t('affiliate.withdrawal.all') }}
                  </button>
                </div>
                <p class="mt-1 text-xs text-gray-400">
                  {{ t('affiliate.withdrawal.minimum', { amount: formatCurrency(detail.withdrawal_config.min_amount) }) }}
                </p>
              </div>
              <div>
                <label class="input-label" for="affiliate-withdrawal-account">{{ t('affiliate.withdrawal.alipayAccount') }}</label>
                <Select
                  id="affiliate-withdrawal-account"
                  v-model="selectedWithdrawalAccountID"
                  :options="withdrawalAccountOptions"
                  :disabled="withdrawalAccountsLoading || withdrawalAccounts.length === 0"
                  :placeholder="t('affiliate.withdrawal.accounts.selectPlaceholder')"
                  :empty-text="t('affiliate.withdrawal.accounts.empty')"
                  :aria-label="t('affiliate.withdrawal.alipayAccount')"
                />
                <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
                  <template v-if="withdrawalAccounts.length === 0">
                    {{ t('affiliate.withdrawal.accounts.emptyHint') }}
                    <button type="button" class="ml-1 text-primary-600 hover:underline dark:text-primary-400" @click="openWithdrawalAccounts">
                      {{ t('affiliate.withdrawal.accounts.addNow') }}
                    </button>
                  </template>
                  <template v-else>{{ t('affiliate.withdrawal.accounts.selectionHint') }}</template>
                </p>
              </div>
            </div>

            <div class="grid gap-px overflow-hidden rounded-lg border border-gray-200 bg-gray-200 dark:border-dark-700 dark:bg-dark-700 sm:grid-cols-3">
              <div class="bg-gray-50 px-4 py-3 dark:bg-dark-900">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.withdrawal.requestAmount') }}</p>
                <p class="mt-1 font-semibold text-gray-900 dark:text-white">{{ formatCurrency(normalizedWithdrawalAmount) }}</p>
              </div>
              <div class="bg-gray-50 px-4 py-3 dark:bg-dark-900">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.withdrawal.fee', { rate: formattedWithdrawalFeeRate }) }}</p>
                <p class="mt-1 font-semibold text-amber-600 dark:text-amber-400">{{ formatCurrency(withdrawalFeeAmount) }}</p>
              </div>
              <div class="bg-gray-50 px-4 py-3 dark:bg-dark-900">
                <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('affiliate.withdrawal.payoutAmount') }}</p>
                <p class="mt-1 font-semibold text-emerald-600 dark:text-emerald-400">{{ formatCurrency(withdrawalPayoutAmount) }}</p>
              </div>
            </div>

            <div class="flex justify-end">
              <button
                type="button"
                class="btn btn-primary"
                :disabled="!canSubmitWithdrawal || submittingWithdrawal"
                @click="submitWithdrawal"
              >
                <Icon v-if="submittingWithdrawal" name="refresh" size="sm" class="animate-spin" />
                <Icon v-else name="dollar" size="sm" />
                <span>{{ submittingWithdrawal ? t('affiliate.withdrawal.submitting') : t('affiliate.withdrawal.submit') }}</span>
              </button>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.invitees.title') }}</h3>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
            {{ t('affiliate.invitees.empty') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700 dark:text-dark-400">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.email') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.username') }}</th>
                  <th class="px-3 py-2 font-medium text-right">{{ t('affiliate.invitees.columns.rebate') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.invitees.columns.joinedAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="item in detail.invitees"
                  :key="item.user_id"
                  class="border-b border-gray-100 last:border-b-0 dark:border-dark-800"
                >
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>

    <BaseDialog
      :show="withdrawalRecordsOpen"
      :title="t('affiliate.withdrawal.recordsTitle')"
      width="wide"
      @close="withdrawalRecordsOpen = false"
    >
      <div class="space-y-4">
        <div class="w-full sm:w-52">
          <Select
            v-model="withdrawalStatusFilter"
            :options="withdrawalStatusOptions"
            :aria-label="t('affiliate.withdrawal.statusFilter')"
            @change="reloadWithdrawalRecords"
          />
        </div>

        <div v-if="withdrawalRecordsLoading" class="flex justify-center py-10">
          <div class="h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
        </div>
        <div v-else-if="withdrawalRecords.length === 0" class="rounded-lg border border-dashed border-gray-300 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
          {{ t('affiliate.withdrawal.noRecords') }}
        </div>
        <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
          <table class="w-full min-w-[860px] text-left text-sm">
            <thead class="bg-gray-50 text-gray-500 dark:bg-dark-900 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.requestNo') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.amount') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.payout') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.alipay') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.status') }}</th>
                <th class="px-4 py-3 font-medium">{{ t('affiliate.withdrawal.columns.time') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="record in withdrawalRecords" :key="record.id" class="border-t border-gray-100 dark:border-dark-800">
                <td class="px-4 py-3 font-mono text-xs text-gray-700 dark:text-gray-300">{{ record.request_no }}</td>
                <td class="px-4 py-3">
                  <p class="font-medium text-gray-900 dark:text-white">{{ formatCurrency(record.amount) }}</p>
                  <p class="text-xs text-gray-400">{{ t('affiliate.withdrawal.fee', { rate: formatPlainAmount(record.fee_rate) }) }}: {{ formatCurrency(record.fee_amount) }}</p>
                </td>
                <td class="px-4 py-3 font-medium text-emerald-600 dark:text-emerald-400">{{ formatCurrency(record.payout_amount) }}</td>
                <td class="px-4 py-3 text-gray-700 dark:text-gray-300">{{ record.alipay_account_masked }}</td>
                <td class="px-4 py-3">
                  <span :class="withdrawalStatusClass(record.status)" class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium">
                    {{ t(`affiliate.withdrawal.status.${record.status}`) }}
                  </span>
                  <p v-if="record.reject_reason" class="mt-1 max-w-52 text-xs text-red-500">{{ record.reject_reason }}</p>
                </td>
                <td class="px-4 py-3 text-gray-500 dark:text-dark-400">{{ formatDateTime(record.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <Pagination
          v-if="withdrawalPagination.total > withdrawalPagination.page_size"
          :total="withdrawalPagination.total"
          :page="withdrawalPagination.page"
          :page-size="withdrawalPagination.page_size"
          :show-page-size-selector="false"
          @update:page="changeWithdrawalPage"
        />
      </div>
    </BaseDialog>

    <AffiliateWithdrawalAccountsDialog
      :show="withdrawalAccountsOpen"
      :accounts="withdrawalAccounts"
      :loading="withdrawalAccountsLoading"
      @close="withdrawalAccountsOpen = false"
      @changed="reloadWithdrawalAccountsAfterChange"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Pagination from '@/components/common/Pagination.vue'
import AffiliateWithdrawalAccountsDialog from '@/components/user/AffiliateWithdrawalAccountsDialog.vue'
import userAPI from '@/api/user'
import type { AffiliateWithdrawal, AffiliateWithdrawalAccount, AffiliateWithdrawalStatus, UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const transferring = ref(false)
const submittingWithdrawal = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const withdrawalAmount = ref<number | null>(null)
const withdrawalAccountsOpen = ref(false)
const withdrawalAccountsLoading = ref(false)
const withdrawalAccounts = ref<AffiliateWithdrawalAccount[]>([])
const selectedWithdrawalAccountID = ref<number | null>(null)
const withdrawalRecordsOpen = ref(false)
const withdrawalRecordsLoading = ref(false)
const withdrawalRecords = ref<AffiliateWithdrawal[]>([])
const withdrawalStatusFilter = ref<AffiliateWithdrawalStatus | ''>('')
const withdrawalPagination = reactive({ page: 1, page_size: 10, total: 0 })

const withdrawalStatusOptions = computed(() => [
  { value: '', label: t('affiliate.withdrawal.status.all') },
  { value: 'pending', label: t('affiliate.withdrawal.status.pending') },
  { value: 'paid', label: t('affiliate.withdrawal.status.paid') },
  { value: 'rejected', label: t('affiliate.withdrawal.status.rejected') },
])

const withdrawalAccountOptions = computed(() => withdrawalAccounts.value.map((account) => ({
  value: account.id,
  label: account.is_default
    ? `${account.account_masked} (${t('affiliate.withdrawal.accounts.default')})`
    : account.account_masked,
})))

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

// Rebate rate is a percentage in the range [0, 100]; backend already clamps it.
// We trim trailing zeros (e.g. 20.00 → "20", 12.50 → "12.5") for a cleaner UI.
const formattedRebateRate = computed(() => {
  const v = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(v * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

const normalizedWithdrawalAmount = computed(() => Math.max(0, Number(withdrawalAmount.value) || 0))
const withdrawalFeeAmount = computed(() => {
  const rate = detail.value?.withdrawal_config?.fee_rate ?? 0
  return Math.round(normalizedWithdrawalAmount.value * (rate / 100) * 1e8) / 1e8
})
const withdrawalPayoutAmount = computed(() => Math.max(0, normalizedWithdrawalAmount.value - withdrawalFeeAmount.value))
const formattedWithdrawalFeeRate = computed(() => formatPlainAmount(detail.value?.withdrawal_config?.fee_rate ?? 0))
const canSubmitWithdrawal = computed(() => {
  if (!detail.value?.withdrawal_config?.enabled) return false
  const amount = normalizedWithdrawalAmount.value
  return amount >= detail.value.withdrawal_config.min_amount
    && amount <= detail.value.aff_quota
    && withdrawalPayoutAmount.value > 0
    && Number(selectedWithdrawalAccountID.value) > 0
})

function formatCount(value: number): string {
  return value.toLocaleString()
}

function formatPlainAmount(value: number): string {
  return Number(value || 0).toFixed(2).replace(/\.00$/, '').replace(/(\.\d)0$/, '$1')
}

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) {
    loading.value = true
  }
  try {
    detail.value = await userAPI.getAffiliateDetail()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.loadFailed')))
  } finally {
    if (!silent) {
      loading.value = false
    }
  }
}

async function loadWithdrawalAccounts(preferDefault = false): Promise<void> {
  withdrawalAccountsLoading.value = true
  try {
    const items = await userAPI.listAffiliateWithdrawalAccounts()
    withdrawalAccounts.value = items || []
    const selectedStillExists = withdrawalAccounts.value.some((account) => account.id === selectedWithdrawalAccountID.value)
    if (preferDefault || !selectedStillExists) {
      const preferred = withdrawalAccounts.value.find((account) => account.is_default) || withdrawalAccounts.value[0]
      selectedWithdrawalAccountID.value = preferred?.id ?? null
    }
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.accounts.loadFailed')))
  } finally {
    withdrawalAccountsLoading.value = false
  }
}

async function copyCode(): Promise<void> {
  if (!detail.value?.aff_code) return
  await copyToClipboard(detail.value.aff_code, t('affiliate.codeCopied'))
}

async function copyInviteLink(): Promise<void> {
  if (!inviteLink.value) return
  await copyToClipboard(inviteLink.value, t('affiliate.linkCopied'))
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transfer.success', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

function fillAllWithdrawalQuota(): void {
  withdrawalAmount.value = detail.value ? Math.max(0, detail.value.aff_quota) : 0
}

async function submitWithdrawal(): Promise<void> {
  if (!canSubmitWithdrawal.value || submittingWithdrawal.value) return
  submittingWithdrawal.value = true
  try {
    const record = await userAPI.createAffiliateWithdrawal({
      amount: normalizedWithdrawalAmount.value,
      withdrawal_account_id: Number(selectedWithdrawalAccountID.value),
    })
    appStore.showSuccess(t('affiliate.withdrawal.success', { amount: formatCurrency(record.payout_amount) }))
    withdrawalAmount.value = null
    await loadAffiliateDetail(true)
    if (withdrawalRecordsOpen.value) await loadWithdrawalRecords()
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.submitFailed')))
  } finally {
    submittingWithdrawal.value = false
  }
}

async function openWithdrawalAccounts(): Promise<void> {
  withdrawalAccountsOpen.value = true
  await loadWithdrawalAccounts()
}

function reloadWithdrawalAccountsAfterChange(): void {
  void loadWithdrawalAccounts(true)
}

async function openWithdrawalRecords(): Promise<void> {
  withdrawalRecordsOpen.value = true
  withdrawalPagination.page = 1
  await loadWithdrawalRecords()
}

async function loadWithdrawalRecords(): Promise<void> {
  withdrawalRecordsLoading.value = true
  try {
    const result = await userAPI.listAffiliateWithdrawals({
      page: withdrawalPagination.page,
      page_size: withdrawalPagination.page_size,
      status: withdrawalStatusFilter.value,
    })
    withdrawalRecords.value = result.items || []
    withdrawalPagination.total = result.total || 0
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.recordsFailed')))
  } finally {
    withdrawalRecordsLoading.value = false
  }
}

function reloadWithdrawalRecords(): void {
  withdrawalPagination.page = 1
  void loadWithdrawalRecords()
}

function changeWithdrawalPage(page: number): void {
  withdrawalPagination.page = page
  void loadWithdrawalRecords()
}

function withdrawalStatusClass(status: AffiliateWithdrawalStatus): string {
  if (status === 'paid') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  if (status === 'rejected') return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

onMounted(() => {
  void Promise.all([
    loadAffiliateDetail(),
    loadWithdrawalAccounts(true),
  ])
})
</script>
