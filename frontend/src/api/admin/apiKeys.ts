/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupIds - Ordered group IDs (empty to unbind all)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupIds: number[]): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_ids: [...groupIds]
  })
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup
}

export default apiKeysAPI
