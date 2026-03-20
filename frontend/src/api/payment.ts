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
    return apiClient.post<CreateOrderResponse>('/payment/orders', data)
  },
  getOrders() {
    return apiClient.get<PaymentOrder[]>('/payment/orders')
  },
  cancelOrder(orderId: number) {
    return apiClient.post(`/payment/orders/${orderId}/cancel`)
  },
  retryPayment(orderId: number) {
    return apiClient.post<CreateOrderResponse>(`/payment/orders/${orderId}/pay`)
  },
  getExchangeRate() {
    return apiClient.get<{ rate: number }>('/payment/exchange-rate')
  },
  getLimits() {
    return apiClient.get<{ min_usd: number; max_usd: number; preset_amounts: number[] }>('/payment/limits')
  }
}
