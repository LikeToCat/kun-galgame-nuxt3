export interface SectionTopic {
  id: number
  title: string
  content: string
  view: number
  likeCount: number
  replyCount: number
  hasBestAnswer: boolean
  isNSFWTopic: boolean
  user: KunUser
  created: Date | string
}
