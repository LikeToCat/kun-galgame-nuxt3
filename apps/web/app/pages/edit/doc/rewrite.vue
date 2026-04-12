<script setup lang="ts">
definePageMeta({
  middleware: 'auth'
})

const route = useRoute()
const slug = computed(() => (route.query.slug as string) || '')

if (!slug.value) {
  await navigateTo('/doc')
}

const { data: article } = await useKunFetch<DocArticleDetail>(
  `/doc/article/${slug.value}`
)

useKunSeoMeta({ title: '重新编辑文档' })
</script>

<template>
  <div>
    <EditDocLayout v-if="article" mode="rewrite" :initial-article="article" />
    <KunNull v-else description="未找到对应的文档" />
  </div>
</template>
