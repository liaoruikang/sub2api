import { describe, expect, it } from 'vitest'

import {
  HIGHEST_SCHEDULING_EXTRA_KEYS,
  applyHighestSchedulingExtra,
  buildHighestSchedulingExtraPatch,
  readHighestSchedulingState
} from '../highestScheduling'

const legacyHighestSchedulingExtra = {
  highest_scheduling_recovery_minutes: 30,
  highest_scheduling_suppressed: true,
  highest_scheduling_suppressed_until: '2026-06-09T12:15:00Z',
  highest_scheduling_suppressed_at: '2026-06-09T12:00:00Z',
  highest_scheduling_suppressed_reason: 'boom'
}

describe('highest scheduling account extra helpers', () => {
  it('strictly reads only boolean true as enabled', () => {
    expect(readHighestSchedulingState({ highest_scheduling_mode: true })).toEqual({ enabled: true })
    expect(readHighestSchedulingState({ highest_scheduling_mode: false })).toEqual({ enabled: false })
    expect(readHighestSchedulingState({ highest_scheduling_mode: 1 })).toEqual({ enabled: false })
    expect(readHighestSchedulingState({ highest_scheduling_mode: 'true' })).toEqual({ enabled: false })
    expect(readHighestSchedulingState(null)).toEqual({ enabled: false })
  })

  it('applies enabled mode while preserving unrelated extra and removing all legacy keys', () => {
    const extra = applyHighestSchedulingExtra(
      {
        unrelated: 'kept',
        ...legacyHighestSchedulingExtra
      },
      { enabled: true }
    )

    expect(extra).toEqual({
      unrelated: 'kept',
      highest_scheduling_mode: true
    })
  })

  it('applies disabled mode by removing mode and all legacy keys from complete extra', () => {
    const extra = applyHighestSchedulingExtra(
      {
        unrelated: 1,
        highest_scheduling_mode: true,
        ...legacyHighestSchedulingExtra
      },
      { enabled: false }
    )

    expect(extra).toEqual({ unrelated: 1 })
  })

  it('builds a mode-only incremental patch and sends explicit false when disabled', () => {
    expect(buildHighestSchedulingExtraPatch(true)).toEqual({ highest_scheduling_mode: true })
    expect(buildHighestSchedulingExtraPatch(false)).toEqual({ highest_scheduling_mode: false })
  })

  it('exports only the current backend extra key', () => {
    expect(HIGHEST_SCHEDULING_EXTRA_KEYS).toEqual({
      mode: 'highest_scheduling_mode'
    })
  })
})
