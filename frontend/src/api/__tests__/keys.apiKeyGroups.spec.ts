import { afterEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '@/api/client'
import { keysAPI } from '@/api/keys'
import { adminAPI } from '@/api/admin'

vi.mock('@/api/client', () => ({
  apiClient: {
    post: vi.fn(),
    put: vi.fn()
  }
}))

describe('API key ordered group payloads', () => {
  afterEach(() => vi.clearAllMocks())

  it('sends ordered group_ids when creating a key', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1 } })

    await keysAPI.create('ordered', undefined, undefined, undefined, undefined, undefined, undefined, undefined, [8, 3])

    expect(apiClient.post).toHaveBeenCalledWith('/keys', expect.objectContaining({
      name: 'ordered',
      group_ids: [8, 3]
    }))
    expect(apiClient.post).not.toHaveBeenCalledWith('/keys', expect.objectContaining({ group_id: expect.anything() }))
  })

  it('keeps legacy positional creation compatible', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { id: 1 } })

    await keysAPI.create('legacy', 8)

    expect(apiClient.post).toHaveBeenCalledWith('/keys', expect.objectContaining({ name: 'legacy', group_id: 8 }))
  })

  it('sends an empty array when an administrator unbinds all groups', async () => {
    vi.mocked(apiClient.put).mockResolvedValue({ data: { api_key: { id: 1 } } })

    await adminAPI.apiKeys.updateApiKeyGroup(1, [])

    expect(apiClient.put).toHaveBeenCalledWith('/admin/api-keys/1', { group_ids: [] })
  })
})
