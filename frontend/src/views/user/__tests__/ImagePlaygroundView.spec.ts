import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'

import type { ImagePlaygroundGenerateResponse } from '@/api/imagePlayground'
import type { ImageHistoryRecord } from '@/utils/imagePlaygroundHistory'

import ImagePlaygroundView from '../ImagePlaygroundView.vue'

const {
  getImageOptionsMock,
  generateImageMock,
  addImageHistoryRecordMock,
  listImageHistoryRecordsMock,
  deleteImageHistoryRecordMock,
  clearImageHistoryMock,
  authUserMock,
} = vi.hoisted(() => ({
  getImageOptionsMock: vi.fn(),
  generateImageMock: vi.fn(),
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
  generateImage: generateImageMock,
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
