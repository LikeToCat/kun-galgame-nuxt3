<script setup lang="ts">
const route = useRoute()

const { data: articleResponse } = await useKunFetch<DocArticleListResponse>(
  '/doc/article',
  {
    query: {
      page: 1,
      limit: 100,
      orderBy: 'published_time',
      sortOrder: 'desc'
    }
  }
)

const articles = computed(() => articleResponse.value?.items || [])

const currentIndex = computed(() => {
  return articles.value.findIndex((article) => article.path === route.path)
})

const prev = computed(() => {
  if (currentIndex.value > 0) {
    return articles.value[currentIndex.value - 1]
  }
  return null
})

const next = computed(() => {
  if (
    currentIndex.value !== -1 &&
    currentIndex.value < articles.value.length - 1
  ) {
    return articles.value[currentIndex.value + 1]
  }
  return null
})
</script>

<template>
  <div class="flex items-center justify-between">
    <KunButton
      class-name="mr-auto justify-start text-start gap-2"
      v-if="prev"
      color="default"
      variant="light"
      :href="prev.path"
    >
      <KunIcon name="lucide:chevron-left" />
      {{ prev.title }}
    </KunButton>
    <KunButton
      class-name="ml-auto justify-end text-end gap-2"
      color="default"
      v-if="next"
      variant="light"
      :href="next.path"
    >
      {{ next.title }}
      <KunIcon name="lucide:chevron-right" />
    </KunButton>
  </div>
</template>
