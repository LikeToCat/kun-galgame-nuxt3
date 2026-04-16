<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_TYPE_MAP,
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_SORT_FIELD_MAP
} from '~/constants/galgame'
import {
  KUN_GALGAME_PROVIDER_LABEL_MAP,
  PROVIDER_KEY_OPTIONS,
  type ProviderKey
} from '~/constants/galgameResource'
import { usePersistKUNGalgameAdvancedFilterStore } from '~/store/modules/galgame'
import type {
  KunGalgameResourceTypeOptions,
  KunGalgameResourceLanguageOptions,
  KunGalgameResourcePlatformOptions
} from '~/constants/galgame'

withDefaults(
  defineProps<{
    isShowAdvanced?: boolean
  }>(),
  { isShowAdvanced: false }
)

const { page, type, language, platform, sortField, sortOrder } = storeToRefs(
  useTempGalgameStore()
)

const advStore = usePersistKUNGalgameAdvancedFilterStore()
const { includeProviders, excludeOnlyProviders } = storeToRefs(advStore)
const showAdvanced = ref(false)

watch(
  () => [
    type.value,
    language.value,
    platform.value,
    sortField.value,
    sortOrder.value,
    includeProviders.value.join(','),
    excludeOnlyProviders.value.join(',')
  ],
  () => {
    page.value = 1
  }
)

const typeOptions = Object.entries(KUN_GALGAME_RESOURCE_TYPE_MAP)
  .filter(([k]) => k !== 'name')
  .map(([value, label]) => ({ value, label }))

const langOptions = Object.entries(KUN_GALGAME_RESOURCE_LANGUAGE_MAP).map(
  ([value, label]) => ({ value, label })
)

const platformOptions = Object.entries(KUN_GALGAME_RESOURCE_PLATFORM_MAP)
  .filter(([k]) => k !== 'name')
  .map(([value, label]) => ({ value, label }))

const sortOptions = Object.entries(KUN_GALGAME_RESOURCE_SORT_FIELD_MAP).map(
  ([value, label]) => ({
    value: value === 'views' ? 'view' : value,
    label
  })
)
</script>

<template>
  <div class="space-y-1">
    <KunScrollShadow>
      <button
        v-for="opt in typeOptions"
        :key="opt.value"
        class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
        :class="
          type === opt.value
            ? 'bg-primary/15 text-primary font-medium'
            : 'text-default-600 hover:bg-default-100'
        "
        @click="type = opt.value as KunGalgameResourceTypeOptions"
      >
        {{ opt.label }}
      </button>
    </KunScrollShadow>

    <KunScrollShadow>
      <button
        v-for="opt in langOptions"
        :key="opt.value"
        class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
        :class="
          language === opt.value
            ? 'bg-primary/15 text-primary font-medium'
            : 'text-default-600 hover:bg-default-100'
        "
        @click="language = opt.value as KunGalgameResourceLanguageOptions"
      >
        {{ opt.label }}
      </button>
    </KunScrollShadow>

    <KunScrollShadow>
      <button
        v-for="opt in platformOptions"
        :key="opt.value"
        class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
        :class="
          platform === opt.value
            ? 'bg-primary/15 text-primary font-medium'
            : 'text-default-600 hover:bg-default-100'
        "
        @click="platform = opt.value as KunGalgameResourcePlatformOptions"
      >
        {{ opt.label }}
      </button>
    </KunScrollShadow>

    <KunScrollShadow>
      <button
        v-for="opt in sortOptions"
        :key="opt.value"
        class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
        :class="
          sortField === opt.value
            ? 'bg-primary/15 text-primary font-medium'
            : 'text-default-600 hover:bg-default-100'
        "
        @click="sortField = opt.value as 'time' | 'view' | 'created'"
      >
        {{ opt.label }}
      </button>
    </KunScrollShadow>

    <div class="flex items-center gap-1.5">
      <button
        class="cursor-pointer rounded-md p-1 transition-colors"
        :class="
          sortOrder === 'desc'
            ? 'bg-primary/15 text-primary'
            : 'text-default-500 hover:bg-default-100'
        "
        @click="sortOrder = 'desc'"
      >
        <KunIcon name="lucide:arrow-down" />
      </button>
      <button
        class="cursor-pointer rounded-md p-1 transition-colors"
        :class="
          sortOrder === 'asc'
            ? 'bg-primary/15 text-primary'
            : 'text-default-500 hover:bg-default-100'
        "
        @click="sortOrder = 'asc'"
      >
        <KunIcon name="lucide:arrow-up" />
      </button>

      <button
        v-if="isShowAdvanced"
        class="text-default-500 hover:text-primary flex cursor-pointer items-center gap-1 rounded-md px-2 py-1 text-sm transition-colors"
        :class="
          (includeProviders.length || excludeOnlyProviders.length) &&
          'text-warning'
        "
        @click="showAdvanced = !showAdvanced"
      >
        <KunIcon name="lucide:filter" class="text-inherit" />
        <span>网盘筛选</span>
      </button>
    </div>

    <div
      v-if="showAdvanced"
      class="bg-default-50 space-y-3 rounded-lg border p-3"
    >
      <div>
        <div class="text-default-700 mb-1.5 text-xs font-medium">
          必须含有以下网盘
        </div>
        <KunScrollShadow>
          <button
            v-for="key in PROVIDER_KEY_OPTIONS"
            :key="key"
            class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
            :class="
              includeProviders.includes(key)
                ? 'bg-primary/15 text-primary font-medium'
                : 'text-default-600 hover:bg-default-100'
            "
            @click="advStore.toggleIncludeProvider(key as ProviderKey)"
          >
            {{ KUN_GALGAME_PROVIDER_LABEL_MAP[key as ProviderKey] }}
          </button>
        </KunScrollShadow>
      </div>

      <div>
        <div class="text-default-700 mb-1.5 text-xs font-medium">
          排除仅含以下网盘
        </div>
        <KunScrollShadow>
          <button
            v-for="key in PROVIDER_KEY_OPTIONS"
            :key="key + '-ex'"
            class="cursor-pointer rounded-md px-2.5 py-1 text-sm whitespace-nowrap transition-colors"
            :class="
              excludeOnlyProviders.includes(key)
                ? 'bg-danger/15 text-danger font-medium'
                : 'text-default-600 hover:bg-default-100'
            "
            @click="advStore.toggleExcludeOnlyProvider(key as ProviderKey)"
          >
            {{ KUN_GALGAME_PROVIDER_LABEL_MAP[key as ProviderKey] }}
          </button>
        </KunScrollShadow>
      </div>
    </div>
  </div>
</template>
