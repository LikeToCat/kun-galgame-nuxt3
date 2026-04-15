<script setup lang="ts">
const route = useRoute()

const [{ data: categoryResponse }, { data: articleResponse }] =
  await Promise.all([
    useKunFetch<DocCategoryListResponse>('/doc/category', {
      query: { page: 1, limit: 100 }
    }),
    useKunFetch<DocArticleListResponse>('/doc/article', {
      query: {
        page: 1,
        limit: 100,
        orderBy: 'published_time',
        sortOrder: 'desc'
      }
    })
  ])

const expandedCategories = ref<Record<number, boolean>>({})

const categories = computed(() => categoryResponse.value?.items || [])
const articles = computed(() => articleResponse.value?.items || [])

watch(
  categories,
  (list) => {
    list.forEach((category) => {
      if (typeof expandedCategories.value[category.id] === 'undefined') {
        expandedCategories.value[category.id] = true
      }
    })
  },
  { immediate: true }
)

const articlesByCategory = computed<Record<number, DocArticleSummary[]>>(() => {
  const grouped: Record<number, DocArticleSummary[]> = {}
  articles.value.forEach((article) => {
    const categoryId = article.category.id
    if (!grouped[categoryId]) {
      grouped[categoryId] = []
    }
    grouped[categoryId]!.push(article)
  })
  return grouped
})

const toggleCategory = (categoryId: number) => {
  expandedCategories.value[categoryId] = !expandedCategories.value[categoryId]
}
</script>

<template>
  <div class="fixed hidden shrink-0 space-y-1 lg:w-64 xl:block">
    <h3 class="p-3 text-xl font-semibold">文档索引</h3>
    <div class="scrollbar-hide max-h-[calc(100dvh-10rem)] overflow-y-auto">
      <div class="space-y-1" v-for="category in categories" :key="category.id">
        <KunButton
          :full-width="true"
          variant="light"
          size="lg"
          @click="toggleCategory(category.id)"
          class-name="justify-between mb-2"
        >
          <span class="text-foreground">
            {{ category.title }}
          </span>
          <KunIcon
            :name="
              expandedCategories[category.id]
                ? 'lucide:chevron-down'
                : 'lucide:chevron-right'
            "
          />
        </KunButton>

        <div v-if="expandedCategories[category.id]" class="ml-4 space-y-1">
          <KunButton
            :full-width="true"
            :variant="route.fullPath === article.path ? 'flat' : 'light'"
            v-for="article in articlesByCategory[category.id] || []"
            :key="article.id"
            :href="article.path"
            class-name="justify-start text-start"
          >
            <span
              :class="
                cn(
                  'gap-2',
                  route.fullPath === article.path ? '' : 'text-foreground'
                )
              "
            >
              {{ article.title }}
            </span>
          </KunButton>
        </div>
      </div>
    </div>
  </div>
</template>
