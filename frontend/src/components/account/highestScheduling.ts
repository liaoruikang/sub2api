export const HIGHEST_SCHEDULING_RECOVERY_MINUTES_MAX = 10080

export const HIGHEST_SCHEDULING_EXTRA_KEYS = {
  mode: 'highest_scheduling_mode',
  recoveryMinutes: 'highest_scheduling_recovery_minutes',
  suppressed: 'highest_scheduling_suppressed',
  suppressedUntil: 'highest_scheduling_suppressed_until',
  suppressedAt: 'highest_scheduling_suppressed_at',
  suppressedReason: 'highest_scheduling_suppressed_reason'
} as const

export type HighestSchedulingSuppressionType = 'none' | 'timed' | 'manual'

export interface HighestSchedulingInputState {
  enabled: boolean
  recoveryMinutes: unknown
}

export interface HighestSchedulingState {
  enabled: boolean
  recoveryMinutes: number
  suppressionActive: boolean
  suppressionType: HighestSchedulingSuppressionType
  suppressedUntil: Date | null
  suppressedUntilRaw: string | null
  suppressedAt: string | null
  suppressedReason: string | null
}

const parseDate = (value: unknown): Date | null => {
  if (typeof value !== 'string' || !value.trim()) {
    return null
  }
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed
}

const readString = (value: unknown): string | null => {
  if (typeof value !== 'string') {
    return null
  }
  const trimmed = value.trim()
  return trimmed || null
}

export function normalizeHighestSchedulingRecoveryMinutes(value: unknown): number {
  if (value === null || value === undefined || value === '') {
    return 0
  }
  if (typeof value === 'boolean') {
    return 0
  }
  const numeric = typeof value === 'string' ? Number(value.trim()) : Number(value)
  if (!Number.isFinite(numeric)) {
    return 0
  }
  const normalized = Math.trunc(numeric)
  if (normalized < 0) {
    return 0
  }
  if (normalized > HIGHEST_SCHEDULING_RECOVERY_MINUTES_MAX) {
    return HIGHEST_SCHEDULING_RECOVERY_MINUTES_MAX
  }
  return normalized
}

export function readHighestSchedulingState(
  extra?: Record<string, unknown> | null,
  now: Date = new Date()
): HighestSchedulingState {
  const safeExtra = extra || {}
  const suppressedUntilRaw = readString(safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedUntil])
  const suppressedUntil = parseDate(suppressedUntilRaw)
  const manuallySuppressed = safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressed] === true
  const timedSuppressed = !!suppressedUntil && suppressedUntil.getTime() > now.getTime()
  const suppressionType: HighestSchedulingSuppressionType = manuallySuppressed
    ? 'manual'
    : timedSuppressed
      ? 'timed'
      : 'none'

  return {
    enabled: safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.mode] === true,
    recoveryMinutes: normalizeHighestSchedulingRecoveryMinutes(
      safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.recoveryMinutes]
    ),
    suppressionActive: suppressionType !== 'none',
    suppressionType,
    suppressedUntil,
    suppressedUntilRaw,
    suppressedAt: readString(safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedAt]),
    suppressedReason: readString(safeExtra[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedReason])
  }
}

export function clearHighestSchedulingSuppression(
  extra?: Record<string, unknown> | null
): Record<string, unknown> {
  const next: Record<string, unknown> = { ...(extra || {}) }
  delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressed]
  delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedUntil]
  delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedAt]
  delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedReason]
  return next
}

export function applyHighestSchedulingExtra(
  extra: Record<string, unknown> | null | undefined,
  state: HighestSchedulingInputState
): Record<string, unknown> {
  const next = { ...(extra || {}) }

  if (!state.enabled) {
    delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.mode]
    delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.recoveryMinutes]
    return clearHighestSchedulingSuppression(next)
  }

  next[HIGHEST_SCHEDULING_EXTRA_KEYS.mode] = true
  next[HIGHEST_SCHEDULING_EXTRA_KEYS.recoveryMinutes] = normalizeHighestSchedulingRecoveryMinutes(
    state.recoveryMinutes
  )
  return next
}
