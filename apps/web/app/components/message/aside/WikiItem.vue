<script setup lang="ts">
// Wiki notification aside entry — separate component from SystemItem
// because the data source is different: this one talks to wiki-managed
// /galgame/messages/mine (and the kungal-local /galgame/messages/read-state
// for unread counts), whereas SystemItem feeds off /message/nav/system.
//
// See docs/galgame_wiki/08-messages.md for the upstream message shape and
// the rationale for keeping per-consumer read state local rather than in
// the wiki itself (the same message can surface in kungal/moyu/admin UI
// with independent read state).

interface WikiMessageItem {
  id: number
  type: string
  created_at: string
}

interface WikiMessagesEnvelope {
  items: WikiMessageItem[]
  total: number
}

interface ReadStateResp {
  last_read_message_id: number
}

// Pull the top message + the read marker. limit=1 keeps it cheap; we
// only need the latest id to derive `hasUnread` and a small preview.
// Lazy + client-only — aside lives in a layout, the SSR pass shouldn't
// pay for a wiki round-trip per request.
const { data: feed } = useKunFetch<WikiMessagesEnvelope>(
  '/galgame/messages/mine',
  {
    query: { since_id: 0, limit: 1 },
    server: false,
    lazy: true
  }
)
const { data: readState } = useKunFetch<ReadStateResp>(
  '/galgame/messages/read-state',
  { server: false, lazy: true }
)

const latest = computed(() => feed.value?.items?.[0])
const lastReadId = computed(() => readState.value?.last_read_message_id ?? 0)
const hasUnread = computed(() => {
  if (!latest.value) return false
  return latest.value.id > lastReadId.value
})

const typeLabel = (t: string | undefined) => {
  switch (t) {
    case 'approved':
      return '审核通过'
    case 'declined':
      return '审核拒绝'
    case 'banned':
      return '已封禁'
    case 'unbanned':
      return '已恢复'
    default:
      return t ?? ''
  }
}
</script>

<template>
  <KunLink
    color="default"
    underline="none"
    class-name="hover:bg-primary/20 flex cursor-pointer flex-nowrap gap-3 rounded-lg p-2 transition-colors hover:opacity-80"
    to="/message/wiki"
  >
    <KunImage src="/apple-touch-icon.png" class="h-12 w-12 rounded-full" />
    <div class="justify-space flex w-full flex-col">
      <div class="flex items-center justify-between">
        <span class="font-bold">Wiki 通知</span>
        <span class="text-default-500 text-sm" v-if="latest">
          {{ formatTimeDifference(latest.created_at) }}
        </span>
      </div>

      <div class="flex items-center justify-between text-sm">
        <span v-if="!hasUnread" class="zako">杂鱼~♡</span>
        <span v-if="hasUnread" class="new">{{ `「 新消息 」` }}</span>
        <span class="line-clamp-1 break-all">
          {{ latest ? typeLabel(latest.type) : '暂无审核反馈' }}
        </span>
        <KunBadge
          class-name="whitespace-nowrap"
          color="primary"
          v-if="hasUnread"
        >
          新
        </KunBadge>
      </div>
    </div>
  </KunLink>
</template>
