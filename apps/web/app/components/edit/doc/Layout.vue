<script setup lang="ts">
import { provideDocEditorContext } from './context'
import type { DocEditorMode, DocEditorForm } from './type'
import { computeReadingMinute } from '~/utils/doc'
const props = withDefaults(
  defineProps<{
    mode: DocEditorMode
    initialArticle?: DocArticleDetail | null
  }>(),
  {
    initialArticle: null
  }
)

const isRewriteMode = computed(() => props.mode === 'rewrite')

const { data: categoryResponse } = await useKunFetch<DocCategoryListResponse>(
  '/doc/category',
  {
    query: {
      page: 1,
      limit: 100,
      keyword: ''
    }
  }
)

const { data: tagResponse, refresh: refreshTagResponse } =
  await useKunFetch<DocTagListResponse>('/doc/tag', {
    query: {
      page: 1,
      limit: 100,
      keyword: ''
    }
  })

const categories = ref<DocCategoryItem[]>([])
const tags = ref<DocTagItem[]>([])

watch(
  categoryResponse,
  (response) => {
    categories.value = response?.categories ?? []
  },
  { immediate: true }
)

watch(
  tagResponse,
  (response) => {
    tags.value = response?.tags ?? []
  },
  { immediate: true }
)

const createDefaultForm = (): DocEditorForm => ({
  articleId: null,
  title: '',
  slug: '',
  description: '',
  banner: '',
  status: 1,
  isPin: false,
  contentMarkdown: '',
  categoryId: 0,
  tagIds: []
})

const form = reactive<DocEditorForm>(createDefaultForm())
const isSubmitting = ref(false)
const readingMinute = computed(() =>
  form.contentMarkdown.trim() ? computeReadingMinute(form.contentMarkdown) : 0
)

const applyArticleToForm = (article: DocArticleDetail) => {
  form.articleId = article.id
  form.title = article.title
  form.slug = article.slug
  form.description = article.description
  form.banner = article.banner || ''
  form.status = article.status
  form.isPin = article.isPin
  form.contentMarkdown = article.contentMarkdown
  form.categoryId = article.category.id
  form.tagIds = article.tags.map((tag) => tag.id)
}

const resetForm = () => {
  if (isRewriteMode.value && props.initialArticle) {
    applyArticleToForm(props.initialArticle)
    return
  }

  Object.assign(form, createDefaultForm())
}

if (isRewriteMode.value && props.initialArticle) {
  applyArticleToForm(props.initialArticle)
}

watch(
  () => props.initialArticle,
  (article) => {
    if (isRewriteMode.value && article) {
      applyArticleToForm(article)
    }
  }
)

const validateForm = () => {
  if (!form.title.trim()) {
    return '请输入标题'
  }
  if (!form.slug.trim()) {
    return '请输入 slug'
  }
  if (!form.description.trim()) {
    return '请输入简介'
  }
  if (!form.contentMarkdown.trim()) {
    return '请输入正文内容'
  }
  if (!form.categoryId) {
    return '请选择文档分类'
  }
  return true
}

const handleSubmit = async () => {
  if (isSubmitting.value) {
    return
  }

  const validation = validateForm()
  if (validation !== true) {
    useMessage(validation, 'warn')
    return
  }

  if (isRewriteMode.value && !form.articleId) {
    useMessage('未找到文档 ID，无法更新', 'error')
    return
  }

  isSubmitting.value = true
  try {
    const normalizedSlug = form.slug.trim()
    const body: Record<string, unknown> = {
      title: form.title.trim(),
      slug: normalizedSlug,
      description: form.description.trim(),
      banner: form.banner.trim(),
      status: form.status,
      isPin: form.isPin,
      contentMarkdown: form.contentMarkdown,
      categoryId: form.categoryId as number,
      tagIds: Array.from(new Set(form.tagIds))
    }

    if (isRewriteMode.value) {
      body.articleId = form.articleId
    }

    const result = await kunFetch<DocArticleDetail>('/doc/article', {
      method: isRewriteMode.value ? 'PUT' : 'POST',
      body
    })

    if (result) {
      useMessage(
        isRewriteMode.value ? '更新文档成功' : '创建文档成功',
        'success'
      )
      applyArticleToForm(result)
      await navigateTo(result.path)
    }
  } finally {
    isSubmitting.value = false
  }
}

const refreshTags = async () => {
  await refreshTagResponse()
}

provideDocEditorContext({
  form,
  categories,
  tags,
  mode: props.mode,
  isSubmitting,
  handleSubmit,
  resetForm,
  refreshTags,
  readingMinute
})
</script>

<template>
  <div class="contents">
    <ClientOnly>
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <KunCard
          :is-hoverable="false"
          :is-pressable="false"
          :is-transparent="false"
          class-name="lg:col-span-1 order-2 sm:order-1"
          content-class="space-y-6"
        >
          <EditDocMetadataForm />
          <EditDocSubmitActions />
        </KunCard>

        <div class="order-1 space-y-3 sm:order-2 lg:col-span-2">
          <KunCard
            :is-hoverable="false"
            :is-pressable="false"
            :is-transparent="false"
          >
            <EditDocTitle />
          </KunCard>
          <KunCard
            :is-hoverable="false"
            :is-pressable="false"
            :is-transparent="false"
          >
            <EditDocContentEditor />
          </KunCard>
        </div>
      </div>
    </ClientOnly>
  </div>
</template>
