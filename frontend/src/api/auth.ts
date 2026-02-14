import request from '@/utils/request'

export const login = (data: { email: string; password: string }) =>
  request.post('/auth/login', data)

export const register = (data: { email: string; password: string; username: string; invite_code?: string }) =>
  request.post('/auth/register', data)

export const logout = () => request.post('/auth/logout')

export const refreshToken = (data: { refresh_token: string }) => request.post('/auth/refresh', data)

export const sendVerificationCode = (data: { email: string; purpose: string }) =>
  request.post('/auth/verification/send', data)

export const verifyCode = (data: { email: string; code: string }) =>
  request.post('/auth/verification/verify', data)

export const forgotPassword = (data: { email: string }) =>
  request.post('/auth/forgot-password', data)

export const resetPassword = (data: { email: string; code: string; password: string }) =>
  request.post('/auth/reset-password', data)
