<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FormField from '@/components/FormField.vue'
import LoadingSpinner from '@/components/LoadingSpinner.vue'
import { useCrud } from '@/composables/useCrud'
import { useToast } from '@/composables/useToast'
import { useErrorHandler } from '@/composables/useErrorHandler'
import type { User, CreateUserRequest, UpdateUserRequest, ApiError } from '@/types'

const route = useRoute()
const router = useRouter()
const toast = useToast()
const { getById, create, update, loading } = useCrud<User>('/v1/users')

const userId = computed(() => route.params.id as string | undefined)
const isEditMode = computed(() => !!userId.value)

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

const pageTitle = computed(() => (isEditMode.value ? 'Editar Usuário' : 'Novo Usuário'))

async function loadUser() {
  if (!userId.value) return
  fetchError.value = ''
  notFound.value = false
  try {
    const user = await getById(userId.value)
    name.value = user.name
    role.value = user.role as 'admin' | 'member'
  } catch (e) {
    const err = e as ApiError
    const handled = handleError(err, loadUser)
    if (!handled) {
      fetchError.value = err.message || 'Erro ao carregar usuário'
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
      await update(userId.value!, payload as unknown as Partial<User>)
      toast.success('Usuário atualizado com sucesso')
    } else {
      const payload: CreateUserRequest = {
        name: name.value,
        email: email.value,
        password: password.value,
        role: role.value,
      }
      await create(payload as unknown as Partial<User>)
      toast.success('Usuário criado com sucesso')
    }
    router.push('/users')
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
        formError.value = err.message || 'Erro ao salvar usuário'
      }
    }
  }
}

onMounted(() => {
  if (isEditMode.value) {
    loadUser()
  }
})
</script>

<template>
  <div class="user-form">
    <h1 class="user-form__title">{{ pageTitle }}</h1>

    <div v-if="notFound" class="user-form__not-found">
      <p class="user-form__not-found-message">Usuário não encontrado</p>
      <router-link to="/users" class="user-form__back-link">Voltar para listagem</router-link>
    </div>

    <div v-else-if="fetchError" class="user-form__fetch-error">
      <p class="user-form__fetch-error-message">{{ fetchError }}</p>
      <button class="user-form__retry-btn" @click="loadUser">Tentar novamente</button>
    </div>

    <LoadingSpinner v-else-if="loading && isEditMode && !name" size="lg" />

    <form v-else class="user-form__form" @submit.prevent="handleSubmit">
      <div v-if="formError" class="user-form__error">
        {{ formError }}
      </div>

      <FormField
        label="Nome"
        :modelValue="name"
        @update:modelValue="name = $event as string"
        :error="fieldErrors.name"
        :required="true"
        placeholder="Nome do usuário"
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

      <div class="user-form__actions">
        <button
          type="button"
          class="user-form__cancel-btn"
          @click="router.push('/users')"
        >
          Cancelar
        </button>
        <button
          type="submit"
          class="user-form__submit-btn"
          :disabled="loading"
        >
          {{ loading ? 'Salvando...' : (isEditMode ? 'Salvar' : 'Criar') }}
        </button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.user-form {
  max-width: 600px;
  margin: 0 auto;
  padding: 1.5rem;
}

.user-form__title {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--foreground);
  margin-bottom: 1.5rem;
}

.user-form__form {
  display: flex;
  flex-direction: column;
}

.user-form__error {
  background-color: var(--card);
  border: 1px solid var(--destructive);
  color: var(--destructive);
  padding: 0.75rem 1rem;
  border-radius: 0.375rem;
  margin-bottom: 1rem;
  font-size: 0.875rem;
}

.user-form__fetch-error {
  text-align: center;
  padding: 2rem;
}

.user-form__not-found {
  text-align: center;
  padding: 2rem;
  background: var(--card);
  border: 1px solid var(--destructive);
  border-radius: 0.5rem;
}

.user-form__not-found-message {
  color: var(--destructive);
  margin-bottom: 1rem;
  font-size: 1rem;
}

.user-form__back-link {
  color: var(--primary);
  text-decoration: none;
  font-size: 0.875rem;
  font-weight: 500;
}

.user-form__back-link:hover {
  text-decoration: underline;
}

.user-form__fetch-error-message {
  color: var(--destructive);
  margin-bottom: 1rem;
}

.user-form__retry-btn {
  padding: 0.5rem 1rem;
  background-color: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.user-form__retry-btn:hover {
  filter: brightness(1.08);
}

.user-form__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 0.5rem;
}

.user-form__cancel-btn {
  padding: 0.5rem 1rem;
  background-color: var(--secondary);
  color: var(--secondary-foreground);
  border: 1px solid var(--border);
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
}

.user-form__cancel-btn:hover {
  filter: brightness(1.08);
}

.user-form__submit-btn {
  padding: 0.5rem 1.5rem;
  background-color: var(--primary);
  color: var(--primary-foreground);
  border: none;
  border-radius: 0.375rem;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 600;
}

.user-form__submit-btn:hover:not(:disabled) {
  filter: brightness(1.08);
}

.user-form__submit-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
