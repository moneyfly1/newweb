import request from '@/utils/request'

export const listNodes = (params?: any) => request.get('/nodes', { params })

export const testNode = (id: number) => request.post(`/nodes/${id}/test`)
