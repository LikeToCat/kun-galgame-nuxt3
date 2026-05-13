<script setup lang="ts">
// Publish wizard — the first stop in the "发布 Galgame" flow. Goal is to
// keep duplicate submissions out of the moderation queue by surfacing
// existing records before the user fills out a full form.
//
// Three resolutions, in priority order:
//   1. VNDB ID matched (lookup via /galgame/check) → could be any status.
//      The wiki always knows about a vndb id by integer galgame_id.
//   2. Name search → wiki returns status=0 hits in `items` and the
//      caller's own status=3/4 in `pending` (Bearer-aware, see
//      docs/galgame_wiki/07-submission.md §GET /galgame/search 增量参数).
//   3. None of the above → "create new" CTA → /edit/galgame/create form.
//
// Per-status action:
//   status=0 published   → 前往发布资源（直接跳详情页）
//   status=2 VNDB 草稿    → 认领并发布 (POST /:gid/claim, +3 萌萌点)
//   status=3 自己 pending → 继续编辑 (/draft/:gid)
//   status=4 自己 declined → 查看并修改 (/draft/:gid)

interface SearchHit {
  id: number
  vndb_id?: string
  name_zh_cn?: string
  name_ja_jp?: string
  name_en_us?: string
  name_zh_tw?: string
  banner?: string
  banner_image_hash?: string
  status?: number
}

interface WizardSearchResp {
  items: SearchHit[]
  pending?: SearchHit[]
  total: number
}

interface CheckResp {
  exists: boolean
  galgame_id?: number
}

interface DetailGalgame {
  id: number
  vndb_id: string
  status: number
  name_en_us: string
  name_ja_jp: string
  name_zh_cn: string
  name_zh_tw: string
  banner: string
  banner_image_hash: string
}

interface DetailEnvelope {
  galgame: DetailGalgame
}

const q = ref('')
const vndbInput = ref('')

const isSearching = ref(false)
const isVndbLooking = ref(false)
const searchResults = ref<WizardSearchResp | null>(null)

// vndbHit is the "resolved by VNDB ID" result. Separate from the
// text-search results because it's a precise hit and renders distinctly.
const vndbHit = ref<DetailGalgame | null>(null)
const vndbMissed = ref(false)

// Search hits and detail responses arrive in snake_case (wiki wire
// format); shim into the KunLanguage hyphen shape on demand so we can
// reuse getPreferredLanguageText with the same locale priority the rest
// of the site uses.
const nameOfHit = (h: SearchHit): string =>
  getPreferredLanguageText({
    'en-us': h.name_en_us ?? '',
    'ja-jp': h.name_ja_jp ?? '',
    'zh-cn': h.name_zh_cn ?? '',
    'zh-tw': h.name_zh_tw ?? ''
  }) ||
  (h.vndb_id ? `VNDB ${h.vndb_id}` : `#${h.id}`)

const nameOfDetail = (g: DetailGalgame): string =>
  getPreferredLanguageText({
    'en-us': g.name_en_us,
    'ja-jp': g.name_ja_jp,
    'zh-cn': g.name_zh_cn,
    'zh-tw': g.name_zh_tw
  }) ||
  (g.vndb_id ? `VNDB ${g.vndb_id}` : `#${g.id}`)

const statusBadge = (s: number | undefined) => {
  switch (s) {
    case GalgameStatus.Published:
      return { label: '已发布', color: 'success' as const }
    case GalgameStatus.VndbDraft:
      return { label: 'VNDB 草稿', color: 'primary' as const }
    case GalgameStatus.Pending:
      return { label: '待审核', color: 'warning' as const }
    case GalgameStatus.Declined:
      return { label: '已拒绝', color: 'danger' as const }
    case GalgameStatus.Banned:
      return { label: '已封禁', color: 'default' as const }
    default:
      return { label: '已发布', color: 'success' as const }
  }
}

const handleSearch = async () => {
  if (!q.value.trim()) {
    useMessage('请先输入关键词', 'warn')
    return
  }
  isSearching.value = true
  // /galgame/search/wizard already forces include_pending=true server-side;
  // we just pass the query. The Bearer is attached automatically by the
  // session middleware so wiki resolves the caller's pending list.
  const res = await kunFetch<WizardSearchResp>('/galgame/search/wizard', {
    method: 'GET',
    query: { q: q.value.trim(), limit: 12 }
  })
  isSearching.value = false
  searchResults.value = res
}

