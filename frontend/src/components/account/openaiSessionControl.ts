export type OpenAISessionTimeoutUnit = 'seconds' | 'minutes' | 'days'

export const DEFAULT_OPENAI_SESSION_MAX_COUNT = 35
export const MIN_OPENAI_SESSION_MAX_COUNT = 3
export const DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE = 1
export const DEFAULT_OPENAI_SESSION_TIMEOUT_UNIT: OpenAISessionTimeoutUnit = 'days'

const unitSeconds: Record<OpenAISessionTimeoutUnit, number> = {
  seconds: 1,
  minutes: 60,
  days: 86400
}

export function openAISessionTimeoutToSeconds(
  value: number | null,
  unit: OpenAISessionTimeoutUnit
): number {
  const normalizedValue = Number.isInteger(value) && Number(value) > 0
    ? Number(value)
    : DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE
  return normalizedValue * unitSeconds[unit]
}

export function openAISessionTimeoutFromSeconds(seconds: number | null | undefined): {
  value: number
  unit: OpenAISessionTimeoutUnit
} {
  const normalized = Number.isInteger(seconds) && Number(seconds) > 0 ? Number(seconds) : 86400
  if (normalized % unitSeconds.days === 0) {
    return { value: normalized / unitSeconds.days, unit: 'days' }
  }
  if (normalized % unitSeconds.minutes === 0) {
    return { value: normalized / unitSeconds.minutes, unit: 'minutes' }
  }
  return { value: normalized, unit: 'seconds' }
}
