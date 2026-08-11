<template>
  <span
    v-if="isExclusive"
    data-testid="group-authorization-type"
    :class="[
      'inline-flex max-w-full shrink-0 items-center truncate rounded px-1.5 py-0.5 text-[10px] font-semibold',
      hasAuthorizationTags
        ? 'bg-cyan-50 text-cyan-700 dark:bg-cyan-950/40 dark:text-cyan-300'
        : 'bg-orange-50 text-orange-700 dark:bg-orange-950/30 dark:text-orange-300'
    ]"
    :title="authorizationLabel"
  >
    {{ authorizationLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

interface Props {
  isExclusive?: boolean
  authorizationTagNames?: string[]
  authorizationTagCount?: number
}

const props = withDefaults(defineProps<Props>(), {
  isExclusive: false,
  authorizationTagNames: () => [],
  authorizationTagCount: 0
})

const { t } = useI18n()

const hasAuthorizationTags = computed(() =>
  props.authorizationTagNames.length > 0 || props.authorizationTagCount > 0
)

const authorizationLabel = computed(() => {
  if (!hasAuthorizationTags.value) return t('admin.groups.exclusive')
  if (props.authorizationTagNames.length === 0) return t('admin.groups.tagExclusive')
  return `${props.authorizationTagNames.join('、')}${t('admin.groups.exclusive')}`
})
</script>
