<script setup lang="ts">
import { replaceAsideItem } from '../aside/asideItemStore'
import type { KunContextMenuItem } from '~/components/kun/context-menu/type'

const props = defineProps<{
  userId: number
}>()

const socket = useSocketIO()
useSocketIOErrorHandler()

const historyContainer = ref<HTMLDivElement | null>(null)
const messageInput = ref('')
const messages = ref<ChatMessage[]>([])
const isLoadHistoryMessageComplete = ref(false)
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
const contextMenuState = reactive({
  visible: false,
  x: 0,
  y: 0,
  message: null as ChatMessage | null
})
const contextMenuItems = computed<KunContextMenuItem[]>(() => {
  if (!contextMenuState.message || contextMenuState.message.isRecall) {
    return []
  }
  return [
    {
      key: 'recall',
      label: '撤回消息',
      icon: 'lucide:rotate-ccw',
      color: 'danger'
    }
  ]
})

const upsertChatMessage = (incoming: ChatMessage) => {
  const targetIndex = messages.value.findIndex(
    (message) => message.id === incoming.id
  )

  if (targetIndex !== -1) {
    messages.value[targetIndex] = incoming
  } else {
    messages.value.push(incoming)
  }
}

const scrollToBottom = () => {
  if (historyContainer.value) {
    historyContainer.value.scrollTo({
      top: historyContainer.value.scrollHeight,
      behavior: 'smooth'
    })
  }
}

const handleMessageSent = (newMessage: ChatMessage) => {
  upsertChatMessage(newMessage)
  replaceAsideItem(newMessage)
  messageInput.value = ''
  nextTick(() => scrollToBottom())
}

const handleMessageReceived = (msg: ChatMessage) => {
  if (msg.receiverUid === currentUserUid && msg.sender.id === uid) {
    upsertChatMessage(msg)
    replaceAsideItem(msg)
    nextTick(() => {
      scrollToBottom()
    })
  }
}

const handleMessageRecalled = (msg: ChatMessage) => {
  upsertChatMessage(msg)
  replaceAsideItem(msg)
  if (msg.sender.id === currentUserUid) {
    useMessage('撤回消息成功', 'success')
  }
}

const handleMessageRecallError = (errMsg: string) => {
  useMessage(errMsg, 'error')
}

const sendMessage = () => {
  if (!messageInput.value.trim()) {
    useMessage(10401, 'warn')
    return
  }
  if (messageInput.value.length > 1007) {
    useMessage(10402, 'warn')
    return
  }
  socket.emit('message:sending', uid, messageInput.value)
}

const getMessageHistory = async () => {
  const histories = await kunFetch<ChatMessage[]>('/message/chat/history', {
    method: 'GET',
    query: { receiverUid: uid, ...pageData }
  })
  const results = Array.isArray(histories) ? histories : [] as ChatMessage[]
  return results
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

const handleRecallMessage = async (message: ChatMessage) => {
  if (!message || message.isRecall) {
    return
  }

  const ok = await useComponentMessageStore().alert('确定撤回该消息吗？')
  if (!ok) {
    return
  }

  socket.emit('message:recall', message.id)
}

const closeContextMenu = () => {
  contextMenuState.visible = false
  contextMenuState.message = null
}

const handleMessageContextMenu = ({
  event,
  message
}: {
  event: MouseEvent
  message: ChatMessage
}) => {
  event.preventDefault()
  event.stopPropagation()
  closeContextMenu()
  contextMenuState.message = message
  contextMenuState.x = event.clientX
  contextMenuState.y = event.clientY
  contextMenuState.visible = true
}

const handleContextMenuSelect = async (item: KunContextMenuItem) => {
  if (!contextMenuState.message) {
    return
  }
  if (item.key === 'recall') {
    await handleRecallMessage(contextMenuState.message)
  }
}

watch(
  () => messages.value,
  () => scrollToBottom()
)

const onKeydown = async (event: KeyboardEvent) => {
  if (event.key === 'Enter') {
    event.preventDefault()
    sendMessage()
  }
}

onMounted(async () => {
  socket.emit('private:join', uid)

  socket.on('message:sent', handleMessageSent)
  socket.on('message:received', handleMessageReceived)
  socket.on('message:recalled', handleMessageRecalled)
  socket.on('message:recall:error', handleMessageRecallError)

  messages.value = await getMessageHistory()
  window.addEventListener('keydown', onKeydown)

  nextTick(() => {
    scrollToBottom()
  })
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  socket.emit('private:leave')
  socket.off('message:sent', handleMessageSent)
  socket.off('message:received', handleMessageReceived)
  socket.off('message:recalled', handleMessageRecalled)
  socket.off('message:recall:error', handleMessageRecallError)
})
</script>

<template>
  <div
    ref="historyContainer"
    class="h-[calc(100dvh-14rem)] overflow-y-auto py-3"
  >
    <div class="flex justify-center">
      <KunButton
        variant="light"
        v-if="isShowLoader"
        @click="handleLoadHistoryMessages"
      >
        加载历史消息
      </KunButton>
    </div>

    <KunNull v-if="!isShowLoader && messages.length > 30" />

    <div class="space-y-2" v-if="messages.length">
      <MessagePmItem
        v-for="message in messages"
        :key="message.id"
        :message="message"
        :is-sent="uid !== message.sender.id"
        @context-menu="handleMessageContextMenu"
      />
    </div>

    <KunNull v-if="!messages.length" />
  </div>

  <div class="flex justify-between gap-2">
    <KunInput size="lg" v-model="messageInput" />
    <KunButton
      size="xl"
      :is-icon-only="true"
      class-name="shrink-0"
      @click="sendMessage"
    >
      <KunIcon name="lucide:send" />
    </KunButton>
  </div>

  <KunContextMenu
    :visible="contextMenuState.visible && !!contextMenuState.message"
    :position="{ x: contextMenuState.x, y: contextMenuState.y }"
    :items="contextMenuItems"
    @close="closeContextMenu"
    @select="handleContextMenuSelect"
  />
</template>
