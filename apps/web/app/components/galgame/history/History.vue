<script setup lang="ts">
const KUN_REVISION_ACTION_MAP: Record<string, string> = {
  created: '创建',
  updated: '编辑',
  merged: '合并 PR',
  reverted: '回滚',
  declined: '拒绝 PR'
}

const route = useRoute()
const gid = computed(() => {
  return parseInt((route.params as { gid: string }).gid)
})

const pageData = reactive({
  page: 1,
  limit: 10,
  galgameId: gid.value
})

const { data, status } = await useKunFetch<{
  items: GalgameRevision[]
  total: number
}>(`/galgame/${gid.value}/history/all`, {
  lazy: true,
  method: 'GET',
  query: pageData
})
</script>

<template>
  <div class="flex flex-col space-y-3" v-if="data">
    <KunHeader
      name="版本历史"
      description="这里记录了这个 Galgame 项目发生的所有更改历史"
      scale="h3"
    />

    <KunLoading v-if="status === 'pending'" />

    <div
      class="flex items-center gap-2 text-sm"
      v-for="(rev, index) in data.items"
      :key="index"
    >
      <KunAvatar :user="rev.user" />

      <div class="space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <span>{{ rev.user.name }}</span>
          <KunBadge size="sm">
            {{ KUN_REVISION_ACTION_MAP[rev.action] || rev.action }}
          </KunBadge>
          <span
            v-if="rev.isMinor"
            class="text-default-400 text-xs"
          >
            (小修改)
          </span>
          <span class="text-default-500">
            {{ formatTimeDifference(rev.created) }}
          </span>
        </div>

        <div class="text-default-500" v-if="rev.note">
          {{ rev.note }}
        </div>
      </div>
    </div>

    <KunPagination
      v-if="data.total >= pageData.limit"
      v-model:current-page="pageData.page"
      :total-page="Math.ceil(data.total / pageData.limit)"
      :is-loading="status === 'pending'"
    />
  </div>
</template>
