import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const removedLocaleKeys = [
  'highestSchedulingRecoveryMinutes',
  'highestSchedulingRecoveryMinutesHint',
  'highestSchedulingSuppressed',
  'highestSchedulingSuppressedUntil',
  'highestSchedulingSuppressedManual',
  'highestSchedulingManualResume',
  'highestSchedulingSuppressedReason'
] as const

describe('highest scheduling account locale keys', () => {
  it('contains mode-only English labels and hint', () => {
    expect(en.admin.accounts.highestSchedulingMode).toBe('Highest Scheduling Mode')
    expect(en.admin.accounts.highestSchedulingModeHint).toBe(
      'Prioritizes candidates that already satisfy account status and schedulability requirements.'
    )
    for (const key of removedLocaleKeys) {
      expect(en.admin.accounts).not.toHaveProperty(key)
    }
  })

  it('contains mode-only Chinese labels and hint', () => {
    expect(zh.admin.accounts.highestSchedulingMode).toBe('最高调度模式')
    expect(zh.admin.accounts.highestSchedulingModeHint).toBe(
      '仅提升已满足账号状态和可调度条件的候选账号。'
    )
    for (const key of removedLocaleKeys) {
      expect(zh.admin.accounts).not.toHaveProperty(key)
    }
  })
})
