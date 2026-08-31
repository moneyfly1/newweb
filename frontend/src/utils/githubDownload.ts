/**
 * GitHub Release 自动下载直链（移植自 myweb）
 *
 * 统一入口 getClientDownloadUrl(clientKey, config)：
 *   1. 检测当前系统/架构（UA）
 *   2. 若配置值为 pan:// 或为空 → 查 GitHub latest release 匹配安装包 → 返回后端 /download/gh?key=
 *   3. 若配置值为普通 URL → 原样返回（兼容手工配置）
 */

const DEFAULT_GITHUB_PROXY_PREFIXES = [
  'https://ghproxy.com/{url}',
  'https://ghproxy.net/{url}',
  '{url}',
]

interface PlatformPattern { pattern: RegExp; installer: boolean }
interface ClientConfig {
  name: string
  repo: string
  platforms: Record<string, Record<string, PlatformPattern>>
}

// 客户端仓库映射（与后端 software_sync.Catalog 对应）
const CLIENT_CONFIGS = {
  'clash-party': {
    name: 'Clash Party',
    repo: 'mihomo-party-org/clash-party',
    platforms: {
      windows: { x64: { pattern: /windows.*(x64|[^a-z]64).*\.exe$/i, installer: true } },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.(pkg|dmg)$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.(pkg|dmg)$/i, installer: true },
      },
    },
  },
  'clash-verge-rev': {
    name: 'Clash Verge Rev',
    repo: 'clash-verge-rev/clash-verge-rev',
    platforms: {
      windows: {
        x64: { pattern: /(windows|win).*x64|.*x64.*setup|.*x64.*\.exe$/i, installer: true },
        arm64: { pattern: /(windows|win).*arm64|arm64.*\.exe$/i, installer: true },
      },
      macos: {
        intel: { pattern: /(intel|x64|amd64|_x64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64|_aarch64).*\.dmg$/i, installer: true },
      },
      linux: {
        x64: { pattern: /linux.*x64|amd64.*\.(deb|rpm|AppImage)$/i, installer: true },
        arm64: { pattern: /linux.*arm64|aarch64.*\.(deb|rpm|AppImage)$/i, installer: true },
      },
    },
  },
  'hiddify-app': {
    name: 'Hiddify',
    repo: 'hiddify/hiddify-app',
    platforms: {
      windows: { x64: { pattern: /(windows|win).*x64|.*x64.*\.exe$/i, installer: true } },
      android: { universal: { pattern: /android.*\.apk|\.apk$/i, installer: true } },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.dmg$/i, installer: true },
      },
    },
  },
  'FlClash': {
    name: 'FlClash',
    repo: 'chen08209/FlClash',
    platforms: {
      windows: { x64: { pattern: /(windows|win).*x64|.*x64.*\.exe$/i, installer: true } },
      macos: {
        intel: { pattern: /(intel|x64|amd64).*\.dmg$/i, installer: true },
        apple: { pattern: /(apple|silicon|m\d+|arm64|aarch64).*\.dmg$/i, installer: true },
      },
      android: { universal: { pattern: /android.*arm64.*v8a|arm64.*v8a.*\.apk|android.*\.apk$/i, installer: true } },
    },
  },
  'v2rayNG': {
    name: 'V2rayNG',
    repo: '2dust/v2rayNG',
    platforms: { android: { universal: { pattern: /android.*\.apk|\.apk$/i, installer: true } } },
  },
  'v2rayN': {
    name: 'V2rayN',
    repo: '2dust/v2rayN',
    platforms: {
      windows: {
        x64: { pattern: /windows.*64|win64|.*64.*\.zip$/i, installer: false },
        x32: { pattern: /windows.*32|win32|x32.*\.zip$/i, installer: false },
      },
      macos: {
        apple: { pattern: /macos.*arm64|mac.*arm64|arm64.*\.dmg$/i, installer: true },
        intel: { pattern: /macos.*intel|mac.*intel|intel.*\.dmg$/i, installer: true },
      },
    },
  },
  'clash-meta': {
    name: 'Clash Meta',
    repo: 'MetaCubeX/ClashMetaForAndroid',
    platforms: { android: { universal: { pattern: /\.apk$/i, installer: true } } },
  },
}

