export interface TopicCard {
  id: number
  title: string
  view: number
  tag: string[]
  section: string[]
  user: KunUser
  status: number
  hasBestAnswer: boolean
  isPollTopic: boolean
  isNSFWTopic: boolean
  likeCount: number
  replyCount: number
  commentCount: number
  statusUpdateTime: Date | string
  upvoteTime: Date | string | null
}

export interface TopicAside {
  title: string
  tid: number
}

// TODO: bestAnswer is no longer embedded in TopicDetail from Go backend.
// Best answers are replies with isBestAnswer: true, loaded in reply list.
// Kept for backward compatibility with BestAnswer.vue component.
export interface TopicBestAnswer {
  id: number
  topicId: number
  floor: number
  user: KunUser & { moemoepoint: number }
  created: Date | string
  edited: Date | string | null
}

export interface TopicDetail {
  id: number
  title: string
  contentMarkdown: string
  contentHtml: string
  view: number
  status: number
  isNSFW: boolean
  category: string
  section: string[]
  tag: string[]
  user: KunUser & { moemoepoint: number }

  likeCount: number
  isLiked: boolean
  dislikeCount: number
  isDisliked: boolean
  favoriteCount: number
  isFavorited: boolean
  upvoteCount: number
  isUpvoted: boolean

  replyCount: number
  isPollTopic: boolean

  statusUpdateTime: Date | string
  upvoteTime: Date | string | null
  edited: Date | string | null
  created: Date | string
}
