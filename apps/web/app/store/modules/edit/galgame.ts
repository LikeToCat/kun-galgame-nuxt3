import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import { createEmptyLocaleMap, resetReactiveState } from '~/store/index'
import type { GalgameStorePersist } from '~/store/types/edit/galgame'

export const usePersistEditGalgameStore = defineStore(
  'KUNGalgameEditGalgame',
  () => {
    const vndbId = ref<GalgameStorePersist['vndbId']>('')
    const name = reactive<GalgameStorePersist['name']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    const introduction = reactive<GalgameStorePersist['introduction']>({
      'en-us': '',
      'ja-jp': '',
      'zh-cn': '',
      'zh-tw': ''
    })
    const contentLimit = ref<GalgameStorePersist['contentLimit']>('sfw')
    // Wiki defaults: original_language=ja-jp, age_limit=r18. We default
    // age_limit to 'all' instead, because publishing R18 without the user
    // opting in is a content-policy risk on a default-SFW site (per audit
    // §10). User must consciously flip to r18 if applicable.
    const ageLimit = ref<GalgameStorePersist['ageLimit']>('all')
    const originalLanguage =
      ref<GalgameStorePersist['originalLanguage']>('ja-jp')
    const aliases = ref<GalgameStorePersist['aliases']>([])

    const resetEditGalgameStore = () => {
      vndbId.value = ''
      resetReactiveState(name, createEmptyLocaleMap())
      resetReactiveState(introduction, createEmptyLocaleMap())
      contentLimit.value = 'sfw'
      ageLimit.value = 'all'
      originalLanguage.value = 'ja-jp'
      aliases.value = []
    }

    return {
      vndbId,
      name,
      introduction,
      contentLimit,
      ageLimit,
      originalLanguage,
      aliases,

      resetEditGalgameStore
    }
  },
  {
    persist: {
      storage: piniaPluginPersistedstate.localStorage()
    }
  }
)
