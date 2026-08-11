import type { Group } from '@/types'

type TranslateFn = (key: string, params?: Record<string, unknown>) => string

export type LimitedTimeCronFrequency = 'daily' | 'weekly' | 'monthly'

export interface ParsedLimitedTimeCron {
  frequency: LimitedTimeCronFrequency
  weekday: number
  monthDay: number
  hour: number
  minute: number
}

type LimitedTimeGroup = Pick<
  Group,
  | 'platform'
  | 'subscription_type'
  | 'rate_multiplier'
  | 'limited_time_multiplier_enabled'
  | 'limited_time_multiplier_cron'
  | 'limited_time_multiplier_duration_minutes'
  | 'limited_time_multiplier_value'
>

export function parseLimitedTimeCron(cron: string | null | undefined): ParsedLimitedTimeCron {
  const parts = String(cron || '').trim().split(/\s+/)
  if (parts.length !== 5) {
    return { frequency: 'daily', weekday: 1, monthDay: 1, hour: 9, minute: 0 }
  }

  const [minutePart, hourPart, dayPart, monthPart, weekdayPart] = parts
  const minute = Number(minutePart)
  const hour = Number(hourPart)
  const isValidMinute = Number.isInteger(minute) && minute >= 0 && minute <= 59
  const isValidHour = Number.isInteger(hour) && hour >= 0 && hour <= 23
  const base = { hour: isValidHour ? hour : 9, minute: isValidMinute ? minute : 0 }

  if (monthPart === '*' && dayPart === '*' && weekdayPart === '*') {
    return { frequency: 'daily', weekday: 1, monthDay: 1, ...base }
  }

  const weekday = Number(weekdayPart)
  if (monthPart === '*' && dayPart === '*' && Number.isInteger(weekday) && weekday >= 0 && weekday <= 6) {
    return { frequency: 'weekly', weekday, monthDay: 1, ...base }
  }

  const monthDay = Number(dayPart)
  if (monthPart === '*' && weekdayPart === '*' && Number.isInteger(monthDay) && monthDay >= 1 && monthDay <= 31) {
    return { frequency: 'monthly', weekday: 1, monthDay, ...base }
  }

  return { frequency: 'daily', weekday: 1, monthDay: 1, ...base }
}

const formatTimeOfDay = (hour: number, minute: number) => {
  const normalizedHour = Math.min(23, Math.max(0, Math.floor(Number(hour) || 0)))
  const normalizedMinute = Math.min(59, Math.max(0, Math.floor(Number(minute) || 0)))
  return `${normalizedHour}:${String(normalizedMinute).padStart(2, '0')}`
}

export function isLimitedTimeMultiplierActive(
  group: LimitedTimeGroup,
  current = new Date(),
  userRateMultiplier?: number | null
): boolean {
  if (
    group.subscription_type === 'subscription' ||
    !group.limited_time_multiplier_enabled ||
    !group.limited_time_multiplier_cron ||
    !group.limited_time_multiplier_duration_minutes
  ) {
    return false
  }

  const normalRate = userRateMultiplier ?? group.rate_multiplier
  if (
    group.limited_time_multiplier_value === null ||
    group.limited_time_multiplier_value === undefined ||
    group.limited_time_multiplier_value >= normalRate
  ) {
    return false
  }

  const cron = parseLimitedTimeCron(group.limited_time_multiplier_cron)
  const durationMinutes = Math.max(0, Math.floor(Number(group.limited_time_multiplier_duration_minutes) || 0))
  if (cron.frequency === 'weekly' && current.getDay() !== cron.weekday) return false
  if (cron.frequency === 'monthly' && current.getDate() !== cron.monthDay) return false

  const startTotalMinutes = cron.hour * 60 + cron.minute
  const currentTotalMinutes = current.getHours() * 60 + current.getMinutes()
  const endTotalMinutes = startTotalMinutes + durationMinutes
  if (endTotalMinutes <= 24 * 60) {
    return currentTotalMinutes >= startTotalMinutes && currentTotalMinutes < endTotalMinutes
  }
  return currentTotalMinutes >= startTotalMinutes || currentTotalMinutes < endTotalMinutes % (24 * 60)
}

export function formatLimitedTimeMultiplier(
  group: LimitedTimeGroup,
  t: TranslateFn,
  current = new Date(),
  userRateMultiplier?: number | null
): string | null {
  if (
    group.subscription_type === 'subscription' ||
    !group.limited_time_multiplier_enabled ||
    !group.limited_time_multiplier_cron
  ) {
    return null
  }

  const cron = parseLimitedTimeCron(group.limited_time_multiplier_cron)
  const durationMinutes = Math.max(0, Math.floor(Number(group.limited_time_multiplier_duration_minutes) || 0))
  const startTotalMinutes = cron.hour * 60 + cron.minute
  const endTotalMinutes = startTotalMinutes + durationMinutes
  const endHour = Math.floor((endTotalMinutes % (24 * 60)) / 60)
  const endMinute = endTotalMinutes % 60
  const timeRange = `${formatTimeOfDay(cron.hour, cron.minute)}-${formatTimeOfDay(endHour, endMinute)}`

  let schedule = t('admin.groups.limitedTimeMultiplier.schedule.daily')
  if (cron.frequency === 'weekly') {
    const weekdayKeys = [
      'sunday',
      'monday',
      'tuesday',
      'wednesday',
      'thursday',
      'friday',
      'saturday'
    ]
    const weekday = t(`admin.groups.limitedTimeMultiplier.weekdays.${weekdayKeys[cron.weekday]}`)
    schedule = t('admin.groups.limitedTimeMultiplier.schedule.weekly', { weekday })
  } else if (cron.frequency === 'monthly') {
    schedule = t('admin.groups.limitedTimeMultiplier.schedule.monthly', { day: cron.monthDay })
  }

  return t('admin.groups.limitedTimeMultiplier.tableBadge', {
    value: group.limited_time_multiplier_value,
    schedule,
    timeRange,
    active: isLimitedTimeMultiplierActive(group, current, userRateMultiplier)
      ? t('admin.groups.limitedTimeMultiplier.activeBadge')
      : ''
  }).trim()
}
