import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import type { ImagePlaygroundGenerateResponse } from '@/api/imagePlayground'
import type { ImageHistoryRecord } from '@/utils/imagePlaygroundHistory'

import ImagePlaygroundView from '../ImagePlaygroundView.vue'

const {
  getImageOptionsMock,
  generateImageMock,
  generateImageAdvancedMock,
  generateImageStreamMock,
  extractImageGenerationErrorMessageMock,
  addImageHistoryRecordMock,
  listImageHistoryRecordsMock,
  deleteImageHistoryRecordMock,
  clearImageHistoryMock,
  authUserMock,
} = vi.hoisted(() => ({
  getImageOptionsMock: vi.fn(),
  generateImageMock: vi.fn(),
  generateImageAdvancedMock: vi.fn(),
  generateImageStreamMock: vi.fn(),
  extractImageGenerationErrorMessageMock: vi.fn(() => ''),
  addImageHistoryRecordMock: vi.fn(),
  listImageHistoryRecordsMock: vi.fn(),
  deleteImageHistoryRecordMock: vi.fn(),
  clearImageHistoryMock: vi.fn(),
  authUserMock: {
    value: {
      id: 7,
      username: 'image-user',
      email: 'image-user@example.com',
      role: 'user',
    },
  },
}))

vi.mock('@/api/imagePlayground', () => ({
  getImageOptions: getImageOptionsMock,
  imageGeneratePayload: (input: Record<string, unknown>) => ({
    api_key_id: input.api_key_id,
    model: input.model,
    prompt: input.prompt,
    size: input.size,
    quality: input.quality,
    output_format: input.output_format,
    moderation: input.moderation,
    n: input.n,
  }),
  generateImage: generateImageMock,
  generateImageAdvanced: generateImageAdvancedMock,
  generateImageStream: generateImageStreamMock,
  extractImageGenerationErrorMessage: extractImageGenerationErrorMessageMock,
}))

vi.mock('@/utils/imagePlaygroundHistory', () => ({
  addImageHistoryRecord: addImageHistoryRecordMock,
  listImageHistoryRecords: listImageHistoryRecordsMock,
  deleteImageHistoryRecord: deleteImageHistoryRecordMock,
  clearImageHistory: clearImageHistoryMock,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: authUserMock.value,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<div data-testid="app-layout"><slot /></div>',
  },
}))

vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: {
    name: 'BaseDialog',
    props: {
      show: Boolean,
      title: String,
    },
    emits: ['close'],
    template: `
      <div v-if="show" role="dialog" :aria-label="title" tabindex="-1" @keydown.esc="$emit('close')">
        <button type="button" aria-label="Close modal" @click="$emit('close')">close</button>
        <slot />
      </div>
    `,
  },
}))

const createOptions = () => ({
  keys: [
    {
      id: 101,
      name: 'Primary image key',
      masked_key: 'sk-1234',
      group_id: 9,
      group_name: 'Images',
      allow_image_generation: true,
      models: ['gpt-image-1', 'gpt-image-edit'],
      default_model: 'gpt-image-1',
    },
    {
      id: 202,
      name: 'Secondary key',
      masked_key: 'sk-5678',
      group_id: null,
      models: ['gpt-image-2'],
      default_model: 'gpt-image-2',
      allow_image_generation: true,
    },
  ],
  fallback_models: ['gpt-image-1', 'gpt-image-2'],
})

const createHistoryRecord = (overrides: Partial<ImageHistoryRecord> = {}): ImageHistoryRecord => ({
  id: overrides.id ?? 'history-1',
  user_id: overrides.user_id ?? 7,
  created_at: overrides.created_at ?? '2026-06-06T09:00:00.000Z',
  api_key_id: overrides.api_key_id ?? 101,
  key_name: overrides.key_name ?? 'Primary image key',
  model: overrides.model ?? 'gpt-image-1',
  prompt: overrides.prompt ?? 'A calm ocean at sunrise',
  params: {
    size: '1024x1024',
    quality: 'high',
    output_format: 'png',
    moderation: 'auto',
    n: 1,
    ...overrides.params,
  },
  price: overrides.price ?? {
    actual_cost: 0.18,
  },
  images: overrides.images ?? [
    {
      url_or_base64: 'https://example.com/history.png',
      mime_type: 'image/png',
    },
  ],
})

