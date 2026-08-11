import { describe, expect, it } from 'vitest'

import {
  formatLimitedTimeMultiplier,
  isLimitedTimeMultiplierActive
} from '@/utils/limitedTimeMultiplier'

const makeGroup = (overrides: Partial<{
  rate_multiplier: number
  limited_time_multiplier_enabled: boolean
  limited_time_multiplier_cron: string
  limited_time_multiplier_duration_minutes: number
  limited_time_multiplier_value: number
  subscription_type: 'standard' | 'subscription'
  platform: 'openai'
}> = {}) => ({
  platform: 'openai' as const,
  subscription_type: 'standard' as const,
  rate_multiplier: 1,
  limited_time_multiplier_enabled: true,
  limited_time_multiplier_cron: '0 9 * * *',
  limited_time_multiplier_duration_minutes: 60,
  limited_time_multiplier_value: 0.5,
  ...overrides
})

const translate = (key: string, params?: Record<string, unknown>) => {
  if (key.endsWith('.tableBadge')) {
    return `${params?.value}x ${params?.active ?? ''}`.trim()
  }
  if (key.endsWith('.activeBadge')) return 'active'
  if (key.endsWith('.schedule.daily')) return 'daily'
  return key
}

describe('limited-time multiplier effective rate', () => {
  it('uses the user-specific rate as the comparison baseline', () => {
    const group = makeGroup({ rate_multiplier: 1, limited_time_multiplier_value: 0.8 })
    const current = new Date(2026, 7, 9, 9, 30)

    expect(isLimitedTimeMultiplierActive(group, current, 0.7)).toBe(false)
    expect(isLimitedTimeMultiplierActive(group, current, 1)).toBe(true)
  })

  it('marks the limited-time value active in formatted text only when effective', () => {
    const group = makeGroup({ rate_multiplier: 1, limited_time_multiplier_value: 0.8 })
    const current = new Date(2026, 7, 9, 9, 30)

    expect(formatLimitedTimeMultiplier(group, translate, current, 0.7)).toBe('0.8x')
    expect(formatLimitedTimeMultiplier(group, translate, current, 1)).toBe('0.8x active')
  })

  it('does not apply limited-time rates to subscription groups', () => {
    const group = makeGroup({ subscription_type: 'subscription' })
    const current = new Date(2026, 7, 9, 9, 30)

    expect(isLimitedTimeMultiplierActive(group, current, 2)).toBe(false)
  })
})
