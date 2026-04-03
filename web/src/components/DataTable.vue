<script setup lang="ts">
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableEmpty,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

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
  <div class="w-full overflow-x-auto">
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead v-for="col in props.columns" :key="col.key">{{ col.label }}</TableHead>
          <TableHead v-if="props.showActions" class="w-px whitespace-nowrap">Ações</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody v-if="props.loading">
        <TableRow>
          <TableCell :colspan="props.columns.length + (props.showActions ? 1 : 0)" class="py-8 text-center">
            <LoadingSpinner />
          </TableCell>
        </TableRow>
      </TableBody>
      <TableEmpty v-else-if="props.data.length === 0">
        Nenhum registro encontrado
      </TableEmpty>
      <TableBody v-else>
        <TableRow
          v-for="(row, index) in props.data"
          :key="index"
          class="cursor-pointer"
          @click="emit('row-click', row)"
        >
          <TableCell v-for="col in props.columns" :key="col.key">
            {{ getCellValue(row, col) }}
          </TableCell>
          <TableCell v-if="props.showActions" class="whitespace-nowrap">
            <Button
              variant="outline"
              size="sm"
              class="mr-2"
              aria-label="Editar"
              @click.stop="emit('edit', row)"
            >
              Editar
            </Button>
            <Button
              variant="destructive"
              size="sm"
              aria-label="Excluir"
              @click.stop="emit('delete', row)"
            >
              Excluir
            </Button>
          </TableCell>
        </TableRow>
      </TableBody>
    </Table>
  </div>
</template>
