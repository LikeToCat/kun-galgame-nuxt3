<script setup lang="ts">
const route = useRoute()

const docSlug = computed(() => (route.params.slug as string) || '')

const { images, isLightboxOpen, currentImageIndex } = useKunLightbox()

const { data } = await useKunFetch<DocArticleDetail>(
  `/doc/article/${docSlug.value}`
)

useKunSeoMeta({
  title: data.value?.title,
  description: data.value?.description,
  ogImage: data.value?.banner,
  ogType: 'article',
  articleAuthor: [`${kungal.domain.main}/user/${data.value?.author.id}/info`],
  articlePublishedTime: data.value?.publishedTime.toString(),
  articleModifiedTime: data.value?.editedTime?.toString()
})
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    v-if="data"
    class-name="backdrop-blur-none pb-6 min-h-[calc(100dvh-6rem)]"
  >
    <div class="flex">
      <DocDetailCategoryTree />

      <article class="flex-1 space-y-6 pl-0 lg:pr-67 xl:pl-67">
        <KunLightbox
          :images="images"
          v-model:is-open="isLightboxOpen"
          :initial-index="currentImageIndex"
        />

        <DocDetailHeader :metadata="data" />
        <KunContent :content="data.contentHtml" />
        <DocDetailFooter />
      </article>

      <div v-if="data.toc.length" class="hidden lg:block">
        <div class="fixed -translate-x-67">
          <DocDetailTableOfContent :links="data.toc" />
        </div>
      </div>
    </div>
  </KunCard>
</template>
