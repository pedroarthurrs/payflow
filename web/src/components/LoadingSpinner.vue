<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = withDefaults(defineProps<{
  size?: 'sm' | 'md' | 'lg'
}>(), {
  size: 'md',
})

const sizeClass = {
  sm: 'size-4',
  md: 'size-8',
  lg: 'size-12',
}
</script>

<template>
  <div class="flex items-center justify-center" role="status">
    <Loader2 :class="cn('animate-spin text-primary', sizeClass[props.size])" />
    <span class="sr-only">Carregando...</span>
  </div>
</template>
