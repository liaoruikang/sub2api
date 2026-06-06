<template>
  <AppLayout>
    <div class="flex w-full flex-col gap-6">
      <section
        v-if="!loadingOptions && !hasAvailableKeys"
        class="rounded-2xl border border-dashed border-gray-300 bg-white p-8 text-center shadow-sm dark:border-dark-600 dark:bg-dark-800"
      >
        <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
          {{ t('imagePlayground.emptyStateTitle') }}
        </h2>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">
          {{ t('imagePlayground.emptyStateDescription') }}
        </p>
      </section>

      <div v-else-if="!loadingOptions" class="grid items-start gap-6 xl:grid-cols-[440px_minmax(0,1fr)]">
        <div class="xl:relative">
          <section
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800 xl:fixed xl:top-24 xl:w-[440px]"
          >
          <div class="mb-5 flex items-start justify-between gap-4">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('imagePlayground.generatorTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                {{ t('imagePlayground.generatorDescription') }}
              </p>
            </div>
          </div>

          <form class="space-y-5" @submit.prevent="handleGenerate">
            <div class="grid gap-5 md:grid-cols-2">
              <div>
                <label for="image-key" class="input-label">
                  {{ t('imagePlayground.apiKeyLabel') }}
                </label>
                <select
                  id="image-key"
                  v-model.number="selectedKeyId"
                  data-test="image-key"
                  class="image-playground-select"
                >
                  <option v-for="key in availableKeys" :key="key.id" :value="key.id">
                    {{ key.name }}
                  </option>
                </select>
                <div
                  v-if="selectedKey"
                  class="mt-2 rounded-xl bg-gray-50 px-3 py-2 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400"
                >
                  <div class="flex items-center justify-between gap-3">
                    <span class="truncate">{{ selectedKey.group_name || t('imagePlayground.defaultGroupLabel') }}</span>
                    <span class="shrink-0 font-mono">{{ selectedKey.masked_key }}</span>
                  </div>
                </div>
              </div>

              <div>
                <label for="image-model" class="input-label">
                  {{ t('imagePlayground.modelLabel') }}
                </label>
                <input
                  id="image-model"
                  v-model="model"
                  data-test="image-model"
                  list="image-model-options"
                  type="text"
                  autocomplete="off"
                  class="input"
                  :placeholder="t('imagePlayground.modelPlaceholder')"
                />
                <datalist id="image-model-options">
                  <option v-for="modelOption in modelSuggestions" :key="modelOption" :value="modelOption" />
                </datalist>
              </div>
            </div>

            <div>
              <label for="image-prompt" class="input-label">
                {{ t('imagePlayground.promptLabel') }}
              </label>
              <textarea
                id="image-prompt"
                v-model="prompt"
                data-test="image-prompt"
                rows="5"
                class="input min-h-[132px] resize-y leading-6"
                :placeholder="t('imagePlayground.promptPlaceholder')"
              />
            </div>

            <div class="space-y-3">
              <div>
                <label for="image-reference" class="input-label">
                  {{ t('imagePlayground.referenceLabel') }}
                </label>
                <div
                  data-test="image-reference-dropzone"
                  class="rounded-2xl border-2 border-dashed p-4 transition"
                  :class="referenceDragging ? 'border-primary-400 bg-primary-50/80 dark:border-primary-500 dark:bg-primary-900/20' : 'border-gray-200 bg-gray-50 hover:border-primary-300 hover:bg-primary-50/50 dark:border-dark-600 dark:bg-dark-700/70 dark:hover:border-primary-700 dark:hover:bg-primary-900/10'"
                  @dragenter.prevent="referenceDragging = true"
                  @dragover.prevent="referenceDragging = true"
                  @dragleave.prevent="referenceDragging = false"
                  @drop.prevent="handleReferenceDrop"
                >
                  <input
                    id="image-reference"
                    ref="referenceInputRef"
                    data-test="image-reference"
                    type="file"
                    accept="image/*"
                    multiple
                    class="sr-only"
                    @change="handleReferenceInput"
                  />
                  <div class="flex flex-col items-center justify-center gap-3 text-center sm:flex-row sm:text-left">
                    <span class="inline-flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-2xl bg-white text-primary-600 shadow-sm dark:bg-dark-800 dark:text-primary-300">
                      <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
                        <path stroke-linecap="round" stroke-linejoin="round" d="M3 16.5v2.25A2.25 2.25 0 005.25 21h13.5A2.25 2.25 0 0021 18.75V16.5m-13.5-9L12 3m0 0l4.5 4.5M12 3v13.5" />
                      </svg>
                    </span>
                    <div class="min-w-0 flex-1">
                      <p class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ t('imagePlayground.referenceDropTitle') }}
                      </p>
                      <p class="mt-1 text-xs leading-5 text-gray-500 dark:text-gray-400">
                        {{ t('imagePlayground.referenceHint') }}
                      </p>
                    </div>
                    <label for="image-reference" class="btn btn-secondary btn-sm cursor-pointer">
                      {{ t('imagePlayground.referenceBrowseAction') }}
                    </label>
                  </div>
                </div>
                <p
                  v-if="referenceItems.length > 0"
                  class="mt-2 text-sm text-amber-700 dark:text-amber-300"
                >
                  {{ t('imagePlayground.referenceEditHint') }}
                </p>
              </div>

              <ul
                v-if="referenceItems.length > 0"
                class="flex flex-wrap gap-2"
                :aria-label="t('imagePlayground.referenceListLabel')"
              >
                <li
                  v-for="item in referenceItems"
                  :key="item.id"
                  class="flex items-center gap-2 rounded-full border border-gray-200 bg-gray-50 px-3 py-2 text-sm text-gray-700 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200"
                >
                  <span class="truncate max-w-[180px]">{{ item.file.name }}</span>
                  <span class="text-xs text-gray-500 dark:text-gray-400">
                    {{ formatFileSize(item.file.size) }}
                  </span>
                  <span
                    v-if="item.tooLarge"
                    class="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700 dark:bg-red-900/30 dark:text-red-300"
                  >
                    {{ t('imagePlayground.fileTooLarge') }}
                  </span>
                  <button
                    type="button"
                    class="rounded-full px-2 py-1 text-xs font-medium text-gray-600 transition hover:bg-gray-200 hover:text-gray-900 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:text-gray-300 dark:hover:bg-dark-600 dark:hover:text-white"
                    :aria-label="`${t('imagePlayground.removeReferenceLabel')} ${item.file.name}`"
                    @click="removeReference(item.id)"
                  >
                    {{ t('imagePlayground.removeReferenceAction') }}
                  </button>
                </li>
              </ul>
            </div>

            <div class="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
              <div>
                <label for="image-size" class="input-label">
                  {{ t('imagePlayground.sizeLabel') }}
                </label>
                <button
                  id="image-size"
                  type="button"
                  data-test="image-size-picker"
                  class="image-playground-select text-left"
                  aria-haspopup="dialog"
                  :aria-expanded="sizePickerOpen"
                  @click="openSizePicker"
                >
                  <span data-test="image-size-value">{{ size }}</span>
                </button>
              </div>

              <div>
                <label for="image-quality" class="input-label">
                  {{ t('imagePlayground.qualityLabel') }}
                </label>
                <select
                  id="image-quality"
                  v-model="quality"
                  class="image-playground-select"
                >
                  <option v-for="qualityOption in qualityOptions" :key="qualityOption" :value="qualityOption">
                    {{ qualityOption }}
                  </option>
                </select>
              </div>

              <div>
                <label
                  for="image-format"
                  class="input-label"
                >
                  {{ t('imagePlayground.outputFormatLabel') }}
                </label>
                <select
                  id="image-format"
                  v-model="outputFormat"
                  data-test="image-format"
                  class="image-playground-select"
                >
                  <option v-for="formatOption in outputFormatOptions" :key="formatOption" :value="formatOption">
                    {{ formatOption }}
                  </option>
                </select>
              </div>

              <div>
                <label
                  for="image-compression"
                  class="input-label"
                >
                  {{ t('imagePlayground.compressionLabel') }}
                </label>
                <input
                  id="image-compression"
                  v-model="compressionInput"
                  data-test="image-compression"
                  type="number"
                  inputmode="numeric"
                  min="0"
                  max="100"
                  step="1"
                  :disabled="!compressionEnabled"
                  :aria-invalid="invalidCompression ? 'true' : 'false'"
                  :aria-describedby="invalidCompression ? 'image-compression-error' : undefined"
                  class="input disabled:text-gray-400 dark:disabled:text-gray-500"
                  :placeholder="t('imagePlayground.compressionPlaceholder')"
                />
                <p
                  v-if="invalidCompression"
                  id="image-compression-error"
                  class="mt-2 text-xs text-red-600 dark:text-red-400"
                >
                  {{ t('imagePlayground.compressionInvalid') }}
                </p>
              </div>

              <div>
                <label
                  for="image-moderation"
                  class="input-label"
                >
                  {{ t('imagePlayground.moderationLabel') }}
                </label>
                <select
                  id="image-moderation"
                  v-model="moderation"
                  class="image-playground-select"
                >
                  <option v-for="moderationOption in moderationOptions" :key="moderationOption" :value="moderationOption">
                    {{ moderationOption }}
                  </option>
                </select>
              </div>

              <div>
                <label for="image-count" class="input-label">
                  {{ t('imagePlayground.countLabel') }}
                </label>
                <input
                  id="image-count"
                  v-model.number="count"
                  type="number"
                  inputmode="numeric"
                  min="1"
                  max="4"
                  step="1"
                  :aria-invalid="invalidCount ? 'true' : 'false'"
                  :aria-describedby="invalidCount ? 'image-count-error' : undefined"
                  class="input"
                />
                <p
                  v-if="invalidCount"
                  id="image-count-error"
                  class="mt-2 text-xs text-red-600 dark:text-red-400"
                >
                  {{ t('imagePlayground.countInvalid') }}
                </p>
              </div>
            </div>

            <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
              <button
                type="submit"
                data-test="image-generate"
                :disabled="generateDisabled"
                class="btn btn-primary btn-lg w-full"
              >
                <span
                  v-if="generating"
                  class="inline-block h-5 w-5 animate-spin rounded-full border-2 border-white/30 border-t-white"
                />
                {{ generating ? t('imagePlayground.generatingButton') : t('imagePlayground.generateButton') }}
              </button>
            </div>
          </form>
          </section>
        </div>

        <aside class="space-y-6">
          <section
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="flex items-start justify-between gap-4">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('imagePlayground.resultsTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                  {{ t('imagePlayground.resultsDescription') }}
                </p>
              </div>
              <p
                v-if="formattedCurrentPrice"
                data-test="image-price"
                class="rounded-full bg-emerald-50 px-3 py-1 text-sm font-medium text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300"
              >
                {{ formattedCurrentPrice }}
              </p>
            </div>

            <div
              v-if="statusMessage || errorMessage"
              class="mt-4 rounded-xl border px-4 py-3 text-sm"
              :class="errorMessage ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300' : 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300'"
              aria-live="polite"
            >
              <p v-if="errorMessage" aria-live="assertive">
                {{ errorMessage }}
              </p>
              <p v-else>
                {{ statusMessage }}
              </p>
            </div>

            <div v-if="currentResults.length === 0" class="mt-4 rounded-xl bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              {{ t('imagePlayground.resultsEmpty') }}
            </div>

            <div v-else class="mt-4 grid gap-4 sm:grid-cols-2">
              <article
                v-for="(result, index) in currentResults"
                :key="`${currentRequestId}-${index}`"
                class="overflow-hidden rounded-2xl border border-gray-200 bg-gray-50 dark:border-dark-500 dark:bg-dark-700"
              >
                <div class="group relative">
                  <button
                    type="button"
                    class="block w-full overflow-hidden focus:outline-none focus:ring-2 focus:ring-primary-500/40"
                    :aria-label="t('imagePlayground.previewImageAction')"
                    @click="openGeneratedResultPreview(result, index)"
                  >
                    <img
                      :src="imageSrc(result, currentOutputFormat)"
                      :alt="generatedResultAlt(index)"
                      class="aspect-square w-full object-cover transition group-hover:scale-[1.01]"
                    />
                  </button>
                  <button
                    type="button"
                    class="absolute right-3 top-3 rounded-full bg-white/90 px-3 py-1.5 text-xs font-medium text-gray-700 shadow-sm backdrop-blur transition hover:bg-white focus:outline-none focus:ring-2 focus:ring-primary-500/40 dark:bg-dark-800/90 dark:text-gray-200 dark:hover:bg-dark-800"
                    @click.stop="downloadGeneratedResult(result, index)"
                  >
                    {{ t('imagePlayground.downloadImageAction') }}
                  </button>
                </div>
                <div class="space-y-2 p-4">
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ currentPrompt || t('imagePlayground.generatedImageAlt') }}
                  </p>
                  <p
                    v-if="result.revised_prompt"
                    class="text-xs text-gray-500 dark:text-gray-400"
                  >
                    {{ result.revised_prompt }}
                  </p>
                </div>
              </article>
            </div>
          </section>

          <section
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
          >
            <div class="space-y-2">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('imagePlayground.tipsTitle') }}
              </h2>
              <ul class="grid gap-2 text-sm text-gray-600 dark:text-gray-300 sm:grid-cols-3">
                <li class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700">{{ t('imagePlayground.tipPrompt') }}</li>
                <li class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700">{{ t('imagePlayground.tipReference') }}</li>
                <li class="rounded-xl bg-gray-50 p-3 dark:bg-dark-700">{{ t('imagePlayground.tipReuse') }}</li>
              </ul>
            </div>
          </section>

          <section class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
            <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t('imagePlayground.historyTitle') }}
                </h2>
                <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                  {{ t('imagePlayground.historyDescription') }}
                </p>
              </div>
              <button
                type="button"
                class="btn btn-secondary"
                :disabled="historyRecords.length === 0"
                @click="handleClearHistory"
              >
                {{ t('imagePlayground.clearHistoryButton') }}
              </button>
            </div>

            <div v-if="historyRecords.length === 0" class="mt-4 rounded-xl bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              {{ t('imagePlayground.historyEmpty') }}
            </div>

            <div v-else class="mt-4 grid gap-4 md:grid-cols-2 2xl:grid-cols-3">
              <article
                v-for="record in historyRecords"
                :key="record.id"
                class="flex h-full flex-col rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-500 dark:bg-dark-700"
              >
                <div class="flex flex-1 flex-col gap-3">
                  <div class="space-y-2">
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ record.prompt }}
                      </span>
                      <span
                        v-if="formatPrice(record.price)"
                        class="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-medium text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300"
                      >
                        {{ formatPrice(record.price) }}
                      </span>
                    </div>
                    <p class="text-sm text-gray-600 dark:text-gray-300">
                      {{ record.key_name }} · {{ record.model }} · {{ formatRecordTimestamp(record.created_at) }}
                    </p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">
                      {{ record.params.size }} · {{ record.params.quality }} · {{ record.params.output_format }} ·
                      {{ record.params.moderation }} · n={{ record.params.n }}
                      <template v-if="typeof record.params.output_compression === 'number'">
                        · {{ t('imagePlayground.compressionLabel') }} {{ record.params.output_compression }}
                      </template>
                    </p>
                  </div>

                  <div class="mt-auto flex flex-wrap gap-2">
                    <button
                      type="button"
                      class="btn btn-primary btn-sm"
                      @click="reuseHistoryRecord(record)"
                    >
                      {{ t('imagePlayground.reuseParamsButton') }}
                    </button>
                    <button
                      type="button"
                      class="btn btn-secondary btn-sm"
                      @click="handleDeleteHistory(record.id)"
                    >
                      {{ t('imagePlayground.deleteHistoryButton') }}
                    </button>
                  </div>
                </div>

                <div class="mt-4 grid grid-cols-2 gap-3">
                  <figure
                    v-for="(image, index) in record.images"
                    :key="`${record.id}-${index}`"
                    class="group relative overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-500 dark:bg-dark-800"
                  >
                    <button
                      type="button"
                      class="block w-full focus:outline-none focus:ring-2 focus:ring-primary-500/40"
                      :aria-label="t('imagePlayground.previewImageAction')"
                      @click="openHistoryImagePreview(record, image, index)"
                    >
                      <img
                        :src="image.url_or_base64"
                        :alt="historyImageAlt(record, index)"
                        class="aspect-square w-full object-cover transition group-hover:scale-[1.01]"
                      />
                    </button>
                    <button
                      type="button"
                      class="absolute right-2 top-2 rounded-full bg-white/90 px-2.5 py-1 text-xs font-medium text-gray-700 opacity-0 shadow-sm backdrop-blur transition hover:bg-white focus:opacity-100 focus:outline-none focus:ring-2 focus:ring-primary-500/40 group-hover:opacity-100 dark:bg-dark-800/90 dark:text-gray-200 dark:hover:bg-dark-800"
                      @click.stop="downloadHistoryImage(record, image, index)"
                    >
                      {{ t('imagePlayground.downloadImageAction') }}
                    </button>
                  </figure>
                </div>
              </article>
            </div>
          </section>
        </aside>
      </div>
    </div>

    <div
      v-if="sizePickerOpen"
      class="fixed inset-0 z-50 flex items-center justify-center bg-gray-950/40 px-4 py-6 backdrop-blur-sm"
      role="presentation"
      @click.self="closeSizePicker"
    >
      <section
        class="flex max-h-[calc(100vh-3rem)] w-full max-w-[460px] flex-col overflow-hidden rounded-3xl bg-white shadow-2xl dark:bg-dark-800"
        role="dialog"
        aria-modal="true"
        :aria-label="t('imagePlayground.sizePickerTitle')"
      >
        <header class="flex items-start justify-between gap-4 p-6 pb-4">
          <div>
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('imagePlayground.sizePickerTitle') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('imagePlayground.sizePickerCurrent') }}: {{ size }}
            </p>
          </div>
          <button
            type="button"
            class="rounded-full p-2 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:hover:bg-dark-700 dark:hover:text-gray-200"
            :aria-label="t('common.close')"
            @click="closeSizePicker"
          >
            <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </header>

        <div class="flex-1 overflow-y-auto px-6 pb-4">
          <div class="grid grid-cols-3 rounded-xl bg-gray-100 p-1 text-sm font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300">
            <button
              type="button"
              data-test="image-size-tab-auto"
              class="rounded-lg px-3 py-2 transition"
              :class="sizePickerMode === 'auto' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white' : 'hover:text-gray-900 dark:hover:text-white'"
              @click="sizePickerMode = 'auto'"
            >
              {{ t('imagePlayground.sizePickerAutoTab') }}
            </button>
            <button
              type="button"
              data-test="image-size-tab-ratio"
              class="rounded-lg px-3 py-2 transition"
              :class="sizePickerMode === 'ratio' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white' : 'hover:text-gray-900 dark:hover:text-white'"
              @click="sizePickerMode = 'ratio'"
            >
              {{ t('imagePlayground.sizePickerRatioTab') }}
            </button>
            <button
              type="button"
              data-test="image-size-tab-custom"
              class="rounded-lg px-3 py-2 transition"
              :class="sizePickerMode === 'custom' ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-800 dark:text-white' : 'hover:text-gray-900 dark:hover:text-white'"
              @click="sizePickerMode = 'custom'"
            >
              {{ t('imagePlayground.sizePickerCustomTab') }}
            </button>
          </div>

          <div v-if="sizePickerMode === 'auto'" class="flex min-h-[360px] flex-col items-center justify-center text-center">
            <div class="flex h-16 w-16 items-center justify-center rounded-full bg-primary-50 text-primary-600 dark:bg-primary-900/20 dark:text-primary-300">
              <svg class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
            <h3 class="mt-5 text-base font-semibold text-gray-900 dark:text-white">
              {{ t('imagePlayground.sizePickerAutoTitle') }}
            </h3>
            <p class="mt-2 max-w-xs text-sm leading-6 text-gray-500 dark:text-gray-400">
              {{ t('imagePlayground.sizePickerAutoDescription') }}
            </p>
          </div>

          <div v-else-if="sizePickerMode === 'ratio'" class="space-y-5 pt-5">
            <div>
              <p class="mb-3 text-sm font-medium text-gray-500 dark:text-gray-400">
                {{ t('imagePlayground.sizePickerResolutionLabel') }}
              </p>
              <div class="grid grid-cols-3 gap-2">
                <button
                  v-for="option in sizeResolutionOptions"
                  :key="option.key"
                  type="button"
                  :data-test="`image-size-resolution-${option.key}`"
                  class="rounded-xl border px-4 py-2.5 text-sm font-medium transition focus:outline-none focus:ring-2 focus:ring-primary-500/30"
                  :class="sizePickerResolution === option.value ? 'border-primary-400 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-600'"
                  @click="sizePickerResolution = option.value"
                >
                  {{ option.label }}
                </button>
              </div>
            </div>

            <div>
              <p class="mb-3 text-sm font-medium text-gray-500 dark:text-gray-400">
                {{ t('imagePlayground.sizePickerRatioLabel') }}
              </p>
              <div class="grid grid-cols-4 gap-2">
                <button
                  v-for="option in sizeRatioOptions"
                  :key="option.label"
                  type="button"
                  :data-test="`image-size-ratio-${option.label}`"
                  class="flex flex-col items-center justify-center rounded-xl border px-3 py-3 text-sm transition focus:outline-none focus:ring-2 focus:ring-primary-500/30"
                  :class="sizePickerRatio === option.label ? 'border-primary-400 bg-primary-50 text-primary-700 dark:border-primary-500 dark:bg-primary-900/20 dark:text-primary-300' : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-600'"
                  @click="sizePickerRatio = option.label"
                >
                  <span class="mb-1 block rounded border border-current" :style="ratioPreviewStyle(option)" />
                  {{ option.label }}
                </button>
              </div>
            </div>

            <button
              type="button"
              class="w-full rounded-xl border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-700 transition hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200 dark:hover:bg-dark-700"
              @click="sizePickerMode = 'custom'"
            >
              {{ t('imagePlayground.sizePickerCustomRatioAction') }}
            </button>
          </div>

          <div v-else class="space-y-5 pt-5">
            <p class="text-sm font-medium text-gray-500 dark:text-gray-400">
              {{ t('imagePlayground.sizePickerCustomInputLabel') }}
            </p>
            <div class="grid grid-cols-[1fr_auto_1fr] items-end gap-3">
              <label class="block">
                <span class="input-label">{{ t('imagePlayground.sizePickerWidthLabel') }}</span>
                <input
                  v-model="sizePickerCustomWidth"
                  data-test="image-size-custom-width"
                  type="number"
                  min="1"
                  step="1"
                  class="input"
                />
              </label>
              <span class="pb-3 text-gray-400">×</span>
              <label class="block">
                <span class="input-label">{{ t('imagePlayground.sizePickerHeightLabel') }}</span>
                <input
                  v-model="sizePickerCustomHeight"
                  data-test="image-size-custom-height"
                  type="number"
                  min="1"
                  step="1"
                  class="input"
                />
              </label>
            </div>
            <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm leading-6 text-gray-600 dark:border-dark-600 dark:bg-dark-700 dark:text-gray-300">
              {{ t('imagePlayground.sizePickerLimitHint') }}
            </div>
          </div>
        </div>

        <footer class="space-y-4 px-6 pb-6 pt-3">
          <div>
            <p class="text-sm text-gray-400 dark:text-gray-500">
              {{ t('imagePlayground.sizePickerWillUse') }}
            </p>
            <p data-test="image-size-pending" class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
              {{ sizePickerPendingSize }}
            </p>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <button type="button" class="btn btn-secondary" @click="closeSizePicker">
              {{ t('imagePlayground.sizePickerCancel') }}
            </button>
            <button
              type="button"
              data-test="image-size-confirm"
              class="btn btn-primary"
              :disabled="!sizePickerPendingSize"
              @click="confirmSizePicker"
            >
              {{ t('imagePlayground.sizePickerConfirm') }}
            </button>
          </div>
        </footer>
      </section>
    </div>

    <div
      v-if="imagePreview"
      class="fixed inset-0 z-[60] flex items-center justify-center bg-gray-950/80 px-4 py-6 backdrop-blur-sm"
      role="presentation"
      @click.self="closeImagePreview"
    >
      <section
        class="flex max-h-[calc(100vh-3rem)] w-full max-w-5xl flex-col overflow-hidden rounded-3xl bg-white shadow-2xl dark:bg-dark-800"
        role="dialog"
        aria-modal="true"
        :aria-label="t('imagePlayground.imagePreviewTitle')"
      >
        <header class="flex items-start justify-between gap-4 border-b border-gray-200 p-5 dark:border-dark-600">
          <div class="min-w-0">
            <p class="text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
              {{ t('imagePlayground.imagePreviewTitle') }}
            </p>
            <h2 class="mt-1 truncate text-lg font-semibold text-gray-900 dark:text-white">
              {{ imagePreview.title }}
            </h2>
            <p v-if="imagePreview.meta" class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ imagePreview.meta }}
            </p>
          </div>
          <div class="flex shrink-0 items-center gap-2">
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              @click="downloadPreviewImage"
            >
              {{ t('imagePlayground.downloadImageAction') }}
            </button>
            <button
              type="button"
              class="rounded-full p-2 text-gray-400 transition hover:bg-gray-100 hover:text-gray-700 focus:outline-none focus:ring-2 focus:ring-primary-500/30 dark:hover:bg-dark-700 dark:hover:text-gray-200"
              :aria-label="t('imagePlayground.closePreviewAction')"
              @click="closeImagePreview"
            >
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.8" aria-hidden="true">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </header>

        <div class="flex min-h-0 flex-1 items-center justify-center bg-gray-100 p-4 dark:bg-dark-900">
          <img
            :src="imagePreview.source"
            :alt="imagePreview.alt"
            class="max-h-[calc(100vh-14rem)] w-full object-contain"
          />
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'
import {
  generateImage,
  getImageOptions,
  type ImagePlaygroundCostMetadata,
  type ImagePlaygroundGenerateResponse,
  type ImagePlaygroundImageResult,
  type ImagePlaygroundKeyOption,
} from '@/api/imagePlayground'
import {
  addImageHistoryRecord,
  clearImageHistory,
  deleteImageHistoryRecord,
  listImageHistoryRecords,
  type ImageHistoryRecord,
} from '@/utils/imagePlaygroundHistory'