// resolveClientConfig 大小写不敏感查找
function resolveClientConfig(clientKey: string): ClientConfig | null {
  if (!clientKey) return null
  if (CLIENT_CONFIGS[clientKey as keyof typeof CLIENT_CONFIGS]) return CLIENT_CONFIGS[clientKey as keyof typeof CLIENT_CONFIGS]
  const lower = clientKey.toLowerCase()
  for (const [key, cfg] of Object.entries(CLIENT_CONFIGS)) {
    if (key.toLowerCase() === lower) return cfg
  }
  return null
}

/** 检测当前系统与架构 */
export function detectSystem() {
  const userAgent = navigator.userAgent.toLowerCase()
  const platform = navigator.platform?.toLowerCase() || ''
  let os = 'unknown'
  let arch = 'unknown'
  if (userAgent.includes('android')) os = 'android'
  else if (userAgent.includes('iphone') || userAgent.includes('ipad') || userAgent.includes('ios')) os = 'ios'
  else if (userAgent.includes('win') || platform.includes('win')) os = 'windows'
  else if (userAgent.includes('mac') || platform.includes('mac')) os = 'macos'
  else if (userAgent.includes('linux') || platform.includes('linux')) os = 'linux'

  if (os === 'windows') {
    if (userAgent.includes('arm64')) arch = 'arm64'
    else if (userAgent.includes('wow64') || userAgent.includes('x64')) arch = 'x64'
    else arch = 'x32'
  } else if (os === 'macos') {
    if (userAgent.includes('intel') && !userAgent.includes('apple')) arch = 'intel'
    else if (userAgent.includes('apple') || userAgent.includes('silicon') || userAgent.includes('arm')) arch = 'apple'
    else arch = (navigator.hardwareConcurrency || 0) >= 8 ? 'apple' : 'intel'
  } else if (os === 'linux') {
    arch = userAgent.includes('arm64') || userAgent.includes('aarch64') ? 'arm64' : 'x64'
  } else if (os === 'android') {
    arch = 'universal'
  }
  return { os, arch }
}

/** pan:// 标记 → 后端自动解析接口 */
export function resolvePanDownloadUrl(url: string) {
  if (typeof url === 'string' && url.startsWith('pan://')) {
    const value = url.slice('pan://'.length)
    return `/api/v1/download/gh?key=${encodeURIComponent(value)}`
  }
  return url
}

function toResolverURL(target: string) {
  return `/api/v1/download/resolve?target=${encodeURIComponent(target)}`
}

function getProxyPrefixes(softwareConfig: Record<string, any> = {}) {
  const raw = softwareConfig?.download_proxy_prefixes
  if (!raw) return DEFAULT_GITHUB_PROXY_PREFIXES
  if (Array.isArray(raw)) {
    const normalized = normalizeProxyPrefixes(raw)
    return normalized.length ? normalized : DEFAULT_GITHUB_PROXY_PREFIXES
  }
  const text = String(raw).trim()
  if (!text) return DEFAULT_GITHUB_PROXY_PREFIXES
  try {
    if (text.startsWith('[')) {
      const parsed = JSON.parse(text)
      if (Array.isArray(parsed)) {
        const normalized = normalizeProxyPrefixes(parsed)
        return normalized.length ? normalized : DEFAULT_GITHUB_PROXY_PREFIXES
      }
    }
  } catch { /* fallthrough */ }
  const list = text.split(/[\n,;]+/).map((i) => i.trim()).filter(Boolean)
  const normalized = normalizeProxyPrefixes(list)
  return normalized.length ? normalized : DEFAULT_GITHUB_PROXY_PREFIXES
}

function normalizeProxyPrefixes(prefixes: string[] = []) {
  const seen = new Set<string>()
  const out: string[] = []
  prefixes.forEach((item) => {
    const value = (item || '').trim()
    if (!value || seen.has(value)) return
    seen.add(value)
    out.push(value)
  })
  if (!out.some((item) => item === '{url}' || item.toLowerCase() === 'direct')) out.push('{url}')
  return out
}

function applyProxyPrefix(url: string, prefix: string) {
  if (!prefix || prefix === '{url}' || String(prefix).toLowerCase() === 'direct') return url
  if (prefix.includes('{url}')) return prefix.split('{url}').join(url)
  return `${prefix.replace(/\/+$/, '')}/${url}`
}

