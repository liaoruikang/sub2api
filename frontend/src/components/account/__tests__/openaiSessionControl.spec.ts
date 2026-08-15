import { describe, expect, it } from 'vitest'
import {
  DEFAULT_OPENAI_SESSION_TIMEOUT_UNIT,
  DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE,
  openAISessionTimeoutFromSeconds,
  openAISessionTimeoutToSeconds
} from '../openaiSessionControl'

describe('OpenAI SessionID timeout conversion', () => {
  it('converts the supported units to seconds', () => {
    expect(openAISessionTimeoutToSeconds(30, 'seconds')).toBe(30)
    expect(openAISessionTimeoutToSeconds(2, 'minutes')).toBe(120)
    expect(openAISessionTimeoutToSeconds(3, 'days')).toBe(259200)
  })

  it('uses one day for an invalid value', () => {
    expect(openAISessionTimeoutToSeconds(null, DEFAULT_OPENAI_SESSION_TIMEOUT_UNIT)).toBe(86400)
    expect(openAISessionTimeoutToSeconds(0, 'days')).toBe(
      DEFAULT_OPENAI_SESSION_TIMEOUT_VALUE * 86400
    )
  })

  it('chooses the largest exact display unit', () => {
    expect(openAISessionTimeoutFromSeconds(172800)).toEqual({ value: 2, unit: 'days' })
    expect(openAISessionTimeoutFromSeconds(120)).toEqual({ value: 2, unit: 'minutes' })
    expect(openAISessionTimeoutFromSeconds(61)).toEqual({ value: 61, unit: 'seconds' })
  })
})