type OutputFormat = 'png' | 'jpeg' | 'webp'
type SizePickerMode = 'auto' | 'ratio' | 'custom'

interface SizeRatioOption {
  label: string
  width: number
  height: number
}

interface SizeResolutionOption {
  key: string
  label: string
  value: number
}

interface ReferenceImageItem {
  id: string
  file: File
  tooLarge: boolean
}

interface GenerationSnapshot {
  userId: number
  apiKeyId: number
  keyName: string
  model: string
  prompt: string
  size: string
  quality: string
  outputFormat: OutputFormat
  outputCompression?: number
  moderation: string
  count: number
  referenceImages: File[]
}

interface ImagePreviewState {
  source: string
  alt: string
  title: string
  meta: string
  filename: string
}

type ImageHistoryItem = ImageHistoryRecord['images'][number]

const MAX_REFERENCE_FILE_SIZE = 20 * 1024 * 1024
const qualityOptions = ['auto', 'low', 'medium', 'high']
const outputFormatOptions: OutputFormat[] = ['png', 'jpeg', 'webp']
const moderationOptions = ['auto', 'low']
const sizeResolutionOptions: SizeResolutionOption[] = [
  { key: '1k', label: '1K', value: 1024 },
  { key: '2k', label: '2K', value: 2048 },
  { key: '4k', label: '4K', value: 4096 },
]
const sizeRatioOptions: SizeRatioOption[] = [
  { label: '1:1', width: 1, height: 1 },
  { label: '3:2', width: 3, height: 2 },
  { label: '2:3', width: 2, height: 3 },
  { label: '16:9', width: 16, height: 9 },
  { label: '9:16', width: 9, height: 16 },
  { label: '4:3', width: 4, height: 3 },
  { label: '3:4', width: 3, height: 4 },
  { label: '21:9', width: 21, height: 9 },
]

