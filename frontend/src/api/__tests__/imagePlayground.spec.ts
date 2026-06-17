import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

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
  generateImageStream,
  getImageOptions,
  type ImagePlaygroundGenerateInput,
  type ImagePlaygroundGenerateResponse,
} from '../imagePlayground'

const streamResponse = (body: string): Response => {
  return new Response(
    new ReadableStream({
      start(controller) {
        controller.enqueue(new TextEncoder().encode(body))
        controller.close()
      },
    }),
    {
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
    }
  )
}

const baseInput = (overrides: Partial<ImagePlaygroundGenerateInput> = {}): ImagePlaygroundGenerateInput => ({
  api_key_id: 1,
  model: 'gpt-image-2',
  prompt: 'Render a neon fox',
  size: '1024x1024',
  quality: 'high',
  output_format: 'png',
  moderation: 'auto',
  n: 2,
  output_compression: undefined,
  reference_images: [],
  ...overrides,
})

describe('imagePlayground api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    localStorage.setItem('auth_token', 'token-123')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    localStorage.clear()
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

    const input = baseInput({
      model: 'gpt-image-1',
      n: 2,
    })

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
    expect(payload).not.toHaveProperty('stream')
  })

  it('posts multipart form data with reference images and output compression', async () => {
    post.mockResolvedValue({ data: {} })

    const fileA = new File(['first'], 'first.png', { type: 'image/png' })
    const fileB = new File(['second'], 'second.jpg', { type: 'image/jpeg' })

    await generateImage(
      baseInput({
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
    )

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
    expect(formData.get('stream')).toBeNull()
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

    const result = await generateImage(
      baseInput({
        api_key_id: 3,
        model: 'gpt-image-1',
        prompt: 'Create an icon',
        n: 1,
      })
    )

    expect(result._sub2api_image_playground?.actual_cost).toBe(0.42)
  })

  it('streams image generation with auth headers and parses completed events', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      streamResponse(
        ': keepalive\n\n' +
          'event: image_generation.partial_image\n' +
          'data: {"type":"image_generation.partial_image","b64_json":"cGFydGlhbA=="}\n\n' +
          'event: image_generation.completed\n' +
          'data: {"type":"image_generation.completed","b64_json":"ZmluYWw=","revised_prompt":"done"}\n\n' +
          'data: [DONE]\n\n'
      )
    )
    vi.stubGlobal('fetch', fetchMock)

    const result = await generateImageStream(baseInput({ n: 4 }))

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0] as [string, RequestInit]
    expect(url).toBe('/api/v1/user/images/generations')
    expect(options.method).toBe('POST')
    expect(options.credentials).toBe('include')
    expect(options.headers).toMatchObject({
      Accept: 'text/event-stream',
      'Accept-Language': 'en',
      Authorization: 'Bearer token-123',
      'Content-Type': 'application/json',
    })
    expect(JSON.parse(String(options.body))).toEqual({
      api_key_id: 1,
      model: 'gpt-image-2',
      prompt: 'Render a neon fox',
      size: '1024x1024',
      quality: 'high',
      output_format: 'png',
      moderation: 'auto',
      n: 4,
      stream: true,
    })
    expect(result.data).toEqual([
      {
        b64_json: 'ZmluYWw=',
        revised_prompt: 'done',
      },
    ])
  })

  it('parses multi-line streamed data events', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        streamResponse(
          'event: image_generation.completed\n' +
            'data: {"type":"image_generation.completed",\n' +
            'data: "url":"data:image/png;base64,QUJD"}\n\n'
        )
      )
    )

    const result = await generateImageStream(baseInput())

    expect(result.data).toEqual([{ url: 'data:image/png;base64,QUJD' }])
  })

  it('parses OpenAI image generation result message events', async () => {
    const onProgress = vi.fn()
    const onImage = vi.fn()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        streamResponse(
          'data: {"object":"image.generation.chunk","index":1,"total":2,"progress_text":"rendering"}\n\n' +
            'data: {"object":"image.generation.result","index":1,"total":2,"data":[{"b64_json":"Zmlyc3Q=","revised_prompt":"first done"}]}\n\n' +
            'data: {"object":"image.generation.result","index":2,"total":2,"data":[{"url":"https://example.com/second.png"}]}\n\n' +
            'data: [DONE]\n\n'
        )
      )
    )

    const result = await generateImageStream(baseInput({ n: 2 }), { onProgress, onImage })

    expect(result.data).toEqual([
      { b64_json: 'Zmlyc3Q=', revised_prompt: 'first done' },
      { url: 'https://example.com/second.png' },
    ])
    expect(onProgress).toHaveBeenCalledWith(expect.objectContaining({ object: 'image.generation.chunk' }))
    expect(onImage).toHaveBeenNthCalledWith(1, { b64_json: 'Zmlyc3Q=', revised_prompt: 'first done' }, expect.any(Object))
    expect(onImage).toHaveBeenNthCalledWith(2, { url: 'https://example.com/second.png' }, expect.any(Object))
  })

  it('falls back to JSON parsing when stream endpoint returns JSON', async () => {
    const response: ImagePlaygroundGenerateResponse = {
      data: [{ url: 'https://example.com/fallback.png' }],
    }
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(response), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        })
      )
    )

    await expect(generateImageStream(baseInput())).resolves.toEqual(response)
  })
})
