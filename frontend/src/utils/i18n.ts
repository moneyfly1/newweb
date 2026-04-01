// 中文化翻译工具

// 邮件类型翻译
export const emailTypeMap: Record<string, string> = {
  verification: '验证码',
  reset_password: '重置密码',
  welcome: '欢迎邮件',
  subscription: '订阅信息',
  payment_success: '支付成功',
  recharge_success: '充值成功',
  expiry_reminder: '到期提醒',
  expiry_notice: '到期通知',
  test: '测试邮件',
  admin_create_user: '管理员创建账号',
  account_disabled: '账号已禁用',
  account_enabled: '账号已启用',
  account_deleted: '账号已删除',
  subscription_reset: '订阅重置',
  abnormal_login: '异常登录',
  unpaid_order: '未支付订单',
  new_order: '新订单通知'
}

// 余额变动类型翻译
export const balanceChangeTypeMap: Record<string, string> = {
  recharge: '充值',
  purchase: '购买套餐',
  refund: '退款',
  checkin: '签到奖励',
  invite_reward: '邀请奖励',
  invite_commission: '邀请佣金',
  mystery_box: '盲盒消费',
  mystery_box_reward: '盲盒奖励',
  redeem: '卡密兑换',
  admin_adjust: '管理员调整',
  subscription_convert: '订阅转余额'
}

// 佣金类型翻译
export const commissionTypeMap: Record<string, string> = {
  purchase: '购买佣金',
  recharge: '充值佣金',
  invite: '邀请佣金'
}

// 登录状态翻译
export const loginStatusMap: Record<string, string> = {
  success: '成功',
  failed: '失败',
  error: '错误'
}

// 设备类型解析
export function parseDeviceType(userAgent: string): string {
  if (!userAgent) return '未知设备'

  const ua = userAgent.toLowerCase()

  // 移动设备
  if (ua.includes('iphone')) return 'iPhone'
  if (ua.includes('ipad')) return 'iPad'
  if (ua.includes('android')) {
    if (ua.includes('mobile')) return 'Android 手机'
    return 'Android 平板'
  }

  // 桌面设备
  if (ua.includes('windows')) {
    if (ua.includes('windows nt 10')) return 'Windows 10/11'
    if (ua.includes('windows nt 6.3')) return 'Windows 8.1'
    if (ua.includes('windows nt 6.2')) return 'Windows 8'
    if (ua.includes('windows nt 6.1')) return 'Windows 7'
    return 'Windows'
  }
  if (ua.includes('macintosh') || ua.includes('mac os x')) return 'macOS'
  if (ua.includes('linux')) {
    if (ua.includes('ubuntu')) return 'Ubuntu'
    if (ua.includes('fedora')) return 'Fedora'
    if (ua.includes('debian')) return 'Debian'
    return 'Linux'
  }

  // 其他
  if (ua.includes('cros')) return 'Chrome OS'

  return '未知设备'
}

// 浏览器类型解析
export function parseBrowserType(userAgent: string): string {
  if (!userAgent) return '未知浏览器'

  const ua = userAgent.toLowerCase()

  if (ua.includes('edg/')) return 'Edge'
  if (ua.includes('chrome/') && !ua.includes('edg')) return 'Chrome'
  if (ua.includes('safari/') && !ua.includes('chrome')) return 'Safari'
  if (ua.includes('firefox/')) return 'Firefox'
  if (ua.includes('opera/') || ua.includes('opr/')) return 'Opera'
  if (ua.includes('msie') || ua.includes('trident/')) return 'IE'

  return '未知浏览器'
}

// 完整设备信息解析
export function parseDeviceInfo(userAgent: string): string {
  if (!userAgent) return '未知设备'

  const device = parseDeviceType(userAgent)
  const browser = parseBrowserType(userAgent)

  if (device === '未知设备' && browser === '未知浏览器') {
    return '未知设备'
  }

  if (browser === '未知浏览器') {
    return device
  }

  return `${device} · ${browser}`
}