const { t } = useI18n()
const authStore = useAuthStore()

const loadingOptions = ref(true)
const generating = ref(false)
const availableKeys = ref<ImagePlaygroundKeyOption[]>([])
const fallbackModels = ref<string[]>([])
const historyRecords = ref<ImageHistoryRecord[]>([])
const selectedKeyId = ref<number | null>(null)
const model = ref('')
const prompt = ref('')
const size = ref('auto')
const sizePickerOpen = ref(false)
const sizePickerMode = ref<SizePickerMode>('auto')
const sizePickerResolution = ref(1024)
const sizePickerRatio = ref('1:1')
const sizePickerCustomWidth = ref('1024')
const sizePickerCustomHeight = ref('1024')
const quality = ref('auto')
const outputFormat = ref<OutputFormat>('png')
const compressionInput = ref('')
const moderation = ref('auto')
const count = ref(1)
const referenceItems = ref<ReferenceImageItem[]>([])
const currentResults = ref<ImagePlaygroundImageResult[]>([])
const currentPrice = ref<ImagePlaygroundCostMetadata | undefined>(undefined)
const currentPrompt = ref('')
const currentOutputFormat = ref<OutputFormat>('png')
const currentRequestId = ref('initial')
const statusMessage = ref('')
const errorMessage = ref('')
const referenceInputRef = ref<HTMLInputElement | null>(null)
const referenceDragging = ref(false)
const pendingReuseModel = ref<string | null>(null)
const imagePreview = ref<ImagePreviewState | null>(null)

