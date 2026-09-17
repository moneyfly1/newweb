/**
 * 下载链接解析。
 *
 * 这里原来还有一整套「前端自己查 GitHub Release 匹配安装包」的实现
 * （getClientDownloadUrl / detectSystem / getClientReleasesUrl + 代理前缀、候选 URL、
 * 客户端配置表等约 280 行）。该链路已由后端 /api/v1/download/gh 与 /download/resolve
 * 承担，前端只剩下面这一个函数被真正调用，其余全是死代码，已删除。
 */

/** 把配置里的 pan:// 下载地址转成后端解析接口；普通 URL 原样返回 */
export function resolvePanDownloadUrl(url: string) {
  if (typeof url === 'string' && url.startsWith('pan://')) {
    const value = url.slice('pan://'.length)
    return `/api/v1/download/gh?key=${encodeURIComponent(value)}`
  }
  return url
}
