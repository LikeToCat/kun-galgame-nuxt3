<script setup lang="ts">
import { replaceAsideItem } from '../aside/asideItemStore'

const props = defineProps<{
  userId: number
}>()

const historyContainer = ref<HTMLDivElement | null>(null)
const messageInput = ref('')
const messages = ref<ChatMessage[]>([])
const isLoadHistoryMessageComplete = ref(false)
const isSending = ref(false)
const isShowLoader = computed(() => {
  if (isLoadHistoryMessageComplete.value) {
    return false
  }
  if (messages.value.length < 30) {
    return false
  }
  return true
})
const currentUserUid = usePersistUserStore().id
const uid = props.userId
const pageData = reactive({
  page: 1,
  limit: 30
})

const scrollToBottom = () => {
  if (historyContainer.value) {
    historyContainer.value.scrollTo({
      top: historyContainer.value.scrollHeight,
      behavior: 'smooth'
    })
  }
}

const getMessageHistory = async () => {
  const histories = await kunFetch<ChatMessage[]>('/message/chat/history', {
    method: 'GET',
    query: {
      receiverUid: uid,
      page: pageData.page,
      limit: pageData.limit
    }
  })
  return Array.isArray(histories) ? histories : ([] as ChatMessage[])
}

const sendMessage = async () => {
  if (!messageInput.value.trim()) {
    useMessage(10401, 'warn')
    return
  }
  if (messageInput.value.length > 1007) {
    useMessage(10402, 'warn')
    return
  }

  isSending.value = true
  const result = await kunFetch('/message/chat/send', {
    method: 'POST',
    body: { receiverUid: uid, content: messageInput.value }
  })
  isSending.value = false

  if (result) {
    messageInput.value = ''
    // Reload latest messages to get the new one
    pageData.page = 1
    messages.value = await getMessageHistory()
    nextTick(() => scrollToBottom())
  }
}

const handleLoadHistoryMessages = async () => {
  if (!historyContainer.value) {
    return
  }

  const previousScrollHeight = historyContainer.value.scrollHeight
  const previousScrollTop = historyContainer.value.scrollTop

  pageData.page += 1
  const histories = await getMessageHistory()

  if (histories.length > 0) {
    messages.value.unshift(...histories)

    nextTick(() => {
      if (historyContainer.value) {
        const newScrollHeight = historyContainer.value.scrollHeight
        historyContainer.value.scrollTo({
          top: previousScrollTop + (newScrollHeight - previousScrollHeight)
        })
      }
    })
  } else {
    isLoadHistoryMessageComplete.value = true
  }
}

const onKeydown = async (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    event.preventDefault()
    sendMessage()
  }
}

onMounted(async () => {
  messages.value = await getMessageHistory()
  window.addEventListener('keydown', onKeydown)

  nextTick(() => {
    scrollToBottom()
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div
    ref="historyContainer"
    class="h-[calc(100dvh-14rem)] overflow-y-auto py-3"
  >
    <div class="flex justify-center">
      <KunButton
        v-if="isShowLoader"
        @click="handleLoadHistoryMessages"
        size="sm"
        variant="light"
      >
        加载更多
      </KunButton>
    </div>

    <MessagePmMessage
      v-for="message in messages"
      :key="message.id"
      :message="message"
      :current-user-uid="currentUserUid"
    />

    <div v-if="!messages.length" class="text-default-500 py-10 text-center">
      暂无消息，发送一条消息开始聊天吧
    </div>
  </div>

  <div class="flex gap-2 border-t px-3 pt-3">
    <KunInput
      v-model="messageInput"
      placeholder="输入消息..."
      class="flex-1"
      @keydown.enter.prevent="sendMessage"
    />
    <KunButton @click="sendMessage" :loading="isSending"> 发送 </KunButton>
  </div>
</template>