const clearMessages = (): void => {
  statusMessage.value = ''
  errorMessage.value = ''
}

const showStatus = (message: string): void => {
  statusMessage.value = message
  errorMessage.value = ''
}

const showError = (message: string): void => {
  errorMessage.value = message
  statusMessage.value = ''
}

const hasAvailableKeys = computed(() => availableKeys.value.length > 0)
const selectedKey = computed(() => {
  if (selectedKeyId.value == null) {
    return null
  }

  return availableKeys.value.find((key) => key.id === selectedKeyId.value) ?? null
})

const modelSuggestions = computed(() => {
  const suggestions = new Set<string>()

  selectedKey.value?.models.forEach((item) => {
    if (item) {
      suggestions.add(item)
    }
  })

  fallbackModels.value.forEach((item) => {
    if (item) {
      suggestions.add(item)
    }
  })

  return Array.from(suggestions)
})

const compressionEnabled = computed(() => outputFormat.value !== 'png')

const parsedCompression = computed<number | undefined>(() => {
  if (!compressionEnabled.value) {
    return undefined
  }

  const normalized = String(compressionInput.value ?? '').trim()
  if (!/^\d+$/.test(normalized)) {
    return undefined
  }

  return Number(normalized)
})

const invalidCompression = computed(() => {
  if (!compressionEnabled.value) {
    return false
  }

  return (
    parsedCompression.value == null ||
    parsedCompression.value < 0 ||
    parsedCompression.value > 100
  )
})

