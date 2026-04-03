<script setup lang="ts">
import { computed } from 'vue'

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

function onInput(event: Event) {
  const target = event.target as HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
  if (props.type === 'checkbox') {
    emit('update:modelValue', (target as HTMLInputElement).checked)
  } else {
    emit('update:modelValue', target.value)
  }
}
</script>

<template>
  <div class="form-field" :class="{ 'form-field--checkbox': type === 'checkbox', 'form-field--error': !!error }">
    <template v-if="type === 'checkbox'">
      <div class="form-field__checkbox-row">
        <input
          :id="inputId"
          type="checkbox"
          :checked="!!modelValue"
          @change="onInput"
          class="form-field__checkbox"
        />
        <label :for="inputId" class="form-field__label form-field__label--inline">
          {{ label }}
          <span v-if="required" class="form-field__required">*</span>
        </label>
      </div>
    </template>

    <template v-else>
      <label :for="inputId" class="form-field__label">
        {{ label }}
        <span v-if="required" class="form-field__required">*</span>
      </label>

      <select
        v-if="type === 'select'"
        :id="inputId"
        :value="modelValue"
        @change="onInput"
        class="form-field__input form-field__select"
      >
        <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
        <option
          v-for="opt in options"
          :key="opt.value"
          :value="opt.value"
        >
          {{ opt.label }}
        </option>
      </select>

      <textarea
        v-else-if="type === 'textarea'"
        :id="inputId"
        :value="modelValue as string"
        :placeholder="placeholder"
        @input="onInput"
        class="form-field__input form-field__textarea"
        rows="4"
      />

      <input
        v-else
        :id="inputId"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        @input="onInput"
        class="form-field__input"
      />
    </template>

    <p v-if="error" class="form-field__error">{{ error }}</p>
  </div>
</template>

<style scoped>
.form-field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
  margin-bottom: 1rem;
}

.form-field__label {
  font-size: 0.875rem;
  font-weight: 600;
  color: #374151;
}

.form-field__label--inline {
  font-weight: 400;
  cursor: pointer;
}

.form-field__required {
  color: #dc2626;
  margin-left: 0.125rem;
}

.form-field__input {
  padding: 0.5rem 0.75rem;
  border: 1px solid #d1d5db;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  color: #111827;
  background-color: #fff;
  transition: border-color 0.15s, box-shadow 0.15s;
}

.form-field__input:focus {
  outline: none;
  border-color: #2563eb;
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.2);
}

.form-field--error .form-field__input {
  border-color: #dc2626;
}

.form-field--error .form-field__input:focus {
  box-shadow: 0 0 0 2px rgba(220, 38, 38, 0.2);
}

.form-field__select {
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%236b7280' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 0.75rem center;
  padding-right: 2rem;
}

.form-field__textarea {
  resize: vertical;
  min-height: 5rem;
}

.form-field__checkbox-row {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.form-field__checkbox {
  width: 1rem;
  height: 1rem;
  cursor: pointer;
}

.form-field__error {
  font-size: 0.8125rem;
  color: #dc2626;
  margin: 0;
}
</style>
