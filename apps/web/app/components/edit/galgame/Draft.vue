<script setup lang="ts">
// Draft editor for the user's own pending (status=3) or declined
// (status=4) galgame submission. Backed by PATCH /api/galgame/:gid.
//
// Distinct from:
//   - Galgame.vue (the "create new" form, POST /galgame/submit) which
//     uses usePersistEditGalgameStore for draft autosave;
//   - PullRequest.vue (PR form for published galgames, POST /:gid/prs).
//
// We reuse useTempGalgamePRStore as the form-state container so that
// EditGalgameEditor (its 'rewrite' branch reads from
// galgamePR[0].introduction) just works without a third store type.
// The store is session-scoped (persist:false) so leftover draft state
// from a previous edit doesn't bleed into a fresh open.
//
// Wiki PATCH semantics (see docs/galgame_wiki/07-submission.md):
//   - All fields optional; only supplied fields are merged.
//   - status=4 (declined) automatically flips back to status=3 (pending)
//     on a successful patch.

import { languageItems } from '~/constants/edit'
import { patchDraftSchema } from '~/validations/galgame'

interface WikiDraftDetail {
  id: number
  vndb_id: string
  name_en_us: string
  name_ja_jp: string
  name_zh_cn: string
  name_zh_tw: string
  banner: string
  banner_image_hash: string
  intro_en_us: string
  intro_ja_jp: string
  intro_zh_cn: string
  intro_zh_tw: string
  content_limit: 'sfw' | 'nsfw'
  age_limit: 'all' | 'r18'
  original_language: string
  status: number
  user_id: number
  created: string
  updated: string
}

interface WikiDraftEnvelope {
  galgame: WikiDraftDetail
}

const route = useRoute()
const gid = computed(() => Number(route.params.gid))

const { data, status, error } = await useKunFetch<WikiDraftEnvelope>(
  `/galgame/${gid.value}`
)

const introductionLanguage = ref<Language>('zh-cn')

// Hydrate the temp PR store from the fetched draft so EditGalgameEditor's
// 'rewrite' branch picks up the right markdown source.
const { galgamePR } = storeToRefs(useTempGalgamePRStore())
const isMinor = ref(false)
const originalLanguageLocal = ref<Language>('ja-jp')

// Wiki returns `original_language` as a free string. Map back to the
// Language union; anything else collapses to the ja-jp default rather
// than crashing the KunSelect binding.
const isOriginalLanguage = (s: string): s is Language => {
  return s === 'ja-jp' || s === 'en-us' || s === 'zh-cn' || s === 'zh-tw'
}

// Mirrors Meta.vue's option shape so the wire-format keeps aligned with
// what the create flow emits. Kept inline (rather than reusing Meta.vue)
// because Meta.vue binds to usePersistEditGalgameStore and we want draft
// state isolated from the create-flow's autosaved draft.
const contentLimitOptions = [
  { value: 'sfw', label: 'SFW (本游戏不含成人内容)' },
  { value: 'nsfw', label: 'NSFW (本游戏含成人内容)' }
] as const

const ageLimitOptions = [
  { value: 'all', label: '全年龄' },
  { value: 'r18', label: 'R18' }
] as const

const originalLanguageOptions = [
  { value: 'ja-jp', label: '日语' },
  { value: 'en-us', label: '英语' },
  { value: 'zh-cn', label: '简体中文' },
  { value: 'zh-tw', label: '繁体中文' }
] as const

watch(
  data,
  (val) => {
    if (!val?.galgame) {
      return
    }
    const g = val.galgame
    galgamePR.value = [
      {
        id: g.id,
        vndbId: g.vndb_id,
        name: {
          'en-us': g.name_en_us,
          'ja-jp': g.name_ja_jp,
          'zh-cn': g.name_zh_cn,
          'zh-tw': g.name_zh_tw
        },
        introduction: {
          'en-us': g.intro_en_us,
          'ja-jp': g.intro_ja_jp,
          'zh-cn': g.intro_zh_cn,
          'zh-tw': g.intro_zh_tw
        },
        contentLimit: g.content_limit,
        ageLimit: g.age_limit,
        originalLanguage: isOriginalLanguage(g.original_language)
          ? g.original_language
          : 'ja-jp',
        alias: []
      }
    ]
    originalLanguageLocal.value = isOriginalLanguage(g.original_language)
      ? g.original_language
      : 'ja-jp'
  },
  { immediate: true }
)

const draft = computed(() => galgamePR.value[0])

const statusBadge = computed(() => {
  const s = data.value?.galgame.status
  switch (s) {
    case GalgameStatus.Pending:
      return { label: '待审核', color: 'warning' as const }
    case GalgameStatus.Declined:
      return { label: '已拒绝', color: 'danger' as const }
    case GalgameStatus.Published:
      return { label: '已发布', color: 'success' as const }
    case GalgameStatus.Banned:
      return { label: '已封禁', color: 'default' as const }
    default:
      return { label: '未知', color: 'default' as const }
  }
})

