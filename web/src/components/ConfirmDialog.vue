<script setup lang="ts">
import { computed } from 'vue'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'

const props = withDefaults(defineProps<{
  visible: boolean
  title: string
  message: string
  confirmLabel?: string
  variant?: 'danger' | 'warning'
}>(), {
  confirmLabel: 'Confirmar',
  variant: 'danger',
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const open = computed({
  get: () => props.visible,
  set: (val) => { if (!val) emit('cancel') },
})
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>{{ title }}</DialogTitle>
        <DialogDescription>{{ message }}</DialogDescription>
      </DialogHeader>
      <slot />
      <DialogFooter>
        <Button variant="outline" @click="emit('cancel')">Cancelar</Button>
        <Button
          :variant="variant === 'danger' ? 'destructive' : 'default'"
          @click="emit('confirm')"
        >
          {{ confirmLabel }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
