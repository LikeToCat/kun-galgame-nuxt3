export interface GalgameStorePersist {
  vndbId: string
  name: KunLanguage
  introduction: KunLanguage
  contentLimit: 'sfw' | 'nsfw'
  ageLimit: 'all' | 'r18'
  originalLanguage: Language
  aliases: string[]
}

export interface GalgameEditStoreTemp {
  id: number
  vndbId: string
  name: KunLanguage
  introduction: KunLanguage
  contentLimit: 'sfw' | 'nsfw'
  ageLimit: 'all' | 'r18'
  originalLanguage: Language
  alias: string[]
}
