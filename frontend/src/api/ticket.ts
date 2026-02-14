import request from '@/utils/request'

export const listTickets = (params?: any) => request.get('/tickets', { params })
export const createTicket = (data: { title: string; content: string; type: string; priority?: string }) =>
  request.post('/tickets', data)
export const getTicket = (id: number) => request.get(`/tickets/${id}`)
export const replyTicket = (id: number, data: { content: string }) =>
  request.post(`/tickets/${id}/reply`, data)
export const closeTicket = (id: number) => request.put(`/tickets/${id}`, { status: 'closed' })
