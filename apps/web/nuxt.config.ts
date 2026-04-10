import tailwindcss from '@tailwindcss/vite'
import fs from 'fs'
import path from 'path'
import { ICON_COLLECTIONS, ICON_NAMES } from './lib/icon'
import type { TSConfig } from 'pkg-types'

const packageJson = JSON.parse(
  fs.readFileSync(path.resolve(__dirname, '..', '..', 'package.json'), 'utf-8')
)
const appVersion = packageJson.version

console.log(appVersion)

const sharedTsConfig: TSConfig = {
  exclude: ['**/backup/**', '**/dist/**', '**/node_modules/**', '**/prisma/**']
}

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: false },

  app: {
    pageTransition: { name: 'kun-page', mode: 'out-in' },
    layoutTransition: { name: 'kun-page', mode: 'out-in' },

    // https://github.com/nuxt/nuxt/issues/26565#issuecomment-3448517709
    baseURL: '/'
    // buildAssetsDir: `/_nuxt/v${Math.floor(Date.now() / 1000).toString()}/`
  },

  experimental: {
    scanPageMeta: true,
    typescriptPlugin: true
  },

  compatibilityDate: '2025-07-15',

  devServer: {
    host: '127.0.0.1',
    port: 2333
  },

  modules: [
    '@nuxt/image',
    '@nuxt/icon',
    '@nuxt/eslint',
    '@nuxtjs/color-mode',
    '@pinia/nuxt',
    '@dxup/nuxt',
    'pinia-plugin-persistedstate/nuxt',
    'nuxt-schema-org',
    'nuxt-umami'
  ],

  runtimeConfig: {
    // Server-only: used by useFetch during SSR to directly reach Go API
    apiBaseUrl: process.env.API_BASE_URL || 'http://127.0.0.1:2334',

    public: {
      KUN_GALGAME_URL: process.env.KUN_GALGAME_URL,
      KUN_VISUAL_NOVEL_FORUM_YANDEX_VERIFICATION:
        process.env.KUN_VISUAL_NOVEL_FORUM_YANDEX_VERIFICATION,
      KUN_VISUAL_NOVEL_VERSION: appVersion,

      // Go API base URL (client-side, relative — goes through browser)
      apiBaseUrl: process.env.API_BASE_URL || 'http://127.0.0.1:2334',

      // OAuth
      oauthServerUrl:
        process.env.OAUTH_SERVER_URL || 'http://127.0.0.1:9277/api/v1',
      oauthClientId: process.env.OAUTH_CLIENT_ID || '',
      oauthRedirectUri:
        process.env.OAUTH_REDIRECT_URI || 'http://127.0.0.1:2333/auth/callback'
    }
  },

  routeRules: {},

  imports: {
    dirs: ['./composables', './config', './utils']
  },

  site: {
    url: process.env.KUN_GALGAME_URL
  },

  umami: {
    id: process.env.KUN_VISUAL_NOVEL_FORUM_UMAMI_ID,
    host: 'https://stats.kungal.org/',
    autoTrack: true
  },

  // Frontend
  css: ['~/styles/index.css'],

  icon: {
    mode: 'svg',
    serverBundle: {
      collections: ICON_COLLECTIONS
    },
    clientBundle: {
      icons: ICON_NAMES,
      scan: false
    }
  },

  typescript: {
    tsConfig: {
      ...sharedTsConfig
    },
    nodeTsConfig: {
      ...sharedTsConfig
    },
    sharedTsConfig: {
      ...sharedTsConfig
    }
  },

  vite: {
    plugins: [tailwindcss()]
  },

  // $production: {
  //   vite: {
  //     esbuild: {
  //       drop: ['console', 'debugger']
  //     }
  //   }
  // },

  // ogImage: {
  //   fonts: [
  //     {
  //       name: 'Lolita',
  //       weight: 400,
  //       path: '/fonts/Lolita.ttf'
  //     }
  //   ]
  // },

  eslint: {
    config: {
      stylistic: false
    }
  },

  // Pinia store functions auto imports
  pinia: {
    storesDirs: ['./store/**']
  },

  piniaPluginPersistedstate: {
    cookieOptions: {
      maxAge: 60 * 60 * 24 * 7,
      sameSite: 'strict'
    }
  },

  colorMode: {
    preference: 'system',
    fallback: 'light',
    globalName: '__KUNGALGAME_COLOR_MODE__',
    componentName: 'ColorScheme',
    classPrefix: 'kun-',
    classSuffix: '-mode',
    storageKey: 'kungalgame-color-mode'
  },

  nitro: {
    typescript: {
      tsConfig: {
        ...sharedTsConfig
      }
    }
  }
})