const parsePositiveIntegerInput = (value: string): number | null => {
  const normalized = String(value ?? '').trim()
  if (!/^\d+$/.test(normalized)) {
    return null
  }

  const parsed = Number(normalized)
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
}

const selectedSizeRatioOption = computed(
  () => sizeRatioOptions.find((option) => option.label === sizePickerRatio.value) ?? sizeRatioOptions[0]
)
const sizePickerPendingSize = computed(() => {
  if (sizePickerMode.value === 'auto') {
    return 'auto'
  }

  if (sizePickerMode.value === 'custom') {
    const width = parsePositiveIntegerInput(sizePickerCustomWidth.value)
    const height = parsePositiveIntegerInput(sizePickerCustomHeight.value)
    return width && height ? `${width}x${height}` : ''
  }

  const ratio = selectedSizeRatioOption.value
  if (!ratio) {
    return ''
  }

  if (ratio.width >= ratio.height) {
    return `${Math.round((sizePickerResolution.value * ratio.width) / ratio.height)}x${sizePickerResolution.value}`
  }

  return `${sizePickerResolution.value}x${Math.round((sizePickerResolution.value * ratio.height) / ratio.width)}`
})

const ratioPreviewStyle = (option: SizeRatioOption): Record<string, string> => {
  const maxSize = 22
  if (option.width >= option.height) {
    return {
      width: `${maxSize}px`,
      height: `${Math.max(8, Math.round((maxSize * option.height) / option.width))}px`,
    }
  }

  return {
    width: `${Math.max(8, Math.round((maxSize * option.width) / option.height))}px`,
    height: `${maxSize}px`,
  }
}

