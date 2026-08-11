<template>
  <div>
    <label class="input-label">
      {{ t('admin.users.groups') }}
      <span class="font-normal text-gray-400">{{ t('common.selectedCount', { count: selectedIds.length }) }}</span>
    </label>
    <div
      v-if="ordered && selectedIds.length > 0"
      class="mb-3 max-h-52 overflow-y-auto rounded-xl border border-gray-200 bg-gray-50/80 p-2.5 shadow-sm dark:border-dark-600 dark:bg-dark-800/70"
    >
      <div class="text-xs font-medium text-primary-700 dark:text-primary-300">
        {{ t('keys.groupOrderHint') }}
      </div>
      <VueDraggable
        v-model="localOrderedIds"
        :animation="200"
        handle=".drag-handle"
        class="mt-2 space-y-1.5"
        @end="handleOrderedDragEnd"
      >
        <div
          v-for="(groupId, index) in localOrderedIds"
          :key="groupId"
          class="flex min-h-10 items-center gap-2 rounded-lg border border-gray-200/80 bg-white px-2 py-1.5 shadow-sm dark:border-dark-600 dark:bg-dark-700"
        >
          <div
            class="drag-handle flex h-7 w-5 shrink-0 cursor-grab items-center justify-center text-gray-400 hover:text-gray-700 active:cursor-grabbing dark:text-dark-300 dark:hover:text-gray-200"
            :title="t('keys.dragGroup')"
          >
            <Icon name="menu" size="sm" />
          </div>
          <span class="w-5 shrink-0 text-center text-xs font-semibold tabular-nums text-gray-400">{{ index + 1 }}</span>
          <GroupOptionItem
            v-if="selectedGroups[index]"
            :name="selectedGroups[index]!.name"
            :platform="selectedGroups[index]!.platform"
            :subscription-type="selectedGroups[index]!.subscription_type"
            :is-exclusive="selectedGroups[index]!.is_exclusive"
            :authorization-tag-count="selectedGroups[index]!.tags?.length ?? selectedGroups[index]!.tag_ids?.length ?? 0"
            :authorization-tag-names="selectedGroups[index]!.tags?.map((tag) => tag.name) ?? []"
            :rate-multiplier="selectedGroups[index]!.rate_multiplier"
            :user-rate-multiplier="userGroupRates[selectedGroups[index]!.id]"
            :peak-rate-enabled="selectedGroups[index]!.peak_rate_enabled"
            :peak-start="selectedGroups[index]!.peak_start"
            :peak-end="selectedGroups[index]!.peak_end"
            :peak-rate-multiplier="selectedGroups[index]!.peak_rate_multiplier"
            :description="selectedGroups[index]!.description"
            :limited-time-multiplier="formatLimitedTimeMultiplier(selectedGroups[index]!, t, current, userGroupRates[selectedGroups[index]!.id])"
            :limited-time-multiplier-value="selectedGroups[index]!.limited_time_multiplier_value"
            :limited-time-multiplier-active="isLimitedTimeMultiplierActive(selectedGroups[index]!, current, userGroupRates[selectedGroups[index]!.id])"
            :show-checkmark="false"
            class="min-w-0 flex-1"
          />
          <span v-else class="min-w-0 flex-1 truncate text-sm text-gray-500">#{{ groupId }}</span>
          <button
            type="button"
            class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-lg text-gray-400 transition-colors hover:bg-red-50 hover:text-red-600 focus:outline-none focus:ring-2 focus:ring-red-500/40 dark:hover:bg-red-900/20"
            :title="t('keys.removeGroup')"
            @click="removeGroup(groupId)"
          >
            <Icon name="x" size="xs" />
          </button>
        </div>
      </VueDraggable>
    </div>
    <template v-if="dropdown">
      <button
        ref="dropdownTriggerRef"
        type="button"
        class="flex w-full items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-left text-sm text-gray-700 transition-colors hover:border-gray-300 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500"
        :aria-expanded="isDropdownOpen"
        aria-haspopup="listbox"
        @click="toggleDropdown"
      >
        <span class="min-w-0 flex-1 truncate">{{ t('keys.selectGroup') }}</span>
        <span class="flex shrink-0 items-center gap-2 text-xs text-gray-400 dark:text-dark-400">
          {{ t('common.selectedCount', { count: selectedIds.length }) }}
          <Icon
            name="chevronDown"
            size="sm"
            :class="['transition-transform duration-200', isDropdownOpen && 'rotate-180']"
          />
        </span>
      </button>

      <Teleport to="body">
        <div
          v-if="isDropdownOpen"
          ref="dropdownRef"
          class="fixed z-[100000020] w-[min(24rem,calc(100vw-1rem))] overflow-hidden rounded-xl border border-gray-200 bg-white shadow-xl shadow-black/10 dark:border-dark-600 dark:bg-dark-800 dark:shadow-black/30"
          :style="dropdownStyle"
          role="listbox"
          @click.stop
        >
          <div
            v-if="isSearchable"
            class="flex items-center gap-2 border-b border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-dark-600 dark:bg-dark-800"
          >
            <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
            <input
              v-model="searchText"
              type="text"
              :placeholder="t('common.searchPlaceholder')"
              class="min-w-0 flex-1 bg-transparent text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none dark:text-gray-100 dark:placeholder:text-dark-400"
              @click.stop
            />
          </div>
          <div class="grid max-h-72 grid-cols-1 gap-1 overflow-y-auto bg-gray-50 p-1.5 dark:bg-dark-800">
            <label
              v-for="group in filteredGroups"
              :key="group.id"
              :class="[
                'flex cursor-pointer items-center gap-2 rounded-lg border px-2.5 py-2 text-sm transition-colors',
                selectedIds.includes(group.id)
                  ? 'border-primary-200 bg-primary-50/70 dark:border-primary-800 dark:bg-primary-900/20'
                  : 'border-transparent hover:border-gray-200 hover:bg-white dark:hover:border-dark-600 dark:hover:bg-dark-700'
              ]"
            >
              <input
                type="checkbox"
                :value="group.id"
                :checked="selectedIds.includes(group.id)"
                @change="handleChange(group.id, ($event.target as HTMLInputElement).checked)"
                class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
              />
              <GroupOptionItem
                :name="group.name"
                :platform="group.platform"
                :subscription-type="group.subscription_type"
                :is-exclusive="group.is_exclusive"
                :authorization-tag-count="group.tags?.length ?? group.tag_ids?.length ?? 0"
                :authorization-tag-names="group.tags?.map((tag) => tag.name) ?? []"
                :rate-multiplier="group.rate_multiplier"
                :user-rate-multiplier="userGroupRates[group.id]"
                :peak-rate-enabled="group.peak_rate_enabled"
                :peak-start="group.peak_start"
                :peak-end="group.peak_end"
                :peak-rate-multiplier="group.peak_rate_multiplier"
                :description="group.description"
                :limited-time-multiplier="formatLimitedTimeMultiplier(group, t, current, userGroupRates[group.id])"
                :limited-time-multiplier-value="group.limited_time_multiplier_value"
                :limited-time-multiplier-active="isLimitedTimeMultiplierActive(group, current, userGroupRates[group.id])"
                :selected="selectedIds.includes(group.id)"
                class="min-w-0 flex-1"
              />
            </label>
            <div
              v-if="filteredGroups.length === 0"
              class="py-2 text-center text-sm text-gray-500 dark:text-gray-400"
            >
              {{ t('common.noGroupsAvailable') }}
            </div>
          </div>
        </div>
      </Teleport>
    </template>
    <template v-else>
      <button
        v-if="collapsible"
        ref="dropdownTriggerRef"
        type="button"
        class="flex w-full items-center justify-between gap-2 rounded-xl border border-gray-200 bg-white px-3 py-2.5 text-left text-sm text-gray-700 transition-colors hover:border-gray-300 focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-dark-500"
        :aria-expanded="isDropdownOpen"
        aria-haspopup="listbox"
        @click="toggleDropdown"
      >
        <span class="min-w-0 flex-1 truncate">{{ t('keys.selectGroup') }}</span>
        <span class="flex shrink-0 items-center gap-2 text-xs text-gray-400 dark:text-dark-400">
          {{ t('common.selectedCount', { count: selectedIds.length }) }}
          <Icon
            name="chevronDown"
            size="sm"
            :class="['transition-transform duration-200', isDropdownOpen && 'rotate-180']"
          />
        </span>
      </button>
      <template v-if="!collapsible || isDropdownOpen">
        <div
          v-if="isSearchable"
          class="flex items-center gap-2 rounded-t-xl border border-b-0 border-gray-200 bg-gray-50 px-3 py-2.5 dark:border-dark-600 dark:bg-dark-800"
        >
          <Icon name="search" size="sm" class="shrink-0 text-gray-400" />
          <input
            v-model="searchText"
            type="text"
            :placeholder="t('common.searchPlaceholder')"
            class="min-w-0 flex-1 bg-transparent text-sm text-gray-900 placeholder:text-gray-400 focus:outline-none dark:text-gray-100 dark:placeholder:text-dark-400"
          />
        </div>
        <div
          :class="[
            'grid max-h-56 grid-cols-1 gap-1 overflow-y-auto p-1.5',
            isSearchable
              ? 'rounded-b-lg border border-t-0 border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'
              : 'rounded-lg border border-gray-200 bg-gray-50 dark:border-dark-600 dark:bg-dark-800'
          ]"
        >
          <label
            v-for="group in filteredGroups"
            :key="group.id"
            :class="[
              'flex cursor-pointer items-center gap-2 rounded-lg border px-2.5 py-2 text-sm transition-colors',
              selectedIds.includes(group.id)
                ? 'border-primary-200 bg-primary-50/70 dark:border-primary-800 dark:bg-primary-900/20'
                : 'border-transparent hover:border-gray-200 hover:bg-white dark:hover:border-dark-600 dark:hover:bg-dark-700'
            ]"
          >
            <input
              type="checkbox"
              :value="group.id"
              :checked="selectedIds.includes(group.id)"
              @change="handleChange(group.id, ($event.target as HTMLInputElement).checked)"
              class="h-3.5 w-3.5 shrink-0 rounded border-gray-300 text-primary-500 focus:ring-primary-500 dark:border-dark-500"
            />
            <GroupOptionItem
              :name="group.name"
              :platform="group.platform"
              :subscription-type="group.subscription_type"
              :is-exclusive="group.is_exclusive"
              :authorization-tag-count="group.tags?.length ?? group.tag_ids?.length ?? 0"
              :authorization-tag-names="group.tags?.map((tag) => tag.name) ?? []"
              :rate-multiplier="group.rate_multiplier"
              :user-rate-multiplier="userGroupRates[group.id]"
              :peak-rate-enabled="group.peak_rate_enabled"
              :peak-start="group.peak_start"
              :peak-end="group.peak_end"
              :peak-rate-multiplier="group.peak_rate_multiplier"
              :description="group.description"
              :limited-time-multiplier="formatLimitedTimeMultiplier(group, t, current, userGroupRates[group.id])"
              :limited-time-multiplier-value="group.limited_time_multiplier_value"
              :limited-time-multiplier-active="isLimitedTimeMultiplierActive(group, current, userGroupRates[group.id])"
              :selected="selectedIds.includes(group.id)"
              class="min-w-0 flex-1"
            />
          </label>
          <div
            v-if="filteredGroups.length === 0"
            class="py-2 text-center text-sm text-gray-500 dark:text-gray-400"
          >
            {{ t('common.noGroupsAvailable') }}
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { VueDraggable } from 'vue-draggable-plus'
import GroupOptionItem from './GroupOptionItem.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Group, GroupPlatform } from '@/types'
import {
  formatLimitedTimeMultiplier,
  isLimitedTimeMultiplierActive
} from '@/utils/limitedTimeMultiplier'

