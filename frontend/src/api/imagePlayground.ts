import { getLocale } from '@/i18n'

import { apiClient } from './client'

export interface ImagePlaygroundKeyOption {
  id: number
  name: string
  masked_key: string
  group_id: number | null
  group_name?: string
  allow_image_generation: boolean
  models: string[]
  default_model: string
}

export interface ImagePlaygroundOptions {
  keys: ImagePlaygroundKeyOption[]
  fallback_models: string[]
}

export interface ImagePlaygroundGenerateInput {
  api_key_id: number
  model: string
  prompt: string
  size: string
  quality: string
  output_format: string
  output_compression?: number
  moderation: string
  n: number
  reference_images: File[]
}

export interface ImagePlaygroundCostMetadata {
  estimated_price?: number
  actual_cost?: number
  total_cost?: number
  image_count?: number
  image_size?: string
  billing_mode?: string
}

export interface ImagePlaygroundImageResult {
  b64_json?: string
  url?: string
  revised_prompt?: string
}

export interface ImagePlaygroundGenerateResponse {
  data?: ImagePlaygroundImageResult[]
  _sub2api_image_playground?: ImagePlaygroundCostMetadata
  [key: string]: unknown
}

export interface ImagePlaygroundStreamEvent {
  type?: string
  object?: string
  data?: unknown
  b64_json?: string
  url?: string
  revised_prompt?: string
  result?: unknown
  message?: string
  error?: string | { message?: string; code?: string }
  [key: string]: unknown
}

export interface ImagePlaygroundStreamOptions {
  signal?: AbortSignal
  onProgress?: (event: ImagePlaygroundStreamEvent) => void
  onImage?: (image: ImagePlaygroundImageResult, event: ImagePlaygroundStreamEvent) => void
}

export interface ImagePlaygroundAdvancedGenerateInput {
  body: Record<string, unknown>
  reference_images: File[]
}

export async function getImageOptions(): Promise<ImagePlaygroundOptions> {
  const { data } = await apiClient.get<ImagePlaygroundOptions>('/user/images/options')
  return data
}

// Axios treats 0 as no timeout; image generation can legitimately exceed the shared 30s client timeout.
const IMAGE_GENERATION_TIMEOUT_MS = 0
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

const appendScalar = (formData: FormData, key: string, value: string | number | boolean): void => {
  formData.append(key, String(value))
}

export const imageGeneratePayload = (input: ImagePlaygroundGenerateInput): Record<string, unknown> => {
  const payload: Record<string, unknown> = {
    api_key_id: input.api_key_id,
    model: input.model,
    prompt: input.prompt,
    size: input.size,
    quality: input.quality,
    output_format: input.output_format,
    moderation: input.moderation,
    n: input.n,
  }

  if (typeof input.output_compression === 'number') {
    payload.output_compression = input.output_compression
  }

  return payload
}

const appendAdvancedBody = (formData: FormData, body: Record<string, unknown>): void => {
  Object.entries(body).forEach(([key, value]) => {
    if (key === 'image' || value == null) {
      return
    }
    if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
      appendScalar(formData, key, value)
    }
  })
}

export async function generateImage(
  input: ImagePlaygroundGenerateInput
): Promise<ImagePlaygroundGenerateResponse> {
  if (input.reference_images.length > 0) {
    return generateImageAdvanced({
      body: imageGeneratePayload(input),
      reference_images: input.reference_images,
    })
  }

  const { data } = await apiClient.post<ImagePlaygroundGenerateResponse>(
    '/user/images/generations',
    imageGeneratePayload(input),
    { timeout: IMAGE_GENERATION_TIMEOUT_MS }
  )
  return data
}

export async function generateImageAdvanced(
  input: ImagePlaygroundAdvancedGenerateInput
): Promise<ImagePlaygroundGenerateResponse> {
  if (input.reference_images.length > 0) {
    const formData = new FormData()
    appendAdvancedBody(formData, input.body)
    input.reference_images.forEach((file) => {
      formData.append('image', file)
    })

    const { data } = await apiClient.post<ImagePlaygroundGenerateResponse>(
      '/user/images/generations',
      formData,
      {
        headers: { 'Content-Type': undefined },
        timeout: IMAGE_GENERATION_TIMEOUT_MS,
      }
    )
    return data
  }

  const { data } = await apiClient.post<ImagePlaygroundGenerateResponse>(
    '/user/images/generations',
    input.body,
    { timeout: IMAGE_GENERATION_TIMEOUT_MS }
  )
  return data
}

