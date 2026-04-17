import type { GalgameCard } from './galgame'

export interface UserInfo {
  id: number
  name: string
  avatar: string
  role: number
  status: number
  moemoepoint: number
  bio: string
  created: Date | string

  upvote: number
  like: number
  dislike: number

  replyCreated: number
  commentCreated: number
  topic: number
  topicPoll: number

  galgame: number
  contributeGalgame: number
  galgameComment: number
  galgameRating: number

  galgameResource: number
  galgameToolset: number
  galgameToolsetResource: number

  dailyTopicCount: number
  dailyGalgameCount: number
}

export interface UserTopic {
  id: number
  title: string
  created: Date | string
}

export type UserGalgame = GalgameCard

export interface UserGalgameResource {
  id: number
  galgameId: number
  galgameName: KunLanguage
  type: string
  language: string
  platform: string
  size: string
  link: string[]
  code: string
  password: string
  note: string
  status: number
  created: Date | string
}

export interface UserReply {
  topicId: number
  content: string
  created: Date | string
}

export interface UserGetUserReplyRequestData {
  uid: number
  ridArray: number[]
}

export interface UserComment {
  topicId: number
  content: string
  created: Date | string
}

export interface UserGetUserCommentRequestData {
  uid: number
  cidArray: number[]
}

export interface UserFloatingCard extends KunUser {
  moemoepoint: number
  topicCount: number
  topicReplyCount: number
  topicCommentCount: number
  galgameResourceCount: number
}

export type UserUpdateAvatarResponseData = {
  avatar: string
  avatarMin: string
}

export type UserGetUserEmailResponseData = {
  email: string
}
