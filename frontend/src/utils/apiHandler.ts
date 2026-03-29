import { useMessage } from 'naive-ui'

export interface ApiOptions {
  successMsg?: string
  errorMsg?: string
  showSuccess?: boolean
  showError?: boolean
}

export async function handleApiCall<T>(
  apiCall: () => Promise<T>,
  options: ApiOptions = {}
): Promise<T | null> {
  const message = useMessage()
  const {
    successMsg,
    errorMsg = '操作失败',
    showSuccess = !!successMsg,
    showError = true
  } = options

  try {
    const result = await apiCall()
    if (showSuccess && successMsg) {
      message.success(successMsg)
    }
    return result
  } catch (error: any) {
    if (showError) {
      const msg = error?.response?.data?.message || error?.message || errorMsg
      message.error(msg)
    }
    return null
  }
}
