import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type {
  Medicine,
  MedicineLocation,
  MedicineAlert,
  AIFARecord,
  DatamatrixParseResult
} from '../api/types'

export const useMedicineStore = defineStore('medicine', () => {
  const medicines = ref<Medicine[]>([])
  const alerts = ref<MedicineAlert[]>([])
  const aifaSearchResults = ref<AIFARecord[]>([])

  const selectedLocation = ref<MedicineLocation | 'all'>('all')
  const selectedStatus = ref<'all' | 'expired' | 'expiring_7d' | 'expiring_30d' | 'cold_chain' | 'low_stock'>('all')
  const searchQuery = ref('')

  const loading = ref(false)
  const error = ref<string | null>(null)

  // --- COMPUTED ---

  const totalCount = computed(() => medicines.value.length)

  const expiredCount = computed(() => {
    const today = new Date().toISOString().slice(0, 10)
    return medicines.value.filter((m) => m.effective_expiration_date < today).length
  })

  const expiringCount = computed(() => {
    const today = new Date()
    const in30d = new Date()
    in30d.setDate(today.getDate() + 30)
    const todayStr = today.toISOString().slice(0, 10)
    const in30dStr = in30d.toISOString().slice(0, 10)

    return medicines.value.filter((m) => {
      return m.effective_expiration_date >= todayStr && m.effective_expiration_date <= in30dStr
    }).length
  })

  const coldChainCount = computed(() => {
    return medicines.value.filter((m) => m.requires_cold_chain).length
  })

  const lowStockCount = computed(() => {
    return medicines.value.filter((m) => m.min_threshold > 0 && m.quantity <= m.min_threshold).length
  })

  const filteredMedicines = computed(() => {
    const todayStr = new Date().toISOString().slice(0, 10)
    const in7d = new Date()
    in7d.setDate(new Date().getDate() + 7)
    const in7dStr = in7d.toISOString().slice(0, 10)

    const in30d = new Date()
    in30d.setDate(new Date().getDate() + 30)
    const in30dStr = in30d.toISOString().slice(0, 10)

    return medicines.value.filter((m) => {
      // Filter location
      if (selectedLocation.value !== 'all' && m.location !== selectedLocation.value) {
        return false
      }

      // Filter status
      if (selectedStatus.value === 'expired') {
        if (m.effective_expiration_date >= todayStr) return false
      } else if (selectedStatus.value === 'expiring_7d') {
        if (m.effective_expiration_date < todayStr || m.effective_expiration_date > in7dStr) return false
      } else if (selectedStatus.value === 'expiring_30d') {
        if (m.effective_expiration_date < todayStr || m.effective_expiration_date > in30dStr) return false
      } else if (selectedStatus.value === 'cold_chain') {
        if (!m.requires_cold_chain) return false
      } else if (selectedStatus.value === 'low_stock') {
        if (!(m.min_threshold > 0 && m.quantity <= m.min_threshold)) return false
      }

      // Filter search
      if (searchQuery.value.trim()) {
        const q = searchQuery.value.toLowerCase().trim()
        const matchName = m.name.toLowerCase().includes(q)
        const matchIngredient = m.active_ingredient?.toLowerCase().includes(q) ?? false
        const matchAic = m.aic?.includes(q) ?? false
        const matchLot = m.lot_number?.toLowerCase().includes(q) ?? false
        if (!matchName && !matchIngredient && !matchAic && !matchLot) {
          return false
        }
      }

      return true
    })
  })

  // --- ACTIONS ---

  async function fetchMedicines() {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<Medicine[]>('/api/v1/medicines')
      medicines.value = data || []
    } catch (err: any) {
      error.value = err.message || 'Errore nel caricamento dei farmaci'
    } finally {
      loading.value = false
    }
  }

  async function fetchAlerts(fridgeTemp?: number) {
    try {
      const params = fridgeTemp !== undefined ? { fridge_temp: fridgeTemp } : {}
      const data = await apiClient.get<MedicineAlert[]>('/api/v1/medicines/alerts', params)
      alerts.value = data || []
    } catch (err: any) {
      console.warn('Impossibile recuperare gli alert farmaci:', err)
    }
  }

  async function createMedicine(data: Partial<Medicine>): Promise<Medicine> {
    loading.value = true
    error.value = null
    try {
      const created = await apiClient.post<Medicine>('/api/v1/medicines', data)
      medicines.value.unshift(created)
      return created
    } catch (err: any) {
      error.value = err.message || 'Errore nella creazione del farmaco'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateMedicine(id: string, data: Partial<Medicine>): Promise<Medicine> {
    loading.value = true
    error.value = null
    try {
      const updated = await apiClient.put<Medicine>(`/api/v1/medicines/${id}`, data)
      const idx = medicines.value.findIndex((m) => m.id === id)
      if (idx !== -1) {
        medicines.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err.message || 'Errore nell\'aggiornamento del farmaco'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function deleteMedicine(id: string): Promise<void> {
    loading.value = true
    error.value = null
    try {
      await apiClient.delete(`/api/v1/medicines/${id}`)
      medicines.value = medicines.value.filter((m) => m.id !== id)
    } catch (err: any) {
      error.value = err.message || 'Errore nella cancellazione del farmaco'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function openMedicine(id: string, openedAt?: string): Promise<Medicine> {
    try {
      const updated = await apiClient.post<Medicine>(`/api/v1/medicines/${id}/open`, {
        opened_at: openedAt || new Date().toISOString().slice(0, 10)
      })
      const idx = medicines.value.findIndex((m) => m.id === id)
      if (idx !== -1) {
        medicines.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err.message || 'Errore nella registrazione apertura'
      throw err
    }
  }

  async function consumeDose(id: string, amount: number = 1): Promise<Medicine> {
    try {
      const updated = await apiClient.post<Medicine>(`/api/v1/medicines/${id}/consume`, {
        amount
      })
      const idx = medicines.value.findIndex((m) => m.id === id)
      if (idx !== -1) {
        medicines.value[idx] = updated
      }
      return updated
    } catch (err: any) {
      error.value = err.message || 'Errore nel consumo della dose'
      throw err
    }
  }

  async function parseCode(code: string): Promise<DatamatrixParseResult> {
    return await apiClient.post<DatamatrixParseResult>('/api/v1/medicines/parse-code', { code })
  }

  async function lookupAIFA(query?: string, aic?: string): Promise<AIFARecord[]> {
    const params: Record<string, string> = {}
    if (query) params.q = query
    if (aic) params.aic = aic
    const data = await apiClient.get<AIFARecord[]>('/api/v1/medicines/lookup', params)
    aifaSearchResults.value = data || []
    return aifaSearchResults.value
  }

  return {
    medicines,
    alerts,
    aifaSearchResults,
    selectedLocation,
    selectedStatus,
    searchQuery,
    loading,
    error,
    totalCount,
    expiredCount,
    expiringCount,
    coldChainCount,
    lowStockCount,
    filteredMedicines,
    fetchMedicines,
    fetchAlerts,
    createMedicine,
    updateMedicine,
    deleteMedicine,
    openMedicine,
    consumeDose,
    parseCode,
    lookupAIFA
  }
})