export async function generateImageStream(
  input: ImagePlaygroundGenerateInput,
  options: ImagePlaygroundStreamOptions = {}
): Promise<ImagePlaygroundGenerateResponse> {
  const payload = {
    ...imageGeneratePayload(input),
    stream: true,
  }
  const headers: Record<string, string> = {
    Accept: 'text/event-stream',
    'Accept-Language': getLocale(),
    'Content-Type': 'application/json',
  }
  const token = localStorage.getItem('auth_token')
  if (token) {
    headers.Authorization = `Bearer ${token}`
  }

  const response = await fetch(`${API_BASE_URL}/user/images/generations`, {
    method: 'POST',
    credentials: 'include',
    headers,
    body: JSON.stringify(payload),
    signal: options.signal,
  })

  if (!response.ok) {
    throw new Error(await imageStreamErrorMessage(response))
  }

  if (!response.headers.get('Content-Type')?.toLowerCase().includes('text/event-stream')) {
    return await parseImageGenerateJSONResponse(response)
  }

  return { data: await collectImageStreamResults(response, options) }
}

const parseImageGenerateJSONResponse = async (response: Response): Promise<ImagePlaygroundGenerateResponse> => {
  const json = await response.json()
  if (json && typeof json === 'object' && 'code' in json) {
    const envelope = json as { code?: unknown; data?: unknown; message?: unknown; detail?: unknown }
    if (envelope.code === 0) {
      return (envelope.data ?? {}) as ImagePlaygroundGenerateResponse
    }
    throw new Error(String(envelope.message || envelope.detail || 'Image generation failed'))
  }
  return json as ImagePlaygroundGenerateResponse
}

export const extractImageGenerationErrorMessage = (error: unknown): string => {
  if (error && typeof error === 'object') {
    const record = error as Record<string, any>
    const responseData = record.response?.data
    if (responseData && typeof responseData === 'object') {
      if (typeof responseData.error?.message === 'string' && responseData.error.message.trim()) {
        return responseData.error.message.trim()
      }
      if (typeof responseData.message === 'string' && responseData.message.trim()) {
        return responseData.message.trim()
      }
      if (typeof responseData.detail === 'string' && responseData.detail.trim()) {
        return responseData.detail.trim()
      }
    }
    if (typeof record.error?.message === 'string' && record.error.message.trim()) {
      return record.error.message.trim()
    }
    if (typeof record.message === 'string' && record.message.trim()) {
      return record.message.trim()
    }
  }
  return ''
}

const imageStreamErrorMessage = async (response: Response): Promise<string> => {
  try {
    const json = await response.clone().json()
    if (json && typeof json === 'object') {
      const payload = json as Record<string, any>
      const error = payload.error
      if (error && typeof error === 'object' && typeof error.message === 'string') {
        return error.message
      }
      if (typeof payload.message === 'string') {
        return payload.message
      }
      if (typeof payload.detail === 'string') {
        return payload.detail
      }
    }
  } catch {
    // Fall back to response text/status below.
  }

  const text = await response.text().catch(() => '')
  return text.trim() || `Image stream request failed with status ${response.status}`
}

const collectImageStreamResults = async (
  response: Response,
  options: ImagePlaygroundStreamOptions
): Promise<ImagePlaygroundImageResult[]> => {
  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('Image stream response has no body')
  }

  const decoder = new TextDecoder()
  let buffer = ''
  const results: ImagePlaygroundImageResult[] = []

  const processBlocks = (text: string, final = false): void => {
    buffer += text
    const blocks = buffer.split(/\r?\n\r?\n/)
    buffer = final ? '' : blocks.pop() ?? ''
    const completedBlocks = final ? blocks : blocks
    completedBlocks.forEach((block) => processImageStreamBlock(block, results, options))
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) {
      break
    }
    processBlocks(decoder.decode(value, { stream: true }))
  }
  processBlocks(decoder.decode(), true)

  return results
}

const processImageStreamBlock = (
  block: string,
  results: ImagePlaygroundImageResult[],
  options: ImagePlaygroundStreamOptions
): void => {
  const lines = block.split(/\r?\n/)
  let eventName = ''
  const dataLines: string[] = []

  lines.forEach((line) => {
    if (line.startsWith(':')) {
      return
    }
    if (line.startsWith('event:')) {
      eventName = line.slice(6).trim()
      return
    }
    if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).replace(/^ /, ''))
    }
  })

  const data = dataLines.join('\n').trim()
  if (!data || data === '[DONE]') {
    return
  }

  const event = JSON.parse(data) as ImagePlaygroundStreamEvent
  const type = imageStreamEventType(eventName, event)
  if (eventName === 'error' || event.type === 'error') {
    throw new Error(imageStreamEventErrorMessage(event))
  }

  options.onProgress?.(event)

  const images = imageResultsFromStreamEvent(event, type)
  images.forEach((image) => {
    results.push(image)
    options.onImage?.(image, event)
  })
}