type SelectableGroup = Group & { account_count?: number }

const { t } = useI18n()

interface Props {
  modelValue: number[]
  groups: SelectableGroup[]
  ordered?: boolean
  platform?: GroupPlatform
  mixedScheduling?: boolean
  searchable?: boolean | 'auto'
  dropdown?: boolean
  collapsible?: boolean
  defaultOpen?: boolean
  userGroupRates?: Record<number, number>
  current?: Date
}

const props = withDefaults(defineProps<Props>(), {
  ordered: false,
  searchable: 'auto',
  dropdown: false,
  defaultOpen: false,
  userGroupRates: () => ({})
})

const userGroupRates = computed(() => props.userGroupRates)
const current = computed(() => props.current ?? new Date())
const emit = defineEmits<{
  'update:modelValue': [value: number[]]
}>()

const searchText = ref('')
const localOrderedIds = ref<number[]>([])
const isDropdownOpen = ref(props.defaultOpen)
const dropdownTriggerRef = ref<HTMLButtonElement | null>(null)
const dropdownRef = ref<HTMLElement | null>(null)
const dropdownPosition = ref<'bottom' | 'top'>('bottom')
const dropdownRect = ref<DOMRect | null>(null)

watch(
  () => [props.modelValue, props.ordered] as const,
  ([value, ordered]) => {
    if (ordered) localOrderedIds.value = [...value]
  },
  { immediate: true }
)

