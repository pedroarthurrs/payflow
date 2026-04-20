<script setup lang="ts">
import { Checkbox } from '@/core/components/ui/checkbox'
import { Input } from '@/core/components/ui/input'
import { Label } from '@/core/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/core/components/ui/select'
import { Textarea } from '@/core/components/ui/textarea'
import { Eye, EyeOff } from 'lucide-vue-next'
import { computed, ref } from 'vue'

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
  showPasswordToggle?: boolean
}>(), {
  type: 'text',
  required: false,
  showPasswordToggle: false,
})

const showPassword = ref(false)
const effectiveType = computed(() =>
  props.type === 'password' && props.showPasswordToggle && showPassword.value ? 'text' : props.type,
)

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
          <span v-if="required" class="form-field__required text-destructive ml-0.5">*</span>
        </Label>
      </div>
    </template>

    <template v-else>
      <Label :for="inputId">
        {{ label }}
        <span v-if="required" class="form-field__required text-destructive ml-0.5">*</span>
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
      <div v-else :class="showPasswordToggle && type === 'password' ? 'relative' : undefined">
        <Input
          :id="inputId"
          :type="effectiveType"
          :model-value="modelValue as string"
          :placeholder="placeholder"
          :class="{ 'border-destructive': !!error, 'pr-10': showPasswordToggle && type === 'password' }"
          @update:model-value="emit('update:modelValue', String($event ?? ''))"
        />
        <button
          v-if="showPasswordToggle && type === 'password'"
          type="button"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground focus:outline-none"
          :aria-label="showPassword ? 'Ocultar senha' : 'Mostrar senha'"
          @click="showPassword = !showPassword"
        >
          <EyeOff v-if="showPassword" :size="16" />
          <Eye v-else :size="16" />
        </button>
      </div>
    </template>

    <p v-if="error" class="form-field__error text-sm text-destructive">{{ error }}</p>
  </div>
</template>
