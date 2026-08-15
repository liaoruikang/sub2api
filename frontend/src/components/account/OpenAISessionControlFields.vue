<template>
  <div :class="['space-y-4 pt-4', bordered && 'border-t border-gray-200 dark:border-dark-600']">
    <div class="flex items-start justify-between gap-4">
      <div class="min-w-0">
        <label class="input-label mb-0">{{ t('admin.accounts.openai.sessionControl.title') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.openai.sessionControl.description') }}
        </p>
      </div>
      <Toggle
        v-model="enabled"
        :disabled="disabled"
        data-testid="openai-session-control-toggle"
      />
    </div>

    <div
      v-if="enabled"
      class="grid grid-cols-1 gap-4 border-l-2 border-gray-200 pl-4 sm:grid-cols-2 dark:border-dark-600"
    >
      <div>
        <label class="input-label">{{ t('admin.accounts.openai.sessionControl.maxCount') }}</label>
        <input
          v-model.number="maxCount"
          type="number"
          class="input"
          :min="MIN_OPENAI_SESSION_MAX_COUNT"
          step="1"
          :disabled="disabled"
          data-testid="openai-session-control-max-count"
        />
        <p class="input-hint">{{ t('admin.accounts.openai.sessionControl.maxCountHint') }}</p>
      </div>

      <div>
        <label class="input-label">{{ t('admin.accounts.openai.sessionControl.idleTimeout') }}</label>
        <div class="grid grid-cols-[minmax(0,1fr)_8rem] gap-2">
          <input
            v-model.number="timeoutValue"
            type="number"
            class="input"
            min="1"
            step="1"
            :disabled="disabled"
            data-testid="openai-session-control-timeout-value"
          />
          <Select
            v-model="timeoutUnit"
            :options="timeoutUnitOptions"
            :disabled="disabled"
            data-testid="openai-session-control-timeout-unit"
          />
        </div>
        <p class="input-hint">{{ t('admin.accounts.openai.sessionControl.idleTimeoutHint') }}</p>
      </div>

      <div
        class="flex items-start justify-between gap-4 rounded-md bg-gray-50 p-3 sm:col-span-2 dark:bg-dark-700/50"
      >
        <div class="min-w-0">
          <label class="input-label mb-0">
            {{ t('admin.accounts.openai.sessionControl.slotRotation') }}
          </label>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.openai.sessionControl.slotRotationHint') }}
          </p>
        </div>
        <Toggle
          v-model="slotRotationEnabled"
          :disabled="disabled"
          data-testid="openai-session-slot-rotation-toggle"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import {
  DEFAULT_OPENAI_SESSION_MAX_COUNT,
  DEFAULT_OPENAI_SESSION_TIMEOUT_UNIT,
  DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE,
  MIN_OPENAI_SESSION_MAX_COUNT,
  type OpenAISessionTimeoutUnit
} from './openaiSessionControl'

withDefaults(defineProps<{ disabled?: boolean; bordered?: boolean }>(), { disabled: false, bordered: true })

const enabled = defineModel<boolean>('enabled', { required: true })
const maxCount = defineModel<number | null>('maxCount', { required: true })
const timeoutValue = defineModel<number | null>('timeoutValue', { required: true })
const timeoutUnit = defineModel<OpenAISessionTimeoutUnit>('timeoutUnit', { required: true })
const slotRotationEnabled = defineModel<boolean>('slotRotationEnabled', { required: true })

const { t } = useI18n()
const timeoutUnitOptions = computed(() => [
  { value: 'seconds' as const, label: t('admin.accounts.openai.sessionControl.units.seconds') },
  { value: 'minutes' as const, label: t('admin.accounts.openai.sessionControl.units.minutes') },
  { value: 'days' as const, label: t('admin.accounts.openai.sessionControl.units.days') }
])

watch(enabled, (value) => {
  if (!value) return
  if (!Number.isInteger(maxCount.value) || Number(maxCount.value) < MIN_OPENAI_SESSION_MAX_COUNT) {
    maxCount.value = DEFAULT_OPENAI_SESSION_MAX_COUNT
  }
  if (!Number.isInteger(timeoutValue.value) || Number(timeoutValue.value) <= 0) {
    timeoutValue.value = DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE
  }
  if (!timeoutUnit.value) {
    timeoutUnit.value = DEFAULT_OPENAI_SESSION_TIMEOUT_UNIT
  }
})
</script>
