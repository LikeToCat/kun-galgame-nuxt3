<script setup lang="ts">
import {
  spoilerOptions,
  playStatusOptions,
  typeOptions,
  sortFieldOptions
} from './_sort'
const params = reactive({
  page: 1,
  limit: 24,
  sortField: 'time',
  sortOrder: 'desc',
  spoilerLevel: 'all',
  playStatus: 'all',
  galgameType: 'all'
})

const { data, status } = await useKunFetch(`/galgame-rating/all`, {
  method: 'GET',
  query: params
})
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-transparent="false"
    content-class="space-y-3"
  >
    <div class="space-y-2">
      <KunHeader name="Galgame 评分列表">
        <template #description>
          <p class="text-default-500">
            浏览所有用户对所有 Galgame 的评分与短评，支持按 Galgame 剧透等级,
            Galgame 游玩状态, Galgame 游戏类型筛选, 并按时间, Galgame 热度或
            Galgame 评分进行排序。
            <GalgameCardNSFWHint />
          </p>
        </template>
      </KunHeader>

      <div
        class="flex w-full shrink-0 flex-wrap items-center justify-between gap-3 rounded-lg border border-transparent sm:flex-nowrap"
      >
        <div class="grid w-full grid-cols-2 gap-3 lg:grid-cols-4">
          <KunSelect v-model="params.spoilerLevel" :options="spoilerOptions" />
          <KunSelect v-model="params.playStatus" :options="playStatusOptions" />
          <KunSelect v-model="params.galgameType" :options="typeOptions" />
          <KunSelect v-model="params.sortField" :options="sortFieldOptions" />
        </div>

        <div class="flex items-center gap-2">
          <KunButton
            :is-icon-only="true"
            :variant="params.sortOrder === 'desc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sortOrder = 'desc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-down" />
          </KunButton>

          <KunButton
            :is-icon-only="true"
            :variant="params.sortOrder === 'asc' ? 'flat' : 'light'"
            size="lg"
            @click="params.sortOrder = 'asc'"
          >
            <KunIcon class="text-inherit" name="lucide:arrow-up" />
          </KunButton>
        </div>
      </div>
    </div>

    <GalgameRatingCard v-if="data" :ratings="data.ratingData" />

    <KunPagination
      v-if="(data?.total || 0) > params.limit"
      v-model:current-page="params.page"
      :total-page="Math.ceil((data?.total || 0) / params.limit)"
      :is-loading="status === 'pending'"
    />
  </KunCard>
</template>
