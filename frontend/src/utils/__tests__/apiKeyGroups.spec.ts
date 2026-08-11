import { describe, expect, it } from 'vitest'
import type { ApiKey, Group } from '@/types'
import { getApiKeyGroupIds, resolveApiKeyGroups } from '@/utils/apiKeyGroups'

const group = (id: number, name = `Group ${id}`) => ({ id, name } as Group)

describe('API key group compatibility helpers', () => {
  it('preserves an explicit empty group_ids array', () => {
    const key = { group_id: 7, group_ids: [] } as ApiKey

    expect(getApiKeyGroupIds(key)).toEqual([])
  })

  it('falls back to groups and then the legacy group field', () => {
    expect(getApiKeyGroupIds({ group_id: 7, groups: [group(3), group(8)] } as ApiKey)).toEqual([3, 8])
    expect(getApiKeyGroupIds({ group_id: 7 } as ApiKey)).toEqual([7])
    expect(getApiKeyGroupIds({ group_id: null } as ApiKey)).toEqual([])
  })

  it('resolves metadata without changing configured order', () => {
    const key = { group_ids: [8, 3] } as ApiKey
    const resolved = resolveApiKeyGroups(key, [group(3), group(8)])

    expect(resolved.map((entry) => entry.id)).toEqual([8, 3])
    expect(resolved.map((entry) => entry.group?.name)).toEqual(['Group 8', 'Group 3'])
  })
})