const selectedIds = computed(() => (props.ordered ? localOrderedIds.value : props.modelValue))
const selectedGroups = computed(() =>
  selectedIds.value.map((id) => props.groups.find((group) => group.id === id))
)

const isSearchable = computed(() => {
  if (props.searchable === 'auto') return props.groups.length > 5
  return props.searchable
})

const dropdownStyle = computed(() => {
  if (!dropdownRect.value) return {}
  const rect = dropdownRect.value
  const viewportPadding = 8
  const right = Math.max(viewportPadding, window.innerWidth - viewportPadding)
  const left = Math.min(Math.max(viewportPadding, rect.left), right)
  const width = Math.min(Math.max(rect.width, 280), right - left)
  const style: Record<string, string> = {
    left: `${left}px`,
    width: `${width}px`
  }
  if (dropdownPosition.value === 'top') {
    style.bottom = `${window.innerHeight - rect.top + 4}px`
  } else {
    style.top = `${rect.bottom + 4}px`
  }
  return style
})

const updateDropdownRect = () => {
  if (dropdownTriggerRef.value) dropdownRect.value = dropdownTriggerRef.value.getBoundingClientRect()
}

const calculateDropdownPosition = () => {
  updateDropdownRect()
  nextTick(() => {
    if (!dropdownRect.value) return
    const dropdownHeight = dropdownRef.value?.offsetHeight || 280
    const spaceBelow = window.innerHeight - dropdownRect.value.bottom
    dropdownPosition.value = spaceBelow < dropdownHeight && dropdownRect.value.top > dropdownHeight
      ? 'top'
      : 'bottom'
  })
}