const mountView = async () => {
  const wrapper = mount(ImagePlaygroundView)
  await flushPromises()
  return wrapper
}

const getAdvancedDialog = (wrapper: VueWrapper) => {
  const dialog = wrapper.find('[role="dialog"]')
  if (!dialog.exists()) {
    throw new Error('Advanced details dialog was not found')
  }

  return dialog
}

const getGenerateButton = (wrapper: VueWrapper): HTMLButtonElement =>
  wrapper.get('[data-test="image-generate"]').element as HTMLButtonElement

const getButtonByText = (wrapper: VueWrapper, text: string) => {
  const button = wrapper.findAll('button').find((candidate) => candidate.text() === text)
  if (!button) {
    throw new Error(`Button with text ${text} was not found`)
  }

  return button
}

const uploadReferenceFiles = async (wrapper: VueWrapper, files: File[]): Promise<void> => {
  const fileInput = wrapper.get('[data-test="image-reference"]')

  Object.defineProperty(fileInput.element, 'files', {
    value: files,
    configurable: true,
  })

  await fileInput.trigger('change')
  await flushPromises()
}

const dropReferenceFiles = async (wrapper: VueWrapper, files: File[]): Promise<void> => {
  await wrapper.get('[data-test="image-reference-dropzone"]').trigger('drop', {
    dataTransfer: {
      files,
    },
  })
  await flushPromises()
}

const submitGeneration = async (wrapper: VueWrapper): Promise<void> => {
  await wrapper.get('form').trigger('submit')
  for (let index = 0; index < 6; index += 1) {
    await flushPromises()
  }
}

const createOversizedReferenceFile = (): File =>
  new File([new Uint8Array(20 * 1024 * 1024 + 1)], 'too-large.png', { type: 'image/png' })

const createDeferred = <T>() => {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve
    reject = promiseReject
  })

  return { promise, resolve, reject }
}

