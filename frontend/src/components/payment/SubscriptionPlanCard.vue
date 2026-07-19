<template>
  <div
    :class="[
      'group relative flex flex-col overflow-hidden rounded-2xl border transition-all',
      'hover:shadow-xl hover:-translate-y-0.5',
      borderClass,
      'bg-white dark:bg-dark-800',
    ]"
  >
    <!-- Colored top accent bar -->
    <div :class="['h-1.5', accentClass]" />

    <div class="flex flex-1 flex-col p-4">
      <!-- Header: name + badge + price -->
      <div class="mb-3 flex items-start justify-between gap-2">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <h3 class="truncate text-base font-bold text-gray-900 dark:text-white">{{ plan.name }}</h3>
            <span :class="['shrink-0 rounded-full px-2 py-0.5 text-[11px] font-medium', badgeLightClass]">
              {{ pLabel }}
            </span>
            <span v-if="plan.new_user_only" class="shrink-0 rounded-full border border-primary-200 bg-primary-50 px-2 py-0.5 text-[10px] font-semibold text-primary-600 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-300">
              {{ t('payment.planCard.newUserOnly') }}
            </span>
          </div>
          <p v-if="plan.description" class="mt-0.5 text-xs leading-relaxed text-gray-500 dark:text-dark-400 line-clamp-2">
            {{ plan.description }}
          </p>
        </div>
        <div class="shrink-0 text-right">
          <div class="flex items-center justify-end gap-1.5">
            <span v-if="firstPurchaseLabel" class="rounded-full border border-primary-200 bg-primary-50 px-2 py-0.5 text-[10px] font-semibold text-primary-600 dark:border-primary-500/30 dark:bg-primary-500/10 dark:text-primary-300">
              {{ firstPurchaseLabel }}
            </span>
            <div class="flex items-baseline gap-1">
              <span class="text-xs text-gray-400 dark:text-dark-500">$</span>
              <span :class="['text-2xl font-extrabold tracking-tight', textClass]">{{ effectivePrice }}</span>
              <span v-if="plan.currency" class="text-xs font-medium text-gray-400 dark:text-dark-500">{{ plan.currency }}</span>
            </div>
          </div>
          <span class="text-[11px] text-gray-400 dark:text-dark-500">/ {{ validitySuffix }}</span>
          <div v-if="plan.original_price" class="mt-0.5 flex items-center justify-end gap-1.5">
            <span class="text-xs text-gray-400 line-through dark:text-dark-500">${{ plan.original_price }}<template v-if="plan.currency"> {{ plan.currency }}</template></span>
            <span :class="['rounded px-1 py-0.5 text-[10px] font-semibold', discountClass]">{{ discountText }}</span>
          </div>
        </div>
      </div>

      <!-- Group quota info (compact) -->
      <div class="mb-3 space-y-2 rounded-xl border border-gray-100 bg-gray-50/80 p-3 text-xs dark:border-dark-600 dark:bg-dark-700/40">
        <div class="grid grid-cols-2 gap-2">
          <div class="rounded-lg bg-white px-3 py-2 shadow-sm dark:bg-dark-800/70">
            <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">{{ t('payment.planCard.purchaseLimit') }}</div>
            <div class="mt-0.5 text-lg font-semibold text-gray-900 dark:text-white">{{ purchaseLimitText }}</div>
          </div>
          <div class="rounded-lg bg-white px-3 py-2 shadow-sm dark:bg-dark-800/70">
            <div class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-dark-500">{{ t('payment.planCard.stock') }}</div>
            <div class="mt-0.5 text-lg font-semibold" :class="stockStatusClass">{{ stockText }}</div>
          </div>
        </div>
        <div class="grid grid-cols-2 gap-x-3 gap-y-2 pt-1 text-[11px] text-gray-500 dark:text-dark-400">
          <div class="flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.rate') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ rateDisplay }}</span>
          </div>
          <div v-if="hasPeakRate" class="col-span-2 flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.peakRate') }}</span>
            <span class="text-right font-medium text-amber-700 dark:text-amber-300">{{ peakRateDisplay }}</span>
          </div>
          <div v-if="plan.daily_limit_usd != null" class="flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.dailyLimit') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.daily_limit_usd }}</span>
          </div>
          <div v-if="plan.weekly_limit_usd != null" class="flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.weeklyLimit') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.weekly_limit_usd }}</span>
          </div>
          <div v-if="plan.monthly_limit_usd != null" class="flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.monthlyLimit') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">${{ plan.monthly_limit_usd }}</span>
          </div>
          <div v-if="plan.daily_limit_usd == null && plan.weekly_limit_usd == null && plan.monthly_limit_usd == null" class="flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.quota') }}</span>
            <span class="font-medium text-gray-700 dark:text-gray-300">{{ t('payment.planCard.unlimited') }}</span>
          </div>
          <div v-if="modelScopeLabels.length > 0" class="col-span-2 flex items-center justify-between gap-2">
            <span>{{ t('payment.planCard.models') }}</span>
            <div class="flex flex-wrap justify-end gap-1">
              <span v-for="scope in modelScopeLabels" :key="scope"
                class="rounded bg-gray-200/80 px-1.5 py-0.5 text-[10px] font-medium text-gray-600 dark:bg-dark-600 dark:text-gray-300">
                {{ scope }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Features list (compact) -->
      <div v-if="plan.features.length > 0" class="mb-3 space-y-1">
        <div v-for="feature in plan.features" :key="feature" class="flex items-start gap-1.5">
          <svg :class="['mt-0.5 h-3.5 w-3.5 flex-shrink-0', iconClass]" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M4.5 12.75l6 6 9-13.5" />
          </svg>
          <span class="text-xs text-gray-600 dark:text-gray-300">{{ feature }}</span>
        </div>
      </div>

      <div class="flex-1" />

      <div v-if="offSaleStatusText" :class="['mb-3 rounded-lg border px-3 py-2 text-xs font-medium', offSaleStatusClass]">
        <div class="flex items-center justify-between gap-2">
          <span>{{ t('payment.planCard.offSaleCountdown') }}</span>
          <span>{{ offSaleStatusText }}</span>
        </div>
      </div>

      <!-- Subscribe Button -->
      <button
        type="button"
        :disabled="!canPurchase"
        :class="buttonClass"
        @click="handleSelect"
      >
        {{ buttonText }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SubscriptionPlan } from '@/types/payment'
import type { UserSubscription } from '@/types'
import { useAppStore } from '@/stores/app'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'
import {
  platformAccentBarClass,
  platformBadgeLightClass,
  platformBorderClass,
  platformTextClass,
  platformIconClass,
  platformButtonClass,
  platformDiscountClass,
  platformLabel,
} from '@/utils/platformColors'

const props = defineProps<{ plan: SubscriptionPlan; activeSubscriptions?: UserSubscription[]; userCreatedAt?: string | null }>()
const emit = defineEmits<{ select: [plan: SubscriptionPlan] }>()
const { t } = useI18n()
const now = ref(Date.now())
let timer: ReturnType<typeof setInterval> | null = null

const platform = computed(() => props.plan.group_platform || '')
const isRenewal = computed(() =>
  props.activeSubscriptions?.some(s => s.group_id === props.plan.group_id && s.status === 'active') ?? false
)

// Derived color classes from central config
const accentClass = computed(() => platformAccentBarClass(platform.value))
const borderClass = computed(() => platformBorderClass(platform.value))
const badgeLightClass = computed(() => platformBadgeLightClass(platform.value))
const textClass = computed(() => platformTextClass(platform.value))
const iconClass = computed(() => platformIconClass(platform.value))
const btnClass = computed(() => platformButtonClass(platform.value))
const discountClass = computed(() => platformDiscountClass(platform.value))
const pLabel = computed(() => platformLabel(platform.value))

const effectivePrice = computed(() => props.plan.effective_price ?? props.plan.price)
const discountText = computed(() => {
  if (!props.plan.original_price || props.plan.original_price <= 0) return ''
  const pct = Math.round((1 - effectivePrice.value / props.plan.original_price) * 100)
  return pct > 0 ? `-${pct}%` : ''
})
const purchaseLimitText = computed(() => {
  const limit = props.plan.purchase_limit_count ?? 0
  return limit > 0 ? `${limit}` : t('payment.planCard.unlimited')
})
const stockCount = computed(() => props.plan.stock_count ?? 0)
const stockText = computed(() => formatPlanStockText(stockCount.value))
const stockStatusClass = computed(() => formatPlanStockClass(stockCount.value))
const stockIsSoldOut = computed(() => stockCount.value <= 0)
const purchaseLimitReached = computed(() => {
  const limit = props.plan.purchase_limit_count ?? 0
  if (limit <= 0) return false
  return (props.plan.current_purchase_count ?? 0) >= limit
})
const newUserOnlyNotEligible = computed(() => {
  if (!props.plan.new_user_only) return false
  const userCreatedAt = parseDateMs(props.userCreatedAt)
  const listedAt = parseDateMs(props.plan.listed_at)
  if (userCreatedAt === null || listedAt === null) return false
  return userCreatedAt < listedAt
})
const ipPurchaseLimitReached = computed(() => {
  const limit = props.plan.ip_purchase_limit_count ?? 0
  if (limit <= 0) return false
  return (props.plan.current_ip_purchase_count ?? 0) >= limit
})
const firstPurchaseLabel = computed(() => props.plan.first_purchase_discount_available ? t('payment.firstOrderSpecial') : '')

function formatPlanStockText(stock: number): string {
  if (stock <= 0) return t('payment.planCard.soldOut')
  if (stock <= 3) return t('payment.planCard.stockLowWithCount', { count: stock })
  if (stock <= 10) return t('payment.planCard.stockLow')
  return t('payment.planCard.stockEnough')
}

function formatPlanStockClass(stock: number): string {
  if (stock <= 0) return 'text-red-500 dark:text-red-400'
  if (stock <= 3) return 'text-orange-500 dark:text-orange-400'
  if (stock <= 10) return 'text-amber-500 dark:text-amber-400'
  return 'text-emerald-600 dark:text-emerald-400'
}

function parseDateMs(value?: string | null): number | null {
  if (!value) return null
  const time = new Date(value).getTime()
  return Number.isNaN(time) ? null : time
}

const rateDisplay = computed(() => {
  const rate = props.plan.rate_multiplier ?? 1
  return `×${Number(rate.toPrecision(10))}`
})

const appStore = useAppStore()

const hasPeakRate = computed(() => groupHasPeakRate(props.plan))

const peakRateDisplay = computed(() => {
  return formatPeakRateWindow(props.plan, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
})

const MODEL_SCOPE_LABELS: Record<string, string> = {
  claude: 'Claude',
  gemini_text: 'Gemini',
  gemini_image: 'Imagen',
}

const modelScopeLabels = computed(() => {
  if (platform.value !== 'antigravity') return []
  const scopes = props.plan.supported_model_scopes
  if (!scopes || scopes.length === 0) return []
  return scopes.map(s => MODEL_SCOPE_LABELS[s] || s)
})

const validitySuffix = computed(() => {
  const u = props.plan.validity_unit || 'day'
  if (u === 'month') return t('payment.perMonth')
  if (u === 'year') return t('payment.perYear')
  return `${props.plan.validity_days}${t('payment.days')}`
})

const offSaleAtMs = computed(() => {
  if (!props.plan.off_sale_at) return null
  const value = new Date(props.plan.off_sale_at).getTime()
  return Number.isNaN(value) ? null : value
})
const offSaleExpired = computed(() => offSaleAtMs.value !== null && now.value >= offSaleAtMs.value)
const offSaleCountdownText = computed(() => {
  if (offSaleAtMs.value === null) return ''
  const diff = offSaleAtMs.value - now.value
  if (diff <= 0) return t('payment.planCard.offSold')
  const totalSeconds = Math.floor(diff / 1000)
  const days = Math.floor(totalSeconds / 86400)
  const hours = Math.floor((totalSeconds % 86400) / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  if (days > 0) return t('payment.planCard.countdownDays', { days, time: `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}` })
  return t('payment.planCard.countdownTime', {
    time: `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`,
  })
})
const offSaleStatusText = computed(() => (offSaleExpired.value ? t('payment.planCard.offSold') : offSaleCountdownText.value))
const offSaleStatusClass = computed(() => {
  if (offSaleExpired.value) {
    return 'border-red-200 bg-red-50 text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300'
  }
  return 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200'
})
const canPurchase = computed(() => !offSaleExpired.value && !stockIsSoldOut.value && !purchaseLimitReached.value && !newUserOnlyNotEligible.value && !ipPurchaseLimitReached.value)
const buttonText = computed(() => {
  if (offSaleExpired.value) return t('payment.planCard.offSold')
  if (stockIsSoldOut.value) return t('payment.planCard.soldOut')
  if (purchaseLimitReached.value) return t('payment.planCard.limitReached')
  if (newUserOnlyNotEligible.value) return t('payment.planCard.notEligible')
  if (ipPurchaseLimitReached.value) return t('payment.planCard.ipLimitReached')
  return isRenewal.value ? t('payment.renewNow') : t('payment.subscribeNow')
})
const buttonClass = computed(() => {
  if (!canPurchase.value) {
    return 'w-full rounded-xl bg-gray-300 py-2.5 text-sm font-semibold text-gray-500 transition-all dark:bg-dark-600 dark:text-dark-300 cursor-not-allowed'
  }
  return ['w-full rounded-xl py-2.5 text-sm font-semibold transition-all active:scale-[0.98]', btnClass.value]
})

function handleSelect() {
  if (!canPurchase.value) return
  emit('select', props.plan)
}

onMounted(() => {
  timer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
})
</script>
