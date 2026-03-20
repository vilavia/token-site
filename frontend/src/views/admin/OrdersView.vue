<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Left: Search + Filters -->
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <!-- User ID Filter -->
            <div class="relative w-full md:w-48">
              <input
                v-model="userIdFilter"
                type="text"
                :placeholder="t('admin.orders.filterByUser')"
                class="input"
                @keyup.enter="loadOrders"
              />
            </div>

            <!-- Status Filter Tabs -->
            <div class="flex gap-1">
              <button
                v-for="tab in statusTabs"
                :key="tab.value"
                @click="statusFilter = tab.value; loadOrders()"
                :class="[
                  'rounded-lg px-3 py-1.5 text-sm font-medium transition-colors',
                  statusFilter === tab.value
                    ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                    : 'text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-dark-200'
                ]"
              >
                {{ tab.label }}
              </button>
            </div>
          </div>

          <!-- Right: Refresh -->
          <div class="flex items-center gap-2">
            <button
              @click="loadOrders"
              :disabled="loading"
              class="btn btn-secondary"
              :title="t('common.refresh')"
            >
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredOrders" :loading="loading">
          <template #cell-id="{ value }">
            <span class="font-mono text-xs text-gray-600 dark:text-dark-300">{{ value }}</span>
          </template>

          <template #cell-user_id="{ value }">
            <button
              class="font-mono text-xs text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
              @click="userIdFilter = String(value); loadOrders()"
            >
              {{ value }}
            </button>
          </template>

          <template #cell-order_no="{ value }">
            <span class="font-mono text-xs text-gray-600 dark:text-dark-300">
              {{ value.slice(0, 16) }}...
            </span>
          </template>

          <template #cell-amount_usd="{ value }">
            <span class="font-medium text-gray-900 dark:text-white">${{ value.toFixed(2) }}</span>
          </template>

          <template #cell-amount_rmb="{ value }">
            <span class="text-gray-600 dark:text-dark-300">&yen;{{ value.toFixed(2) }}</span>
          </template>

          <template #cell-status="{ value }">
            <span
              :class="[
                'inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium',
                value === 'paid'
                  ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                  : value === 'pending'
                    ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                    : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'
              ]"
            >
              {{ value }}
            </span>
          </template>

          <template #cell-pay_type="{ value }">
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ value || '-' }}</span>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <button
              v-if="row.status === 'pending'"
              @click="handleMarkPaid(row)"
              class="btn btn-secondary px-2 py-1 text-xs"
            >
              {{ t('admin.orders.markAsPaid') }}
            </button>
          </template>

          <template #empty>
            <EmptyState :title="t('admin.orders.noOrders')" icon="inbox" />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { apiClient } from '@/api/client'
import { formatDateTime } from '@/utils/format'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()

interface PaymentOrder {
  id: number
  user_id: number
  order_no: string
  trade_no?: string
  amount_usd: number
  amount_rmb: number
  exchange_rate: number
  status: string
  pay_type?: string
  paid_at?: string
  created_at: string
  updated_at: string
}

const loading = ref(false)
const orders = ref<PaymentOrder[]>([])
const userIdFilter = ref('')
const statusFilter = ref('')

const statusTabs = computed(() => [
  { value: '', label: t('admin.orders.allOrders') },
  { value: 'pending', label: t('admin.orders.pending') },
  { value: 'paid', label: t('admin.orders.paid') },
  { value: 'cancelled', label: t('admin.orders.cancelled') },
])

const columns = computed<Column[]>(() => [
  { key: 'id', label: 'ID', sortable: true },
  { key: 'user_id', label: t('admin.orders.userId'), sortable: true },
  { key: 'order_no', label: t('admin.orders.orderNo') },
  { key: 'amount_usd', label: t('admin.orders.amountUsd'), sortable: true },
  { key: 'amount_rmb', label: t('admin.orders.amountRmb'), sortable: true },
  { key: 'status', label: t('admin.orders.status') },
  { key: 'pay_type', label: t('admin.orders.payType') },
  { key: 'created_at', label: t('admin.orders.time'), sortable: true },
  { key: 'actions', label: t('admin.orders.actions') },
])

const filteredOrders = computed(() => {
  if (!statusFilter.value) return orders.value
  return orders.value.filter(o => o.status === statusFilter.value)
})

async function loadOrders() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (userIdFilter.value) {
      params.user_id = userIdFilter.value
    }
    const queryString = new URLSearchParams(params).toString()
    const url = '/admin/payment/orders' + (queryString ? '?' + queryString : '')
    const res = await apiClient.get<PaymentOrder[]>(url)
    orders.value = res.data
  } catch (error: any) {
    appStore.showError(error?.message || 'Failed to load orders')
  } finally {
    loading.value = false
  }
}

async function handleMarkPaid(order: PaymentOrder) {
  if (!confirm(t('admin.orders.confirmMarkPaid'))) return
  try {
    await apiClient.put(`/admin/payment/orders/${order.id}/status`, { status: 'paid' })
    appStore.showSuccess(t('admin.orders.markPaidSuccess'))
    await loadOrders()
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.orders.markPaidFailed'))
  }
}

onMounted(() => {
  // Check for user_id query param (from user action menu)
  const queryUserId = route.query.user_id as string
  if (queryUserId) {
    userIdFilter.value = queryUserId
  }
  loadOrders()
})
</script>