const handleVndbLookup = async () => {
  vndbHit.value = null
  vndbMissed.value = false
  const vndbId = vndbInput.value.trim()
  if (!vndbId) {
    useMessage('请先输入 VNDB ID', 'warn')
    return
  }
  if (!VNDBPattern.test(vndbId)) {
    useMessage('非法的 VNDB ID 格式 (例如 v19658)', 'warn')
    return
  }
  isVndbLooking.value = true
  const check = await kunFetch<CheckResp>('/galgame/check', {
    method: 'GET',
    query: { vndb_id: vndbId }
  })
  if (!check?.exists || !check.galgame_id) {
    isVndbLooking.value = false
    vndbMissed.value = true
    return
  }
  // VNDB id matched — pull the full detail so we know the current status
  // and can label the action button correctly.
  const detail = await kunFetch<DetailEnvelope>(`/galgame/${check.galgame_id}`, {
    method: 'GET'
  })
  isVndbLooking.value = false
  if (detail?.galgame) {
    vndbHit.value = detail.galgame
  }
}

const isClaiming = ref(false)

const handleClaim = async (gid: number) => {
  const ok = await useComponentMessageStore().alert(
    '认领此 VNDB 草稿吗?',
    '认领后该条目立即变为已发布状态, 您将成为该 Galgame 的创建者, 并获得 +3 萌萌点。'
  )
  if (!ok) return

  isClaiming.value = true
  // POST /galgame/:gid/claim — wiki flips status 2→0 and adds the
  // claimer as contributor; kungal awards moemoepoint in a local tx.
  const result = await kunFetch<{ id: number }>(`/galgame/${gid}/claim`, {
    method: 'POST',
    body: {}
  })
  isClaiming.value = false
  if (result?.id) {
    useKunLoliInfo('认领成功, 已发布', 5)
    await navigateTo(`/galgame/${result.id}`)
  }
}

