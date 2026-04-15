<script setup lang="ts">
import { usePersistKUNGalgameAdvancedFilterStore } from '~/store/modules/galgame'

const pageData = storeToRefs(useTempGalgameStore())
const { includeProviders, excludeOnlyProviders } = storeToRefs(
  usePersistKUNGalgameAdvancedFilterStore()
)

const { data, status } = await useKunFetch(`/galgame`, {
  method: 'GET',
  query: {
    ...pageData,
    includeProviders: includeProviders.value,
    excludeOnlyProviders: excludeOnlyProviders.value
  }
})
</script>

<template>
  <div v-if="data" class="flex flex-col gap-3">
    <KunCard class-name="z-10" :is-hoverable="false" :is-transparent="false">
      <KunHeader name="Galgame 资源 Wiki">
        <template #endContent>
          <GalgameCardNav :is-show-advanced="true" />
        </template>

        <template #description>
          <p class="text-default-500">
            Galgame 资源页面, 提供各类 Galgame 下载。我们不是资源的提供者,
            我们只是资源的指路人。
            <GalgameCardNSFWHint />
          </p>
        </template>
      </KunHeader>
    </KunCard>

    <KunLoading :loading="status === 'pending'">
      <GalgameCard v-if="data.galgames" :galgames="data.galgames" />
    </KunLoading>

    <KunCard
      :is-hoverable="false"
      :is-transparent="false"
      content-class="gap-3"
    >
      <KunPagination
        v-model:current-page="pageData.page.value"
        :total-page="Math.ceil(data.total / pageData.limit.value)"
        :is-loading="status === 'pending'"
      />
    </KunCard>

    <KunAdDZMMBanner />
  </div>
</template>
