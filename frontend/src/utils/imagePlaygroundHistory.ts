import type { ImagePlaygroundCostMetadata } from '@/api/imagePlayground'

export interface ImageHistoryRecord {
  id: string
  user_id: number
  created_at: string
  api_key_id: number
  key_name: string
  model: string
  prompt: string
  params: {
    size: string
    quality: string
    output_format: string
    output_compression?: number
    moderation: string
    n: number
  }
  price?: ImagePlaygroundCostMetadata
  images: Array<{
    url_or_base64: string
    mime_type: string
  }>
}

const DB_NAME = 'sub2api-image-playground-history'
const STORE_NAME = 'image-history-records'
const MAX_RECORDS = 50

const memoryStore = new Map<string, ImageHistoryRecord>()
let storageFallbackActive = false

const cloneRecord = (record: ImageHistoryRecord): ImageHistoryRecord => ({
  ...record,
  params: { ...record.params },
  price: record.price ? { ...record.price } : undefined,
  images: record.images.map((image) => ({ ...image })),
})

const sortNewestFirst = (records: ImageHistoryRecord[]): ImageHistoryRecord[] => {
  return records.sort((left, right) => {
    const timeDiff = new Date(right.created_at).getTime() - new Date(left.created_at).getTime()
    if (timeDiff !== 0) {
      return timeDiff
    }
    return right.id.localeCompare(left.id)
  })
}

const matchesUser = (record: ImageHistoryRecord, userId?: number): boolean => {
  if (typeof userId !== 'number') {
    return true
  }

  return record.user_id === userId
}

const trimNewest = (records: ImageHistoryRecord[]): ImageHistoryRecord[] => {
  return sortNewestFirst(records).slice(0, MAX_RECORDS)
}

const trimRecordsForStorage = (records: ImageHistoryRecord[], userId?: number): ImageHistoryRecord[] => {
  if (typeof userId !== 'number') {
    return trimNewest(records)
  }

  const scoped = trimNewest(records.filter((record) => matchesUser(record, userId)))
  const otherUsers = records.filter((record) => !matchesUser(record, userId)).map(cloneRecord)
  return [...otherUsers, ...scoped]
}

const listMemoryRecords = (userId?: number): ImageHistoryRecord[] => {
  return trimNewest(Array.from(memoryStore.values()).filter((record) => matchesUser(record, userId)).map(cloneRecord))
}

const writeMemoryRecords = (records: ImageHistoryRecord[]): void => {
  memoryStore.clear()
  records.forEach((record) => {
    memoryStore.set(record.id, cloneRecord(record))
  })
}

const hasIndexedDB = (): boolean => typeof indexedDB !== 'undefined'

const openIndexedDB = async (): Promise<IDBDatabase> => {
  return await new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, 1)

    request.onupgradeneeded = () => {
      const database = request.result
      if (!database.objectStoreNames.contains(STORE_NAME)) {
        database.createObjectStore(STORE_NAME, { keyPath: 'id' })
      }
    }

    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('Failed to open IndexedDB'))
  })
}

const runTransaction = async <T>(
  mode: IDBTransactionMode,
  action: (store: IDBObjectStore) => Promise<T>
): Promise<T> => {
  const database = await openIndexedDB()
  const transaction = database.transaction(STORE_NAME, mode)
  const store = transaction.objectStore(STORE_NAME)
  const completion = new Promise<void>((resolve, reject) => {
    transaction.oncomplete = () => resolve()
    transaction.onerror = () => reject(transaction.error ?? new Error('IndexedDB transaction failed'))
    transaction.onabort = () => reject(transaction.error ?? new Error('IndexedDB transaction aborted'))
  })

  try {
    const result = await action(store)
    await completion
    return result
  } catch (error) {
    try {
      transaction.abort()
    } catch {
      // Transaction may already be completed, aborted, or inactive.
    }
    try {
      await completion
    } catch {
      // Preserve the original action/request error.
    }
    throw error
  } finally {
    database.close()
  }
}

const requestToPromise = async <T>(request: IDBRequest<T>): Promise<T> => {
  return await new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error ?? new Error('IndexedDB request failed'))
  })
}

