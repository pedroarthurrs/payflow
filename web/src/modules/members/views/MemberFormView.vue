<script setup lang="ts">
import { api } from '@/core/api/http'
import FormField from '@/core/components/FormField.vue'
import LoadingSpinner from '@/core/components/LoadingSpinner.vue'
import { Button } from '@/core/components/ui/button'
import { useErrorHandler } from '@/core/composables/useErrorHandler'
import { useToast } from '@/core/composables/useToast'
import type { ApiError, User } from '@/core/types'
import { useCreateMember, useUpdateMember } from '@/modules/members/composables/useMembers'
import { createMemberSchema, updateMemberSchema } from '@/modules/members/schemas'
import { toTypedSchema } from '@vee-validate/zod'
import { useForm } from 'vee-validate'
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()
const toast = useToast()

const memberId = computed(() => route.params.id as string | undefined)
const isEditMode = computed(() => !!memberId.value)

const schema = computed(() => isEditMode.value ? updateMemberSchema : createMemberSchema)

const { handleSubmit, errors, defineField, setFieldError, resetForm, setValues } = useForm({
  validationSchema: computed(() => toTypedSchema(schema.value)),
})

const [name] = defineField('name')
const [email] = defineField('email')
const [password] = defineField('password')
const [role] = defineField('role')

const formError = ref('')
const fetchError = ref('')
const notFound = ref(false)
const fetchLoading = ref(false)

const createMutation = useCreateMember()
const updateMutation = useUpdateMember()

const loading = computed(() => createMutation.isPending.value || updateMutation.isPending.value)

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
  fetchLoading.value = true
  try {
    const member = await api.get<User>(`/v1/members/${memberId.value}`)
    setValues({
      name: member.name,
      role: member.role as 'admin' | 'member',
    })
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadMember)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar member'
    }
  } finally {
    fetchLoading.value = false
  }
}

const onSubmit = handleSubmit(async (values) => {
  formError.value = ''

  try {
    if (isEditMode.value) {
      const payload = {
        name: values.name,
        role: values.role,
      }
      await updateMutation.mutateAsync({ id: memberId.value!, data: payload })
      toast.success('Member atualizado com sucesso')
    } else {
      const payload = {
        name: values.name,
        email: values.email,
        password: values.password,
        role: values.role,
      }
      await createMutation.mutateAsync(payload)
      toast.success('Member criado com sucesso')
      resetForm()
    }
    router.push('/members')
  } catch (e) {
    const err = e as ApiError
    if (err.status === 400 && err.validationErrors) {
      for (const [field, message] of Object.entries(err.validationErrors)) {
        setFieldError(field, message)
      }
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
})

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

    <LoadingSpinner v-else-if="fetchLoading && isEditMode && !name" size="lg" />

    <form v-else class="flex flex-col" @submit.prevent="onSubmit">
      <div v-if="formError" class="bg-card border border-destructive text-destructive px-4 py-3 rounded-md mb-4 text-sm">
        {{ formError }}
      </div>

      <FormField
        label="Nome"
        :modelValue="name ?? ''"
        @update:modelValue="name = $event as string"
        :error="errors.name"
        :required="true"
        placeholder="Nome do member"
      />

      <template v-if="!isEditMode">
        <FormField
          label="Email"
          type="email"
          :modelValue="email ?? ''"
          @update:modelValue="email = $event as string"
          :error="errors.email"
          :required="true"
          placeholder="email@exemplo.com"
        />

        <FormField
          label="Senha"
          type="password"
          :modelValue="password ?? ''"
          @update:modelValue="password = $event as string"
          :error="errors.password"
          :required="true"
          placeholder="Mínimo 8 caracteres"
        />
      </template>

      <FormField
        label="Role"
        type="select"
        :modelValue="role ?? 'member'"
        @update:modelValue="role = $event as string"
        :error="errors.role"
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
