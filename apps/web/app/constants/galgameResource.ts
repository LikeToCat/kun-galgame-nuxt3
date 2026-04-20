export const GALGAME_RESOURCE_TYPE_ICON_MAP: Record<string, string> = {
  game: 'lucide:box',
  patch: 'lucide:puzzle',
  collection: 'lucide:boxes',
  voice: 'lucide:music-4',
  image: 'lucide:image',
  ai: 'simple-icons:openai',
  video: 'lucide:video',
  others: 'lucide:ellipsis'
}

export const GALGAME_RESOURCE_PLATFORM_ICON_MAP: Record<string, string> = {
  windows: 'ant-design:windows-outlined',
  mac: 'iconoir:apple-mac',
  linux: 'ant-design:linux-outlined',
  emulator: 'lucide:terminal',
  app: 'lucide:smartphone',
  others: 'lucide:ellipsis'
}

export type ProviderKey =
  | 'baidu'
  | 'aliyun'
  | 'quark'
  | 'pan123'
  | 'tianyiyun'
  | 'caiyun'
  | 'xunlei'
  | 'uc'
  | 'lanzou'
  | 'other'

export const PROVIDER_PATTERNS: Record<ProviderKey, string[]> = {
  baidu: ['pan.baidu.com', 'tieba.baidu.com', 'pan.baidu.', 'baidu.com'],
  aliyun: ['alipan.com', 'aliyun', 'aliyundrive', 'aliyuncs', 'aliyunpan'],
  quark: ['pan.quark.cn', 'quark.cn', 'quark'],
  pan123: [
    '123pan',
    '123684',
    '123865',
    '123912',
    '123912.com',
    '123684.com',
    '123865.com',
    '123pan.cn',
    'vip.123pan'
  ],
  tianyiyun: ['cloud.189.cn', '189.cn', 'ecloud.189.cn'],
  caiyun: ['caiyun.139.com', 'yun.139.com', '139.com'],
  xunlei: ['pan.xunlei.com', 'xunlei.com'],
  uc: ['drive.uc.cn', 'uc.cn'],
  lanzou: [
    'lanzou.com',
    'lanzous.com',
    'lanzoux.com',
    'lanzoui.com',
    'lanzouw.com',
    'lanzouj.com',
    'lanzouu.com',
    'lanzouq.com'
  ],
  other: []
}

export const PROVIDER_KEY_OPTIONS = [
  'baidu',
  'aliyun',
  'quark',
  'pan123',
  'tianyiyun',
  'caiyun',
  'xunlei',
  'uc',
  'lanzou',
  'other'
] as const satisfies ProviderKey[]

export const KUN_GALGAME_PROVIDER_LABEL_MAP: Record<ProviderKey, string> = {
  baidu: '百度网盘',
  aliyun: '阿里云盘',
  quark: '夸克网盘',
  pan123: '123盘',
  tianyiyun: '天翼云盘',
  caiyun: '和彩云',
  xunlei: '迅雷网盘',
  uc: 'UC网盘',
  lanzou: '蓝奏云',
  other: '其他 (自建网盘等不限速)'
}
