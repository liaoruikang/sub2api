<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.tagManagement')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-5">
      <form class="flex flex-col gap-3 sm:flex-row" @submit.prevent="createTag">
        <input
          v-model="newName"
          type="text"
          maxlength="100"
          class="input flex-1"
          :placeholder="t('admin.users.tagNamePlaceholder')"
        />
        <button type="submit" class="btn btn-primary sm:min-w-28" :disabled="saving">
          {{ t('common.create') }}
        </button>
      </form>

      <div v-if="loading" class="flex justify-center py-8">
        <svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 0 12 12h4zm2 5.291A7.962 7.962 0 013 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
        </svg>
      </div>
      <p v-else-if="tags.length === 0" class="rounded-lg border border-dashed border-gray-300 px-4 py-8 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400">
        {{ t('admin.users.noTags') }}
      </p>
      <div v-else class="space-y-2">
        <div
          v-for="tag in tags"
          :key="tag.id"
          class="flex items-center gap-3 rounded-lg border border-gray-200 px-3 py-2.5 dark:border-dark-600"
        >
          <template v-if="editingId === tag.id">
            <input
              v-model="editingName"
              type="text"
              maxlength="100"
              class="input min-w-0 flex-1"
              @keyup.enter="saveRename(tag.id)"
            />
            <button type="button" class="btn btn-primary px-3" :disabled="saving" @click="saveRename(tag.id)">
              {{ t('common.save') }}
            </button>
            <button type="button" class="btn btn-secondary px-3" :disabled="saving" @click="cancelRename">
              {{ t('common.cancel') }}
            </button>
          </template>
          <template v-else>
            <span class="min-w-0 flex-1 truncate font-medium text-gray-800 dark:text-gray-200">{{ tag.name }}</span>
            <button type="button" class="btn btn-secondary px-3" :disabled="saving" @click="openTagUsers(tag)">
              {{ t('admin.users.viewTagUsers') }}
            </button>
            <button type="button" class="btn btn-secondary px-3" :disabled="saving" @click="startRename(tag)">
              {{ t('common.edit') }}
            </button>
            <button type="button" class="btn px-3 text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20" :disabled="saving" @click="pendingDelete = tag">
              {{ t('common.delete') }}
            </button>
          </template>
        </div>
      </div>
    </div>
  </BaseDialog>

  <TagUsersManagementModal
    :show="tagUsersTag !== null"
    :tag="tagUsersTag"
    @close="tagUsersTag = null"
    @success="$emit('success')"
  />

  <ConfirmDialog
    :show="pendingDelete !== null"
    :title="t('admin.users.deleteTag')"
    :message="t('admin.users.deleteTagConfirm', { name: pendingDelete?.name })"
    :danger="true"
    @confirm="deleteTag"
    @cancel="pendingDelete = null"
  />
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UserTag } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TagUsersManagementModal from './TagUsersManagementModal.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ (event: 'close'): void; (event: 'success'): void }>()
const { t } = useI18n()
const appStore = useAppStore()

const tags = ref<UserTag[]>([])
const newName = ref('')
const editingId = ref<number | null>(null)
const editingName = ref('')
const pendingDelete = ref<UserTag | null>(null)
const tagUsersTag = ref<UserTag | null>(null)
const loading = ref(false)
const saving = ref(false)

const loadTags = async () => {
  loading.value = true
  try {
    tags.value = await adminAPI.tags.list()
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || error?.message || t('admin.users.failedToLoadTags'))
  } finally {
    loading.value = false
  }
}

const createTag = async () => {
  const name = newName.value.trim()
  if (!name || saving.value) return
  saving.value = true
  try {
    await adminAPI.tags.create(name)
    newName.value = ''
    await loadTags()
    emit('success')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || error?.message || t('admin.users.failedToCreateTag'))
  } finally {
    saving.value = false
  }
}

const openTagUsers = (tag: UserTag) => {
  tagUsersTag.value = tag
}

const handleClose = () => {
  tagUsersTag.value = null
  pendingDelete.value = null
  cancelRename()
  emit('close')
}

const startRename = (tag: UserTag) => {
  editingId.value = tag.id
  editingName.value = tag.name
}

const cancelRename = () => {
  editingId.value = null
  editingName.value = ''
}

const saveRename = async (id: number) => {
  const name = editingName.value.trim()
  if (!name || saving.value) return
  saving.value = true
  try {
    await adminAPI.tags.update(id, name)
    cancelRename()
    await loadTags()
    emit('success')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || error?.message || t('admin.users.failedToUpdateTag'))
  } finally {
    saving.value = false
  }
}

const deleteTag = async () => {
  if (!pendingDelete.value || saving.value) return
  saving.value = true
  try {
    await adminAPI.tags.delete(pendingDelete.value.id)
    pendingDelete.value = null
    await loadTags()
    emit('success')
  } catch (error: any) {
    appStore.showError(error?.response?.data?.detail || error?.message || t('admin.users.failedToDeleteTag'))
  } finally {
    saving.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      newName.value = ''
      cancelRename()
      pendingDelete.value = null
      tagUsersTag.value = null
      void loadTags()
    }
  }
)
</script>
