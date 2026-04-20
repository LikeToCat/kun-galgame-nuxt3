export interface GalgameResource {
  id: number
  view: number
  galgameId: number
  user: KunUser
  type: string
  language: string
  platform: string
  size: string
  status: number
  download: number
  likeCount: number
  isLiked: boolean
  linkDomain: string
  /**
   * Pre-computed display labels for the resource's hosting providers
   * (e.g. ["百度网盘", "OneDrive"]). Resolved by the backend at write time
   * — do not re-derive from `linkDomain` in the UI.
   */
  providerNames: string[]
  note: string
  created: Date | string
  edited: Date | string | null
}

export interface GalgameResourceDetailLink extends GalgameResource {
  link: string[]
  code: string
  note: string
  password: string
}

export interface GalgameResourceCard extends GalgameResource {
  galgameName: KunLanguage
}

export interface GalgameResourceSummary {
  id: number
  name: KunLanguage
  banner: string
  contentLimit: string
  resourceUpdateTime: Date | string
  view: number
  originalLanguage: string
  ageLimit: KunAgeLimit
  platform: string[]
  language: string[]
  type: string[]
}

export interface GalgameResourcePageData {
  galgame: GalgameResourceSummary
  resource: GalgameResource
  recommendations: GalgameResourceCard[]
}
