export const HIGHEST_SCHEDULING_EXTRA_KEYS = {
  mode: 'highest_scheduling_mode'
} as const

const HIGHEST_SCHEDULING_LEGACY_EXTRA_KEYS = [
  'highest_scheduling_recovery_minutes',
  'highest_scheduling_suppressed',
  'highest_scheduling_suppressed_until',
  'highest_scheduling_suppressed_at',
  'highest_scheduling_suppressed_reason'
] as const

export interface HighestSchedulingInputState {
  enabled: boolean
}

export interface HighestSchedulingState {
  enabled: boolean
}

export function readHighestSchedulingState(
  extra?: Record<string, unknown> | null
): HighestSchedulingState {
  return {
    enabled: extra?.[HIGHEST_SCHEDULING_EXTRA_KEYS.mode] === true
  }
}

export function applyHighestSchedulingExtra(
  extra: Record<string, unknown> | null | undefined,
  state: HighestSchedulingInputState
): Record<string, unknown> {
  const next = { ...(extra || {}) }

  delete next[HIGHEST_SCHEDULING_EXTRA_KEYS.mode]
  for (const key of HIGHEST_SCHEDULING_LEGACY_EXTRA_KEYS) {
    delete next[key]
  }

  if (state.enabled) {
    next[HIGHEST_SCHEDULING_EXTRA_KEYS.mode] = true
  }

  return next
}

export function buildHighestSchedulingExtraPatch(enabled: boolean): Record<string, boolean> {
  return {
    [HIGHEST_SCHEDULING_EXTRA_KEYS.mode]: enabled
  }
}
