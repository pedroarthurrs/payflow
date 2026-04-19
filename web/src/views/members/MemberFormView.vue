<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { Button } from '@/components/ui/button'
import { useCrud } from '@/composables/useCrud'
import { api } from '@/services/http'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
import type { User, CreateUserRequest, UpdateUserRequest, ApiError } from '@/types'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { getById, create, loading } = useCrud<User>('/v1/members')

const memberId = computed(() => route.params.id as string | undefined)
const isEditMode = computed(() => !!memberId.value)

const name = ref('')
const email = ref('')
const password = ref('')
const role = ref<'admin' | 'member'>('member')

const fieldErrors = ref<Record<string, string>>({})
const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)

const { handleError } = useErrorHandler({
  onNotFound: () => {
    notFound.value = true
  },
})

const roleOptions = [
  { value: 'admin', label: 'Admin' },
  { value: 'member', label: 'Member' },
]

const pageTitle = computed(() => (isEditMode.value ? 'Editar Member' : 'Novo Member'))

async function loadMember() {
  if (!memberId.value) return
  fetchError.value = ''
  notFound.value = false
  try {
    const member = await getById(memberId.value)
    name.value = member.name
    role.value = member.role as 'admin' | 'member'
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadMember)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar member'
    }
  }
}

async function handleSubmit() {
  fieldErrors.value = {}
  formError.value = ''

  try {
    if (isEditMode.value) {
      const payload: UpdateUserRequest = {
        name: name.value,
        role: role.value,
      }
      await api.put(`/v1/members/${memberId.value}/role`, payload)
      toast.success('Member atualizado com sucesso')
    } else {
      const payload: CreateUserRequest = {
        name: name.value,
        email: email.value,
        password: password.value,
        role: role.value,
      }
      await create(payload)
      toast.success('Member criado com sucesso')
    }
    router.push('/members')
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      fieldErrors.value = { ...err.validationErrors }
    } else if (err.status === 403) {
      formError.value = err.message || 'Operação não permitida'
    } else if (err.status === 409) {
      formError.value = err.message || 'Conflito: email já cadastrado'
    } else if (err.status === 429) {
      toast.warning('Limite de requisições excedido. Aguarde antes de tentar novamente.')
    } else {
      const handled = handleError(err)
      if (!handled) {
        formError.value = err.message || 'Erro ao salvar member'
      }
    }
  }
}

onMounted(() => {
  if (isEditMode.value) {
    loadMember()
  }
})
</script>

<template>
  <div class="mx-auto max-w-[600px] p-6">
    <h1 class="text-2xl font-bold text-foreground mb-6">{{ pageTitle }}</h1>

    <div v-if="notFound" class="text-center p-8 bg-card border border-destructive rounded-lg">
      <p class="text-destructive mb-4">Member não encontrado</p>
      <router-link to="/members" class="text-primary text-sm font-medium hover:underline">Voltar para listagem</router-link>
    </div>

    <div v-else-if="fetchError" class="text-center p-8">
      <p class="text-destructive mb-4">{{ fetchError }}</p>
      <Button @click="loadMember">Tentar novamente</Button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !name" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="handleSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
        {{ formError }}
      </div>

      <FormField
        label="Nome"
        :modelValue="name"
        @update:modelValue="name = $event as string"
        :error="fieldErrors.name"
        :required="true"
        placeholder="Nome do member"
      />

      <template v-if="!isEditMode">
        <FormField
          label="Email"
          type="email"
          :modelValue="email"
          @update:modelValue="email = $event as string"
          :error="fieldErrors.email"
          :required="true"
          placeholder="email@exemplo.com"
        />

        <FormField
          label="Senha"
          type="password"
          :modelValue="password"
          @update:modelValue="password = $event as string"
          :error="fieldErrors.password"
          :required="true"
          placeholder="Mínimo 8 caracteres"
        />
      </template>

      <FormField
        label="Role"
        type="select"
        :modelValue="role"
        @update:modelValue="role = $event as 'admin' | 'member'"
        :error="fieldErrors.role"
        :required="true"
        :options="roleOptions"
      />

      <div class="flex justify-end gap-3 mt-2">
        <Button type="button" variant="outline" @click="router.push('/members')">Cancelar</Button>
        <Button type="submit" :disabled="loading">
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </Button>
      </div>
    </form>
  </div>
</template>
