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

export async function getImageOptions(): Promise<ImagePlaygroundOptions> {
  const { data } = await apiClient.get<ImagePlaygroundOptions>('/user/images/options')
  return data
}

// Axios treats 0 as no timeout; image generation can legitimately exceed the shared 30s client timeout.
const IMAGE_GENERATION_TIMEOUT_MS = 0

const appendScalar = (formData: FormData, key: string, value: string | number): void => {
  formData.append(key, String(value))
}

export async function generateImage(
  input: ImagePlaygroundGenerateInput
): Promise<ImagePlaygroundGenerateResponse> {
  if (input.reference_images.length > 0) {
    const formData = new FormData()

    appendScalar(formData, 'api_key_id', input.api_key_id)
    appendScalar(formData, 'model', input.model)
    appendScalar(formData, 'prompt', input.prompt)
    appendScalar(formData, 'size', input.size)
    appendScalar(formData, 'quality', input.quality)
    appendScalar(formData, 'output_format', input.output_format)
    appendScalar(formData, 'moderation', input.moderation)
    appendScalar(formData, 'n', input.n)

    if (typeof input.output_compression === 'number') {
      appendScalar(formData, 'output_compression', input.output_compression)
    }

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

  const payload: Omit<ImagePlaygroundGenerateInput, 'reference_images'> = {
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

  const { data } = await apiClient.post<ImagePlaygroundGenerateResponse>(
    '/user/images/generations',
    payload,
    { timeout: IMAGE_GENERATION_TIMEOUT_MS }
  )
  return data
}

export const imagePlaygroundAPI = {
  getImageOptions,
  generateImage,
}

export default imagePlaygroundAPI