// Pre-populate the create form's persisted store with whatever the user
// typed in the wizard, then jump to the form. Avoids forcing them to
// re-type. The store is the same one Galgame.vue reads from.
const handleCreateNew = async () => {
  const store = usePersistEditGalgameStore()
  if (vndbInput.value && VNDBPattern.test(vndbInput.value)) {
    store.vndbId = vndbInput.value
  }
  // Best-effort: stash the query in the zh-cn name slot so the user sees
  // their own search term carried over. They can edit before submitting.
  if (q.value.trim() && !store.name['zh-cn']) {
    store.name['zh-cn'] = q.value.trim()
  }
  await navigateTo('/edit/galgame/create')
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-pressable="false"
    :is-transparent="false"
    content-class="space-y-6"
  >
    <KunHeader
      name="发布 Galgame"
      description="请先搜索您想发布的 Galgame, 已存在的条目可直接跳转或认领, 系统中没有的再走「新建申请」流程, 减少重复提交。"
    >
      <template #endContent>
        <KunLink to="/edit/galgame/mine">
          <KunButton size="sm" variant="flat">我的提交</KunButton>
        </KunLink>
      </template>
    </KunHeader>

    <KunDivider>
      <span class="mx-2">按名称搜索</span>
    </KunDivider>

    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <KunInput
          v-model="q"
          placeholder="输入游戏名 (任意语言)"
          @keydown.enter="handleSearch"
        />
        <KunButton
          class-name="whitespace-nowrap"
          :loading="isSearching"
          @click="handleSearch"
        >
          搜索
        </KunButton>
      </div>
      <p class="text-default-500 text-sm">
        搜索覆盖已发布的 Galgame, 同时会显示您自己的待审核 / 已拒绝草稿。
      </p>
    </div>

    <div v-if="searchResults" class="space-y-4">
      <!-- pending: own status=3/4 hits, shown first as the most actionable -->
      <div
        v-if="searchResults.pending && searchResults.pending.length"
        class="space-y-2"
      >
        <h3 class="text-default-700 text-sm font-bold">您的待审 / 已拒草稿</h3>
        <div
          v-for="hit in searchResults.pending"
          :key="`pending-${hit.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">{{ nameOfHit(hit) }}</h4>
              <KunBadge
                size="xs"
                variant="flat"
                :color="statusBadge(hit.status).color"
              >
                {{ statusBadge(hit.status).label }}
              </KunBadge>
            </div>
            <p class="text-default-500 text-sm">
              VNDB: {{ hit.vndb_id || '—' }}
            </p>
          </div>
          <KunLink :to="`/edit/galgame/draft/${hit.id}`">
            <KunButton size="sm" variant="flat">继续编辑</KunButton>
          </KunLink>
        </div>
      </div>

      <!-- items: public status=0 hits -->
      <div v-if="searchResults.items.length" class="space-y-2">
        <h3 class="text-default-700 text-sm font-bold">已发布的 Galgame</h3>
        <div
          v-for="hit in searchResults.items"
          :key="`item-${hit.id}`"
          class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
        >
          <KunImage
            v-if="hit.banner"
            :src="hit.banner"
            loading="lazy"
            placeholder="/placeholder.webp"
            class="h-16 w-28 shrink-0 rounded object-cover"
            :style="{ aspectRatio: '16/9' }"
          />
          <div class="min-w-0 flex-1 space-y-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="truncate font-medium">{{ nameOfHit(hit) }}</h4>
              <KunBadge
                size="xs"
                variant="flat"
                :color="statusBadge(hit.status).color"
              >
                {{ statusBadge(hit.status).label }}
              </KunBadge>
            </div>
            <p class="text-default-500 text-sm">
              VNDB: {{ hit.vndb_id || '—' }}
            </p>
          </div>
          <KunLink :to="`/galgame/${hit.id}`">
            <KunButton size="sm" variant="flat">前往发布资源</KunButton>
          </KunLink>
        </div>
      </div>

      <KunInfo
        v-if="!searchResults.items.length && !searchResults.pending?.length"
        color="info"
        title="没有找到匹配的 Galgame"
        description="您可以试试 VNDB ID 精确查找, 或直接新建一个申请。"
      />
    </div>

    <KunDivider>
      <span class="mx-2">按 VNDB ID 精确查找</span>
    </KunDivider>

    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <KunInput
          v-model="vndbInput"
          placeholder="例如: v19658"
          @keydown.enter="handleVndbLookup"
        />
        <KunButton
          class-name="whitespace-nowrap"
          :loading="isVndbLooking"
          @click="handleVndbLookup"
        >
          查找
        </KunButton>
      </div>
      <p class="text-default-500 text-sm">
        若 wiki 已通过 VNDB 同步任务自动建立了草稿 (VNDB 草稿状态),
        您可以「认领」直接发布并成为该 Galgame 的创建者。
      </p>
    </div>

    <div
      v-if="vndbHit"
      class="dark:border-default-200 flex flex-col gap-3 rounded-lg border border-transparent p-3 backdrop-blur-none transition-all duration-200 sm:flex-row sm:items-center"
    >
      <KunImage
        v-if="vndbHit.banner"
        :src="vndbHit.banner"
        loading="lazy"
        placeholder="/placeholder.webp"
        class="h-16 w-28 shrink-0 rounded object-cover"
        :style="{ aspectRatio: '16/9' }"
      />
      <div class="min-w-0 flex-1 space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <h4 class="truncate font-medium">{{ nameOfDetail(vndbHit) }}</h4>
          <KunBadge
            size="xs"
            variant="flat"
            :color="statusBadge(vndbHit.status).color"
          >
            {{ statusBadge(vndbHit.status).label }}
          </KunBadge>
        </div>
        <p class="text-default-500 text-sm">VNDB: {{ vndbHit.vndb_id || '—' }}</p>
      </div>
      <KunButton
        v-if="vndbHit.status === GalgameStatus.VndbDraft"
        size="sm"
        :loading="isClaiming"
        :disabled="isClaiming"
        @click="handleClaim(vndbHit.id)"
      >
        认领并发布
      </KunButton>
      <KunLink
        v-else-if="vndbHit.status === GalgameStatus.Published"
        :to="`/galgame/${vndbHit.id}`"
      >
        <KunButton size="sm" variant="flat">前往发布资源</KunButton>
      </KunLink>
      <KunLink
        v-else-if="
          vndbHit.status === GalgameStatus.Pending ||
          vndbHit.status === GalgameStatus.Declined
        "
        :to="`/edit/galgame/draft/${vndbHit.id}`"
      >
        <KunButton size="sm" variant="flat">继续编辑</KunButton>
      </KunLink>
    </div>

    <KunInfo
      v-if="vndbMissed"
      color="warning"
      title="未找到 VNDB ID 对应的 Galgame"
      description="该 VNDB ID 当前未被收录, 您可以填写完整信息后新建申请。"
    />

    <KunDivider>
      <span class="mx-2">都不匹配?</span>
    </KunDivider>

    <KunInfo
      color="info"
      title="提交一份新的 Galgame 申请"
      description="确认 wiki 中确实没有该 Galgame 后, 点击下方按钮填写完整信息。普通用户提交后将进入审核队列。"
    />
    <div class="flex justify-end">
      <KunButton size="lg" @click="handleCreateNew">新建 Galgame 申请</KunButton>
    </div>
  </KunCard>
</template>
