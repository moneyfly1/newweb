import request from '@/utils/request'

export const listTickets = (params?: any) => request.get('/tickets', { params })
export const createTicket = (data: { title: string; content: string; type: string; priority?: string; attachment_ids?: number[] }) =>
  request.post('/tickets', data)
export const getTicket = (id: number) => request.get(`/tickets/${id}`)
export const replyTicket = (id: number, data: { content: string; attachment_ids?: number[] }) =>
  request.post(`/tickets/${id}/reply`, data)
export const closeTicket = (id: number) => request.put(`/tickets/${id}`, { status: 'closed' })

// 工单附件（用户端与管理端共用上传/下载端点；下载需鉴权）
export const uploadTicketAttachment = (data: FormData) =>
  request.post('/tickets/attachments', data, { headers: { 'Content-Type': 'multipart/form-data' } })
export const deleteTicketAttachment = (attId: number) => request.delete(`/tickets/attachments/${attId}`)