function buildCandidateUrls(url: string, prefixes: string[]) {
  const seen = new Set<string>()
  const candidates: string[] = []
  prefixes.forEach((prefix) => {
    const candidate = applyProxyPrefix(url, prefix)
    if (!seen.has(candidate)) { seen.add(candidate); candidates.push(candidate) }
  })
  if (!seen.has(url)) candidates.push(url)
  return candidates
}

async function fetchJSONWithCandidates(url: string, prefixes: string[]) {
  const candidates = buildCandidateUrls(url, prefixes)
  for (const candidate of candidates) {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 8000)
    try {
      const response = await fetch(candidate, {
        signal: controller.signal,
        headers: { Accept: 'application/vnd.github.v3+json' },
      })
      clearTimeout(timeoutId)
      if (response.ok) return await response.json()
    } catch {
      clearTimeout(timeoutId)
    }
  }
  throw new Error('获取发布信息失败，请稍后重试')
}

/**
 * 获取客户端最新下载地址。
 * - 配置值为 pan:// 或空 → 查 GitHub release 匹配 → 返回后端解析接口（自动加速镜像）
 * - 配置值为普通 URL → 原样返回
 * forcedArch 可选：强制指定架构（'apple' | 'intel' | ...）
 */
export async function getClientDownloadUrl(clientKey: string, softwareConfig: Record<string, any> = {}, forcedArch: string | null = null) {
  const { os, arch } = detectSystem()
  const resolvedArch = forcedArch || arch
  const client = resolveClientConfig(clientKey)
  if (!client) throw new Error(`未知的客户端: ${clientKey}`)

  // Android 特判：优先 arm64-v8a APK
  if (os === 'android') {
    try {
      const data = await fetchJSONWithCandidates(`https://api.github.com/repos/${client.repo}/releases/latest`, getProxyPrefixes(softwareConfig))
      if (data?.assets) {
        const apkAssets = data.assets.filter((a: any) => a.name.endsWith('.apk'))
        let apkAsset = apkAssets.find((a: any) => /arm64[-_]?v8a/i.test(a.name))
        if (!apkAsset) apkAsset = apkAssets.find((a: any) => /arm64/i.test(a.name))
        if (!apkAsset) apkAsset = apkAssets[0]
        if (apkAsset) return toResolverURL(apkAsset.browser_download_url)
      }
    } catch { /* fallthrough */ }
    return toResolverURL(`https://github.com/${client.repo}/releases/latest`)
  }

  try {
    const prefixes = getProxyPrefixes(softwareConfig)
    const apiUrl = `https://api.github.com/repos/${client.repo}/releases/latest`
    const data = await fetchJSONWithCandidates(apiUrl, prefixes)
    const platformConfig = client.platforms[os as keyof typeof client.platforms]
    if (!platformConfig) throw new Error(`不支持的操作系统: ${os}`)
    const archConfig = platformConfig[resolvedArch]
    if (!archConfig) {
      const firstArch = Object.keys(platformConfig)[0]
      if (firstArch) {
        const fallbackConfig = platformConfig[firstArch]
        const asset = data.assets.find((a: any) => fallbackConfig.pattern.test(a.name))
        if (asset) return toResolverURL(asset.browser_download_url)
      }
      throw new Error(`不支持的架构: ${resolvedArch}`)
    }
    let asset = data.assets.find((a: any) => archConfig.pattern.test(a.name))
    if (!asset) {
      const fallbackAsset = data.assets.find((a: any) => {
        const name = a.name.toLowerCase()
        if (os === 'windows' && (name.includes('.exe') || name.includes('.zip'))) return true
        if (os === 'macos' && (name.includes('.dmg') || name.includes('.pkg'))) return true
        if (os === 'linux' && (name.includes('.deb') || name.includes('.rpm') || name.includes('.appimage'))) return true
        return false
      })
      if (fallbackAsset) asset = fallbackAsset
      else throw new Error('未找到匹配的下载文件')
    }
    return toResolverURL(asset.browser_download_url)
  } catch (error) {
    console.error('获取 GitHub 下载链接失败:', error)
    return toResolverURL(`https://github.com/${client.repo}/releases/latest`)
  }
}

/** 客户端 Release 页面地址 */
export function getClientReleasesUrl(clientKey: string) {
  const client = resolveClientConfig(clientKey)
  if (!client) return null
  return toResolverURL(`https://github.com/${client.repo}/releases/latest`)
}
