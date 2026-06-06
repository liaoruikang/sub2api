import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('../client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { apiClient } from '../client'
import {
  generateImage,
  getImageOptions,
  type ImagePlaygroundGenerateInput,
  type ImagePlaygroundGenerateResponse,
} from '../imagePlayground'

describe('imagePlayground api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
  })

  it('loads image options from the user options endpoint', async () => {
    const response = {
      keys: [
        {
          id: 1,
          name: 'Primary key',
          masked_key: 'sk-1234',
          group_id: 10,
          group_name: 'Image team',
          allow_image_generation: true,
          models: ['gpt-image-1'],
          default_model: 'gpt-image-1',
        },
      ],
      fallback_models: ['gpt-image-1'],
    }
    get.mockResolvedValue({ data: response })

    await expect(getImageOptions()).resolves.toEqual(response)
    expect(apiClient.get).toHaveBeenCalledWith('/user/images/options')
  })

  it('posts JSON without reference images when no files are provided', async () => {
    post.mockResolvedValue({ data: {} })

    const input: ImagePlaygroundGenerateInput = {
      api_key_id: 1,
      model: 'gpt-image-1',
      prompt: 'Render a neon fox',
      size: '1024x1024',
      quality: 'high',
      output_format: 'png',
      moderation: 'auto',
      n: 2,
      output_compression: undefined,
      reference_images: [],
    }

    await generateImage(input)

    expect(apiClient.post).toHaveBeenCalledTimes(1)
    expect(apiClient.post).toHaveBeenCalledWith(
      '/user/images/generations',
      {
        api_key_id: 1,
        model: 'gpt-image-1',
        prompt: 'Render a neon fox',
        size: '1024x1024',
        quality: 'high',
        output_format: 'png',
        moderation: 'auto',
        n: 2,
      },
      { timeout: 0 }
    )

    const payload = post.mock.calls[0][1] as Record<string, unknown>
    expect(payload).not.toHaveProperty('reference_images')
    expect(payload).not.toHaveProperty('output_compression')
  })

  it('posts multipart form data with reference images and output compression', async () => {
    post.mockResolvedValue({ data: {} })

    const fileA = new File(['first'], 'first.png', { type: 'image/png' })
    const fileB = new File(['second'], 'second.jpg', { type: 'image/jpeg' })

    await generateImage({
      api_key_id: 2,
      model: 'gpt-image-1',
      prompt: 'Create a collage',
      size: '1536x1024',
      quality: 'medium',
      output_format: 'webp',
      output_compression: 90,
      moderation: 'low',
      n: 1,
      reference_images: [fileA, fileB],
    })

    expect(apiClient.post).toHaveBeenCalledTimes(1)
    expect(apiClient.post).toHaveBeenCalledWith('/user/images/generations', expect.any(FormData), {
      headers: { 'Content-Type': undefined },
      timeout: 0,
    })

    const formData = post.mock.calls[0][1] as FormData
    expect(formData.get('api_key_id')).toBe('2')
    expect(formData.get('model')).toBe('gpt-image-1')
    expect(formData.get('prompt')).toBe('Create a collage')
    expect(formData.get('size')).toBe('1536x1024')
    expect(formData.get('quality')).toBe('medium')
    expect(formData.get('output_format')).toBe('webp')
    expect(formData.get('output_compression')).toBe('90')
    expect(formData.get('moderation')).toBe('low')
    expect(formData.get('n')).toBe('1')
    expect(formData.getAll('image')).toEqual([fileA, fileB])
  })

  it('returns gateway response metadata for image generation billing', async () => {
    const response: ImagePlaygroundGenerateResponse = {
      data: [{ url: 'https://example.com/generated.png' }],
      _sub2api_image_playground: {
        actual_cost: 0.42,
        total_cost: 0.42,
        image_count: 1,
      },
    }
    post.mockResolvedValue({ data: response })

    const result = await generateImage({
      api_key_id: 3,
      model: 'gpt-image-1',
      prompt: 'Create an icon',
      size: '1024x1024',
      quality: 'high',
      output_format: 'png',
      moderation: 'auto',
      n: 1,
      reference_images: [],
    })

    expect(result._sub2api_image_playground?.actual_cost).toBe(0.42)
  })
})
