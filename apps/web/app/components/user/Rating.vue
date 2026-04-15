<script setup lang="ts">
const props = defineProps<{
  uid: number
}>()

const pageData = reactive({
  page: 1,
  limit: 24,
  userId: props.uid
})

const { data, status } = await useKunFetch<{
  ratingData: GalgameRatingCard[]
  total: number
}>(`/user/${props.uid}/ratings`, { query: pageData })
</script>

<template>
  <div class="space-y-3">
    <KunHeader
      name="Galgame 评分"
      description="这是您的 Galgame 评分列表, 您可以在这里查看您发布的所有 Galgame 评分"
    />

    <div v-if="data && data.ratingData.length" class="space-y-3">
      <GalgameRatingCard :ratings="data.ratingData" />

      <KunPagination
        v-if="data.total > pageData.limit"
        v-model:current-page="pageData.page"
        :total-page="Math.ceil(data.total / pageData.limit)"
        :is-loading="status === 'pending'"
      />
    </div>

    <KunNull v-if="data && !data.ratingData.length" description="暂无评分" />
  </div>
</template>
