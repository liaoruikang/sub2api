import { apiClient } from '../client'
import type { PaginatedResponse, UserTag } from '@/types'

export interface TagUser {
  id: number
  email: string
  username: string
  status: 'active' | 'disabled'
}

export async function list(): Promise<UserTag[]> {
  const { data } = await apiClient.get<UserTag[]>('/admin/tags')
  return data
}

export async function create(name: string): Promise<UserTag> {
  const { data } = await apiClient.post<UserTag>('/admin/tags', { name })
  return data
}

export async function update(id: number, name: string): Promise<UserTag> {
  const { data } = await apiClient.put<UserTag>(`/admin/tags/${id}`, { name })
  return data
}

export async function deleteTag(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/tags/${id}`)
  return data
}

export async function getUserTags(userId: number): Promise<UserTag[]> {
  const { data } = await apiClient.get<UserTag[]>(`/admin/users/${userId}/tags`)
  return data
}

export async function updateUserTags(userId: number, tagIds: number[]): Promise<UserTag[]> {
  const { data } = await apiClient.put<UserTag[]>(`/admin/users/${userId}/tags`, { tag_ids: tagIds })
  return data
}

export async function getGroupTags(groupId: number): Promise<UserTag[]> {
  const { data } = await apiClient.get<UserTag[]>(`/admin/groups/${groupId}/tags`)
  return data
}

export async function updateGroupTags(groupId: number, tagIds: number[]): Promise<UserTag[]> {
  const { data } = await apiClient.put<UserTag[]>(`/admin/groups/${groupId}/tags`, { tag_ids: tagIds })
  return data
}

export async function getTagUsers(
  tagId: number,
  page = 1,
  pageSize = 20,
  filters?: {
    search?: string
    status?: 'active' | 'disabled'
  }
): Promise<PaginatedResponse<TagUser>> {
  const { data } = await apiClient.get<PaginatedResponse<TagUser>>(`/admin/tags/${tagId}/users`, {
    params: {
      page,
      page_size: pageSize,
      search: filters?.search,
      status: filters?.status
    }
  })
  return data
}

export async function addUsersToTag(tagId: number, userIds: number[]): Promise<{ affected: number }> {
  const { data } = await apiClient.post<{ affected: number }>(`/admin/tags/${tagId}/users`, {
    user_ids: userIds
  })
  return data
}

const tagsAPI = {
  list,
  create,
  update,
  delete: deleteTag,
  getUserTags,
  updateUserTags,
  getGroupTags,
  updateGroupTags,
  getTagUsers,
  addUsersToTag
}

export default tagsAPI