// 位置格式化（国家+省份+城市，识别不到的自动省略）
export function formatLocation(location: string): string {
  if (!location || location === '-' || location === 'Unknown' || location === '未知' || location === '未知位置') return '未知位置'
  if (location === '本地' || location === '本地网络') return location

  // 如果已经是中文格式（包含中文），直接返回
  if (/[\u4e00-\u9fa5]/.test(location)) {
    return location
  }

  // 尝试解析英文格式（如 "China, Beijing" 或 "United States, New York"）
  const parts = location.split(/[,\s]+/).map(p => p.trim()).filter(Boolean)

  if (parts.length >= 2) {
    const country = parts[0]
    const city = parts[parts.length - 1]

    // 使用浏览器内置 API 自动翻译国家名（支持全球所有国家）
    let translatedCountry = country
    try {
      const displayNames = new Intl.DisplayNames(['zh-CN'], { type: 'region' })
      // 尝试将国家名转为 ISO 代码再翻译
      const countryCode = getCountryCode(country)
      if (countryCode) {
        translatedCountry = displayNames.of(countryCode) || country
      }
    } catch {
      // 降级到手动映射表
      translatedCountry = fallbackCountryMap[country] || country
    }

    return `${translatedCountry} · ${city}`
  }

  return location
}

// 降级用的国家名映射表（当 Intl.DisplayNames 不可用时使用）
const fallbackCountryMap: Record<string, string> = {
  'China': '中国', 'United States': '美国', 'USA': '美国', 'US': '美国',
  'Japan': '日本', 'Korea': '韩国', 'South Korea': '韩国', 'Republic of Korea': '韩国',
  'United Kingdom': '英国', 'UK': '英国', 'Great Britain': '英国',
  'France': '法国', 'Germany': '德国', 'Canada': '加拿大',
  'Australia': '澳大利亚', 'Singapore': '新加坡',
  'Hong Kong': '香港', 'Taiwan': '台湾', 'Macao': '澳门',
  'Russia': '俄罗斯', 'India': '印度', 'Brazil': '巴西',
  'Italy': '意大利', 'Spain': '西班牙', 'Netherlands': '荷兰',
  'Sweden': '瑞典', 'Norway': '挪威', 'Denmark': '丹麦',
  'Finland': '芬兰', 'Switzerland': '瑞士', 'Austria': '奥地利',
  'Belgium': '比利时', 'Portugal': '葡萄牙', 'Ireland': '爱尔兰',
  'New Zealand': '新西兰', 'Mexico': '墨西哥', 'Argentina': '阿根廷',
  'Thailand': '泰国', 'Vietnam': '越南', 'Malaysia': '马来西亚',
  'Indonesia': '印度尼西亚', 'Philippines': '菲律宾',
  'Turkey': '土耳其', 'Israel': '以色列', 'UAE': '阿联酋',
  'Saudi Arabia': '沙特阿拉伯', 'Egypt': '埃及', 'South Africa': '南非'
}

// 常见国家名到 ISO 3166-1 alpha-2 代码的映射
function getCountryCode(name: string): string | null {
  const codeMap: Record<string, string> = {
    'China': 'CN', 'United States': 'US', 'USA': 'US', 'US': 'US',
    'Japan': 'JP', 'Korea': 'KR', 'South Korea': 'KR', 'Republic of Korea': 'KR',
    'United Kingdom': 'GB', 'UK': 'GB', 'Great Britain': 'GB',
    'France': 'FR', 'Germany': 'DE', 'Canada': 'CA',
    'Australia': 'AU', 'Singapore': 'SG',
    'Hong Kong': 'HK', 'Taiwan': 'TW', 'Macao': 'MO',
    'Russia': 'RU', 'India': 'IN', 'Brazil': 'BR',
    'Italy': 'IT', 'Spain': 'ES', 'Netherlands': 'NL',
    'Sweden': 'SE', 'Norway': 'NO', 'Denmark': 'DK',
    'Finland': 'FI', 'Switzerland': 'CH', 'Austria': 'AT',
    'Belgium': 'BE', 'Portugal': 'PT', 'Ireland': 'IE',
    'New Zealand': 'NZ', 'Mexico': 'MX', 'Argentina': 'AR',
    'Thailand': 'TH', 'Vietnam': 'VN', 'Malaysia': 'MY',
    'Indonesia': 'ID', 'Philippines': 'PH',
    'Turkey': 'TR', 'Israel': 'IL', 'UAE': 'AE',
    'Saudi Arabia': 'SA', 'Egypt': 'EG', 'South Africa': 'ZA'
  }
  return codeMap[name] || null
}

// 翻译邮件类型
export function translateEmailType(type: string): string {
  return emailTypeMap[type] || type
}

// 翻译余额变动类型
export function translateBalanceChangeType(type: string): string {
  return balanceChangeTypeMap[type] || type
}

// 翻译佣金类型
export function translateCommissionType(type: string): string {
  return commissionTypeMap[type] || type
}

// 翻译登录状态
export function translateLoginStatus(status: string): string {
  return loginStatusMap[status] || status
}
