<script setup lang="ts">
const props = defineProps<{
  topicId?: number
  replyId?: number
  targetUserId: number
  likeCount: number
  isLiked: boolean
}>()

const { id } = usePersistUserStore()
// Trust the backend — it returns isLiked=false for anonymous requests. The
// previous `id && ...` guard clobbered highlight state during the brief
// window before the persisted store hydrates (id = 0).
const isLiked = ref(props.isLiked)
const likeCount = ref(props.likeCount)

const toggleLike = async () => {
  const result = props.topicId
    ? await kunFetch<string>(`/topic/${props.topicId}/like`, {
        method: 'PUT',
        body: { topicId: props.topicId }
      })
    : // Reply-scoped variant: the path needs a valid :tid. Using
      // `props.topicId` here produced `/topic/undefined/reply/like` —
      // the backend ignored the path param so it worked by accident,
      // but the URL is misleading in logs/traces. Fall back to 0 so
      // the path is always a number.
      await kunFetch<string>(`/topic/0/reply/like`, {
        method: 'PUT',
        body: { replyId: props.replyId }
      })

  if (!result) return

  likeCount.value += isLiked.value ? -1 : 1
  useMessage(isLiked.value ? 10234 : 10233, 'success')
  isLiked.value = !isLiked.value
}

const handleClickLikeThrottled = throttle(toggleLike, 1007, () =>
  useMessage(10227, 'warn')
)

const handleClickLike = () => {
  if (!id) {
    useMessage(10235, 'warn', 5000)
    return
  }
  if (id === props.targetUserId) {
    useMessage(10236, 'warn')
    return
  }
  handleClickLikeThrottled()
}
</script>

<template>
  <KunTooltip text="点赞">
    <KunButton
      :is-icon-only="true"
      :variant="isLiked ? 'flat' : 'light'"
      :color="isLiked ? 'secondary' : 'default'"
      :size="likeCount ? 'md' : 'lg'"
      class-name="gap-1"
      @click="handleClickLike"
    >
      <KunIcon name="lucide:thumbs-up" />
      <span v-if="likeCount">{{ likeCount }}</span>
    </KunButton>
  </KunTooltip>
</template>
