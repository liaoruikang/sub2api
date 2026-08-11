<template>
  <BaseDialog :show="show" :title="t('admin.users.userApiKeys')" width="wide" @close="handleClose">
    <div v-if="user" class="space-y-4">
      <div class="flex items-center gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex h-10 w-10 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30">
          <span class="text-lg font-medium text-primary-700 dark:text-primary-300">{{ user.email.charAt(0).toUpperCase() }}</span>
        </div>
        <div><p class="font-medium text-gray-900 dark:text-white">{{ user.email }}</p><p class="text-sm text-gray-500 dark:text-dark-400">{{ user.username }}</p></div>
      </div>
      <div v-if="loading" class="flex justify-center py-8"><svg class="h-8 w-8 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg></div>
      <div v-else-if="apiKeys.length === 0" class="py-8 text-center"><p class="text-sm text-gray-500">{{ t('admin.users.noApiKeys') }}</p></div>
      <div v-else ref="scrollContainerRef" class="max-h-96 space-y-3 overflow-y-auto" @scroll="closeGroupSelector">
        <div v-for="key in apiKeys" :key="key.id" class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
          <div class="flex items-start justify-between">
            <div class="min-w-0 flex-1">
              <div class="mb-1 flex items-center gap-2"><span class="font-medium text-gray-900 dark:text-white">{{ key.name }}</span><span :class="['badge text-xs', key.status === 'active' ? 'badge-success' : 'badge-danger']">{{ key.status }}</span></div>
              <p class="truncate font-mono text-sm text-gray-500">{{ key.key.substring(0, 20) }}...{{ key.key.substring(key.key.length - 8) }}</p>
            </div>
          </div>
          <div class="mt-3 flex flex-wrap gap-4 text-xs text-gray-500">
            <div class="flex items-center gap-1">
              <span>{{ t('admin.users.group') }}:</span>
              <button
                :ref="(el) => setGroupButtonRef(key.id, el)"
                @click="openGroupSelector(key)"
                class="group/key-group -mx-1 -my-0.5 flex max-w-full cursor-pointer items-center gap-2 rounded-lg border border-transparent px-2 py-1.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500/40 dark:hover:border-dark-600 dark:hover:bg-dark-700"
                :disabled="updatingKeyIds.has(key.id)"
              >
                <div v-if="getApiKeyGroups(key).length" class="flex max-h-28 min-w-0 flex-1 flex-col gap-1.5 overflow-y-auto">
                  <div
                    v-for="(entry, index) in getApiKeyGroups(key)"
                    :key="`${key.id}-${entry.id}`"
                    class="flex min-w-0 items-center gap-1.5"
                  >
                    <span class="w-4 shrink-0 text-center text-xs font-semibold tabular-nums text-gray-400">{{ index + 1 }}</span>
                    <GroupBadge
                      v-if="entry.group"
                      :name="entry.group.name"
                      :platform="entry.group.platform"
                      :subscription-type="entry.group.subscription_type"
                      :rate-multiplier="entry.group.rate_multiplier"
                      class="min-w-0 max-w-full"
                    />
                    <GroupAuthorizationBadge
                      v-if="entry.group"
                      :is-exclusive="entry.group.is_exclusive"
                      :authorization-tag-names="entry.group.tags?.map((tag) => tag.name) ?? []"
                      :authorization-tag-count="entry.group.tags?.length ?? entry.group.tag_ids?.length ?? 0"
                    />
                    <span v-if="!entry.group" class="min-w-0 pt-1 text-gray-500">#{{ entry.id }}</span>
                  </div>
                </div>
                <span v-else class="text-gray-400 italic">{{ t('admin.users.none') }}</span>
                <svg v-if="updatingKeyIds.has(key.id)" class="h-3 w-3 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
                <Icon v-else name="chevronDown" size="xs" class="shrink-0 text-gray-400 transition-colors group-hover/key-group:text-primary-500" />
              </button>
            </div>
            <div class="flex items-center gap-1"><span>{{ t('admin.users.columns.created') }}: {{ formatDateTime(key.created_at) }}</span></div>
          </div>
        </div>
      </div>
    </div>
  </BaseDialog>

  <!-- Group Selector Dropdown -->
  <Teleport to="body">
    <div
      v-if="groupSelectorKeyId !== null && dropdownPosition"
      ref="dropdownRef"
      data-test="inline-group-editor"
      class="animate-in fade-in slide-in-from-top-2 fixed z-[100000020] w-[min(24rem,calc(100vw-1rem))] max-h-[min(32rem,calc(100vh-1rem))] overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl shadow-black/10 duration-200 dark:border-dark-600 dark:bg-dark-800 dark:shadow-black/30"
      :style="{ top: dropdownPosition.top + 'px', left: dropdownPosition.left + 'px' }"
    >
      <div class="border-b border-gray-200 bg-gray-50/80 p-3 dark:border-dark-600 dark:bg-dark-800/80">
        <div class="text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('keys.groupOrderHint') }}</div>
        <VueDraggable
          v-if="groupDraftIds.length"
          v-model="groupDraftIds"
          :animation="200"
          handle=".drag-handle"
          class="mt-2 space-y-1"
        >
          <div v-for="(groupId, index) in groupDraftIds" :key="groupId" class="flex min-h-10 items-center gap-1.5 rounded-lg border border-primary-100 bg-white px-2 py-1.5 shadow-sm dark:border-primary-900/50 dark:bg-dark-700">
            <div
              class="drag-handle flex h-7 w-4 shrink-0 cursor-grab items-center justify-center text-gray-400 hover:text-gray-700 active:cursor-grabbing dark:text-dark-300 dark:hover:text-gray-200"
              :title="t('keys.dragGroup')"
            >
              <Icon name="menu" size="sm" />
            </div>
            <span class="w-4 text-center text-xs text-gray-400">{{ index + 1 }}</span>
            <GroupOptionItem
              v-if="allGroups.find((group) => group.id === groupId)"
              :name="allGroups.find((group) => group.id === groupId)!.name"
              :platform="allGroups.find((group) => group.id === groupId)!.platform"
              :subscription-type="allGroups.find((group) => group.id === groupId)!.subscription_type"
              :is-exclusive="allGroups.find((group) => group.id === groupId)!.is_exclusive"
              :authorization-tag-names="allGroups.find((group) => group.id === groupId)!.tags?.map((tag) => tag.name) ?? []"
              :authorization-tag-count="allGroups.find((group) => group.id === groupId)!.tags?.length ?? allGroups.find((group) => group.id === groupId)!.tag_ids?.length ?? 0"
              :rate-multiplier="allGroups.find((group) => group.id === groupId)!.rate_multiplier"
              :peak-rate-enabled="allGroups.find((group) => group.id === groupId)!.peak_rate_enabled"
              :peak-start="allGroups.find((group) => group.id === groupId)!.peak_start"
              :peak-end="allGroups.find((group) => group.id === groupId)!.peak_end"
              :peak-rate-multiplier="allGroups.find((group) => group.id === groupId)!.peak_rate_multiplier"
              :description="allGroups.find((group) => group.id === groupId)!.description"
              :limited-time-multiplier="formatLimitedTimeMultiplier(allGroups.find((group) => group.id === groupId)!, t)"
              :limited-time-multiplier-value="allGroups.find((group) => group.id === groupId)!.limited_time_multiplier_value"
              :limited-time-multiplier-active="isLimitedTimeMultiplierActive(allGroups.find((group) => group.id === groupId)!)"
              :show-checkmark="false"
              class="min-w-0 flex-1"
            />
            <span v-else class="min-w-0 flex-1 truncate text-xs text-gray-500">#{{ groupId }}</span>
            <button type="button" class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 focus:outline-none focus:ring-2 focus:ring-red-500/40 dark:hover:bg-red-900/20" :title="t('keys.removeGroup')" @click.stop="removeDraftGroup(groupId)"><Icon name="x" size="xs" /></button>
          </div>
        </VueDraggable>
      </div>
      <button
        type="button"
        class="flex w-full items-center justify-between gap-2 border-b border-gray-200 bg-white px-3 py-2.5 text-left text-sm text-gray-700 transition-colors hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-inset focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700"
        :aria-expanded="groupOptionsOpen"
        aria-haspopup="listbox"
        @click.stop="groupOptionsOpen = !groupOptionsOpen"
      >
        <span class="min-w-0 flex-1 truncate">{{ t('keys.selectGroup') }}</span>
        <span class="flex shrink-0 items-center gap-2 text-xs text-gray-400 dark:text-dark-400">
          {{ t('common.selectedCount', { count: groupDraftIds.length }) }}
          <Icon
            name="chevronDown"
            size="sm"
            :class="['transition-transform duration-200', groupOptionsOpen && 'rotate-180']"
          />
        </span>
      </button>
      <div
        v-if="groupOptionsOpen"
        class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-dark-600 dark:bg-dark-800"
      >
        <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
        <input
          v-model="groupSearchText"
          type="text"
          :placeholder="t('common.searchPlaceholder')"
          class="min-w-0 flex-1 bg-transparent text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none dark:text-gray-100 dark:placeholder:text-dark-400"
          @click.stop
        />
      </div>
      <div
        v-if="groupOptionsOpen"
        class="max-h-64 overflow-y-auto bg-white p-2 dark:bg-dark-800"
        role="listbox"
      >
        <label
          v-for="group in filteredGroupOptions"
          :key="group.id"
          :class="[
            'flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors',
            groupDraftIds.includes(group.id)
              ? 'border-primary-200 bg-primary-50/70 dark:border-primary-800 dark:bg-primary-900/20'
              : 'border-transparent hover:border-gray-200 hover:bg-gray-50 dark:hover:border-dark-600 dark:hover:bg-dark-700'
          ]"
          @click.stop
        >
          <input type="checkbox" :checked="groupDraftIds.includes(group.id)" class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-2 focus:ring-primary-500/40 dark:border-dark-500" @change="toggleDraftGroup(group.id, ($event.target as HTMLInputElement).checked)" />
          <GroupOptionItem
            :name="group.name"
            :platform="group.platform"
            :subscription-type="group.subscription_type"
            :is-exclusive="group.is_exclusive"
            :authorization-tag-names="group.tags?.map((tag) => tag.name) ?? []"
            :authorization-tag-count="group.tags?.length ?? group.tag_ids?.length ?? 0"
            :rate-multiplier="group.rate_multiplier"
            :peak-rate-enabled="group.peak_rate_enabled"
            :peak-start="group.peak_start"
            :peak-end="group.peak_end"
            :peak-rate-multiplier="group.peak_rate_multiplier"
            :description="group.description"
            :limited-time-multiplier="formatLimitedTimeMultiplier(group, t)"
            :limited-time-multiplier-value="group.limited_time_multiplier_value"
            :limited-time-multiplier-active="isLimitedTimeMultiplierActive(group)"
            :selected="groupDraftIds.includes(group.id)"
          />
        </label>
        <div v-if="filteredGroupOptions.length === 0" class="py-4 text-center text-sm text-gray-400 dark:text-gray-500">
          {{ t('keys.noGroupFound') }}
        </div>
      </div>
      <div class="flex justify-end gap-2 border-t border-gray-200 bg-gray-50/80 p-3 dark:border-dark-600 dark:bg-dark-800/80">
        <button type="button" class="btn btn-secondary px-3 py-1.5 text-xs" @click.stop="closeGroupSelector">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary px-3 py-1.5 text-xs" :disabled="updatingKeyIds.has(selectedKeyForGroup?.id || 0)" @click.stop="saveGroupDraft">
          {{ updatingKeyIds.has(selectedKeyForGroup?.id || 0) ? t('keys.saving') : t('common.save') }}
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, type ComponentPublicInstance } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import { VueDraggable } from 'vue-draggable-plus'
import { formatDateTime } from '@/utils/format'
import type { AdminUser, AdminGroup, ApiKey } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupAuthorizationBadge from '@/components/common/GroupAuthorizationBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Icon from '@/components/icons/Icon.vue'
import { getApiKeyGroupIds, resolveApiKeyGroups, type ResolvedApiKeyGroup } from '@/utils/apiKeyGroups'
import {
  formatLimitedTimeMultiplier,
  isLimitedTimeMultiplierActive
} from '@/utils/limitedTimeMultiplier'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits(['close'])
const { t } = useI18n()
const appStore = useAppStore()

