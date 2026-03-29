export interface User {
  id: number
  username: string
  email: string
  balance: number
  is_admin: boolean
  is_active: boolean
  notes?: string
  expire_time?: string
  device_limit: number
  level?: number
  level_name?: string
  created_at: string
  last_login?: string
}

export interface Subscription {
  id: number
  user_id: number
  username?: string
  user_email?: string
  package_name?: string
  status: 'active' | 'expiring' | 'expired' | 'disabled'
  is_active: boolean
  expire_time?: string
  device_limit: number
  current_devices: number
  subscription_url?: string
  universal_url?: string
  clash_url?: string
  universal_count?: number
  clash_count?: number
  user_notes?: string
}

export interface Order {
  id: number
  order_no: string
  user_id: number
  user_email?: string
  order_type: string
  order_type_text?: string
  order_summary?: string
  package_name?: string
  amount: number
  discount_amount: number
  final_amount: number
  status: 'pending' | 'paid' | 'completed' | 'cancelled' | 'refunded'
  payment_method_name?: string
  gateway_order_id?: string
  created_at: string
  payment_time?: string
}

export interface PaginationParams {
  page: number
  page_size: number
  search?: string
  status?: string
}

export interface ApiResponse<T> {
  data: T
  message?: string
}

export interface ListResponse<T> {
  items: T[]
  total: number
}
