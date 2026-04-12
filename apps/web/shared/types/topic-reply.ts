import type { TopicComment } from './topic-comment'

export interface TopicReplyTargetInfo {
  id: number
  floor: number
  user: KunUser
  contentPreview: string
  replyContentMarkdown: string
  replyContentHtml: string
}

export interface TopicReply {
  id: number
  topicId: number
  floor: number
  user: KunUser & { moemoepoint: number }
  contentMarkdown: string
  contentHtml: string

  likeCount: number
  isLiked: boolean
  dislikeCount: number
  isDisliked: boolean

  comment: TopicComment[]
  created: Date | string
  edited: Date | string | null

  targets: TopicReplyTargetInfo[]
  isPinned: boolean
  isBestAnswer: boolean
}

export type TopicReplyDetail = Omit<TopicReply, 'comment'>