const apiKeys = ref<ApiKey[]>([])
const allGroups = ref<AdminGroup[]>([])
const loading = ref(false)
const updatingKeyIds = ref(new Set<number>())
const groupSelectorKeyId = ref<number | null>(null)
const groupDraftIds = ref<number[]>([])
const groupOptionsOpen = ref(false)
const groupSearchText = ref('')
const dropdownPosition = ref<{ top: number; left: number } | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const scrollContainerRef = ref<HTMLElement | null>(null)
const groupButtonRefs = ref<Map<number, HTMLElement>>(new Map())

const selectedKeyForGroup = computed(() => {
  if (groupSelectorKeyId.value === null) return null
  return apiKeys.value.find((k) => k.id === groupSelectorKeyId.value) || null
})

const getApiKeyGroups = (key: ApiKey): ResolvedApiKeyGroup[] =>
  resolveApiKeyGroups(key, allGroups.value)

const filteredGroupOptions = computed(() => {
  const query = groupSearchText.value.trim().toLowerCase()
  if (!query) return allGroups.value
  return allGroups.value.filter((group) =>
    group.name.toLowerCase().includes(query) || group.description?.toLowerCase().includes(query)
  )
})

const setGroupButtonRef = (keyId: number, el: Element | ComponentPublicInstance | null) => {
  if (el instanceof HTMLElement) {
    groupButtonRefs.value.set(keyId, el)
  } else {
    groupButtonRefs.value.delete(keyId)
  }
}

