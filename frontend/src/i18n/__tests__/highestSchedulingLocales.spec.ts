import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const localeKeys = [
  'highestSchedulingMode',
  'highestSchedulingModeHint',
  'highestSchedulingRecoveryMinutes',
  'highestSchedulingRecoveryMinutesHint',
  'highestSchedulingSuppressed',
  'highestSchedulingSuppressedUntil',
  'highestSchedulingSuppressedManual',
  'highestSchedulingManualResume',
  'highestSchedulingSuppressedReason'
] as const

describe('highest scheduling account locale keys', () => {
  it('contains English labels and hints', () => {
    for (const key of localeKeys) {
      expect(en.admin.accounts[key]).toBeTruthy()
    }
    expect(en.admin.accounts.highestSchedulingMode).toBe('Highest Scheduling Mode')
    expect(en.admin.accounts.highestSchedulingManualResume).toBe('Resume highest scheduling now')
  })

  it('contains Chinese labels and hints', () => {
    for (const key of localeKeys) {
      expect(zh.admin.accounts[key]).toBeTruthy()
    }
    expect(zh.admin.accounts.highestSchedulingMode).toBe('最高调度模式')
    expect(zh.admin.accounts.highestSchedulingManualResume).toBe('立即恢复最高调度')
  })
})