const isEditable = computed(() => {
  const s = data.value?.galgame.status
  return s === GalgameStatus.Pending || s === GalgameStatus.Declined
})

const isSaving = ref(false)

const handleSave = async () => {
  const cur = draft.value
  if (!cur) return

  // Always include every editable field — wiki merges so this is safe
  // even when most are unchanged, and it sidesteps the "did the user
  // intend an empty string or a no-op?" ambiguity. is_minor stays opt-in.
  const payload = {
    vndb_id: cur.vndbId,
    name_en_us: cur.name['en-us'],
    name_ja_jp: cur.name['ja-jp'],
    name_zh_cn: cur.name['zh-cn'],
    name_zh_tw: cur.name['zh-tw'],
    intro_en_us: cur.introduction['en-us'],
    intro_ja_jp: cur.introduction['ja-jp'],
    intro_zh_cn: cur.introduction['zh-cn'],
    intro_zh_tw: cur.introduction['zh-tw'],
    content_limit: cur.contentLimit,
    age_limit: cur.ageLimit,
    original_language: originalLanguageLocal.value,
    aliases: cur.alias.join(','),
    is_minor: isMinor.value
  }

  const parsed = patchDraftSchema.safeParse(payload)
  if (!parsed.success) {
    const message = JSON.parse(parsed.error.message)[0]
    useMessage(
      `位置: ${message.path[0]} - 错误提示: ${message.message}`,
      'warn'
    )
    return
  }

  const ok = await useComponentMessageStore().alert(
    '保存修改?',
    '保存后, 已拒绝的草稿会自动重新进入审核队列。'
  )
  if (!ok) {
    return
  }

  isSaving.value = true
  const res = await kunFetch<WikiDraftDetail>(`/galgame/${gid.value}`, {
    method: 'PATCH',
    body: payload
  })
  isSaving.value = false

  if (res) {
    useMessage('已保存, 等待审核', 'success')
    await navigateTo('/edit/galgame/mine')
  }
}
</script>

<template>
  <KunCard
    :is-hoverable="false"
    :is-pressable="false"
    :is-transparent="false"
    content-class="space-y-6"
  >
    <KunHeader name="编辑 Galgame 草稿" description="保存后将进入审核队列。">
      <template #endContent>
        <div class="flex items-center gap-3">
          <KunBadge size="xs" variant="flat" :color="statusBadge.color">
            {{ statusBadge.label }}
          </KunBadge>
          <KunLink to="/edit/galgame/mine">
            <KunButton size="sm" variant="flat">返回我的提交</KunButton>
          </KunLink>
        </div>
      </template>
    </KunHeader>

    <KunLoading v-if="status === 'pending'" />

    <KunInfo
      v-else-if="error || !data"
      color="danger"
      title="无法加载草稿"
      description="该草稿可能不存在或您无权编辑。"
    />

    <KunInfo
      v-else-if="!isEditable"
      color="warning"
      title="当前状态不可编辑"
      description="已发布的条目请通过 PR 流程修改; 已封禁的条目请联系管理员。"
    />

    <template v-if="isEditable && draft">
      <KunDivider>
        <span class="mx-2">基本信息</span>
      </KunDivider>

      <div class="space-y-2">
        <h2 class="text-xl">VNDB 编号 (可选)</h2>
        <KunInput
          v-model="draft.vndbId"
          placeholder="例如: v19658 (无 VNDB 可留空)"
        />
      </div>

      <div class="space-y-2">
        <h2 class="text-xl">游戏名</h2>
        <KunInput placeholder="英语" v-model="draft.name['en-us']" />
        <KunInput placeholder="日语" v-model="draft.name['ja-jp']" />
        <KunInput placeholder="简体中文" v-model="draft.name['zh-cn']" />
        <KunInput placeholder="繁体中文" v-model="draft.name['zh-tw']" />
      </div>

      <div class="space-y-2">
        <h2 class="text-xl">介绍</h2>
        <KunTab
          :items="languageItems"
          v-model="introductionLanguage"
          variant="underlined"
          color="primary"
          size="sm"
        />
        <EditGalgameEditor :lang="introductionLanguage" type="rewrite" />
      </div>

      <KunDivider>
        <span class="mx-2">其它</span>
      </KunDivider>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <KunSelect
          v-model="draft.contentLimit"
          label="内容限制"
          :options="contentLimitOptions"
        />
        <KunSelect
          v-model="draft.ageLimit"
          label="年龄分级"
          :options="ageLimitOptions"
        />
        <KunSelect
          v-model="originalLanguageLocal"
          label="原始语言"
          :options="originalLanguageOptions"
        />
      </div>

      <KunSwitch
        v-model="isMinor"
        label="小修改 (不更改实际内容, 仅订正)"
      />

      <div class="flex justify-end">
        <KunButton
          :disabled="isSaving"
          :loading="isSaving"
          size="lg"
          @click="handleSave"
        >
          保存修改
        </KunButton>
      </div>
    </template>
  </KunCard>
</template>