const toggleDropdown = () => {
  isDropdownOpen.value = !isDropdownOpen.value
}

const closeDropdown = () => {
  isDropdownOpen.value = false
}

const handleDropdownClickOutside = (event: MouseEvent) => {
  const target = event.target as Node
  if (
    isDropdownOpen.value &&
    !dropdownRef.value?.contains(target) &&
    !dropdownTriggerRef.value?.contains(target)
  ) {
    closeDropdown()
  }
}

watch(isDropdownOpen, (open) => {
  if (open) {
    calculateDropdownPosition()
    window.addEventListener('scroll', updateDropdownRect, { capture: true, passive: true })
    window.addEventListener('resize', calculateDropdownPosition)
  } else {
    searchText.value = ''
    window.removeEventListener('scroll', updateDropdownRect, { capture: true })
    window.removeEventListener('resize', calculateDropdownPosition)
  }
})

const filteredGroups = computed(() => {
  let result: SelectableGroup[] = props.groups
  if (props.platform) {
    if (props.platform === 'antigravity' && props.mixedScheduling) {
      result = result.filter(
        (group) =>
          group.platform === 'antigravity' ||
          group.platform === 'anthropic' ||
          group.platform === 'gemini' ||
          group.platform === 'composite'
      )
    } else {
      result = result.filter(
        (group) => group.platform === props.platform || group.platform === 'composite'
      )
    }
  }
  if (isSearchable.value && searchText.value) {
    const query = searchText.value.toLowerCase()
    result = result.filter(
      (group) =>
        group.name.toLowerCase().includes(query) ||
        group.description?.toLowerCase().includes(query)
    )
  }
  return result
})

const handleOrderedDragEnd = () => {
  emit('update:modelValue', [...localOrderedIds.value])
}

const handleChange = (groupId: number, checked: boolean) => {
  const current = selectedIds.value
  const newValue = checked
    ? [...current, groupId]
    : current.filter((id) => id !== groupId)
  if (props.ordered) localOrderedIds.value = [...newValue]
  emit('update:modelValue', newValue)
}

const removeGroup = (groupId: number) => {
  const newValue = selectedIds.value.filter((id) => id !== groupId)
  if (props.ordered) localOrderedIds.value = [...newValue]
  emit('update:modelValue', newValue)
}

onMounted(() => document.addEventListener('click', handleDropdownClickOutside))

onUnmounted(() => {
  document.removeEventListener('click', handleDropdownClickOutside)
  window.removeEventListener('scroll', updateDropdownRect, { capture: true })
  window.removeEventListener('resize', calculateDropdownPosition)
})
</script>
