import { describe, expect, it } from 'vitest'

import {
  HIGHEST_SCHEDULING_EXTRA_KEYS,
  applyHighestSchedulingExtra,
  clearHighestSchedulingSuppression,
  normalizeHighestSchedulingRecoveryMinutes,
  readHighestSchedulingState
} from '../highestScheduling'

describe('highest scheduling account extra helpers', () => {
  it('writes enabled highest scheduling config while preserving unrelated extra', () => {
    const extra = applyHighestSchedulingExtra(
      { unrelated: 'kept' },
      { enabled: true, recoveryMinutes: '15' }
    )

    expect(extra).toEqual({
      unrelated: 'kept',
      highest_scheduling_mode: true,
      highest_scheduling_recovery_minutes: 15
    })
  })

  it('normalizes recovery minutes into the backend accepted range', () => {
    expect(normalizeHighestSchedulingRecoveryMinutes(undefined)).toBe(0)
    expect(normalizeHighestSchedulingRecoveryMinutes('')).toBe(0)
    expect(normalizeHighestSchedulingRecoveryMinutes('-3')).toBe(0)
    expect(normalizeHighestSchedulingRecoveryMinutes('12.8')).toBe(12)
    expect(normalizeHighestSchedulingRecoveryMinutes(10081)).toBe(10080)
  })

  it('clears mode and suppression metadata when disabled', () => {
    const extra = applyHighestSchedulingExtra(
      {
        unrelated: 1,
        highest_scheduling_mode: true,
        highest_scheduling_recovery_minutes: 30,
        highest_scheduling_suppressed: true,
        highest_scheduling_suppressed_until: '2026-06-09T12:15:00Z',
        highest_scheduling_suppressed_at: '2026-06-09T12:00:00Z',
        highest_scheduling_suppressed_reason: 'boom'
      },
      { enabled: false, recoveryMinutes: 30 }
    )

    expect(extra).toEqual({ unrelated: 1 })
  })

  it('clears only suppression metadata for manual resume', () => {
    const extra = clearHighestSchedulingSuppression({
      unrelated: 'kept',
      highest_scheduling_mode: true,
      highest_scheduling_recovery_minutes: 15,
      highest_scheduling_suppressed: true,
      highest_scheduling_suppressed_until: '2026-06-09T12:15:00Z',
      highest_scheduling_suppressed_at: '2026-06-09T12:00:00Z',
      highest_scheduling_suppressed_reason: 'boom'
    })

    expect(extra).toEqual({
      unrelated: 'kept',
      highest_scheduling_mode: true,
      highest_scheduling_recovery_minutes: 15
    })
  })

  it('reads active timed and manual suppression state', () => {
    const now = new Date('2026-06-09T12:00:00Z')

    const timed = readHighestSchedulingState(
      {
        highest_scheduling_mode: true,
        highest_scheduling_recovery_minutes: '15',
        highest_scheduling_suppressed_until: '2026-06-09T12:10:00Z',
        highest_scheduling_suppressed_reason: 'temporary error'
      },
      now
    )
    expect(timed.enabled).toBe(true)
    expect(timed.recoveryMinutes).toBe(15)
    expect(timed.suppressionActive).toBe(true)
    expect(timed.suppressionType).toBe('timed')
    expect(timed.suppressedReason).toBe('temporary error')

    const manual = readHighestSchedulingState(
      {
        highest_scheduling_mode: true,
        highest_scheduling_suppressed: true
      },
      now
    )
    expect(manual.suppressionActive).toBe(true)
    expect(manual.suppressionType).toBe('manual')

    const expired = readHighestSchedulingState(
      {
        highest_scheduling_mode: true,
        highest_scheduling_suppressed_until: '2026-06-09T11:59:00Z'
      },
      now
    )
    expect(expired.suppressionActive).toBe(false)
    expect(expired.suppressionType).toBe('none')
  })

  it('exports the exact backend extra keys', () => {
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.mode).toBe('highest_scheduling_mode')
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.recoveryMinutes).toBe('highest_scheduling_recovery_minutes')
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.suppressed).toBe('highest_scheduling_suppressed')
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedUntil).toBe('highest_scheduling_suppressed_until')
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedAt).toBe('highest_scheduling_suppressed_at')
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS.suppressedReason).toBe('highest_scheduling_suppressed_reason')
  })
})