const imageStreamEventType = (eventName: string, event: ImagePlaygroundStreamEvent): string => {
  const normalizedEventName = eventName.trim()
  if (normalizedEventName && normalizedEventName !== 'message') {
    return normalizedEventName
  }
  if (typeof event.type === 'string' && event.type.trim()) {
    return event.type.trim()
  }
  if (typeof event.object === 'string' && event.object.trim()) {
    return event.object.trim()
  }
  return normalizedEventName
}

const imageResultsFromStreamEvent = (
  event: ImagePlaygroundStreamEvent,
  type: string
): ImagePlaygroundImageResult[] => {
  const objectType = typeof event.object === 'string' ? event.object.trim() : ''
  const isCompletedEvent =
    type === 'image_generation.completed' ||
    type === 'image_edit.completed' ||
    objectType === 'image.generation.result'

  if (!isCompletedEvent) {
    return []
  }

  const results: ImagePlaygroundImageResult[] = []
  const addValue = (value: unknown): void => {
    imageResultsFromValue(value).forEach((image) => results.push(image))
  }

  addValue(event.data)
  addValue(event.result)
  addValue(event.output)
  addValue((event as { item?: unknown }).item)
  addValue(event)

  return dedupeImageResults(results)
}

const imageResultsFromValue = (value: unknown): ImagePlaygroundImageResult[] => {
  if (!value) {
    return []
  }
  if (Array.isArray(value)) {
    return value.flatMap((item) => imageResultsFromValue(item))
  }
  if (typeof value === 'string') {
    const result = imageResultFromString(value)
    return result ? [result] : []
  }
  if (typeof value !== 'object') {
    return []
  }

  const record = value as Record<string, unknown>
  const direct = imageResultFromRecord(record)
  const nested = [record.data, record.result, record.output, record.item, record.response]
    .flatMap((item) => imageResultsFromValue(item))

  return direct ? [direct, ...nested] : nested
}

const imageResultFromRecord = (record: Record<string, unknown>): ImagePlaygroundImageResult | null => {
  const result: ImagePlaygroundImageResult = {}
  const b64JSON = stringField(record.b64_json)
  const url = stringField(record.url)
  const revisedPrompt = stringField(record.revised_prompt)

  if (b64JSON) {
    result.b64_json = b64JSON
  }
  if (url) {
    result.url = url
  }
  if (revisedPrompt) {
    result.revised_prompt = revisedPrompt
  }

  const rawResult = stringField(record.result)
  if (!result.b64_json && !result.url && rawResult) {
    const parsed = imageResultFromString(rawResult)
    if (parsed) {
      Object.assign(result, parsed)
    }
  }

  return result.b64_json || result.url ? result : null
}

const imageResultFromString = (value: string): ImagePlaygroundImageResult | null => {
  const trimmed = value.trim()
  if (!trimmed) {
    return null
  }
  if (/^(data:image\/|https?:\/\/)/i.test(trimmed)) {
    return { url: trimmed }
  }
  return { b64_json: trimmed }
}

const stringField = (value: unknown): string => (typeof value === 'string' ? value.trim() : '')

const dedupeImageResults = (results: ImagePlaygroundImageResult[]): ImagePlaygroundImageResult[] => {
  const seen = new Set<string>()
  return results.filter((result) => {
    const key = result.url || result.b64_json
    if (!key || seen.has(key)) {
      return false
    }
    seen.add(key)
    return true
  })
}

const imageStreamEventErrorMessage = (event: ImagePlaygroundStreamEvent): string => {
  if (event.error && typeof event.error === 'object' && typeof event.error.message === 'string') {
    return event.error.message
  }
  if (typeof event.error === 'string' && event.error.trim()) {
    return event.error.trim()
  }
  if (typeof event.message === 'string' && event.message.trim()) {
    return event.message.trim()
  }
  return 'Image stream failed'
}

export const imagePlaygroundAPI = {
  getImageOptions,
  generateImage,
  generateImageStream,
}

export default imagePlaygroundAPI
