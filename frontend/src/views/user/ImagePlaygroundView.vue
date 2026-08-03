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

      <div
        v-else-if="!loadingOptions"
        class="grid items-start gap-6"
        :class="advancedEnabled ? 'xl:grid-cols-[420px_minmax(360px,520px)_minmax(0,1fr)] xl:h-[calc(100vh-8rem)] xl:min-h-0 xl:items-stretch xl:overflow-hidden' : 'xl:grid-cols-[440px_minmax(0,1fr)]'"
      >
        <div :class="advancedEnabled ? 'xl:min-h-0' : 'xl:relative'">
          <section
            data-test="image-generator-panel"
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
            :class="advancedEnabled ? 'xl:flex xl:h-full xl:min-h-0 xl:flex-col' : 'xl:fixed xl:top-24 xl:flex xl:max-h-[calc(100vh-8rem)] xl:w-[440px] xl:flex-col'"
          >
            <div
              data-test="image-generator-scroll"
              :class="advancedEnabled ? 'xl:flex xl:min-h-0 xl:flex-1 xl:flex-col xl:overflow-y-auto xl:pr-1' : 'xl:min-h-0 xl:flex-1 xl:overflow-y-auto xl:pr-2'"
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

                <label
                  for="image-advanced"
                  class="inline-flex min-w-0 shrink-0 cursor-pointer items-center gap-3 rounded-2xl border border-gray-200 bg-white px-3 py-2 shadow-sm transition hover:border-primary-300 dark:border-dark-600 dark:bg-dark-700"
                >
                  <span class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ t('imagePlayground.advancedModeLabel') }}
                  </span>
                  <span
                    class="relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition"
                    :class="advancedEnabled ? 'bg-primary-600' : 'bg-gray-300 dark:bg-dark-500'"
                  >
                    <span
                      class="inline-block h-5 w-5 transform rounded-full bg-white shadow transition"
                      :class="advancedEnabled ? 'translate-x-5' : 'translate-x-0.5'"
                    />
                  </span>
                  <input
                    id="image-advanced"
                    v-model="advancedEnabled"
                    data-test="image-advanced"
                    type="checkbox"
                    class="sr-only"
                  />
                </label>
              </div>

              <form class="flex min-h-0 flex-1 flex-col gap-5" @submit.prevent="handleGenerate">
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

            <div class="flex min-h-0 flex-1 flex-col">
              <label for="image-prompt" class="input-label">
                {{ t('imagePlayground.promptLabel') }}
              </label>
              <textarea
                id="image-prompt"
                v-model="prompt"
                data-test="image-prompt"
                rows="5"
                class="input min-h-0 flex-1 resize-y leading-6"
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

            <div class="rounded-2xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-600 dark:bg-dark-700/70">
              <label for="image-stream" class="flex cursor-pointer items-start gap-3">
                <input
                  id="image-stream"
                  v-model="streamEnabled"
                  data-test="image-stream"
                  type="checkbox"
                  :disabled="!streamSupported"
                  class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-50"
                />
                <span class="min-w-0 flex-1">
                  <span class="block text-sm font-medium text-gray-900 dark:text-white">
                    {{ t('imagePlayground.streamLabel') }}
                  </span>
                  <span class="mt-1 block text-xs leading-5 text-gray-500 dark:text-gray-400">
                    {{ streamSupported ? t('imagePlayground.streamHint') : t('imagePlayground.streamUnsupportedHint') }}
                  </span>
                </span>
              </label>
            </div>

                <div
                  data-test="image-generator-actions"
                  class="border-t border-gray-200 pt-4 dark:border-dark-600 xl:sticky xl:bottom-0 xl:mt-5 xl:bg-white xl:pb-1 dark:xl:bg-dark-800"
                >
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
            </div>
          </section>
        </div>

        <section
          v-if="advancedEnabled"
          class="flex flex-col gap-6 xl:h-full xl:min-h-0"
        >
          <div class="flex min-h-0 flex-1 flex-col rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('imagePlayground.advancedModeLabel') }}
              </h2>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                {{ t('imagePlayground.advancedModeHint') }}
              </p>
            </div>

            <div class="mt-5 flex min-h-0 flex-1 flex-col space-y-4">
              <div>
                <label for="image-advanced-path" class="input-label">
                  {{ t('imagePlayground.advancedPathLabel') }}
                </label>
                <input
                  id="image-advanced-path"
                  v-model="advancedRequestPath"
                  data-test="image-advanced-path"
                  type="text"
                  class="input font-mono text-xs"
                  :aria-invalid="advancedRequestError ? 'true' : 'false'"
                  @input="handleAdvancedRequestPathInput"
                />
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('imagePlayground.advancedPathHint') }}
                </p>
              </div>

              <div class="flex min-h-0 flex-1 flex-col">
                <div class="mb-2 flex items-center justify-between gap-3">
                  <label for="image-advanced-body" class="input-label">
                    {{ t('imagePlayground.advancedBodyLabel') }}
                  </label>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    @click="formatAdvancedRequestBody"
                  >
                    {{ t('imagePlayground.advancedFormatAction') }}
                  </button>
                </div>
                <textarea
                  id="image-advanced-body"
                  v-model="advancedRequestBodyText"
                  data-test="image-advanced-body"
                  rows="18"
                  class="input min-h-0 flex-1 resize-none font-mono text-xs leading-5"
                  spellcheck="false"
                  :aria-invalid="advancedRequestError ? 'true' : 'false'"
                  @input="handleAdvancedRequestBodyInput"
                />
              </div>

              <p
                v-if="advancedRequestError"
                data-test="image-advanced-error"
                class="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300"
              >
                {{ advancedRequestError }}
              </p>
            </div>
          </div>

          <div class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800">
            <div>
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t('imagePlayground.advancedResultTitle') }}
              </h2>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">
                {{ t('imagePlayground.advancedResultDescription') }}
              </p>
            </div>

            <div class="mt-4 space-y-4">
              <div>
                <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('imagePlayground.advancedLastRequestLabel') }}
                </p>
                <pre class="max-h-80 overflow-auto rounded-xl bg-gray-950 p-4 text-xs leading-5 text-gray-100"><code>{{ prettyAdvancedRequest }}</code></pre>
              </div>
              <div>
                <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('imagePlayground.advancedLastResponseLabel') }}
                </p>
                <pre class="max-h-80 overflow-auto rounded-xl bg-gray-950 p-4 text-xs leading-5 text-gray-100"><code>{{ prettyAdvancedResponse }}</code></pre>
              </div>
            </div>
          </div>
        </section>

        <aside
          class="space-y-6"
          :class="advancedEnabled ? 'xl:flex xl:h-full xl:min-h-0 xl:flex-col xl:space-y-0 xl:gap-6 xl:overflow-hidden' : ''"
        >
          <section
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
            :class="advancedEnabled ? 'xl:shrink-0 xl:overflow-hidden' : ''"
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
              v-if="statusMessage || errorMessage || generationProgressActive"
              class="mt-4 rounded-xl border px-4 py-3 text-sm"
              :class="errorMessage ? 'border-red-200 bg-red-50 text-red-700 dark:border-red-900/50 dark:bg-red-900/20 dark:text-red-300' : 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-900/20 dark:text-emerald-300'"
              aria-live="polite"
            >
              <p v-if="errorMessage" aria-live="assertive">
                {{ errorMessage }}
              </p>
              <p v-else-if="statusMessage">
                {{ statusMessage }}
              </p>
              <div
                v-if="generationProgressActive"
                class="mt-3 space-y-2"
              >
                <div class="flex items-center justify-between gap-3 text-xs font-medium">
                  <span>{{ t('imagePlayground.generationProgressLabel') }}</span>
                  <span data-test="image-generation-progress-value">
                    {{ generationProgressValueText }}
                  </span>
                </div>
                <div
                  data-test="image-generation-progress"
                  role="progressbar"
                  :aria-label="t('imagePlayground.generationProgressLabel')"
                  aria-valuemin="0"
                  aria-valuemax="100"
                  :aria-valuenow="generationProgressRounded ?? undefined"
                  class="image-generation-progress-track"
                >
                  <div
                    class="image-generation-progress-bar"
                    :class="generationProgressRounded == null ? 'image-generation-progress-bar-indeterminate' : ''"
                    :style="generationProgressBarStyle"
                  />
                </div>
              </div>
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
            :class="advancedEnabled ? 'xl:shrink-0 xl:overflow-hidden' : ''"
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

          <section
            class="rounded-2xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-600 dark:bg-dark-800"
            :class="advancedEnabled ? 'xl:flex xl:min-h-0 xl:flex-1 xl:flex-col xl:overflow-hidden' : ''"
          >
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

            <div
              v-if="historyRecords.length === 0"
              class="mt-4 rounded-xl bg-gray-50 p-4 text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400"
              :class="advancedEnabled ? 'xl:min-h-0 xl:flex-1' : ''"
            >
              {{ t('imagePlayground.historyEmpty') }}
            </div>

            <div
              v-else
              class="mt-4 grid gap-4 md:grid-cols-2 2xl:grid-cols-3"
              :class="advancedEnabled ? 'xl:min-h-0 xl:flex-1 xl:overflow-y-auto xl:pr-1' : ''"
            >
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

        <div
          class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-3 text-sm dark:border-dark-600"
          :aria-label="t('imagePlayground.previewZoomControlsLabel')"
        >
          <div class="flex items-center gap-2">
            <button
              type="button"
              data-test="image-preview-zoom-out"
              class="btn btn-secondary btn-sm"
              :aria-label="t('imagePlayground.previewZoomOutAction')"
              @click="zoomImagePreviewOut"
            >
              −
            </button>
            <span
              data-test="image-preview-zoom-level"
              class="min-w-14 rounded-full bg-gray-100 px-3 py-1 text-center font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200"
            >
              {{ previewZoomPercent }}
            </span>
            <button
              type="button"
              data-test="image-preview-zoom-in"
              class="btn btn-secondary btn-sm"
              :aria-label="t('imagePlayground.previewZoomInAction')"
              @click="zoomImagePreviewIn"
            >
              +
            </button>
            <button
              type="button"
              data-test="image-preview-reset"
              class="btn btn-secondary btn-sm"
              @click="resetImagePreviewTransform"
            >
              {{ t('imagePlayground.previewZoomResetAction') }}
            </button>
          </div>
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('imagePlayground.previewPanHint') }}
          </p>
        </div>

        <div
          data-test="image-preview-pan-area"
          class="flex min-h-0 flex-1 touch-none cursor-grab select-none items-center justify-center overflow-hidden bg-gray-100 p-4 focus:outline-none focus:ring-2 focus:ring-primary-500/40 active:cursor-grabbing dark:bg-dark-900"
          tabindex="0"
          @wheel.prevent="handleImagePreviewWheel"
          @keydown="handleImagePreviewKeydown"
          @pointerdown="startImagePreviewPan"
          @pointermove="moveImagePreviewPan"
          @pointerup="endImagePreviewPan"
          @pointercancel="endImagePreviewPan"
        >
          <img
            data-test="image-preview-img"
            :src="imagePreview.source"
            :alt="imagePreview.alt"
            :style="{ transform: previewImageTransform }"
            :class="previewDragState ? 'transition-none' : 'transition-transform duration-150'"
            draggable="false"
            class="max-h-[calc(100vh-17rem)] w-full select-none object-contain will-change-transform"
          />
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'
import {
  extractImageGenerationErrorMessage,
  generateImage,
  generateImageAdvanced,
  imageGeneratePayload,
  generateImageStream,
  getImageOptions,
  type ImagePlaygroundCostMetadata,
  type ImagePlaygroundGenerateInput,
  type ImagePlaygroundGenerateResponse,
  type ImagePlaygroundImageResult,
  type ImagePlaygroundKeyOption,
  type ImagePlaygroundStreamEvent,
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

interface ImagePreviewOffset {
  x: number
  y: number
}

interface ImagePreviewDragState {
  pointerId: number
  startX: number
  startY: number
  originX: number
  originY: number
}

interface AdvancedRequestState {
  path: string
  body: Record<string, unknown>
  multipart: boolean
}

type ImageHistoryItem = ImageHistoryRecord['images'][number]

const MAX_REFERENCE_FILE_SIZE = 20 * 1024 * 1024
const IMAGE_GENERATION_BATCH_SIZE = 4
const IMAGE_PREVIEW_MIN_ZOOM = 0.5
const IMAGE_PREVIEW_MAX_ZOOM = 4
const IMAGE_PREVIEW_ZOOM_STEP = 0.25
const IMAGE_PREVIEW_KEYBOARD_PAN_STEP = 24
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
const ADVANCED_MODE_STORAGE_KEY = 'image_playground_advanced_mode_enabled'

const readAdvancedModeEnabled = (): boolean => {
  if (typeof window === 'undefined') {
    return false
  }

  return window.localStorage.getItem(ADVANCED_MODE_STORAGE_KEY) === 'true'
}

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
const streamEnabled = ref(false)
const advancedEnabled = ref(readAdvancedModeEnabled())
const advancedRequestPath = ref('/v1/images/generations')
const advancedRequestBodyText = ref('{}')
const advancedRequestError = ref('')
const advancedRequestDirty = ref(false)
const lastAdvancedRequest = ref<AdvancedRequestState | null>(null)
const lastAdvancedResponse = ref<unknown>(null)
const lastAdvancedError = ref<unknown>(null)
const generationProgressActive = ref(false)
const generationProgressPercent = ref<number | null>(null)
const generationProgressText = ref('')
const previewZoom = ref(1)
const previewOffset = ref<ImagePreviewOffset>({ x: 0, y: 0 })
const previewDragState = ref<ImagePreviewDragState | null>(null)
let streamAbortController: AbortController | null = null
let syncingAdvancedRequest = false
let skipNextSelectedKeyModelSync = false

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

const hasOversizeFile = computed(() => referenceItems.value.some((item) => item.tooLarge))
const referenceImagesForSubmit = computed(() =>
  referenceItems.value.filter((item) => !item.tooLarge).map((item) => item.file)
)
const streamSupported = computed(() => {
  return (
    model.value.trim().toLowerCase().startsWith('gpt-image-') &&
    referenceImagesForSubmit.value.length === 0 &&
    !hasOversizeFile.value
  )
})
const effectiveStreamEnabled = computed(() => streamEnabled.value && streamSupported.value)
const invalidCount = computed(() => !Number.isSafeInteger(count.value) || count.value < 1)
const generationProgressRounded = computed(() => {
  if (generationProgressPercent.value == null) {
    return null
  }

  return Math.round(clampPercent(generationProgressPercent.value))
})
const generationProgressValueText = computed(() => {
  if (generationProgressRounded.value != null) {
    return `${generationProgressRounded.value}%`
  }

  return generationProgressText.value || t('imagePlayground.generationProgressIndeterminate')
})
const generationProgressBarStyle = computed(() => {
  if (generationProgressRounded.value == null) {
    return {}
  }

  return { width: `${generationProgressRounded.value}%` }
})
const previewZoomPercent = computed(() => `${Math.round(previewZoom.value * 100)}%`)
const previewImageTransform = computed(
  () => `translate(${formatPreviewTransformNumber(previewOffset.value.x)}px, ${formatPreviewTransformNumber(previewOffset.value.y)}px) scale(${formatPreviewTransformNumber(previewZoom.value)})`
)
const generateDisabled = computed(() => {
  return (
    !selectedKey.value ||
    model.value.trim().length === 0 ||
    prompt.value.trim().length === 0 ||
    generating.value ||
    invalidCompression.value ||
    invalidCount.value ||
    hasOversizeFile.value ||
    (advancedEnabled.value && Boolean(advancedRequestError.value))
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

  if (skipNextSelectedKeyModelSync) {
    skipNextSelectedKeyModelSync = false
    return
  }

  model.value = nextKey.default_model || nextKey.models[0] || fallbackModels.value[0] || ''
})

watch(advancedEnabled, (enabled) => {
  if (typeof window !== 'undefined') {
    window.localStorage.setItem(ADVANCED_MODE_STORAGE_KEY, String(enabled))
  }

  if (enabled && !advancedRequestDirty.value) {
    syncAdvancedRequestFromForm()
  }
})

watch(
  [selectedKeyId, model, prompt, size, quality, outputFormat, compressionInput, moderation, count, streamEnabled, referenceItems],
  () => {
    if (syncingAdvancedRequest || !advancedEnabled.value || advancedRequestDirty.value) {
      return
    }
    syncAdvancedRequestFromForm()
  },
  { deep: true }
)

const handleAdvancedRequestPathInput = (event: Event): void => {
  if (syncingAdvancedRequest || !advancedEnabled.value) {
    return
  }
  advancedRequestPath.value = (event.target as HTMLInputElement).value
  advancedRequestDirty.value = true
  refreshAdvancedValidation()
}

const handleAdvancedRequestBodyInput = (event: Event): void => {
  if (syncingAdvancedRequest || !advancedEnabled.value) {
    return
  }
  advancedRequestBodyText.value = (event.target as HTMLTextAreaElement).value
  advancedRequestDirty.value = true
  refreshAdvancedValidation()
}

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

const formatPreviewTransformNumber = (value: number): string => {
  if (!Number.isFinite(value)) {
    return '0'
  }

  return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
}

const resetImagePreviewTransform = (): void => {
  previewZoom.value = 1
  previewOffset.value = { x: 0, y: 0 }
  previewDragState.value = null
}

const setImagePreviewZoom = (nextZoom: number): void => {
  previewZoom.value = Math.min(IMAGE_PREVIEW_MAX_ZOOM, Math.max(IMAGE_PREVIEW_MIN_ZOOM, nextZoom))
}

const zoomImagePreviewIn = (): void => {
  setImagePreviewZoom(previewZoom.value + IMAGE_PREVIEW_ZOOM_STEP)
}

const zoomImagePreviewOut = (): void => {
  setImagePreviewZoom(previewZoom.value - IMAGE_PREVIEW_ZOOM_STEP)
}

const startImagePreviewPan = (event: PointerEvent): void => {
  previewDragState.value = {
    pointerId: event.pointerId,
    startX: event.clientX,
    startY: event.clientY,
    originX: previewOffset.value.x,
    originY: previewOffset.value.y,
  }
  ;(event.currentTarget as HTMLElement | null)?.setPointerCapture?.(event.pointerId)
}

const moveImagePreviewPan = (event: PointerEvent): void => {
  const dragState = previewDragState.value
  if (!dragState || dragState.pointerId !== event.pointerId) {
    return
  }

  previewOffset.value = {
    x: dragState.originX + event.clientX - dragState.startX,
    y: dragState.originY + event.clientY - dragState.startY,
  }
}

const endImagePreviewPan = (event: PointerEvent): void => {
  if (previewDragState.value?.pointerId === event.pointerId) {
    previewDragState.value = null
    ;(event.currentTarget as HTMLElement | null)?.releasePointerCapture?.(event.pointerId)
  }
}

const handleImagePreviewWheel = (event: WheelEvent): void => {
  setImagePreviewZoom(previewZoom.value + (event.deltaY < 0 ? IMAGE_PREVIEW_ZOOM_STEP : -IMAGE_PREVIEW_ZOOM_STEP))
}

const handleImagePreviewKeydown = (event: KeyboardEvent): void => {
  if (event.key === '+' || event.key === '=') {
    event.preventDefault()
    zoomImagePreviewIn()
    return
  }
  if (event.key === '-') {
    event.preventDefault()
    zoomImagePreviewOut()
    return
  }
  if (event.key === '0') {
    event.preventDefault()
    resetImagePreviewTransform()
    return
  }
  if (event.key === 'Escape') {
    event.preventDefault()
    closeImagePreview()
    return
  }
  const deltaByKey: Record<string, ImagePreviewOffset> = {
    ArrowLeft: { x: IMAGE_PREVIEW_KEYBOARD_PAN_STEP, y: 0 },
    ArrowRight: { x: -IMAGE_PREVIEW_KEYBOARD_PAN_STEP, y: 0 },
    ArrowUp: { x: 0, y: IMAGE_PREVIEW_KEYBOARD_PAN_STEP },
    ArrowDown: { x: 0, y: -IMAGE_PREVIEW_KEYBOARD_PAN_STEP },
  }
  const delta = deltaByKey[event.key]
  if (!delta) {
    return
  }
  event.preventDefault()
  previewOffset.value = {
    x: previewOffset.value.x + delta.x,
    y: previewOffset.value.y + delta.y,
  }
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

  resetImagePreviewTransform()
  imagePreview.value = {
    source,
    alt: generatedResultAlt(index),
    title: currentPrompt.value || t('imagePlayground.generatedImageAlt'),
    meta: formattedCurrentPrice.value,
    filename: generatedResultFilename(result, index),
  }
}

const openHistoryImagePreview = (record: ImageHistoryRecord, image: ImageHistoryItem, index: number): void => {
  resetImagePreviewTransform()
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
  resetImagePreviewTransform()
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

const generationInputFromSnapshot = (snapshot: GenerationSnapshot, countOverride = snapshot.count): ImagePlaygroundGenerateInput => ({
  api_key_id: snapshot.apiKeyId,
  model: snapshot.model,
  prompt: snapshot.prompt,
  size: snapshot.size,
  quality: snapshot.quality,
  output_format: snapshot.outputFormat,
  output_compression: snapshot.outputCompression,
  moderation: snapshot.moderation,
  n: countOverride,
  reference_images: snapshot.referenceImages,
})

const advancedPathFromSnapshot = (snapshot: GenerationSnapshot): string => {
  return snapshot.referenceImages.length > 0 ? '/v1/images/edits' : '/v1/images/generations'
}

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

const prettyJSON = (value: unknown): string => {
  if (value == null) {
    return t('imagePlayground.advancedNoData')
  }

  return JSON.stringify(value, null, 2)
}

const buildAdvancedRequestBody = (snapshot: GenerationSnapshot, base: Record<string, unknown> = {}): Record<string, unknown> => {
  const input = generationInputFromSnapshot(snapshot)
  const nextBody: Record<string, unknown> = {
    ...base,
    ...imageGeneratePayload(input),
    stream: effectiveStreamEnabled.value,
  }

  if (typeof input.output_compression !== 'number') {
    delete nextBody.output_compression
  }

  return nextBody
}

const parseAdvancedRequestBodyObject = (): Record<string, unknown> | null => {
  try {
    const parsed = JSON.parse(advancedRequestBodyText.value)
    return isRecord(parsed) ? parsed : null
  } catch {
    return null
  }
}

const syncAdvancedRequestFromForm = (): void => {
  const snapshot = createGenerationSnapshot()
  if (!snapshot) {
    advancedRequestBodyText.value = '{}'
    advancedRequestPath.value = '/v1/images/generations'
    return
  }

  const base = parseAdvancedRequestBodyObject() ?? {}
  advancedRequestPath.value = advancedPathFromSnapshot(snapshot)
  advancedRequestBodyText.value = prettyJSON(buildAdvancedRequestBody(snapshot, base))
  advancedRequestError.value = ''
  advancedRequestDirty.value = false
}

const fieldString = (body: Record<string, unknown>, key: string): string | null => {
  const value = body[key]
  return typeof value === 'string' ? value : null
}

const fieldInteger = (body: Record<string, unknown>, key: string): number | null => {
  const value = body[key]
  if (typeof value === 'number' && Number.isSafeInteger(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isSafeInteger(parsed) ? parsed : null
  }
  return null
}

const validateAdvancedRequest = (): Record<string, unknown> | null => {
  const path = advancedRequestPath.value.trim()
  if (path !== '/v1/images/generations' && path !== '/v1/images/edits') {
    advancedRequestError.value = t('imagePlayground.advancedInvalidPath')
    return null
  }

  if (path === '/v1/images/edits' && referenceImagesForSubmit.value.length === 0) {
    advancedRequestError.value = t('imagePlayground.advancedEditRequiresReference')
    return null
  }

  if (path === '/v1/images/generations' && referenceImagesForSubmit.value.length > 0) {
    advancedRequestError.value = t('imagePlayground.advancedGenerationRejectsReference')
    return null
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(advancedRequestBodyText.value)
  } catch {
    advancedRequestError.value = t('imagePlayground.advancedInvalidJSON')
    return null
  }

  if (!isRecord(parsed)) {
    advancedRequestError.value = t('imagePlayground.advancedJSONMustBeObject')
    return null
  }

  const apiKeyId = fieldInteger(parsed, 'api_key_id')
  const nextModel = fieldString(parsed, 'model')?.trim()
  const nextPrompt = fieldString(parsed, 'prompt')?.trim()
  if (!apiKeyId || !availableKeys.value.some((key) => key.id === apiKeyId)) {
    advancedRequestError.value = t('imagePlayground.advancedInvalidApiKey')
    return null
  }
  if (!nextModel) {
    advancedRequestError.value = t('imagePlayground.advancedInvalidModel')
    return null
  }
  if (!nextPrompt) {
    advancedRequestError.value = t('imagePlayground.advancedInvalidPrompt')
    return null
  }

  const nextCount = fieldInteger(parsed, 'n')
  if (!nextCount || nextCount < 1 || nextCount > 10) {
    advancedRequestError.value = t('imagePlayground.countInvalid')
    return null
  }

  if ('output_compression' in parsed) {
    const nextCompression = fieldInteger(parsed, 'output_compression')
    if (nextCompression == null || nextCompression < 0 || nextCompression > 100) {
      advancedRequestError.value = t('imagePlayground.compressionInvalid')
      return null
    }
  }

  advancedRequestError.value = ''
  return parsed
}

const applyAdvancedRequestBody = (body: Record<string, unknown>): void => {
  syncingAdvancedRequest = true
  try {
    const nextApiKeyId = fieldInteger(body, 'api_key_id')
    if (nextApiKeyId != null) {
      if (selectedKeyId.value !== nextApiKeyId) {
        skipNextSelectedKeyModelSync = true
      }
      selectedKeyId.value = nextApiKeyId
    }

    const nextModel = fieldString(body, 'model')
    if (nextModel) {
      model.value = nextModel
    }

    const nextPrompt = fieldString(body, 'prompt')
    if (nextPrompt != null) {
      prompt.value = nextPrompt
    }

    const nextSize = fieldString(body, 'size')
    if (nextSize != null) {
      size.value = nextSize
    }

    const nextQuality = fieldString(body, 'quality')
    if (nextQuality != null) {
      quality.value = nextQuality
    }

    const nextModeration = fieldString(body, 'moderation')
    if (nextModeration != null) {
      moderation.value = nextModeration
    }

    const nextCount = fieldInteger(body, 'n')
    if (nextCount != null) {
      count.value = nextCount
    }

    const nextFormat = fieldString(body, 'output_format')
    const nextCompression = fieldInteger(body, 'output_compression')
    if (nextFormat != null) {
      applyOutputSettings(nextFormat, nextCompression ?? undefined)
    }

    if (typeof body.stream === 'boolean') {
      streamEnabled.value = body.stream
    }

    if (Array.isArray(body.reference_images)) {
      referenceItems.value = body.reference_images
        .filter((item): item is File => item instanceof File)
        .map((file) => ({
          id: createReferenceId(file),
          file,
          tooLarge: file.size > MAX_REFERENCE_FILE_SIZE,
        }))
    }

    if (typeof body.path === 'string' && body.path.trim()) {
      advancedRequestPath.value = body.path.trim()
    }
  } finally {
    syncingAdvancedRequest = false
  }
}

const refreshAdvancedValidation = (): void => {
  if (!advancedEnabled.value || !advancedRequestDirty.value) {
    return
  }

  const body = validateAdvancedRequest()
  if (body) {
    applyAdvancedRequestBody(body)
    advancedRequestDirty.value = false
    syncAdvancedRequestFromForm()
  }
}

const formatAdvancedRequestBody = (): void => {
  const body = validateAdvancedRequest()
  if (!body) {
    return
  }

  advancedRequestBodyText.value = prettyJSON(body)
}

const currentAdvancedRequest = (snapshot: GenerationSnapshot): AdvancedRequestState => {
  const base = parseAdvancedRequestBodyObject() ?? {}
  const body = advancedRequestDirty.value ? validateAdvancedRequest() : buildAdvancedRequestBody(snapshot, base)
  return {
    path: advancedRequestPath.value.trim() || advancedPathFromSnapshot(snapshot),
    body: body ?? buildAdvancedRequestBody(snapshot, base),
    multipart: snapshot.referenceImages.length > 0,
  }
}

const advancedBodyForBatch = (body: Record<string, unknown>, batchCount: number): Record<string, unknown> => ({
  ...body,
  n: batchCount,
})

const advancedErrorPayload = (error: unknown): unknown => {
  if (error && typeof error === 'object') {
    const record = error as Record<string, any>
    return record.response?.data ?? record.error ?? record
  }

  return error
}

const prettyAdvancedRequest = computed(() => prettyJSON(lastAdvancedRequest.value))
const prettyAdvancedResponse = computed(() => prettyJSON(lastAdvancedError.value ?? lastAdvancedResponse.value))

const generationBatchCounts = (totalCount: number): number[] => {
  const counts: number[] = []
  let remaining = totalCount

  while (remaining > 0) {
    const batchCount = Math.min(remaining, IMAGE_GENERATION_BATCH_SIZE)
    counts.push(batchCount)
    remaining -= batchCount
  }

  return counts
}

const mergeNumberMetadataField = (
  metadata: ImagePlaygroundCostMetadata[],
  key: keyof ImagePlaygroundCostMetadata
): number | undefined => {
  const values = metadata
    .map((item) => item[key])
    .filter((value): value is number => typeof value === 'number' && !Number.isNaN(value))

  return values.length > 0 ? values.reduce((sum, value) => sum + value, 0) : undefined
}

const mergeImagePlaygroundCostMetadata = (
  responses: ImagePlaygroundGenerateResponse[]
): ImagePlaygroundCostMetadata | undefined => {
  const metadata = responses
    .map((response) => response._sub2api_image_playground)
    .filter((item): item is ImagePlaygroundCostMetadata => Boolean(item))

  if (metadata.length === 0) {
    return undefined
  }

  const merged: ImagePlaygroundCostMetadata = {}
  const estimatedPrice = mergeNumberMetadataField(metadata, 'estimated_price')
  const actualCost = mergeNumberMetadataField(metadata, 'actual_cost')
  const totalCost = mergeNumberMetadataField(metadata, 'total_cost')
  const imageCount = mergeNumberMetadataField(metadata, 'image_count')
  if (estimatedPrice != null) merged.estimated_price = estimatedPrice
  if (actualCost != null) merged.actual_cost = actualCost
  if (totalCost != null) merged.total_cost = totalCost
  if (imageCount != null) merged.image_count = imageCount

  const imageSize = metadata.find((item) => typeof item.image_size === 'string' && item.image_size.trim())?.image_size
  const billingMode = metadata.find((item) => typeof item.billing_mode === 'string' && item.billing_mode.trim())?.billing_mode
  if (imageSize) merged.image_size = imageSize
  if (billingMode) merged.billing_mode = billingMode

  return Object.keys(merged).length > 0 ? merged : undefined
}

const mergeImageGenerateResponses = (
  responses: ImagePlaygroundGenerateResponse[]
): ImagePlaygroundGenerateResponse => ({
  data: responses.flatMap((response) => response.data ?? []),
  _sub2api_image_playground: mergeImagePlaygroundCostMetadata(responses),
})

const clampPercent = (value: number): number => Math.min(100, Math.max(0, value))

const numberField = (record: Record<string, unknown>, key: string): number | null => {
  const value = record[key]
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value
  }
  if (typeof value === 'string' && value.trim()) {
    const parsed = Number(value)
    return Number.isFinite(parsed) ? parsed : null
  }
  return null
}

const progressPercentFromEvent = (event: ImagePlaygroundStreamEvent): number | null => {
  const record = event as Record<string, unknown>
  const directPercent = numberField(record, 'percent') ?? numberField(record, 'percentage') ?? numberField(record, 'progress_percent')
  if (directPercent != null) {
    return clampPercent(directPercent)
  }

  const progress = numberField(record, 'progress')
  if (progress != null) {
    return clampPercent(progress >= 0 && progress <= 1 ? progress * 100 : progress)
  }

  return null
}

const progressTextFromEvent = (event: ImagePlaygroundStreamEvent): string => {
  const record = event as Record<string, unknown>
  const progressText = record.progress_text
  if (typeof progressText === 'string' && progressText.trim()) {
    return progressText.trim()
  }

  return ''
}

const startGenerationProgress = (): void => {
  generationProgressActive.value = true
  generationProgressPercent.value = null
  generationProgressText.value = ''
}

const finishGenerationProgress = (): void => {
  generationProgressActive.value = false
  generationProgressPercent.value = null
  generationProgressText.value = ''
}

const updateGenerationProgress = (percent: number | null): void => {
  if (percent == null) {
    return
  }
  generationProgressPercent.value = clampPercent(percent)
}

const updateGenerationProgressFromImages = (snapshot: GenerationSnapshot): void => {
  if (snapshot.count > 0) {
    generationProgressText.value = t('imagePlayground.generationProgressImages', {
      count: currentResults.value.length,
      total: snapshot.count,
    })
  }
}

const handleStreamProgress = (event: ImagePlaygroundStreamEvent): void => {
  updateGenerationProgress(progressPercentFromEvent(event))
  const progressText = progressTextFromEvent(event)
  if (progressText && generationProgressPercent.value == null) {
    generationProgressText.value = progressText
  }

  const eventType = `${event.type ?? event.object ?? ''}`
  if (
    eventType.includes('partial_image') ||
    eventType.includes('chunk') ||
    eventType.includes('completed') ||
    eventType.includes('result')
  ) {
    showStatus(t('imagePlayground.streamingProgressStatus'))
  }
}

const appendStreamedImage = (image: ImagePlaygroundImageResult, snapshot: GenerationSnapshot): void => {
  if (!image.b64_json && !image.url) {
    return
  }

  if (!currentPrompt.value) {
    currentPrompt.value = snapshot.prompt
    currentOutputFormat.value = snapshot.outputFormat
    currentRequestId.value = globalThis.crypto?.randomUUID?.() ?? String(Date.now())
  }

  const imageKey = image.url || image.b64_json
  if (currentResults.value.some((result) => (result.url || result.b64_json) === imageKey)) {
    return
  }

  currentResults.value = [...currentResults.value, image]
  updateGenerationProgressFromImages(snapshot)
  currentPrice.value = undefined
  showStatus(t('imagePlayground.streamingProgressStatus'))
}

const handleGenerate = async (): Promise<void> => {
  if (advancedEnabled.value && advancedRequestDirty.value) {
    const body = validateAdvancedRequest()
    if (!body) {
      return
    }
    applyAdvancedRequestBody(body)
  }

  const snapshot = createGenerationSnapshot()
  if (generateDisabled.value || !snapshot) {
    return
  }

  const advancedRequest = currentAdvancedRequest(snapshot)
  lastAdvancedRequest.value = advancedRequest
  lastAdvancedResponse.value = null
  lastAdvancedError.value = null

  const useStream = effectiveStreamEnabled.value
  generating.value = true
  startGenerationProgress()
  showStatus(useStream ? t('imagePlayground.streamingStatus') : t('imagePlayground.generatingStatus'))

  if (streamAbortController) {
    streamAbortController.abort()
    streamAbortController = null
  }

  try {
    const batchCounts = generationBatchCounts(snapshot.count)
    const responses: ImagePlaygroundGenerateResponse[] = []
    if (useStream) {
      streamAbortController = new AbortController()
      currentResults.value = []
      currentPrice.value = undefined
      currentPrompt.value = snapshot.prompt
      currentOutputFormat.value = snapshot.outputFormat
      currentRequestId.value = globalThis.crypto?.randomUUID?.() ?? String(Date.now())
    }

    for (const batchCount of batchCounts) {
      const input = generationInputFromSnapshot(snapshot, batchCount)
      const response = useStream
        ? await generateImageStream(input, {
            signal: streamAbortController?.signal,
            onProgress: handleStreamProgress,
            onImage: (image) => appendStreamedImage(image, snapshot),
          })
        : advancedEnabled.value
          ? await generateImageAdvanced({
              body: advancedBodyForBatch(advancedRequest.body, batchCount),
              reference_images: snapshot.referenceImages,
            })
          : await generateImage(input)

      responses.push(response)
      if (!useStream) {
        const receivedCount = responses.reduce((sum, item) => sum + (item.data?.length ?? 0), 0)
        if (receivedCount > 0) {
          generationProgressText.value = t('imagePlayground.generationProgressImages', {
            count: receivedCount,
            total: snapshot.count,
          })
        }
      }
    }

    const response = mergeImageGenerateResponses(responses)
    lastAdvancedResponse.value = response
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
    lastAdvancedError.value = advancedErrorPayload(error)
    const upstreamMessage = extractImageGenerationErrorMessage(error)
    showError(upstreamMessage ? `${t('imagePlayground.generateFailed')}: ${upstreamMessage}` : t('imagePlayground.generateFailed'))
  } finally {
    if (useStream) {
      streamAbortController = null
    }
    finishGenerationProgress()
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

onBeforeUnmount(() => {
  if (streamAbortController) {
    streamAbortController.abort()
    streamAbortController = null
  }
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

.image-generation-progress-track {
  @apply h-2 overflow-hidden rounded-full bg-white/70 dark:bg-dark-900/70;
}

.image-generation-progress-bar {
  @apply h-full rounded-full bg-primary-500 transition-all duration-300;
}

.image-generation-progress-bar-indeterminate {
  width: 45%;
  animation: image-generation-progress-slide 1.2s ease-in-out infinite;
}

@keyframes image-generation-progress-slide {
  0% {
    transform: translateX(-120%);
  }
  50% {
    transform: translateX(60%);
  }
  100% {
    transform: translateX(240%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .image-generation-progress-bar-indeterminate {
    animation: none;
    width: 100%;
  }
}
</style>