const syncSizePickerFromCurrentSize = (): void => {
  if (size.value === 'auto') {
    sizePickerMode.value = 'auto'
    return
  }

  const match = /^(\d+)x(\d+)$/.exec(size.value)
  if (!match) {
    sizePickerMode.value = 'auto'
    return
  }

  const width = Number(match[1])
  const height = Number(match[2])
  sizePickerCustomWidth.value = String(width)
  sizePickerCustomHeight.value = String(height)
  sizePickerMode.value = 'custom'

  const matchedRatio = sizeRatioOptions.find((option) => width * option.height === height * option.width)
  if (matchedRatio) {
    sizePickerRatio.value = matchedRatio.label
    sizePickerResolution.value = Math.min(width, height)
    sizePickerMode.value = 'ratio'
  }
}

const openSizePicker = (): void => {
  syncSizePickerFromCurrentSize()
  sizePickerOpen.value = true
}

const closeSizePicker = (): void => {
  sizePickerOpen.value = false
}

const confirmSizePicker = (): void => {
  if (!sizePickerPendingSize.value) {
    return
  }

  size.value = sizePickerPendingSize.value
  closeSizePicker()
}

const invalidCount = computed(() => !Number.isInteger(count.value) || count.value < 1 || count.value > 4)
const hasOversizeFile = computed(() => referenceItems.value.some((item) => item.tooLarge))
const referenceImagesForSubmit = computed(() =>
  referenceItems.value.filter((item) => !item.tooLarge).map((item) => item.file)
)
const generateDisabled = computed(() => {
  return (
    !selectedKey.value ||
    model.value.trim().length === 0 ||
    prompt.value.trim().length === 0 ||
    generating.value ||
    invalidCompression.value ||
    invalidCount.value ||
    hasOversizeFile.value
  )
})
const formattedCurrentPrice = computed(() => formatPrice(currentPrice.value))
const currentUserId = computed(() => authStore.user?.id)

watch(
  outputFormat,
  (nextFormat) => {
    if (nextFormat === 'png') {
      compressionInput.value = ''
      return
    }

    if (!compressionInput.value) {
      compressionInput.value = '80'
    }
  },
  { immediate: true }
)

watch(selectedKeyId, (nextKeyId) => {
  if (nextKeyId == null) {
    return
  }

  const nextKey = availableKeys.value.find((key) => key.id === nextKeyId)
  if (!nextKey) {
    return
  }

  if (pendingReuseModel.value !== null) {
    model.value = pendingReuseModel.value
    pendingReuseModel.value = null
    return
  }

  model.value = nextKey.default_model || nextKey.models[0] || fallbackModels.value[0] || ''
})

const imageMimeType = (format: string): string => {
  if (format === 'jpeg') {
    return 'image/jpeg'
  }

  return `image/${format}`
}

const inferResultMimeType = (result: ImagePlaygroundImageResult, format: string): string => {
  if (result.url) {
    const normalizedUrl = result.url.toLowerCase()
    if (normalizedUrl.endsWith('.jpg') || normalizedUrl.endsWith('.jpeg')) {
      return 'image/jpeg'
    }
    if (normalizedUrl.endsWith('.webp')) {
      return 'image/webp'
    }
    if (normalizedUrl.endsWith('.png')) {
      return 'image/png'
    }
  }

  return imageMimeType(format)
}

const imageSrc = (result: ImagePlaygroundImageResult, format: string): string => {
  if (result.url) {
    return result.url
  }

  if (result.b64_json) {
    return `data:${imageMimeType(format)};base64,${result.b64_json}`
  }

  return ''
}

const blobToDataUrl = async (blob: Blob, fallbackMimeType: string): Promise<string> => {
  return await new Promise((resolve, reject) => {
    const reader = new FileReader()

    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
        return
      }
      reject(new Error('Image data could not be read'))
    }
    reader.onerror = () => reject(reader.error ?? new Error('Image data could not be read'))
    reader.readAsDataURL(blob.type ? blob : blob.slice(0, blob.size, fallbackMimeType))
  })
}

const mimeTypeFromDataUrl = (source: string): string | null => {
  const match = /^data:([^;,]+)[;,]/.exec(source)
  return match?.[1] ?? null
}

const imageHistorySource = async (
  result: ImagePlaygroundImageResult,
  format: string
): Promise<{ source: string; mimeType: string } | null> => {
  if (result.b64_json) {
    const mimeType = imageMimeType(format)
    return {
      source: `data:${mimeType};base64,${result.b64_json}`,
      mimeType,
    }
  }

  if (!result.url) {
    return null
  }

  if (result.url.startsWith('data:')) {
    return {
      source: result.url,
      mimeType: mimeTypeFromDataUrl(result.url) ?? inferResultMimeType(result, format),
    }
  }

  const fallbackMimeType = inferResultMimeType(result, format)
  try {
    const response = await fetch(result.url)
    if (!response.ok) {
      return null
    }
    const blob = await response.blob()
    return {
      source: await blobToDataUrl(blob, fallbackMimeType),
      mimeType: blob.type || fallbackMimeType,
    }
  } catch {
    return null
  }
}

const localizeGeneratedImages = async (
  generatedImages: ImagePlaygroundImageResult[],
  format: string
): Promise<ImagePlaygroundImageResult[]> => {
  const localized = await Promise.all(
    generatedImages.map(async (result): Promise<ImagePlaygroundImageResult | null> => {
      const historySource = await imageHistorySource(result, format)
      if (!historySource) {
        return null
      }

      return {
        ...result,
        url: historySource.source,
        b64_json: undefined,
      }
    })
  )

  return localized.filter((result): result is ImagePlaygroundImageResult => result !== null)
}

const formatPrice = (metadata?: ImagePlaygroundCostMetadata): string => {
  if (!metadata) {
    return ''
  }

  const value = metadata.actual_cost ?? metadata.total_cost ?? metadata.estimated_price
  if (typeof value !== 'number' || Number.isNaN(value)) {
    return ''
  }

  return `$${value.toFixed(4)}`
}

