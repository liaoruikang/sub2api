import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

const requiredKeys = [
  'eyebrow',
  'title',
  'description',
  'loading',
  'ready',
  'emptyStateTitle',
  'emptyStateDescription',
  'generatorTitle',
  'generatorDescription',
  'apiKeyLabel',
  'modelLabel',
  'modelPlaceholder',
  'promptLabel',
  'promptPlaceholder',
  'referenceLabel',
  'referenceHint',
  'referenceEditHint',
  'referenceListLabel',
  'fileTooLarge',
  'removeReferenceLabel',
  'removeReferenceAction',
  'sizeLabel',
  'qualityLabel',
  'outputFormatLabel',
  'compressionLabel',
  'compressionPlaceholder',
  'compressionInvalid',
  'moderationLabel',
  'countLabel',
  'countInvalid',
  'submitHint',
  'generatingButton',
  'generateButton',
  'resultsTitle',
  'resultsDescription',
  'resultsEmpty',
  'generatedImageAlt',
  'tipsTitle',
  'tipPrompt',
  'tipReference',
  'tipReuse',
  'historyTitle',
  'historyDescription',
  'clearHistoryButton',
  'historyEmpty',
  'reuseParamsButton',
  'deleteHistoryButton',
  'loadOptionsFailed',
  'loadHistoryFailed',
  'referenceTooLargeError',
  'referenceAddedStatus',
  'generatingStatus',
  'generateSuccessStatus',
  'generateEmptyStatus',
  'generateFailed',
  'reuseParamsStatus',
  'historyDeletedStatus',
  'historyDeleteFailed',
  'historyClearedStatus',
  'historyClearFailed',
] as const

describe('image playground locale keys', () => {
  it('contains navigation labels', () => {
    expect(en.nav.imagePlayground).toBe('Image Playground')
    expect(zh.nav.imagePlayground).toBe('在线生图')
  })

  it('contains all page labels in English and Chinese', () => {
    requiredKeys.forEach((key) => {
      expect(en.imagePlayground[key]).toEqual(expect.any(String))
      expect(zh.imagePlayground[key]).toEqual(expect.any(String))
      expect(en.imagePlayground[key].length).toBeGreaterThan(0)
      expect(zh.imagePlayground[key].length).toBeGreaterThan(0)
    })
  })
})
