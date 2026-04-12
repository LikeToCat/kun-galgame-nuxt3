<script setup lang="ts">
const props = defineProps<{
  metadata: DocArticleDetail
}>()

const metadata = computed(() => props.metadata)
const { role } = usePersistUserStore()
const isDeleting = ref(false)

const handleEdit = async () => {
  if (!metadata.value) {
    return
  }
  await navigateTo({
    path: '/edit/doc/rewrite',
    query: { slug: metadata.value.slug }
  })
}

const handleDelete = async () => {
  if (!metadata.value || isDeleting.value) {
    return
  }
  const confirmed = await useComponentMessageStore().alert(
    '确认删除这篇文档吗？',
    '删除操作不可恢复，请慎重。',
    true
  )
  if (!confirmed) {
    return
  }

  isDeleting.value = true
  try {
    await kunFetch('/doc/article', {
      method: 'DELETE',
      params: {
        articleId: metadata.value.id
      }
    })
    useMessage('删除文档成功', 'success')
    await navigateTo('/doc')
  } finally {
    isDeleting.value = false
  }
}
</script>

<template>
  <KunCard :is-hoverable="false" class-name="border-none">
    <div class="relative mb-6 aspect-video h-full w-full">
      <KunImage
        :alt="metadata.title"
        class="size-full rounded-lg object-cover"
        :src="metadata.banner || '/kungalgame.webp'"
        width="100%"
        height="100%"
        data-kun-lazy-image
      />
    </div>

    <div class="flex flex-col gap-3">
      <h1 class="text-2xl font-bold tracking-tight sm:text-4xl">
        {{ metadata.title }}
      </h1>

      <div class="flex flex-wrap items-center gap-3 text-sm">
        <!-- TODO: category is now category_id (flat), need separate fetch for title -->
        <KunBadge color="secondary">
          {{ `分类 #${metadata.category_id}` }}
        </KunBadge>
        <div class="text-default-500 flex items-center gap-1">
          <KunIcon name="lucide:eye" class="h-4 w-4" />
          <span>{{ metadata.view }} 次浏览</span>
        </div>

        <div v-if="role > 2" class="flex flex-wrap items-center gap-2">
          <KunButton
            size="sm"
            variant="flat"
            color="primary"
            @click="handleEdit"
          >
            编辑
          </KunButton>
          <KunButton
            size="sm"
            variant="flat"
            color="danger"
            :loading="isDeleting"
            :disabled="isDeleting"
            @click="handleDelete"
          >
            删除
          </KunButton>
        </div>
      </div>

      <!-- TODO: author is now author_id (flat), need separate fetch for user info -->
      <div class="flex items-center gap-3">
        <div class="flex flex-col gap-1">
          <div class="text-default-500 flex items-center gap-2">
            <KunIcon name="lucide:calendar-days" />
            <p class="text-small text-inherit">
              {{
                formatDate(metadata.published_time, {
                  isPrecise: true,
                  isShowYear: true
                })
              }}
            </p>
          </div>
        </div>
      </div>

      <div class="bg-primary/10 text-primary-700 rounded-lg p-3 text-sm">
        {{ metadata.description }}
      </div>
    </div>
  </KunCard>
</template>