watch(() => props.show, (v) => {
  if (v && props.user) {
    load()
    loadGroups()
  } else {
    closeGroupSelector()
  }
})

const load = async () => {
  if (!props.user) return
  loading.value = true
  groupButtonRefs.value.clear()
  try {
    const res = await adminAPI.users.getUserApiKeys(props.user.id)
    apiKeys.value = res.items || []
  } catch (error) {
    console.error('Failed to load API keys:', error)
  } finally {
    loading.value = false
  }
}

const loadGroups = async () => {
  try {
    const groups = await adminAPI.groups.getAll()
    allGroups.value = groups
  } catch (error) {
    console.error('Failed to load groups:', error)
  }
}

const DROPDOWN_HEIGHT = 272 // max-h-64 = 16rem = 256px + padding
const DROPDOWN_GAP = 4

const openGroupSelector = (key: ApiKey) => {
  if (groupSelectorKeyId.value === key.id) {
    closeGroupSelector()
  } else {
    const buttonEl = groupButtonRefs.value.get(key.id)
    if (buttonEl) {
      const rect = buttonEl.getBoundingClientRect()
      const spaceBelow = window.innerHeight - rect.bottom
      const openUpward = spaceBelow < DROPDOWN_HEIGHT && rect.top > spaceBelow
      dropdownPosition.value = {
        top: openUpward ? rect.top - DROPDOWN_HEIGHT - DROPDOWN_GAP : rect.bottom + DROPDOWN_GAP,
        left: rect.left
      }
    }
    groupSelectorKeyId.value = key.id
    groupDraftIds.value = getApiKeyGroupIds(key)
    groupOptionsOpen.value = false
    groupSearchText.value = ''
  }
}

