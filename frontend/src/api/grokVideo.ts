import { apiClient } from './client'
import type { FetchOptions, PaginatedResponse } from '@/types'

export type GrokVideoJobStatus =
  | 'pending'
  | 'running'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | string

export interface GrokVideoJob {
  id: number
  request_id: string
  user_id: number
  api_key_id?: number | null
  group_id?: number | null
  account_id?: number | null
  model: string
  prompt_preview: string
  status: GrokVideoJobStatus
  progress: number
  progress_text?: string
  result_url?: string | null
  result_urls?: string[]
  cover_image_url?: string | null
  last_error_code?: string | null
  last_error_message?: string | null
  created_at: string
  updated_at: string
  submitted_at: string
  last_polled_at?: string | null
  finished_at?: string | null
}

export interface ListGrokVideoJobsFilters {
  status?: string
  api_key_id?: number | null
  model?: string
  active_only?: boolean
}

export interface RefreshGrokVideoJobsRequest {
  request_ids?: string[]
  active_only?: boolean
  limit?: number
}

export interface RefreshGrokVideoJobsResponse {
  items: GrokVideoJob[]
}

export async function listGrokVideoJobs(
  page: number = 1,
  pageSize: number = 20,
  filters?: ListGrokVideoJobsFilters,
  options?: FetchOptions,
): Promise<PaginatedResponse<GrokVideoJob>> {
  const params: Record<string, string | number | boolean> = {
    page,
    page_size: pageSize,
  }

  if (filters?.status) {
    params.status = filters.status
  }
  if (typeof filters?.api_key_id === 'number') {
    params.api_key_id = filters.api_key_id
  }
  if (filters?.model?.trim()) {
    params.model = filters.model.trim()
  }
  if (filters?.active_only) {
    params.active_only = true
  }

  const { data } = await apiClient.get<PaginatedResponse<GrokVideoJob>>('/user/videos/jobs', {
    params,
    signal: options?.signal,
  })
  return data
}

export async function getGrokVideoJob(requestId: string, options?: FetchOptions): Promise<GrokVideoJob> {
  const { data } = await apiClient.get<GrokVideoJob>(`/user/videos/jobs/${encodeURIComponent(requestId)}`, {
    signal: options?.signal,
  })
  return data
}

export async function refreshGrokVideoJobs(
  payload: RefreshGrokVideoJobsRequest,
  options?: FetchOptions,
): Promise<RefreshGrokVideoJobsResponse> {
  const { data } = await apiClient.post<RefreshGrokVideoJobsResponse>('/user/videos/jobs/refresh', payload, {
    signal: options?.signal,
  })
  return data
}

export const grokVideoAPI = {
  list: listGrokVideoJobs,
  get: getGrokVideoJob,
  refresh: refreshGrokVideoJobs,
}

export default grokVideoAPI
