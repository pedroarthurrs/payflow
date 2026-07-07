import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { createPinia } from 'pinia'
import { createApp } from 'vue'
import App from './App.vue'
import './core/assets/index.css'
import { useAuthStore } from './modules/auth/store/auth'
import router from './router'

document.documentElement.classList.add('dark')

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)

const authStore = useAuthStore()
authStore.loadFromStorage()

const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            staleTime: 60_000,
            gcTime: 5 * 60_000,
            retry: 1,
        },
    },
})

app.use(VueQueryPlugin, { queryClient })
app.use(router)
app.mount('#app')