describe('ImagePlaygroundView', () => {
  beforeEach(() => {
    getImageOptionsMock.mockReset()
    generateImageMock.mockReset()
    generateImageAdvancedMock.mockReset()
    generateImageStreamMock.mockReset()
    addImageHistoryRecordMock.mockReset()
    listImageHistoryRecordsMock.mockReset()
    deleteImageHistoryRecordMock.mockReset()
    clearImageHistoryMock.mockReset()
    authUserMock.value = {
      id: 7,
      username: 'image-user',
      email: 'image-user@example.com',
      role: 'user',
    }
    window.localStorage.removeItem('image_playground_advanced_mode_enabled')
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        blob: async () => new Blob(['generated-bytes'], { type: 'image/webp' }),
      }))
    )

    getImageOptionsMock.mockResolvedValue(createOptions())
    listImageHistoryRecordsMock.mockResolvedValue([createHistoryRecord()])
    addImageHistoryRecordMock.mockResolvedValue(undefined)
    deleteImageHistoryRecordMock.mockResolvedValue(undefined)
    clearImageHistoryMock.mockResolvedValue(undefined)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
    document.body.innerHTML = ''
  })

  it('renders an empty state when no image-capable keys are available', async () => {
    getImageOptionsMock.mockResolvedValue({
      keys: [],
      fallback_models: ['gpt-image-1'],
    })
    listImageHistoryRecordsMock.mockResolvedValue([])

    const wrapper = await mountView()

    expect(wrapper.text()).toContain('imagePlayground.emptyStateTitle')
    expect(wrapper.find('[data-test="image-generate"]').exists()).toBe(false)
  })

  it('shows the selected key default model in the model input', async () => {
    const wrapper = await mountView()

    const modelInput = wrapper.get('[data-test="image-model"]')
    expect((modelInput.element as HTMLInputElement).value).toBe('gpt-image-1')
  })

  it('renders the generator panel with an independent scroll area and fixed action footer', async () => {
    const wrapper = await mountView()

    expect(wrapper.get('[data-test="image-generator-panel"]').classes()).toContain('xl:max-h-[calc(100vh-8rem)]')
    expect(wrapper.get('[data-test="image-generator-scroll"]').classes()).toContain('xl:overflow-y-auto')
    expect(wrapper.get('[data-test="image-generator-actions"]').classes()).toContain('xl:sticky')
  })

  it('disables compression for png and enables it for webp', async () => {
    const wrapper = await mountView()

    const compressionInput = wrapper.get('[data-test="image-compression"]')
    expect((compressionInput.element as HTMLInputElement).disabled).toBe(true)

    await wrapper.get('[data-test="image-format"]').setValue('webp')
    await flushPromises()

    expect((compressionInput.element as HTMLInputElement).disabled).toBe(false)
  })

  it('applies image size from the size picker dialog', async () => {
    generateImageMock.mockResolvedValue({
      data: [{ url: 'https://example.com/generated-image.png' }],
    })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-size-picker"]').trigger('click')
    await wrapper.get('[data-test="image-size-tab-ratio"]').trigger('click')
    await wrapper.get('[data-test="image-size-resolution-1k"]').trigger('click')
    await wrapper.get('[data-test="image-size-ratio-3:2"]').trigger('click')
    await wrapper.get('[data-test="image-size-confirm"]').trigger('click')
    await wrapper.get('[data-test="image-prompt"]').setValue('A cinematic landscape')
    await submitGeneration(wrapper)

    expect(wrapper.get('[data-test="image-size-value"]').text()).toBe('1536x1024')
    expect(generateImageMock).toHaveBeenCalledWith(
      expect.objectContaining({
        size: '1536x1024',
      })
    )
  })

  it('disables generation when compression is outside the accepted range', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('A calm ocean at sunrise')
    await wrapper.get('[data-test="image-format"]').setValue('webp')
    await wrapper.get('[data-test="image-compression"]').setValue('101')
    await flushPromises()

    expect(wrapper.text()).toContain('imagePlayground.compressionInvalid')
    expect(getGenerateButton(wrapper).disabled).toBe(true)
  })

  it('disables generation when count is outside the accepted range', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('A calm ocean at sunrise')
    await wrapper.get('#image-count').setValue('0')
    await flushPromises()

    expect(wrapper.text()).toContain('imagePlayground.countInvalid')
    expect(getGenerateButton(wrapper).disabled).toBe(true)
  })

  it('renders stream toggle and enables it for OpenAI image generation without references', async () => {
    const wrapper = await mountView()

    const streamToggle = wrapper.get('[data-test="image-stream"]')
    expect((streamToggle.element as HTMLInputElement).disabled).toBe(false)
    expect(wrapper.text()).toContain('imagePlayground.streamHint')
  })

  it('disables stream toggle for reference-image edit mode', async () => {
    const wrapper = await mountView()
    const referenceFile = new File(['image-bytes'], 'reference.png', { type: 'image/png' })

    await uploadReferenceFiles(wrapper, [referenceFile])

    const streamToggle = wrapper.get('[data-test="image-stream"]')
    expect((streamToggle.element as HTMLInputElement).disabled).toBe(true)
    expect(wrapper.text()).toContain('imagePlayground.streamUnsupportedHint')
  })

  it('disables stream toggle for Gemini image models', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-model"]').setValue('gemini-2.5-flash-image')
    await flushPromises()

    const streamToggle = wrapper.get('[data-test="image-stream"]')
    expect((streamToggle.element as HTMLInputElement).disabled).toBe(true)
  })

  it('uses streaming API when stream mode is enabled and supported', async () => {
    generateImageStreamMock.mockResolvedValue({
      data: [{ b64_json: 'stream-base64', revised_prompt: 'stream done' }],
    })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-stream"]').setValue(true)
    await wrapper.get('[data-test="image-prompt"]').setValue('A streamable prompt')
    await submitGeneration(wrapper)

    expect(generateImageStreamMock).toHaveBeenCalledWith(
      expect.objectContaining({
        model: 'gpt-image-1',
        prompt: 'A streamable prompt',
        n: 1,
      }),
      expect.objectContaining({
        signal: expect.any(AbortSignal),
        onProgress: expect.any(Function),
        onImage: expect.any(Function),
      })
    )
    expect(generateImageMock).not.toHaveBeenCalled()
    expect(wrapper.get('img[alt="A streamable prompt 1"]').attributes('src')).toBe(
      'data:image/png;base64,stream-base64'
    )
    expect(wrapper.find('[data-test="image-price"]').exists()).toBe(false)
    expect(addImageHistoryRecordMock).toHaveBeenCalledWith(
      expect.objectContaining({
        prompt: 'A streamable prompt',
        price: undefined,
      })
    )
  })

  it('renders streamed images as each image result arrives', async () => {
    const deferred = createDeferred<ImagePlaygroundGenerateResponse>()
    let streamOptions: { onImage?: (image: { b64_json?: string }, event: Record<string, unknown>) => void } | undefined
    generateImageStreamMock.mockImplementation((_input, options) => {
      streamOptions = options
      return deferred.promise
    })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-stream"]').setValue(true)
    await wrapper.get('[data-test="image-prompt"]').setValue('Progressive stream prompt')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    streamOptions?.onImage?.({ b64_json: 'first-stream-base64' }, { object: 'image.generation.result' })
    await flushPromises()

    expect(wrapper.get('img[alt="Progressive stream prompt 1"]').attributes('src')).toBe(
      'data:image/png;base64,first-stream-base64'
    )

    deferred.resolve({
      data: [{ b64_json: 'first-stream-base64' }, { b64_json: 'second-stream-base64' }],
    })
    await flushPromises()

    expect(wrapper.get('img[alt="Progressive stream prompt 2"]').attributes('src')).toBe(
      'data:image/png;base64,second-stream-base64'
    )
  })

  it('splits stream generation counts greater than four into smaller upstream requests', async () => {
    generateImageStreamMock.mockResolvedValue({ data: [{ b64_json: 'stream-base64' }] })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-stream"]').setValue(true)
    await wrapper.get('#image-count').setValue('10')
    await wrapper.get('[data-test="image-prompt"]').setValue('Many streamed images')
    await flushPromises()

    expect(wrapper.text()).not.toContain('imagePlayground.countInvalid')
    expect(getGenerateButton(wrapper).disabled).toBe(false)

    await submitGeneration(wrapper)

    expect(generateImageStreamMock).toHaveBeenCalledTimes(3)
    expect(generateImageStreamMock).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ prompt: 'Many streamed images', n: 4 }),
      expect.any(Object)
    )
    expect(generateImageStreamMock).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ prompt: 'Many streamed images', n: 4 }),
      expect.any(Object)
    )
    expect(generateImageStreamMock).toHaveBeenNthCalledWith(
      3,
      expect.objectContaining({ prompt: 'Many streamed images', n: 2 }),
      expect.any(Object)
    )
    expect(generateImageMock).not.toHaveBeenCalled()
  })

  it('splits non-stream generation counts greater than four into smaller upstream requests', async () => {
    generateImageMock.mockResolvedValue({ data: [{ b64_json: 'many-base64' }] })

    const wrapper = await mountView()

    await wrapper.get('#image-count').setValue('10')
    await wrapper.get('[data-test="image-prompt"]').setValue('Many sync images')
    await flushPromises()

    expect(wrapper.text()).not.toContain('imagePlayground.countInvalid')
    expect(getGenerateButton(wrapper).disabled).toBe(false)

    await submitGeneration(wrapper)

    expect(generateImageMock).toHaveBeenCalledTimes(3)
    expect(generateImageMock).toHaveBeenNthCalledWith(1, expect.objectContaining({ prompt: 'Many sync images', n: 4 }))
    expect(generateImageMock).toHaveBeenNthCalledWith(2, expect.objectContaining({ prompt: 'Many sync images', n: 4 }))
    expect(generateImageMock).toHaveBeenNthCalledWith(3, expect.objectContaining({ prompt: 'Many sync images', n: 2 }))
    expect(generateImageStreamMock).not.toHaveBeenCalled()
  })

  it('shows indeterminate progress while non-stream generation is pending', async () => {
    const deferred = createDeferred<ImagePlaygroundGenerateResponse>()
    generateImageMock.mockReturnValue(deferred.promise)

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('Pending sync generation')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    const progress = wrapper.get('[data-test="image-generation-progress"]')
    expect(progress.attributes('role')).toBe('progressbar')
    expect(progress.attributes('aria-valuenow')).toBeUndefined()
    expect(wrapper.get('[data-test="image-generation-progress-value"]').text()).toContain(
      'imagePlayground.generationProgressIndeterminate'
    )

    deferred.resolve({ data: [{ b64_json: 'sync-done-base64' }] })
    await flushPromises()

    expect(wrapper.find('[data-test="image-generation-progress"]').exists()).toBe(false)
  })

  it('updates stream generation progress from stream events', async () => {
    const deferred = createDeferred<ImagePlaygroundGenerateResponse>()
    let streamOptions: { onProgress?: (event: Record<string, unknown>) => void; onImage?: (image: { b64_json?: string }, event: Record<string, unknown>) => void } | undefined
    generateImageStreamMock.mockImplementation((_input, options) => {
      streamOptions = options
      return deferred.promise
    })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-stream"]').setValue(true)
    await wrapper.get('#image-count').setValue('4')
    await wrapper.get('[data-test="image-prompt"]').setValue('Progress stream prompt')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-test="image-generation-progress"]').attributes('aria-valuenow')).toBeUndefined()

    streamOptions?.onProgress?.({ object: 'image.generation.chunk', index: 1, total: 4, progress_text: 'Rendering fur' })
    await flushPromises()

    expect(wrapper.get('[data-test="image-generation-progress"]').attributes('aria-valuenow')).toBeUndefined()
    expect(wrapper.get('[data-test="image-generation-progress-value"]').text()).toContain('Rendering fur')

    streamOptions?.onProgress?.({ object: 'image.generation.chunk', progress: 0.25 })
    await flushPromises()

    expect(wrapper.get('[data-test="image-generation-progress"]').attributes('aria-valuenow')).toBe('25')
    expect(wrapper.get('[data-test="image-generation-progress-value"]').text()).toContain('25%')

    streamOptions?.onImage?.({ b64_json: 'first-progress-base64' }, { object: 'image.generation.result' })
    await flushPromises()

    expect(wrapper.get('img[alt="Progress stream prompt 1"]').attributes('src')).toBe(
      'data:image/png;base64,first-progress-base64'
    )

    deferred.resolve({ data: [{ b64_json: 'first-progress-base64' }] })
    await flushPromises()

    expect(wrapper.find('[data-test="image-generation-progress"]').exists()).toBe(false)
  })

  it('falls back to non-streaming API when stream mode is enabled but unsupported', async () => {
    generateImageMock.mockResolvedValue({ data: [{ b64_json: 'fallback-base64' }] })

    const wrapper = await mountView()
    const referenceFile = new File(['image-bytes'], 'reference.png', { type: 'image/png' })

    await wrapper.get('[data-test="image-stream"]').setValue(true)
    await uploadReferenceFiles(wrapper, [referenceFile])
    await wrapper.get('[data-test="image-prompt"]').setValue('Edit with reference')
    await submitGeneration(wrapper)

    expect(generateImageStreamMock).not.toHaveBeenCalled()
    expect(generateImageMock).toHaveBeenCalledWith(
      expect.objectContaining({
        prompt: 'Edit with reference',
        reference_images: [referenceFile],
      })
    )
  })

  it('disables generation when any reference image is larger than 20MB', async () => {
    const wrapper = await mountView()
    const oversizedFile = createOversizedReferenceFile()

    await wrapper.get('[data-test="image-prompt"]').setValue('Edit this image into watercolor style')
    await uploadReferenceFiles(wrapper, [oversizedFile])

    expect(wrapper.text()).toContain('imagePlayground.referenceTooLargeError')
    expect(getGenerateButton(wrapper).disabled).toBe(true)
  })

  it('keeps the oversized-file error while a later valid reference is added', async () => {
    const wrapper = await mountView()
    const oversizedFile = createOversizedReferenceFile()
    const validFile = new File(['image-bytes'], 'valid.png', { type: 'image/png' })

    await wrapper.get('[data-test="image-prompt"]').setValue('Edit this image into watercolor style')
    await uploadReferenceFiles(wrapper, [oversizedFile])
    await uploadReferenceFiles(wrapper, [validFile])

    expect(wrapper.text()).toContain('imagePlayground.referenceTooLargeError')
    expect(wrapper.text()).not.toContain('imagePlayground.referenceAddedStatus')
    expect(getGenerateButton(wrapper).disabled).toBe(true)
  })

  it('opens advanced request details in a dialog without adding a fixed middle-column card', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await flushPromises()
    expect(wrapper.get('[data-test="image-advanced-details"]').attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('[data-test="image-advanced-request-details"]').exists()).toBe(false)

    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    await flushPromises()

    expect(getAdvancedDialog(wrapper).text()).toContain('imagePlayground.advancedNoData')
    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-request-details"]').text()).toContain(
      'imagePlayground.advancedNoData'
    )
    expect(getAdvancedDialog(wrapper).find('[data-test="image-advanced-response-details"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="image-advanced-error"]').exists()).toBe(false)
    expect(wrapper.get('[data-test="image-advanced-details"]').attributes('aria-expanded')).toBe('true')
  })

  it('shows the last advanced request and response in the dialog', async () => {
    generateImageAdvancedMock.mockResolvedValue({
      data: [{ b64_json: 'advanced-base64' }],
      _sub2api_image_playground: { actual_cost: 0.2 },
    })

    const wrapper = await mountView()
    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await wrapper.get('[data-test="image-prompt"]').setValue('Advanced ocean prompt')
    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-request-details"]').text()).toContain(
      'Advanced ocean prompt'
    )
    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-response-details"]').text()).toContain(
      'advanced-base64'
    )
    expect(generateImageAdvancedMock).toHaveBeenCalledWith(
      expect.objectContaining({
        body: expect.objectContaining({ prompt: 'Advanced ocean prompt' }),
        reference_images: [],
      })
    )
  })

  it('shows the last advanced error in preference to a response', async () => {
    generateImageAdvancedMock.mockRejectedValue({
      response: { data: { error: { message: 'upstream advanced failure' } } },
    })

    const wrapper = await mountView()
    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await wrapper.get('[data-test="image-prompt"]').setValue('Failing advanced prompt')
    await submitGeneration(wrapper)
    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    await flushPromises()

    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-response-details"]').text()).toContain(
      'upstream advanced failure'
    )
    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-response-details"]').text()).not.toContain(
      'imagePlayground.advancedNoData'
    )
  })

  it('closes advanced details from the dialog controls', async () => {
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    await getAdvancedDialog(wrapper).get('button[aria-label="Close modal"]').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)

    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    await getAdvancedDialog(wrapper).trigger('keydown', { key: 'Escape' })
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('closes advanced details when advanced mode is disabled without clearing the last request', async () => {
    generateImageAdvancedMock.mockResolvedValue({ data: [{ b64_json: 'retained-base64' }] })
    const wrapper = await mountView()

    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await wrapper.get('[data-test="image-prompt"]').setValue('Retained advanced prompt')
    await submitGeneration(wrapper)
    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    expect(getAdvancedDialog(wrapper).exists()).toBe(true)

    await wrapper.get('[data-test="image-advanced"]').setValue(false)
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)

    await wrapper.get('[data-test="image-advanced"]').setValue(true)
    await wrapper.get('[data-test="image-advanced-details"]').trigger('click')
    expect(getAdvancedDialog(wrapper).get('[data-test="image-advanced-request-details"]').text()).toContain(
      'Retained advanced prompt'
    )
  })

  it('shows generated results and price and saves successful generations to history', async () => {
    listImageHistoryRecordsMock.mockResolvedValueOnce([]).mockResolvedValueOnce([createHistoryRecord()])
    generateImageMock.mockResolvedValue({
      data: [
        {
          url: 'https://example.com/generated-image.webp',
          revised_prompt: 'A calm ocean at sunrise with pastel clouds',
        },
      ],
      _sub2api_image_playground: {
        actual_cost: 0.42,
        total_cost: 0.5,
        estimated_price: 0.55,
        image_count: 1,
      },
    })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('A calm ocean at sunrise')
    await submitGeneration(wrapper)

    expect(generateImageMock).toHaveBeenCalledWith(
      expect.objectContaining({
        api_key_id: 101,
        model: 'gpt-image-1',
        prompt: 'A calm ocean at sunrise',
      })
    )
    expect(wrapper.get('img[alt="A calm ocean at sunrise 1"]').attributes('src')).toBe(
      'data:image/webp;base64,Z2VuZXJhdGVkLWJ5dGVz'
    )
    expect(wrapper.get('[data-test="image-price"]').text()).toContain('$0.4200')
    expect(addImageHistoryRecordMock).toHaveBeenCalledWith(
      expect.objectContaining({
        api_key_id: 101,
        key_name: 'Primary image key',
        model: 'gpt-image-1',
        prompt: 'A calm ocean at sunrise',
        user_id: 7,
        price: expect.objectContaining({
          actual_cost: 0.42,
          total_cost: 0.5,
          estimated_price: 0.55,
        }),
        images: [
          expect.objectContaining({
            url_or_base64: 'data:image/webp;base64,Z2VuZXJhdGVkLWJ5dGVz',
            mime_type: 'image/webp',
          }),
        ],
      })
    )
    expect(listImageHistoryRecordsMock).toHaveBeenCalledTimes(2)
  })

  it('zooms and resets the image preview', async () => {
    generateImageMock.mockResolvedValue({ data: [{ b64_json: 'preview-base64' }] })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('Preview zoom prompt')
    await submitGeneration(wrapper)
    await wrapper.get('img[alt="Preview zoom prompt 1"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="image-preview-zoom-level"]').text()).toBe('100%')
    expect(wrapper.get('[data-test="image-preview-img"]').attributes('style')).toContain(
      'translate(0px, 0px) scale(1)'
    )

    await wrapper.get('[data-test="image-preview-zoom-in"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="image-preview-zoom-level"]').text()).toBe('125%')
    expect(wrapper.get('[data-test="image-preview-img"]').attributes('style')).toContain('scale(1.25)')

    await wrapper.get('[data-test="image-preview-reset"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="image-preview-zoom-level"]').text()).toBe('100%')
    expect(wrapper.get('[data-test="image-preview-img"]').attributes('style')).toContain(
      'translate(0px, 0px) scale(1)'
    )
  })

  it('pans the image preview without requiring zoom and clears pan on reset', async () => {
    generateImageMock.mockResolvedValue({ data: [{ b64_json: 'pan-base64' }] })

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('Preview pan prompt')
    await submitGeneration(wrapper)
    await wrapper.get('img[alt="Preview pan prompt 1"]').trigger('click')
    await flushPromises()

    const panArea = wrapper.get('[data-test="image-preview-pan-area"]')
    await panArea.trigger('pointerdown', { clientX: 10, clientY: 10, pointerId: 1 })
    await flushPromises()
    expect(wrapper.get('[data-test="image-preview-img"]').classes()).toContain('transition-none')

    await panArea.trigger('pointermove', { clientX: 30, clientY: 45, pointerId: 1 })
    await panArea.trigger('pointerup', { pointerId: 1 })
    await flushPromises()

    expect(wrapper.get('[data-test="image-preview-img"]').classes()).toContain('transition-transform')
    expect(wrapper.get('[data-test="image-preview-img"]').attributes('style')).toContain(
      'translate(20px, 35px) scale(1)'
    )

    await wrapper.get('[data-test="image-preview-reset"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="image-preview-img"]').attributes('style')).toContain(
      'translate(0px, 0px) scale(1)'
    )
  })

  it('shows edit-mode hint for uploaded references and submits non-empty reference images', async () => {
    generateImageMock.mockResolvedValue({
      data: [{ url: 'https://example.com/edited.png' }],
    })

    const wrapper = await mountView()
    const referenceFile = new File(['image-bytes'], 'reference.png', { type: 'image/png' })

    await uploadReferenceFiles(wrapper, [referenceFile])
    await wrapper.get('[data-test="image-prompt"]').setValue('Edit this image into watercolor style')
    await submitGeneration(wrapper)

    expect(wrapper.text()).toContain('imagePlayground.referenceEditHint')
    expect(generateImageMock).toHaveBeenCalledWith(
      expect.objectContaining({
        reference_images: [referenceFile],
      })
    )
  })

  it('accepts reference images dropped onto the upload area', async () => {
    generateImageMock.mockResolvedValue({
      data: [{ url: 'https://example.com/edited.png' }],
    })

    const wrapper = await mountView()
    const referenceFile = new File(['dragged-image-bytes'], 'dragged-reference.png', { type: 'image/png' })

    await dropReferenceFiles(wrapper, [referenceFile])
    await wrapper.get('[data-test="image-prompt"]').setValue('Turn this into a cinematic poster')
    await submitGeneration(wrapper)

    expect(wrapper.text()).toContain('dragged-reference.png')
    expect(generateImageMock).toHaveBeenCalledWith(
      expect.objectContaining({
        reference_images: [referenceFile],
      })
    )
  })

  it('keeps generated results and history tied to the submitted values when the form changes mid-flight', async () => {
    listImageHistoryRecordsMock.mockResolvedValueOnce([]).mockResolvedValueOnce([])
    const deferred = createDeferred<ImagePlaygroundGenerateResponse>()
    generateImageMock.mockReturnValue(deferred.promise)

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-format"]').setValue('webp')
    await wrapper.get('[data-test="image-compression"]').setValue('72')
    await wrapper.get('[data-test="image-prompt"]').setValue('Submitted prompt')
    await submitGeneration(wrapper)

    await wrapper.get('[data-test="image-model"]').setValue('changed-model')
    await wrapper.get('[data-test="image-prompt"]').setValue('Changed prompt')
    await wrapper.get('[data-test="image-format"]').setValue('png')
    deferred.resolve({
      data: [{ b64_json: 'base64-image' }],
      _sub2api_image_playground: { actual_cost: 0.24 },
    })
    await flushPromises()

    expect(wrapper.get('img[alt="Submitted prompt 1"]').attributes('src')).toBe(
      'data:image/webp;base64,base64-image'
    )
    expect(addImageHistoryRecordMock).toHaveBeenCalledWith(
      expect.objectContaining({
        model: 'gpt-image-1',
        prompt: 'Submitted prompt',
        params: expect.objectContaining({
          output_format: 'webp',
          output_compression: 72,
        }),
        images: [
          expect.objectContaining({
            url_or_base64: 'data:image/webp;base64,base64-image',
            mime_type: 'image/webp',
          }),
        ],
      })
    )
  })

  it('reuses history records without replacing custom models with the selected key default', async () => {
    listImageHistoryRecordsMock.mockResolvedValue([
      createHistoryRecord({
        api_key_id: 202,
        key_name: 'Secondary key',
        model: 'custom-image-model',
        params: {
          size: '1536x1024',
          quality: 'medium',
          output_format: 'webp',
          output_compression: 72,
          moderation: 'low',
          n: 2,
        },
      }),
    ])

    const wrapper = await mountView()

    await getButtonByText(wrapper, 'imagePlayground.reuseParamsButton').trigger('click')
    await flushPromises()

    expect((wrapper.get('[data-test="image-key"]').element as HTMLSelectElement).value).toBe('202')
    expect((wrapper.get('[data-test="image-model"]').element as HTMLInputElement).value).toBe(
      'custom-image-model'
    )
    expect((wrapper.get('[data-test="image-format"]').element as HTMLSelectElement).value).toBe('webp')
    expect((wrapper.get('[data-test="image-compression"]').element as HTMLInputElement).value).toBe('72')
  })

  it('clears stale upload errors after a successful history clear action', async () => {
    const wrapper = await mountView()
    const oversizedFile = createOversizedReferenceFile()

    await uploadReferenceFiles(wrapper, [oversizedFile])
    expect(wrapper.text()).toContain('imagePlayground.referenceTooLargeError')

    await getButtonByText(wrapper, 'imagePlayground.clearHistoryButton').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('imagePlayground.historyClearedStatus')
    expect(wrapper.text()).not.toContain('imagePlayground.referenceTooLargeError')
  })

  it('clears stale progress status when generation fails', async () => {
    generateImageMock.mockRejectedValue(new Error('gateway failed'))

    const wrapper = await mountView()

    await wrapper.get('[data-test="image-prompt"]').setValue('A calm ocean at sunrise')
    await submitGeneration(wrapper)

    expect(wrapper.text()).toContain('imagePlayground.generateFailed')
    expect(wrapper.text()).not.toContain('imagePlayground.generatingStatus')
  })
})
