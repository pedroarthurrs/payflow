import { ref } from 'vue'
import { defineStore } from 'pinia'
import axios from 'axios'

export const useStatusStore = defineStore('status', () => {
  const maintenance = ref(false)

  async function check() {
    try {
      const { data } = await axios.get<{ maintenance: boolean }>('/api/v1/status')
      maintenance.value = data.maintenance
    } catch {
      // Se o endpoint retornar 503, o backend está em manutenção
      maintenance.value = true
    }
  }

  return { maintenance, check }
})
