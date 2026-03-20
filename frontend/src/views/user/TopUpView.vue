<template>
  <AppLayout>
    <div class="mx-auto max-w-2xl space-y-6">
      <!-- Current Balance Card -->
      <div class="card overflow-hidden">
        <div class="bg-gradient-to-br from-primary-500 to-primary-600 px-6 py-8 text-center">
          <div
            class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-white/20 backdrop-blur-sm"
          >
            <Icon name="creditCard" size="xl" class="text-white" />
          </div>
          <p class="text-sm font-medium text-primary-100">{{ t('topup.currentBalance') }}</p>
          <p class="mt-2 text-4xl font-bold text-white">
            ${{ user?.balance?.toFixed(2) || '0.00' }}
          </p>
        </div>
      </div>

      <!-- Top Up Form -->
      <div class="card">
        <div class="p-6 space-y-6">
          <!-- Select Amount -->
          <div>
            <label class="input-label">{{ t('topup.selectAmount') }}</label>
            <div class="mt-2 grid grid-cols-3 gap-2">
              <button
                v-for="preset in presetAmounts"
                :key="preset"
                type="button"
                @click="selectAmount(preset)"
                :class="[
                  'rounded-xl border-2 px-4 py-3 text-sm font-medium transition-all',
                  selectedAmount === preset && !customAmountActive
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 dark:border-primary-400'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-500'
                ]"
              >
                <span class="block font-bold">${{ preset }}</span>
                <span v-if="exchangeRate" class="block text-xs mt-0.5 opacity-70">
                  ≈ ¥{{ (preset * exchangeRate).toFixed(2) }}
                </span>
              </button>
            </div>
          </div>

          <!-- Custom Amount -->
          <div>
            <label class="input-label">{{ t('topup.customAmount') }}</label>
            <div class="relative mt-1">
              <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                <span class="text-gray-500 dark:text-dark-400 font-medium">$</span>
              </div>
              <input
                v-model="customAmount"
                type="number"
                min="0.01"
                step="0.01"
                :placeholder="t('topup.enterAmount')"
                @focus="customAmountActive = true"
                @input="onCustomAmountInput"
                class="input py-3 pl-8"
              />
            </div>
            <p v-if="customAmountActive && customAmount && exchangeRate" class="input-hint">
              ≈ ¥{{ (parseFloat(customAmount) * exchangeRate).toFixed(2) }}
            </p>
          </div>

          <!-- Payment Method -->
          <div>
            <label class="input-label">{{ t('topup.paymentMethod') }}</label>
            <div class="mt-2 flex gap-3">
              <button
                type="button"
                @click="payType = 'alipay'"
                :class="[
                  'flex-1 rounded-xl border-2 px-4 py-3 text-sm font-medium transition-all',
                  payType === 'alipay'
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 dark:border-primary-400'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-500'
                ]"
              >
                {{ t('topup.alipay') }}
              </button>
              <button
                type="button"
                @click="payType = 'wxpay'"
                :class="[
                  'flex-1 rounded-xl border-2 px-4 py-3 text-sm font-medium transition-all',
                  payType === 'wxpay'
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300 dark:border-primary-400'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:border-primary-500'
                ]"
              >
                {{ t('topup.wechatPay') }}
              </button>
            </div>
          </div>

          <!-- Pay Now Button -->
          <button
            type="button"
            :disabled="!finalAmount || finalAmount <= 0 || submitting"
            @click="handlePay"
            class="btn btn-primary w-full py-3"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-5 w-5 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            <template v-else>
              {{ t('topup.payNow') }}
              <span v-if="finalAmount && finalAmount > 0" class="ml-1">
                (${{ finalAmount.toFixed(2) }}<span v-if="exchangeRate"> ≈ ¥{{ (finalAmount * exchangeRate).toFixed(2) }}</span>)
              </span>
            </template>
          </button>
        </div>
      </div>

      <!-- Order History -->
      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('topup.orderHistory') }}
          </h2>
        </div>
        <div class="p-6">
          <!-- Loading State -->
          <div v-if="loadingOrders" class="flex items-center justify-center py-8">
            <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
          </div>

          <!-- Orders Table -->
          <div v-else-if="orders.length > 0" class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-gray-100 dark:border-dark-700">
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('topup.orderNo') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('topup.amount') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('topup.status') }}</th>
                  <th class="pb-3 text-left font-medium text-gray-500 dark:text-dark-400">{{ t('topup.time') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-50 dark:divide-dark-800">
                <tr v-for="order in orders" :key="order.id" class="py-3">
                  <td class="py-3 font-mono text-xs text-gray-600 dark:text-dark-300">
                    {{ order.order_no.slice(0, 12) }}...
                  </td>
                  <td class="py-3 font-medium text-gray-900 dark:text-white">
                    ${{ order.amount_usd.toFixed(2) }}
                    <span class="text-xs text-gray-500 dark:text-dark-400 ml-1">
                      ≈ ¥{{ order.amount_rmb.toFixed(2) }}
                    </span>
                  </td>
                  <td class="py-3">
                    <span
                      :class="[
                        'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
                        order.status === 'paid'
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : order.status === 'pending'
                            ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                            : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
                      ]"
                    >
                      {{ order.status }}
                    </span>
                  </td>
                  <td class="py-3 text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDate(order.created_at) }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Empty State -->
          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('topup.noOrders') }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { paymentAPI, type PaymentOrder } from '@/api/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatDate } from '@/utils/format'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const user = computed(() => authStore.user)

const presetAmounts = [1, 5, 10, 20, 50, 100]
const selectedAmount = ref<number | null>(null)
const customAmount = ref('')
const customAmountActive = ref(false)
const payType = ref<'alipay' | 'wxpay'>('alipay')
const exchangeRate = ref<number | null>(null)
const orders = ref<PaymentOrder[]>([])
const loadingOrders = ref(false)
const submitting = ref(false)

const finalAmount = computed(() => {
  if (customAmountActive.value && customAmount.value) {
    const val = parseFloat(customAmount.value)
    return isNaN(val) ? null : val
  }
  return selectedAmount.value
})

function selectAmount(amount: number) {
  selectedAmount.value = amount
  customAmountActive.value = false
  customAmount.value = ''
}

function onCustomAmountInput() {
  if (customAmount.value) {
    selectedAmount.value = null
    customAmountActive.value = true
  } else {
    customAmountActive.value = false
  }
}

async function fetchExchangeRate() {
  try {
    const res = await paymentAPI.getExchangeRate()
    exchangeRate.value = res.data.usd_to_rmb
  } catch (error) {
    console.error('Failed to fetch exchange rate:', error)
  }
}

async function fetchOrders() {
  loadingOrders.value = true
  try {
    const res = await paymentAPI.getOrders()
    orders.value = res.data
  } catch (error) {
    console.error('Failed to fetch orders:', error)
  } finally {
    loadingOrders.value = false
  }
}

async function handlePay() {
  const amount = finalAmount.value
  if (!amount || amount <= 0) return

  submitting.value = true
  try {
    const res = await paymentAPI.createOrder({
      amount_usd: amount,
      pay_type: payType.value
    })
    window.location.href = res.data.pay_url
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || 'Failed to create order')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  fetchExchangeRate()
  fetchOrders()
})
</script>
