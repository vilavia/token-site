import { apiClient } from './client'

export interface CreateOrderRequest {
  amount_usd: number
  pay_type: 'alipay' | 'wxpay'
}

export interface CreateOrderResponse {
  order_no: string
  pay_url: string
}

export interface PaymentOrder {
  id: number
  order_no: string
  amount_usd: number
  amount_rmb: number
  exchange_rate: number
  status: string
  pay_type: string
  created_at: string
  paid_at?: string
}

export const paymentAPI = {
  createOrder(data: CreateOrderRequest) {
    return apiClient.post<CreateOrderResponse>('/payment/create', data)
  },
  getOrders() {
    return apiClient.get<PaymentOrder[]>('/payment/orders')
  },
  getExchangeRate() {
    return apiClient.get<{ usd_to_rmb: number }>('/payment/exchange-rate')
  }
}
