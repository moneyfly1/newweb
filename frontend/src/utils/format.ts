/**
 * 统一的展示格式化工具。
 *
 * 为什么集中到这里：此前同一份数据在不同页面由各自的实现渲染，导致口径不一致 ——
 *   · 相对时间：用户端输出「5分钟前」，管理端输出「5 分钟前」（多一个空格）；
 *   · 文件大小：附件列表只换算到 MB（大文件显示 1500.0 MB），上传组件却能到 GB（1.46 GB）；
 *   · 地区：用户端订阅页用 Intl.DisplayNames 直出，管理端用户详情另有国家名映射表，
 *     同一份地区数据在两个页面显示不同（如「中国香港」vs「香港」）。
 * 统一后所有页面共用一套规则，避免同类问题再次出现。
 */

/** ISO 3166-1 alpha-2 → 中文名（优先于 Intl，保证港澳台等口径一致） */
const countryNameMap: Record<string, string> = {
  CN: '中国', HK: '中国香港', MO: '中国澳门', TW: '中国台湾',
  US: '美国', JP: '日本', KR: '韩国', SG: '新加坡',
  GB: '英国', UK: '英国', DE: '德国', FR: '法国',
  CA: '加拿大', AU: '澳大利亚', RU: '俄罗斯', IN: '印度',
  TH: '泰国', VN: '越南', MY: '马来西亚', PH: '菲律宾',
  ID: '印度尼西亚',
}

/** 英文名 → 中文名（后端可能存英文国家名，或历史数据来自 MMDB 的 en 字段） */
const countryAliasMap: Record<string, string> = {
  china: '中国', hongkong: '中国香港', 'hong kong': '中国香港',
  macao: '中国澳门', macau: '中国澳门', taiwan: '中国台湾',
  'united states': '美国', usa: '美国', 'united kingdom': '英国',
  uk: '英国', japan: '日本', korea: '韩国', 'south korea': '韩国',
  singapore: '新加坡', germany: '德国', france: '法国',
  canada: '加拿大', australia: '澳大利亚', russia: '俄罗斯',
  india: '印度', thailand: '泰国', vietnam: '越南',
  malaysia: '马来西亚', philippines: '菲律宾', indonesia: '印度尼西亚',
}

const regionDisplayNames = typeof Intl !== 'undefined' && (Intl as any).DisplayNames
  ? new (Intl as any).DisplayNames(['zh-CN'], { type: 'region' })
  : null

/** 把「国家码 / 英文名 / 中文名」统一成中文显示名；无法识别时返回空串 */
export function countryNameFromText(value: unknown): string {
  if (!value || typeof value !== 'string') return ''
  const text = value.trim()
  if (!text) return ''
  const code = text.toUpperCase()
  if (/^[A-Z]{2}$/.test(code)) return countryNameMap[code] || regionDisplayNames?.of(code) || code
  return countryAliasMap[text.toLowerCase()] || ''
}

/** 某些历史数据把 JSON 存成了字符串，先尝试解析再判断 */
function parseMaybeJSON(value: unknown): unknown {
  if (typeof value !== 'string') return value
  const text = value.trim()
  if (!text || !/^[{[]/.test(text)) return value
  try {
    return JSON.parse(text)
  } catch {
    return value
  }
}

/**
 * 只显示国家/地区（不含城市），用于设备地区、登录位置等列。
 * 兼容后端可能返回的多种形态：纯字符串、JSON 字符串、或含多种字段名的对象。
 */
export function formatCountryOnly(location: unknown, empty = '-'): string {
  const parsed = parseMaybeJSON(location)
  if (!parsed) return empty
  if (typeof parsed === 'object') {
    const obj = parsed as Record<string, any>
    const code = obj.country_code || obj.countryCode || obj.country_iso || obj.countryISO || obj.iso_code
    const name = obj.country_name || obj.countryName || obj.country || obj.region || obj.location
    return countryNameFromText(code) || countryNameFromText(name) || name || empty
  }
  if (typeof parsed !== 'string') return empty
  const text = parsed.trim()
  const directName = countryNameFromText(text)
  if (directName) return directName
  const firstSegment = text.split(/[,·|/]+/).filter(Boolean)[0]?.trim() || text
  return countryNameFromText(firstSegment) || firstSegment || empty
}

/**
 * 相对时间：刚刚 / N分钟前 / N小时前 / N天前，超过 7 天显示日期。
 * 统一不带空格（此前管理端多一个空格，与用户端不一致）。
 */
export function formatRelativeTime(input?: string | number | Date | null, empty = '-'): string {
  if (!input) return empty
  const date = new Date(input)
  const ts = date.getTime()
  if (Number.isNaN(ts)) return empty

  const diff = Date.now() - ts
  if (diff < 0) return '刚刚'
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes}分钟前`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}小时前`
  const days = Math.floor(hours / 24)
  if (days < 7) return `${days}天前`
  return date.toLocaleDateString('zh-CN')
}

/**
 * 文件/数据体积：B / KB / MB / GB / TB。
 * 此前附件列表只换算到 MB（大文件显示 1500.0 MB），上传组件到 GB，显示口径不同。
 */
export function formatSize(size?: number | null, empty = '-'): string {
  if (size === null || size === undefined || Number.isNaN(size)) return empty
  if (size < 1024) return `${size} B`
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`
  if (size < 1024 ** 3) return `${(size / 1024 ** 2).toFixed(1)} MB`
  if (size < 1024 ** 4) return `${(size / 1024 ** 3).toFixed(2)} GB`
  return `${(size / 1024 ** 4).toFixed(2)} TB`
}
