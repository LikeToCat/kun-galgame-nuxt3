<script setup lang="ts">
import { KUN_DOC_CATEGORY_MAP, KUN_DOC_STATUS_OPTIONS } from '~/constants/doc'
import { useDocEditorContext } from './context'
import { normalizeDocSlug } from '~/utils/doc'
import type { KunSelectOption } from '~/components/kun/select/type'

const { form, categories, tags, refreshTags, readingMinute } =
  useDocEditorContext()

const categoryOptions = computed<KunSelectOption[]>(() =>
  categories.value.map((category) => ({
    label:
      KUN_DOC_CATEGORY_MAP[category.slug] ||
      `${category.title} (${category.slug})`,
    value: category.id
  }))
)

const statusOptions = computed<KunSelectOption[]>(() =>
  KUN_DOC_STATUS_OPTIONS.map((option) => ({
    label: option.label,
    value: option.value
  }))
)

const isTagSelected = (tagId: number) => form.tagIds.includes(tagId)

const toggleTag = (tagId: number) => {
  if (isTagSelected(tagId)) {
    form.tagIds = form.tagIds.filter((id) => id !== tagId)
  } else {
    form.tagIds = Array.from(new Set([...form.tagIds, tagId]))
  }
}

const normalizeSlug = () => {
  form.slug = normalizeDocSlug(form.slug)
}

const newTagTitle = ref('')
const newTagSlug = ref('')
const newTagDescription = ref('')
const isCreatingTag = ref(false)
const isSlugEdited = ref(false)

watch(newTagTitle, (value) => {
  if (!isSlugEdited.value) {
    newTagSlug.value = normalizeDocSlug(value)
  }
})

const handleSlugInput = () => {
  isSlugEdited.value = true
  newTagSlug.value = normalizeDocSlug(newTagSlug.value)
}

const resetNewTagForm = () => {
  newTagTitle.value = ''
  newTagSlug.value = ''
  newTagDescription.value = ''
  isSlugEdited.value = false
}

const handleCreateTag = async () => {
  if (isCreatingTag.value) {
    return
  }

  const title = newTagTitle.value.trim()
  if (!title) {
    useMessage('请输入标签名称', 'warn')
    return
  }

  const slug = normalizeDocSlug(newTagSlug.value || title)
  if (!slug) {
    useMessage('请输入合法的标签 Slug', 'warn')
    return
  }

  isCreatingTag.value = true
  try {
    const created = await kunFetch<DocTagItem>('/doc/tag', {
      method: 'POST',
      body: {
        slug,
        title,
        description: newTagDescription.value.trim()
      }
    })

    if (created) {
      tags.value = [
        created,
        ...tags.value.filter((tag) => tag.id !== created.id)
      ]
      form.tagIds = Array.from(new Set([...form.tagIds, created.id]))
      useMessage('创建标签成功', 'success')
      resetNewTagForm()
      await refreshTags()
    }
  } finally {
    isCreatingTag.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h3 class="text-lg font-semibold">基础信息</h3>
      <p class="text-default-500 text-sm">
        slug 会自动拼接为 /doc/[slug] 作为访问路径，无需手动填写。
      </p>
    </div>

    <div class="space-y-4">
      <KunInput
        v-model="form.slug"
        label="Slug"
        placeholder="请输入唯一的文档 slug"
        maxlength="233"
        required
        @blur="normalizeSlug"
      />

      <KunInput
        v-model="form.banner"
        label="封面地址"
        placeholder="https://example.com/banner.webp"
        maxlength="777"
      />

      <KunTextarea
        v-model="form.description"
        label="文档简介"
        placeholder="请输入用于展示的简介"
        :rows="4"
        auto-grow
        :maxlength="777"
        required
        show-char-count
      />
    </div>

    <div class="space-y-4">
      <KunSelect
        v-model="form.categoryId"
        :options="categoryOptions"
        label="文档分类"
        placeholder="请选择分类"
      />

      <KunSelect
        v-model="form.status"
        :options="statusOptions"
        label="发布状态"
        placeholder="请选择状态"
      />

      <div
        class="border-default-200 flex flex-wrap items-center justify-between gap-3 rounded-lg border px-4 py-2"
      >
        <div>
          <p class="text-sm font-medium">预计阅读时长</p>
          <p class="text-default-500 text-xs">根据当前正文实时计算</p>
        </div>
        <span class="text-primary font-semibold">
          {{ readingMinute }} 分钟
        </span>
      </div>

      <div
        class="border-default-200 flex items-center justify-between rounded-lg border px-4 py-2"
      >
        <div>
          <p class="text-sm font-medium">首页置顶</p>
          <p class="text-default-500 text-xs">开启后会在首页轮播中固定展示</p>
        </div>
        <KunSwitch v-model="form.isPin" color="primary" />
      </div>
    </div>

    <div class="space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h4 class="font-semibold">文档标签</h4>
          <p class="text-default-500 text-xs">点击任意标签即可切换选中状态。</p>
        </div>
        <KunBadge color="secondary" variant="flat">
          共 {{ tags.length }} 个标签
        </KunBadge>
      </div>

      <div
        class="scrollbar-hide border-default-200 max-h-64 overflow-y-auto rounded-lg border border-dashed p-3"
      >
        <div class="flex flex-wrap gap-2">
          <KunButton
            v-for="tag in tags"
            :key="tag.id"
            size="sm"
            rounded="full"
            :variant="isTagSelected(tag.id) ? 'solid' : 'light'"
            :color="isTagSelected(tag.id) ? 'primary' : 'default'"
            @click="toggleTag(tag.id)"
          >
            {{ tag.title }}
          </KunButton>
        </div>
      </div>

      <div
        v-if="form.tagIds.length"
        class="text-default-500 grid grid-cols-1 gap-2 text-sm"
      >
        <span>已选择：</span>
        <div class="flex flex-wrap gap-2">
          <KunBadge
            v-for="tagId in form.tagIds"
            :key="tagId"
            color="secondary"
            variant="flat"
          >
            {{
              tags.find((tag) => tag.id === tagId)?.title || `标签 #${tagId}`
            }}
          </KunBadge>
        </div>
      </div>

      <div class="border-default-200 space-y-3 rounded-lg border p-4">
        <div>
          <h5 class="text-sm font-semibold">创建新标签</h5>
          <p class="text-default-500 text-xs">
            无需离开当前页面即可补充缺失的标签，方便后续复用。
          </p>
        </div>
        <KunInput
          v-model="newTagTitle"
          label="标签名称"
          placeholder="请输入标签名称"
          maxlength="128"
          required
        />
        <KunInput
          v-model="newTagSlug"
          label="标签 Slug"
          placeholder="请输入标签 slug"
          maxlength="233"
          @input="handleSlugInput"
        />
        <KunTextarea
          v-model="newTagDescription"
          label="标签描述"
          placeholder="可选补充说明"
          :rows="2"
          :maxlength="255"
          auto-grow
        />
        <KunButton
          color="primary"
          :loading="isCreatingTag"
          :disabled="isCreatingTag"
          @click="handleCreateTag"
        >
          创建标签
        </KunButton>
      </div>
    </div>
  </div>
</template>