const formatFileSize = (bytes: number): string => {
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`
  }

  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

const formatRecordTimestamp = (value: string): string => {
  return new Date(value).toLocaleString()
}

const createReferenceId = (file: File): string => {
  return `${file.name}-${file.size}-${file.lastModified}-${Math.random().toString(36).slice(2, 8)}`
}

const generatedResultAlt = (index: number): string => {
  return `${currentPrompt.value || t('imagePlayground.generatedImageAlt')} ${index + 1}`
}

const historyImageAlt = (record: ImageHistoryRecord, index: number): string => {
  return `${record.prompt || t('imagePlayground.generatedImageAlt')} ${index + 1}`
}

const extensionFromMimeType = (mimeType?: string | null): string => {
  if (mimeType === 'image/jpeg') {
    return 'jpg'
  }
  if (mimeType === 'image/png') {
    return 'png'
  }
  if (mimeType === 'image/webp') {
    return 'webp'
  }

  const subtype = mimeType?.split('/')[1]?.split('+')[0]
  return subtype && /^[a-z0-9]+$/i.test(subtype) ? subtype.toLowerCase() : 'png'
}

const extensionFromSource = (source: string, fallbackMimeType?: string | null): string => {
  const dataUrlMimeType = mimeTypeFromDataUrl(source)
  if (dataUrlMimeType) {
    return extensionFromMimeType(dataUrlMimeType)
  }

  try {
    const pathname = new URL(source, window.location.href).pathname
    const match = /\.([a-z0-9]+)$/i.exec(pathname)
    if (match?.[1]) {
      return match[1].toLowerCase() === 'jpeg' ? 'jpg' : match[1].toLowerCase()
    }
  } catch {
    // Fall back to MIME type when the source is not a parseable URL.
  }

  return extensionFromMimeType(fallbackMimeType)
}

const sanitizeFilenamePart = (value: string): string => {
  const normalized = value
    .trim()
    .replace(/[\\/:*?"<>|]+/g, '-')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')
    .slice(0, 80)

  return normalized || 'image'
}

const generatedResultFilename = (result: ImagePlaygroundImageResult, index: number): string => {
  const source = imageSrc(result, currentOutputFormat.value)
  const title = currentPrompt.value || t('imagePlayground.generatedImageAlt')
  const extension = extensionFromSource(source, inferResultMimeType(result, currentOutputFormat.value))
  return `generated-${sanitizeFilenamePart(title)}-${index + 1}.${extension}`
}

const historyImageFilename = (record: ImageHistoryRecord, image: ImageHistoryItem, index: number): string => {
  const extension = extensionFromSource(image.url_or_base64, image.mime_type)
  return `history-${sanitizeFilenamePart(record.prompt || record.id)}-${index + 1}.${extension}`
}

const triggerBrowserDownload = (source: string, filename: string): void => {
  const link = document.createElement('a')
  link.href = source
  link.download = filename
  link.rel = 'noopener noreferrer'
  document.body.appendChild(link)
  link.click()
  link.remove()
}

const downloadImage = async (source: string, filename: string): Promise<void> => {
  if (!source) {
    return
  }

  if (source.startsWith('data:') || source.startsWith('blob:')) {
    triggerBrowserDownload(source, filename)
    return
  }

  try {
    const response = await fetch(source)
    if (!response.ok) {
      triggerBrowserDownload(source, filename)
      return
    }

    const objectUrl = URL.createObjectURL(await response.blob())
    try {
      triggerBrowserDownload(objectUrl, filename)
    } finally {
      URL.revokeObjectURL(objectUrl)
    }
  } catch {
    triggerBrowserDownload(source, filename)
  }
}

const openGeneratedResultPreview = (result: ImagePlaygroundImageResult, index: number): void => {
  const source = imageSrc(result, currentOutputFormat.value)
  if (!source) {
    return
  }

  imagePreview.value = {
    source,
    alt: generatedResultAlt(index),
    title: currentPrompt.value || t('imagePlayground.generatedImageAlt'),
    meta: formattedCurrentPrice.value,
    filename: generatedResultFilename(result, index),
  }
}

const openHistoryImagePreview = (record: ImageHistoryRecord, image: ImageHistoryItem, index: number): void => {
  imagePreview.value = {
    source: image.url_or_base64,
    alt: historyImageAlt(record, index),
    title: record.prompt || t('imagePlayground.generatedImageAlt'),
    meta: `${record.key_name} · ${record.model} · ${formatRecordTimestamp(record.created_at)}`,
    filename: historyImageFilename(record, image, index),
  }
}

const closeImagePreview = (): void => {
  imagePreview.value = null
}

const downloadGeneratedResult = (result: ImagePlaygroundImageResult, index: number): void => {
  void downloadImage(imageSrc(result, currentOutputFormat.value), generatedResultFilename(result, index))
}

const downloadHistoryImage = (record: ImageHistoryRecord, image: ImageHistoryItem, index: number): void => {
  void downloadImage(image.url_or_base64, historyImageFilename(record, image, index))
}

const downloadPreviewImage = (): void => {
  if (!imagePreview.value) {
    return
  }

  void downloadImage(imagePreview.value.source, imagePreview.value.filename)
}

const setInitialKeyAndModel = (): void => {
  const firstKey = availableKeys.value[0]
  if (!firstKey) {
    selectedKeyId.value = null
    model.value = ''
    return
  }

  selectedKeyId.value = firstKey.id
  model.value = firstKey.default_model || firstKey.models[0] || fallbackModels.value[0] || ''
}

const loadHistory = async (): Promise<void> => {
  const userId = currentUserId.value
  historyRecords.value = typeof userId === 'number' ? await listImageHistoryRecords(userId) : []
}

const loadOptions = async (): Promise<void> => {
  const response = await getImageOptions()
  availableKeys.value = response.keys.filter((key) => key.allow_image_generation)
  fallbackModels.value = response.fallback_models ?? []
  setInitialKeyAndModel()
}

const loadPageData = async (): Promise<void> => {
  loadingOptions.value = true
  clearMessages()

  const [optionsResult, historyResult] = await Promise.allSettled([loadOptions(), loadHistory()])

  if (optionsResult.status === 'rejected') {
    showError(t('imagePlayground.loadOptionsFailed'))
  }

  if (historyResult.status === 'rejected' && !errorMessage.value) {
    showError(t('imagePlayground.loadHistoryFailed'))
  }

  loadingOptions.value = false
}

const addReferenceFiles = (files: File[]): void => {
  if (files.length === 0) {
    return
  }

  const nextItems = files.map((file) => ({
    id: createReferenceId(file),
    file,
    tooLarge: file.size > MAX_REFERENCE_FILE_SIZE,
  }))

  referenceItems.value = [...referenceItems.value, ...nextItems]

  if (hasOversizeFile.value) {
    showError(t('imagePlayground.referenceTooLargeError'))
  } else {
    showStatus(t('imagePlayground.referenceAddedStatus'))
  }
}

const handleReferenceInput = (event: Event): void => {
  const input = event.target as HTMLInputElement
  addReferenceFiles(Array.from(input.files ?? []))

  if (input) {
    input.value = ''
  }
}

const handleReferenceDrop = (event: DragEvent): void => {
  referenceDragging.value = false
  addReferenceFiles(Array.from(event.dataTransfer?.files ?? []))
}

const removeReference = (referenceId: string): void => {
  referenceItems.value = referenceItems.value.filter((item) => item.id !== referenceId)

  if (!hasOversizeFile.value && errorMessage.value === t('imagePlayground.referenceTooLargeError')) {
    clearMessages()
  }
}

const createGenerationSnapshot = (): GenerationSnapshot | null => {
  const userId = currentUserId.value
  if (!selectedKey.value || typeof userId !== 'number') {
    return null
  }

  return {
    userId,
    apiKeyId: selectedKey.value.id,
    keyName: selectedKey.value.name,
    model: model.value.trim(),
    prompt: prompt.value.trim(),
    size: size.value,
    quality: quality.value,
    outputFormat: outputFormat.value,
    outputCompression: parsedCompression.value,
    moderation: moderation.value,
    count: count.value,
    referenceImages: [...referenceImagesForSubmit.value],
  }
}

const createHistoryRecordFromResponse = async (
  response: ImagePlaygroundGenerateResponse,
  generatedImages: ImagePlaygroundImageResult[],
  snapshot: GenerationSnapshot
): Promise<ImageHistoryRecord | null> => {
  if (generatedImages.length === 0) {
    return null
  }

  const images = (await Promise.all(
    generatedImages.map(async (result) => {
      const historySource = await imageHistorySource(result, snapshot.outputFormat)
      if (!historySource) {
        return null
      }

      return {
        url_or_base64: historySource.source,
        mime_type: historySource.mimeType,
      }
    })
  )).filter((item): item is { url_or_base64: string; mime_type: string } => item !== null)

  if (images.length === 0) {
    return null
  }

  return {
    id: globalThis.crypto?.randomUUID?.() ?? String(Date.now()),
    user_id: snapshot.userId,
    created_at: new Date().toISOString(),
    api_key_id: snapshot.apiKeyId,
    key_name: snapshot.keyName,
    model: snapshot.model,
    prompt: snapshot.prompt,
    params: {
      size: snapshot.size,
      quality: snapshot.quality,
      output_format: snapshot.outputFormat,
      output_compression: snapshot.outputCompression,
      moderation: snapshot.moderation,
      n: snapshot.count,
    },
    price: response._sub2api_image_playground,
    images,
  }
}

const handleGenerate = async (): Promise<void> => {
  const snapshot = createGenerationSnapshot()
  if (generateDisabled.value || !snapshot) {
    return
  }

  generating.value = true
  showStatus(t('imagePlayground.generatingStatus'))

  try {
    const response = await generateImage({
      api_key_id: snapshot.apiKeyId,
      model: snapshot.model,
      prompt: snapshot.prompt,
      size: snapshot.size,
      quality: snapshot.quality,
      output_format: snapshot.outputFormat,
      output_compression: snapshot.outputCompression,
      moderation: snapshot.moderation,
      n: snapshot.count,
      reference_images: snapshot.referenceImages,
    })

    const generatedImages = await localizeGeneratedImages(response.data ?? [], snapshot.outputFormat)

    currentResults.value = generatedImages
    currentPrice.value = response._sub2api_image_playground
    currentPrompt.value = snapshot.prompt
    currentOutputFormat.value = snapshot.outputFormat
    currentRequestId.value = globalThis.crypto?.randomUUID?.() ?? String(Date.now())
    showStatus(
      generatedImages.length > 0
        ? t('imagePlayground.generateSuccessStatus')
        : t('imagePlayground.generateEmptyStatus')
    )

    const historyRecord = await createHistoryRecordFromResponse(response, generatedImages, snapshot)
    if (historyRecord) {
      await addImageHistoryRecord(historyRecord)
      await loadHistory()
    }
  } catch (error) {
    currentResults.value = []
    currentPrice.value = undefined
    showError(t('imagePlayground.generateFailed'))
  } finally {
    generating.value = false
  }
}

const applyOutputSettings = (format: string, compression?: number): void => {
  if (format === 'jpeg' || format === 'webp') {
    outputFormat.value = format
    compressionInput.value = typeof compression === 'number' ? String(compression) : '80'
    return
  }

  outputFormat.value = 'png'
  compressionInput.value = ''
}

const reuseHistoryRecord = (record: ImageHistoryRecord): void => {
  const recordKeyAvailable = availableKeys.value.some((key) => key.id === record.api_key_id)
  if (recordKeyAvailable && selectedKeyId.value !== record.api_key_id) {
    pendingReuseModel.value = record.model
    selectedKeyId.value = record.api_key_id
  } else if (!recordKeyAvailable) {
    pendingReuseModel.value = null
  }

  model.value = record.model
  prompt.value = record.prompt
  size.value = record.params.size
  quality.value = record.params.quality
  moderation.value = record.params.moderation
  count.value = record.params.n
  applyOutputSettings(record.params.output_format, record.params.output_compression)
  showStatus(t('imagePlayground.reuseParamsStatus'))
}

const handleDeleteHistory = async (recordId: string): Promise<void> => {
  try {
    await deleteImageHistoryRecord(recordId, currentUserId.value)
    await loadHistory()
    showStatus(t('imagePlayground.historyDeletedStatus'))
  } catch (error) {
    showError(t('imagePlayground.historyDeleteFailed'))
  }
}

const handleClearHistory = async (): Promise<void> => {
  try {
    await clearImageHistory(currentUserId.value)
    await loadHistory()
    showStatus(t('imagePlayground.historyClearedStatus'))
  } catch (error) {
    showError(t('imagePlayground.historyClearFailed'))
  }
}

onMounted(() => {
  void loadPageData()
})
</script>

<style scoped>
.image-playground-select {
  @apply w-full appearance-none rounded-xl border border-gray-200 bg-white px-4 py-2.5 pr-10 text-sm text-gray-900 shadow-sm transition-all duration-200;
  @apply focus:border-primary-500 focus:outline-none focus:ring-2 focus:ring-primary-500/30;
  @apply disabled:cursor-not-allowed disabled:bg-gray-100 disabled:text-gray-400;
  @apply dark:border-dark-600 dark:bg-dark-800 dark:text-gray-100 dark:disabled:bg-dark-900 dark:disabled:text-gray-500;
  background-image: linear-gradient(45deg, transparent 50%, currentColor 50%), linear-gradient(135deg, currentColor 50%, transparent 50%);
  background-position: calc(100% - 18px) 50%, calc(100% - 13px) 50%;
  background-size: 5px 5px, 5px 5px;
  background-repeat: no-repeat;
}
</style>
