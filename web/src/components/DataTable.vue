<script setup lang="ts">
import LoadingSpinner from '@/components/LoadingSpinner.vue'

export interface Column {
  key: string
  label: string
  format?: (value: any) => string
}

const props = withDefaults(defineProps<{
  columns: Column[]
  data: any[]
  loading?: boolean
  showActions?: boolean
}>(), {
  loading: false,
  showActions: true,
})

const emit = defineEmits<{
  'row-click': [row: any]
  'edit': [row: any]
  'delete': [row: any]
}>()

function getCellValue(row: any, column: Column): string {
  const value = row[column.key]
  if (column.format) {
    return column.format(value)
  }
  return value == null ? '' : String(value)
}
</script>

<template>
  <div class="data-table-wrapper">
    <table class="data-table">
      <thead>
        <tr>
          <th v-for="col in props.columns" :key="col.key">{{ col.label }}</th>
          <th v-if="props.showActions" class="actions-header">Ações</th>
        </tr>
      </thead>
      <tbody v-if="props.loading">
        <tr>
          <td :colspan="props.columns.length + (props.showActions ? 1 : 0)" class="loading-cell">
            <LoadingSpinner />
          </td>
        </tr>
      </tbody>
      <tbody v-else-if="props.data.length === 0">
        <tr>
          <td :colspan="props.columns.length + (props.showActions ? 1 : 0)" class="empty-cell">
            Nenhum registro encontrado
          </td>
        </tr>
      </tbody>
      <tbody v-else>
        <tr
          v-for="(row, index) in props.data"
          :key="index"
          class="data-row"
          @click="emit('row-click', row)"
        >
          <td v-for="col in props.columns" :key="col.key">
            {{ getCellValue(row, col) }}
          </td>
          <td v-if="props.showActions" class="actions-cell">
            <button
              class="btn-action btn-edit"
              @click.stop="emit('edit', row)"
              aria-label="Editar"
            >
              Editar
            </button>
            <button
              class="btn-action btn-delete"
              @click.stop="emit('delete', row)"
              aria-label="Excluir"
            >
              Excluir
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.data-table-wrapper {
  width: 100%;
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.data-table th,
.data-table td {
  padding: 0.75rem 1rem;
  text-align: left;
  border-bottom: 1px solid #e5e7eb;
}

.data-table th {
  background-color: #f9fafb;
  font-weight: 600;
  color: #374151;
}

.data-row {
  cursor: pointer;
  transition: background-color 0.15s;
}

.data-row:hover {
  background-color: #f3f4f6;
}

.loading-cell,
.empty-cell {
  text-align: center;
  padding: 2rem 1rem;
  color: #6b7280;
}

.actions-header {
  width: 1%;
  white-space: nowrap;
}

.actions-cell {
  white-space: nowrap;
}

.btn-action {
  padding: 0.375rem 0.75rem;
  border: 1px solid transparent;
  border-radius: 0.375rem;
  font-size: 0.8125rem;
  cursor: pointer;
  transition: background-color 0.15s, border-color 0.15s;
}

.btn-edit {
  background-color: #eff6ff;
  color: #2563eb;
  border-color: #bfdbfe;
  margin-right: 0.5rem;
}

.btn-edit:hover {
  background-color: #dbeafe;
}

.btn-delete {
  background-color: #fef2f2;
  color: #dc2626;
  border-color: #fecaca;
}

.btn-delete:hover {
  background-color: #fee2e2;
}
</style>
