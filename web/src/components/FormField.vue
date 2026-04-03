<script setup lang="ts">
import { computed } from 'vue'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

export interface SelectOption {
  value: string
  label: string
}

const props = withDefaults(defineProps<{
  label: string
  modelValue: string | boolean
  type?: string
  error?: string
  required?: boolean
  options?: SelectOption[]
  placeholder?: string
}>(), {
  type: 'text',
  required: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | boolean]
}>()

const inputId = computed(() => `field-${props.label.toLowerCase().replace(/\s+/g, '-')}`)
</script>

<template>
  <div class="flex flex-col gap-1.5 mb-4">
    <!-- Checkbox -->
    <template v-if="type === 'checkbox'">
      <div class="flex items-center gap-2">
        <Checkbox
          :id="inputId"
          :checked="!!modelValue"
          @update:checked="emit('update:modelValue', $event === true)"
        />
        <Label :for="inputId" class="font-normal cursor-pointer">
          {{ label }}
          <span v-if="required" class="text-destructive ml-0.5">*</span>
        </Label>
      </div>
    </template>

    <template v-else>
      <Label :for="inputId">
        {{ label }}
        <span v-if="required" class="text-destructive ml-0.5">*</span>
      </Label>

      <!-- Select -->
      <Select
        v-if="type === 'select'"
        :model-value="modelValue as string"
        @update:model-value="emit('update:modelValue', ($event ?? '') as string)"
      >
        <SelectTrigger :id="inputId" :class="{ 'border-destructive': !!error }">
          <SelectValue :placeholder="placeholder" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem
            v-for="opt in options"
            :key="opt.value"
            :value="opt.value"
          >
            {{ opt.label }}
          </SelectItem>
        </SelectContent>
      </Select>

      <!-- Textarea -->
      <Textarea
        v-else-if="type === 'textarea'"
        :id="inputId"
        :model-value="modelValue as string"
        :placeholder="placeholder"
        :class="{ 'border-destructive': !!error }"
        rows="4"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />

      <!-- Input -->
      <Input
        v-else
        :id="inputId"
        :type="type"
        :model-value="modelValue as string"
        :placeholder="placeholder"
        :class="{ 'border-destructive': !!error }"
        @update:model-value="emit('update:modelValue', String($event ?? ''))"
      />
    </template>

    <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
  </div>
</template>
