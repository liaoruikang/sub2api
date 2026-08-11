import type { ApiKey, Group } from '@/types'

export interface ResolvedApiKeyGroup {
  id: number
  group?: Group
}

export function getApiKeyGroupIds(key: ApiKey): number[] {
  if (Array.isArray(key.group_ids)) {
    return [...key.group_ids]
  }
  if (Array.isArray(key.groups)) {
    return key.groups.map((group) => group.id)
  }
  return key.group_id != null ? [key.group_id] : []
}

export function resolveApiKeyGroups(
  key: ApiKey,
  availableGroups: Group[] = []
): ResolvedApiKeyGroup[] {
  const groupsById = new Map(availableGroups.map((group) => [group.id, group]))
  const embeddedGroups = [...(key.groups ?? []), ...(key.group ? [key.group] : [])]
  const embeddedGroupsById = new Map(embeddedGroups.map((group) => [group.id, group]))

  return getApiKeyGroupIds(key).map((id) => ({
    id,
    group: embeddedGroupsById.get(id) ?? groupsById.get(id)
  }))
}
