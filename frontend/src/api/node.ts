import request from '@/utils/request'

export const listNodes = (params?: any) => request.get('/nodes', { params })
