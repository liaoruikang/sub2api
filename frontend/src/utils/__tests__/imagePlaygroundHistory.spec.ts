import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  addImageHistoryRecord,
  clearImageHistory,
  deleteImageHistoryRecord,
  listImageHistoryRecords,
  type ImageHistoryRecord,
} from '../imagePlaygroundHistory'

const TEST_USER_ID = 7
const OTHER_USER_ID = 8

type TestImageHistoryRecord = ImageHistoryRecord & { user_id: number }

const createRecord = (overrides: Partial<TestImageHistoryRecord> = {}): TestImageHistoryRecord => ({
  id: overrides.id ?? 'record-1',
  user_id: overrides.user_id ?? TEST_USER_ID,
  created_at: overrides.created_at ?? '2026-06-06T10:00:00.000Z',
  api_key_id: overrides.api_key_id ?? 1,
  key_name: overrides.key_name ?? 'Primary image key',
  model: overrides.model ?? 'gpt-image-1',
  prompt: overrides.prompt ?? 'Render a neon fox',
  params: overrides.params ?? {
    size: '1024x1024',
    quality: 'high',
    output_format: 'png',
    moderation: 'auto',
    n: 1,
  },
  price: overrides.price,
  images: overrides.images ?? [
    {
      url_or_base64: 'https://example.com/image.png',
      mime_type: 'image/png',
    },
  ],
})

describe('imagePlaygroundHistory', () => {
  beforeEach(async () => {
    await clearImageHistory()
  })

  it('lists newest records first after adding entries', async () => {
    await addImageHistoryRecord(createRecord({ id: 'older', created_at: '2026-06-06T10:00:00.000Z' }))
    await addImageHistoryRecord(createRecord({ id: 'newer', created_at: '2026-06-06T10:05:00.000Z' }))

    await expect(listImageHistoryRecords()).resolves.toMatchObject([
      { id: 'newer' },
      { id: 'older' },
    ])
  })

  it('deletes a single record by id', async () => {
    await addImageHistoryRecord(createRecord({ id: 'keep-me' }))
    await addImageHistoryRecord(createRecord({ id: 'delete-me', created_at: '2026-06-06T10:01:00.000Z' }))

    await deleteImageHistoryRecord('delete-me')

    await expect(listImageHistoryRecords()).resolves.toMatchObject([{ id: 'keep-me' }])
  })

  it('keeps only the 50 newest records', async () => {
    for (let index = 0; index < 55; index += 1) {
      await addImageHistoryRecord(
        createRecord({
          id: `record-${index}`,
          created_at: `2026-06-06T10:${String(index).padStart(2, '0')}:00.000Z`,
          prompt: `Prompt ${index}`,
        })
      )
    }

    const records = await listImageHistoryRecords()

    expect(records).toHaveLength(50)
    expect(records[0]?.id).toBe('record-54')
    expect(records.at(-1)?.id).toBe('record-5')
  })

  it('preserves price metadata on stored records', async () => {
    await addImageHistoryRecord(
      createRecord({
        id: 'priced',
        price: {
          estimated_price: 0.25,
          actual_cost: 0.42,
          total_cost: 0.42,
          image_count: 1,
          image_size: '1024x1024',
          billing_mode: 'per_image',
        },
      })
    )

    const [record] = await listImageHistoryRecords()

    expect(record?.price).toEqual({
      estimated_price: 0.25,
      actual_cost: 0.42,
      total_cost: 0.42,
      image_count: 1,
      image_size: '1024x1024',
      billing_mode: 'per_image',
    })
  })

  it('lists only records for the requested authenticated user', async () => {
    await addImageHistoryRecord(createRecord({ id: 'mine', user_id: TEST_USER_ID }))
    await addImageHistoryRecord(
      createRecord({
        id: 'someone-else',
        user_id: OTHER_USER_ID,
        created_at: '2026-06-06T10:01:00.000Z',
      })
    )

    await expect(listImageHistoryRecords(TEST_USER_ID)).resolves.toMatchObject([{ id: 'mine' }])
    await expect(listImageHistoryRecords(OTHER_USER_ID)).resolves.toMatchObject([{ id: 'someone-else' }])
  })

  it('clears only records for the requested authenticated user', async () => {
    await addImageHistoryRecord(createRecord({ id: 'mine', user_id: TEST_USER_ID }))
    await addImageHistoryRecord(
      createRecord({
        id: 'someone-else',
        user_id: OTHER_USER_ID,
        created_at: '2026-06-06T10:01:00.000Z',
      })
    )

    await clearImageHistory(TEST_USER_ID)

    await expect(listImageHistoryRecords(TEST_USER_ID)).resolves.toEqual([])
    await expect(listImageHistoryRecords(OTHER_USER_ID)).resolves.toMatchObject([{ id: 'someone-else' }])
  })

  it('clears all records', async () => {
    await addImageHistoryRecord(createRecord({ id: 'first' }))
    await addImageHistoryRecord(createRecord({ id: 'second', created_at: '2026-06-06T10:01:00.000Z' }))

    await clearImageHistory()

    await expect(listImageHistoryRecords()).resolves.toEqual([])
  })

  it('keeps memory fallback visible after an IndexedDB write fails but later reads succeed', async () => {
    const originalIndexedDB = globalThis.indexedDB
    const open = vi.fn()
    let requestNumber = 0

    Object.defineProperty(globalThis, 'indexedDB', {
      configurable: true,
      value: { open },
    })

    open.mockImplementation(() => {
      requestNumber += 1
      const request = {
        result: undefined as unknown,
        error: new Error('quota exceeded'),
        onupgradeneeded: null as (() => void) | null,
        onsuccess: null as (() => void) | null,
        onerror: null as (() => void) | null,
      }

      if (requestNumber === 1) {
        queueMicrotask(() => request.onerror?.())
        return request
      }

      const database = {
        close: vi.fn(),
        transaction: vi.fn(() => {
          const transaction = {
            error: null,
            oncomplete: null as (() => void) | null,
            onerror: null as (() => void) | null,
            onabort: null as (() => void) | null,
            objectStore: vi.fn(() => ({
              getAll: vi.fn(() => {
                const getAllRequest = {
                  result: [],
                  error: null,
                  onsuccess: null as (() => void) | null,
                  onerror: null as (() => void) | null,
                }
                queueMicrotask(() => getAllRequest.onsuccess?.())
                return getAllRequest
              }),
            })),
          }
          queueMicrotask(() => transaction.oncomplete?.())
          return transaction
        }),
      }
      request.result = database
      queueMicrotask(() => request.onsuccess?.())
      return request
    })

    try {
      await addImageHistoryRecord(createRecord({ id: 'memory-only' }))
      await expect(listImageHistoryRecords()).resolves.toMatchObject([{ id: 'memory-only' }])
    } finally {
      Object.defineProperty(globalThis, 'indexedDB', {
        configurable: true,
        value: originalIndexedDB,
      })
    }
  })
})