const listIndexedDBRecordsRaw = async (): Promise<ImageHistoryRecord[]> => {
  return await runTransaction('readonly', async (store) => {
    const result = await requestToPromise(store.getAll() as IDBRequest<ImageHistoryRecord[]>)
    return (result ?? []).map(cloneRecord)
  })
}

const listIndexedDBRecords = async (userId?: number): Promise<ImageHistoryRecord[]> => {
  const records = await listIndexedDBRecordsRaw()
  return trimNewest(records.filter((record) => matchesUser(record, userId)))
}

const putIndexedDBRecord = async (record: ImageHistoryRecord): Promise<void> => {
  await runTransaction('readwrite', async (store) => {
    await requestToPromise(store.put(cloneRecord(record)))
  })
}

const deleteIndexedDBRecord = async (id: string): Promise<void> => {
  await runTransaction('readwrite', async (store) => {
    await requestToPromise(store.delete(id))
  })
}

const clearIndexedDBRecords = async (): Promise<void> => {
  await runTransaction('readwrite', async (store) => {
    await requestToPromise(store.clear())
  })
}

const replaceIndexedDBRecords = async (records: ImageHistoryRecord[]): Promise<void> => {
  await runTransaction('readwrite', async (store) => {
    await requestToPromise(store.clear())
    for (const record of records.map(cloneRecord)) {
      await requestToPromise(store.put(record))
    }
  })
}

const activateStorageFallback = async (): Promise<void> => {
  storageFallbackActive = true

  if (memoryStore.size > 0 || !hasIndexedDB()) {
    return
  }

  try {
    writeMemoryRecords(await listIndexedDBRecordsRaw())
  } catch {
    // If IndexedDB cannot be read either, continue with the in-memory store.
  }
}

const withStorageFallback = async <T>(
  indexedDBAction: () => Promise<T>,
  memoryAction: () => T | Promise<T>
): Promise<T> => {
  if (storageFallbackActive) {
    return await memoryAction()
  }

  if (!hasIndexedDB()) {
    return await memoryAction()
  }

  try {
    return await indexedDBAction()
  } catch {
    await activateStorageFallback()
    return await memoryAction()
  }
}

export async function listImageHistoryRecords(userId?: number): Promise<ImageHistoryRecord[]> {
  return await withStorageFallback(() => listIndexedDBRecords(userId), () => listMemoryRecords(userId))
}

export async function addImageHistoryRecord(record: ImageHistoryRecord): Promise<void> {
  const nextRecord = cloneRecord(record)

  await withStorageFallback(
    async () => {
      await putIndexedDBRecord(nextRecord)
      const records = await listIndexedDBRecordsRaw()
      const trimmedRecords = trimRecordsForStorage(records, nextRecord.user_id)
      if (trimmedRecords.length !== records.length) {
        await replaceIndexedDBRecords(trimmedRecords)
      }
    },
    () => {
      const records = Array.from(memoryStore.values())
        .map(cloneRecord)
        .filter((existingRecord) => existingRecord.id !== nextRecord.id)
      records.push(nextRecord)
      writeMemoryRecords(trimRecordsForStorage(records, nextRecord.user_id))
    }
  )
}

export async function deleteImageHistoryRecord(id: string, userId?: number): Promise<void> {
  await withStorageFallback(
    async () => {
      const [record] = (await listIndexedDBRecordsRaw()).filter((candidate) => candidate.id === id)
      if (!record || !matchesUser(record, userId)) {
        return
      }
      await deleteIndexedDBRecord(id)
    },
    () => {
      const record = memoryStore.get(id)
      if (record && matchesUser(record, userId)) {
        memoryStore.delete(id)
      }
    }
  )
}

export async function clearImageHistory(userId?: number): Promise<void> {
  await withStorageFallback(
    async () => {
      if (typeof userId !== 'number') {
        await clearIndexedDBRecords()
        return
      }

      const remainingRecords = (await listIndexedDBRecordsRaw()).filter((record) => !matchesUser(record, userId))
      await replaceIndexedDBRecords(remainingRecords)
    },
    () => {
      if (typeof userId !== 'number') {
        memoryStore.clear()
        return
      }

      Array.from(memoryStore.entries()).forEach(([id, record]) => {
        if (matchesUser(record, userId)) {
          memoryStore.delete(id)
        }
      })
    }
  )
}
