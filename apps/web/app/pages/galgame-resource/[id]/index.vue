<script setup lang="ts">
import {
  KUN_GALGAME_RESOURCE_LANGUAGE_MAP,
  KUN_GALGAME_RESOURCE_PLATFORM_MAP,
  KUN_GALGAME_RESOURCE_TYPE_MAP
} from '~/constants/galgame'

const route = useRoute()
const resourceId = computed(() => Number((route.params as { id: string }).id))

const { data, refresh } = await useKunFetch(
  `/galgame-resource/${resourceId.value}`,
  {
    query: { resourceId }
  }
)

if (data.value && data.value !== 'not found') {
  const titleBase = getPreferredLanguageText(data.value.galgame.name)
  if (data.value.galgame.contentLimit === 'nsfw') {
    useKunDisableSeo(titleBase)
  }

  const resource = data.value.resource

  const typeLabel =
    KUN_GALGAME_RESOURCE_TYPE_MAP[resource.type] || resource.type
  const languageLabel =
    KUN_GALGAME_RESOURCE_LANGUAGE_MAP[resource.language] || resource.language
  const platformLabel =
    KUN_GALGAME_RESOURCE_PLATFORM_MAP[resource.platform] || resource.platform

  const description = `${typeLabel} · ${languageLabel} · ${platformLabel} · ${resource.size}`

  useKunSeoMeta({
    title: `${titleBase} ${typeLabel}资源下载`,
    description: data.value.resource.note
      ? data.value.resource.note
      : description,
    ogImage: data.value.galgame.banner
  })
} else {
  useKunDisableSeo('未找到 Galgame 资源')
}
</script>

<template>
  <div v-if="data" class="space-y-3">
    <template v-if="data !== 'not found'">
      <GalgameResourceDetailHero :galgame="data.galgame" />

      <KunAdDZMMBanner class-name="hidden lg:block" />

      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <GalgameResourceDetailPanel
          class="lg:col-span-2"
          :galgame="data.galgame"
          :resource="data.resource"
          :refresh="refresh"
        />

        <GalgameResourceDetailRecommendations
          :recommendations="data.recommendations"
        />
      </div>
    </template>

    <KunNull
      v-else
      description="未找到对应的 Galgame 资源，或该资源已被移除。"
    />
  </div>
</template>
