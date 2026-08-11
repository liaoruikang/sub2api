<template>
  <div ref="container" class="relative">
    <button
      ref="trigger"
      type="button"
      class="input flex min-h-11 w-full items-center justify-between gap-2 text-left"
      :disabled="disabled"
      @click="open = !open"
    >
      <span v-if="selectedTags.length === 0" class="text-gray-400 dark:text-gray-500">
        {{ placeholder }}
      </span>
      <span v-else class="flex min-w-0 flex-wrap gap-1.5">
        <span
          v-for="tag in selectedTags"
          :key="tag.id"
          class="inline-flex max-w-full items-center gap-1 rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
        >
          <span class="truncate" :title="tag.name">{{ tag.name }}</span>
          <span
            role="button"
            tabindex="0"
            :aria-label="`Remove ${tag.name}`"
            class="shrink-0 rounded-full p-0.5 text-primary-500 transition-colors hover:bg-primary-200 hover:text-primary-700 dark:text-primary-300 dark:hover:bg-primary-800/50 dark:hover:text-primary-100"
            @click.stop="removeTag(tag.id)"
            @keydown.enter.stop.prevent="removeTag(tag.id)"
            @keydown.space.stop.prevent="removeTag(tag.id)"
          >
            <Icon name="x" size="xs" />
          </span>
        </span>
      </span>
      <Icon name="chevronDown" size="sm" :class="['shrink-0 transition-transform', open && 'rotate-180']" />
    </button>

    <Teleport to="body">
      <div
        v-if="open"
        ref="dropdown"
        class="fixed z-[100000020] max-h-64 overflow-y-auto rounded-lg border border-gray-200 bg-white p-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
        :style="dropdownStyle"
        @click.stop
      >
        <button
          v-for="tag in options"
          :key="tag.id"
          type="button"
          class="flex w-full items-center justify-between rounded-md px-3 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
          @click="toggle(tag.id)"
        >
          <span>{{ tag.name }}</span>
          <Icon v-if="isSelected(tag.id)" name="check" size="sm" class="text-primary-500" :stroke-width="2" />
        </button>
        <p v-if="options.length === 0" class="px-3 py-2 text-sm text-gray-400 dark:text-gray-500">
          {{ emptyText }}
        </p>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import type { UserTag } from '@/types'
import Icon from '@/components/icons/Icon.vue'

const props = withDefaults(
  defineProps<{
    modelValue: number[]
    options: UserTag[]
    placeholder?: string
    emptyText?: string
    disabled?: boolean
  }>(),
  {
    placeholder: 'Select tags',
    emptyText: 'No tags available',
    disabled: false
  }
)

const emit = defineEmits<{ (event: 'update:modelValue', value: number[]): void }>()
const open = ref(false)
const container = ref<HTMLElement | null>(null)
const trigger = ref<HTMLButtonElement | null>(null)
const dropdown = ref<HTMLElement | null>(null)
const dropdownStyle = ref<Record<string, string>>({})

const selectedTags = computed(() => {
  const selected = new Set(props.modelValue)
  return props.options.filter((tag) => selected.has(tag.id))
})

const isSelected = (id: number) => props.modelValue.includes(id)

const toggle = (id: number) => {
  const next = isSelected(id)
    ? props.modelValue.filter((tagId) => tagId !== id)
    : [...props.modelValue, id]
  emit('update:modelValue', next)
}

const removeTag = (id: number) => {
  emit('update:modelValue', props.modelValue.filter((tagId) => tagId !== id))
}

const updateDropdownPosition = () => {
  if (!trigger.value) return
  const rect = trigger.value.getBoundingClientRect()
  const viewportPadding = 8
  const width = Math.min(Math.max(rect.width, 240), window.innerWidth - viewportPadding * 2)
  const left = Math.min(Math.max(rect.left, viewportPadding), window.innerWidth - width - viewportPadding)
  const dropdownHeight = dropdown.value?.offsetHeight ?? 256
  const spaceBelow = window.innerHeight - rect.bottom
  const top = spaceBelow >= dropdownHeight || rect.top < dropdownHeight
    ? rect.bottom + 4
    : Math.max(viewportPadding, rect.top - dropdownHeight - 4)
  dropdownStyle.value = {
    top: `${top}px`,
    left: `${left}px`,
    width: `${width}px`
  }
}

const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as Node
  if (
    open.value &&
    !container.value?.contains(target) &&
    !dropdown.value?.contains(target)
  ) {
    open.value = false
  }
}

watch(open, (isOpen) => {
  if (isOpen) {
    nextTick(updateDropdownPosition)
    window.addEventListener('scroll', updateDropdownPosition, true)
    window.addEventListener('resize', updateDropdownPosition)
  } else {
    window.removeEventListener('scroll', updateDropdownPosition, true)
    window.removeEventListener('resize', updateDropdownPosition)
  }
})

onMounted(() => document.addEventListener('click', handleClickOutside))
onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', updateDropdownPosition, true)
  window.removeEventListener('resize', updateDropdownPosition)
})
</script>
