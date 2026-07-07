<script setup lang="ts">
import type { Column } from '@/core/components/DataTable.vue'
import DataTable from '@/core/components/DataTable.vue'
import { Button } from '@/core/components/ui/button'
import {
  Dialog,
  DialogContent
} from '@/core/components/ui/dialog'
import Can from '@/core/rbac/Can.vue'
import { useRBAC } from '@/core/rbac/composable'
import { Customer } from '@/core/types'
import { formatDocument, formatPhone } from '@/core/utils/formatting'
import { useCustomerList } from '@/modules/customers/composables/useCustomers'
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import CustomerFormView from './CustomerFormView.vue'

const router = useRouter()
const { can } = useRBAC()
const { data: queryData, isPending } = useCustomerList()
const customers = computed(() => queryData.value?.data ?? [])

const fetchError = ref<string | null>(null)

const showCreateModal = ref(false)

const columns: Column[] = [
  { key: 'name', label: 'Nome' },
  { key: 'email', label: 'Email' },
  { key: 'phone', label: 'Telefone', format: (value: string) => formatPhone(value) || '—' },
  { key: 'document', label: 'Documento', format: (value: string, row: Customer) => formatDocument(value, row.document_type) },
  { key: 'status', label: 'Status' },
  {
    key: 'created_at',
    label: 'Data de Criação',
    format: (value: string) => {
      if (!value) return ''
      return new Date(value).toLocaleDateString('pt-BR')
    },
  },
]

function handleRowClick(row: Customer) {
  router.push(`/customers/${row.id}`)
}


function openCreateModal() {
  showCreateModal.value = true
}

function closeCreateModal() {
  showCreateModal.value = false
}
</script>

<template>
  <div class="p-6">
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-2xl font-semibold text-foreground">Clientes</h1>
      <Can permission="customers:write">
        <Button @click="openCreateModal">Novo Cliente</Button>
      </Can>
    </div>

    <div v-if="fetchError" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
    </div>

    <DataTable
      v-else
      :columns="columns"
      :data="customers"
      :loading="isPending"
      @row-click="handleRowClick"
    />

    <Dialog v-model:open="showCreateModal" @update:open="(open) => !open && closeCreateModal()">
      <DialogContent class="max-w-[600px]">
        <CustomerFormView />
      </DialogContent>
    </Dialog>
  </div>
</template>