const closeGroupSelector = () => {
  groupSelectorKeyId.value = null
  groupDraftIds.value = []
  groupOptionsOpen.value = false
  groupSearchText.value = ''
  dropdownPosition.value = null
}

const toggleDraftGroup = (groupId: number, checked: boolean) => {
  groupDraftIds.value = checked
    ? [...groupDraftIds.value, groupId]
    : groupDraftIds.value.filter((id) => id !== groupId)
}

const removeDraftGroup = (groupId: number) => {
  groupDraftIds.value = groupDraftIds.value.filter((id) => id !== groupId)
}

const saveGroupDraft = async () => {
  const key = selectedKeyForGroup.value
  if (!key) return
  updatingKeyIds.value.add(key.id)
  try {
    const result = await adminAPI.apiKeys.updateApiKeyGroup(key.id, [...groupDraftIds.value])
    const idx = apiKeys.value.findIndex((item) => item.id === key.id)
    if (idx !== -1) apiKeys.value[idx] = result.api_key
    if (result.auto_granted_group_access && result.granted_group_name) {
      appStore.showSuccess(t('admin.users.groupChangedWithGrant', { group: result.granted_group_name }))
    } else {
      appStore.showSuccess(t('admin.users.groupChangedSuccess'))
    }
    closeGroupSelector()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.users.groupChangeFailed'))
  } finally {
    updatingKeyIds.value.delete(key.id)
  }
}

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && groupSelectorKeyId.value !== null) {
    event.stopPropagation()
    closeGroupSelector()
  }
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (dropdownRef.value && !dropdownRef.value.contains(target)) {
    // Check if the click is on one of the group trigger buttons
    for (const el of groupButtonRefs.value.values()) {
      if (el.contains(target)) return
    }
    closeGroupSelector()
  }
}

const handleClose = () => {
  closeGroupSelector()
  emit('close')
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  document.addEventListener('keydown', handleKeyDown, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  document.removeEventListener('keydown', handleKeyDown, true)
})
</script>
