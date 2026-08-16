<template>
  <BaseDialog
    :show="show"
    :title="t('affiliate.withdrawal.accounts.title')"
    width="normal"
    @close="emit('close')"
  >
    <div class="space-y-4">
      <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-sm text-gray-500 dark:text-dark-400">
          {{ t('affiliate.withdrawal.accounts.description') }}
        </p>
        <button
          type="button"
          class="btn btn-primary btn-sm shrink-0"
          :disabled="accounts.length >= 10"
          @click="openCreate"
        >
          <Icon name="plus" size="sm" />
          <span>{{ t('affiliate.withdrawal.accounts.add') }}</span>
        </button>
      </div>

      <form
        v-if="editorOpen"
        class="space-y-3 border-y border-gray-200 bg-gray-50 px-4 py-4 dark:border-dark-700 dark:bg-dark-900"
        @submit.prevent="saveAccount"
      >
        <div>
          <label class="input-label" for="affiliate-account-editor">
            {{ editingAccount ? t('affiliate.withdrawal.accounts.editTitle') : t('affiliate.withdrawal.accounts.addTitle') }}
          </label>
          <input
            id="affiliate-account-editor"
            v-model.trim="accountInput"
            type="text"
            maxlength="128"
            autocomplete="off"
            class="input"
            :placeholder="t('affiliate.withdrawal.alipayPlaceholder')"
          />
          <p class="mt-1 text-xs text-gray-400 dark:text-dark-500">
            {{ editingAccount ? t('affiliate.withdrawal.accounts.editHint') : t('affiliate.withdrawal.alipayHint') }}
          </p>
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="closeEditor">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" class="btn btn-primary btn-sm" :disabled="saving || accountInput.trim().length < 5">
            <Icon v-if="saving" name="refresh" size="sm" class="animate-spin" />
            <span>{{ saving ? t('common.saving') : t('common.save') }}</span>
          </button>
        </div>
      </form>

      <div v-if="loading" class="flex justify-center py-10">
        <div class="h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>
      <div
        v-else-if="accounts.length === 0"
        class="rounded-lg border border-dashed border-gray-300 py-10 text-center dark:border-dark-700"
      >
        <Icon name="creditCard" size="lg" class="mx-auto text-gray-400" />
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('affiliate.withdrawal.accounts.empty') }}
        </p>
      </div>
      <div v-else class="divide-y divide-gray-100 overflow-hidden rounded-lg border border-gray-200 dark:divide-dark-800 dark:border-dark-700">
        <div
          v-for="account in accounts"
          :key="account.id"
          class="flex flex-col gap-3 px-4 py-3 sm:flex-row sm:items-center"
        >
          <div class="flex min-w-0 flex-1 items-center gap-3">
            <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-blue-900/20 dark:text-blue-300">
              <Icon name="creditCard" size="sm" />
            </div>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ account.account_masked }}</span>
                <span v-if="account.is_default" class="badge badge-success text-xs">
                  {{ t('affiliate.withdrawal.accounts.default') }}
                </span>
              </div>
              <p class="mt-0.5 text-xs text-gray-400 dark:text-dark-500">
                {{ t('affiliate.withdrawal.alipayAccount') }}
              </p>
            </div>
          </div>

          <div class="flex items-center justify-end gap-1">
            <button
              v-if="!account.is_default"
              type="button"
              class="btn btn-secondary btn-sm mr-1"
              :disabled="settingDefaultID === account.id"
              @click="setDefault(account)"
            >
              <Icon v-if="settingDefaultID === account.id" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="check" size="sm" />
              <span>{{ t('affiliate.withdrawal.accounts.setDefault') }}</span>
            </button>
            <button
              type="button"
              class="rounded-lg p-2 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              :title="t('common.edit')"
              @click="openEdit(account)"
            >
              <Icon name="edit" size="sm" />
            </button>
            <button
              type="button"
              class="rounded-lg p-2 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:text-dark-400 dark:hover:bg-red-900/20 dark:hover:text-red-400"
              :title="t('common.delete')"
              @click="confirmDelete(account)"
            >
              <Icon name="trash" size="sm" />
            </button>
          </div>
        </div>
      </div>

      <p v-if="accounts.length >= 10" class="text-xs text-amber-600 dark:text-amber-400">
        {{ t('affiliate.withdrawal.accounts.limit') }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>

  <ConfirmDialog
    :show="deleteDialogOpen"
    :title="t('affiliate.withdrawal.accounts.deleteTitle')"
    :message="t('affiliate.withdrawal.accounts.deleteConfirm', { account: deletingAccount?.account_masked || '' })"
    :confirm-text="t('common.delete')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="deleteAccount"
    @cancel="closeDeleteDialog"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { AffiliateWithdrawalAccount } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface Props {
  show: boolean
  accounts: AffiliateWithdrawalAccount[]
  loading?: boolean
}

interface Emits {
  (event: 'close'): void
  (event: 'changed'): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
})
const emit = defineEmits<Emits>()
const { t } = useI18n()
const appStore = useAppStore()

const editorOpen = ref(false)
const editingAccount = ref<AffiliateWithdrawalAccount | null>(null)
const accountInput = ref('')
const saving = ref(false)
const settingDefaultID = ref<number | null>(null)
const deleteDialogOpen = ref(false)
const deletingAccount = ref<AffiliateWithdrawalAccount | null>(null)

function openCreate(): void {
  editingAccount.value = null
  accountInput.value = ''
  editorOpen.value = true
}

function openEdit(account: AffiliateWithdrawalAccount): void {
  editingAccount.value = account
  accountInput.value = ''
  editorOpen.value = true
}

function closeEditor(): void {
  editorOpen.value = false
  editingAccount.value = null
  accountInput.value = ''
}

async function saveAccount(): Promise<void> {
  const account = accountInput.value.trim()
  if (account.length < 5 || saving.value) return
  saving.value = true
  try {
    if (editingAccount.value) {
      await userAPI.updateAffiliateWithdrawalAccount(editingAccount.value.id, account)
      appStore.showSuccess(t('affiliate.withdrawal.accounts.updateSuccess'))
    } else {
      await userAPI.createAffiliateWithdrawalAccount(account)
      appStore.showSuccess(t('affiliate.withdrawal.accounts.createSuccess'))
    }
    closeEditor()
    emit('changed')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.accounts.saveFailed')))
  } finally {
    saving.value = false
  }
}

async function setDefault(account: AffiliateWithdrawalAccount): Promise<void> {
  if (settingDefaultID.value !== null) return
  settingDefaultID.value = account.id
  try {
    await userAPI.setDefaultAffiliateWithdrawalAccount(account.id)
    appStore.showSuccess(t('affiliate.withdrawal.accounts.defaultSuccess'))
    emit('changed')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.accounts.defaultFailed')))
  } finally {
    settingDefaultID.value = null
  }
}

function confirmDelete(account: AffiliateWithdrawalAccount): void {
  deletingAccount.value = account
  deleteDialogOpen.value = true
}

function closeDeleteDialog(): void {
  deleteDialogOpen.value = false
  deletingAccount.value = null
}

async function deleteAccount(): Promise<void> {
  if (!deletingAccount.value) return
  const accountID = deletingAccount.value.id
  closeDeleteDialog()
  try {
    await userAPI.deleteAffiliateWithdrawalAccount(accountID)
    appStore.showSuccess(t('affiliate.withdrawal.accounts.deleteSuccess'))
    emit('changed')
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.withdrawal.accounts.deleteFailed')))
  }
}

watch(
  () => props.show,
  (show) => {
    if (!show) {
      closeEditor()
      closeDeleteDialog()
    }
  },
)
</script>
