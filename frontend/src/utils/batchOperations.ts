import { useMessage, useDialog } from 'naive-ui'

export function useBatchOperations() {
  const message = useMessage()
  const dialog = useDialog()

  const batchDelete = async (ids: any[], deleteFn: (id: any) => Promise<any>, itemName = '项') => {
    return new Promise((resolve, reject) => {
      dialog.warning({
        title: '批量删除',
        content: `确定要删除选中的 ${ids.length} 个${itemName}吗？`,
        positiveText: '确定',
        onPositiveClick: async () => {
          try {
            await Promise.all(ids.map(id => deleteFn(id)))
            message.success('批量删除成功')
            resolve(true)
          } catch {
            message.error('批量删除失败')
            reject()
          }
        }
      })
    })
  }

  const batchUpdate = async (ids: any[], updateFn: (id: any, data: any) => Promise<any>, data: any) => {
    try {
      await Promise.all(ids.map(id => updateFn(id, data)))
      message.success('批量操作成功')
      return true
    } catch {
      message.error('批量操作失败')
      return false
    }
  }

  return { batchDelete, batchUpdate }
}
