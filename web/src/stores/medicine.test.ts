import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMedicineStore } from './medicine'
import { apiClient } from '../api/client'
import type { Medicine, DatamatrixParseResult, AIFARecord } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

describe('Medicine Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockMed: Medicine = {
    id: 'med_1',
    name: 'Tachipirina 1000 mg',
    active_ingredient: 'Paracetamolo',
    dosage: '1000 mg',
    form: 'compresse',
    aic: '034234035',
    expiration_date: '2028-10-31',
    pao_days: 0,
    effective_expiration_date: '2028-10-31',
    location: 'armadietto_bagno',
    requires_cold_chain: false,
    quantity: 16,
    unit: 'compresse',
    min_threshold: 4,
    prescription_required: false,
    created_at: '2026-10-18T00:00:00Z',
    updated_at: '2026-10-18T00:00:00Z'
  }

  const mockDrops: Medicine = {
    id: 'med_2',
    name: 'Tobral Collirio 5ml',
    active_ingredient: 'Tobramicina',
    dosage: '0.3%',
    form: 'collirio',
    expiration_date: '2028-12-31',
    pao_days: 28,
    opened_at: '2026-10-01',
    effective_expiration_date: '2026-10-29',
    location: 'frigo',
    requires_cold_chain: true,
    quantity: 1,
    unit: 'ml',
    min_threshold: 0,
    prescription_required: true,
    created_at: '2026-10-18T00:00:00Z',
    updated_at: '2026-10-18T00:00:00Z'
  }

  it('fetches medicines and computes counts and filters', async () => {
    vi.mocked(apiClient.get).mockResolvedValueOnce([mockMed, mockDrops])
    const store = useMedicineStore()

    await store.fetchMedicines()

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/medicines')
    expect(store.medicines).toHaveLength(2)
    expect(store.totalCount).toBe(2)
    expect(store.coldChainCount).toBe(1)

    // Test filter location
    store.selectedLocation = 'frigo'
    expect(store.filteredMedicines).toHaveLength(1)
    expect(store.filteredMedicines[0].id).toBe('med_2')

    // Test search filter
    store.selectedLocation = 'all'
    store.searchQuery = 'paracetamolo'
    expect(store.filteredMedicines).toHaveLength(1)
    expect(store.filteredMedicines[0].id).toBe('med_1')
  })

  it('creates medicine and prepends to state', async () => {
    vi.mocked(apiClient.post).mockResolvedValueOnce(mockMed)
    const store = useMedicineStore()

    const created = await store.createMedicine({ name: 'Tachipirina 1000 mg' })
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/medicines', { name: 'Tachipirina 1000 mg' })
    expect(created.id).toBe('med_1')
    expect(store.medicines).toContainEqual(mockMed)
  })

  it('opens medicine and updates effective expiration', async () => {
    const openedResult: Medicine = {
      ...mockDrops,
      opened_at: '2026-10-18',
      effective_expiration_date: '2026-11-15'
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(openedResult)
    const store = useMedicineStore()
    store.medicines = [mockDrops]

    const updated = await store.openMedicine('med_2', '2026-10-18')
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/medicines/med_2/open', { opened_at: '2026-10-18' })
    expect(updated.effective_expiration_date).toBe('2026-11-15')
    expect(store.medicines[0].effective_expiration_date).toBe('2026-11-15')
  })

  it('consumes dose and reduces stock', async () => {
    const consumedResult: Medicine = {
      ...mockMed,
      quantity: 14
    }
    vi.mocked(apiClient.post).mockResolvedValueOnce(consumedResult)
    const store = useMedicineStore()
    store.medicines = [mockMed]

    const updated = await store.consumeDose('med_1', 2)
    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/medicines/med_1/consume', { amount: 2 })
    expect(updated.quantity).toBe(14)
    expect(store.medicines[0].quantity).toBe(14)
  })

  it('parses Datamatrix GS1 code and looks up AIFA catalog', async () => {
    const mockParseResult: DatamatrixParseResult = {
      aic: '034234035',
      expiration_date: '2027-06-30',
      lot_number: 'LOT123',
      raw_code: '(01)08003423403501(17)270630(10)LOT123'
    }
    const mockAIFARecord: AIFARecord = {
      aic: '034234035',
      name: 'Tachipirina 1000 mg compresse',
      active_ingredient: 'Paracetamolo',
      dosage: '1000 mg',
      form: 'compresse',
      unit: 'compresse',
      default_quantity: 16,
      requires_cold_chain: false,
      default_location: 'armadietto_bagno',
      default_pao_days: 0,
      prescription_required: false
    }

    vi.mocked(apiClient.post).mockResolvedValueOnce(mockParseResult)
    vi.mocked(apiClient.get).mockResolvedValueOnce([mockAIFARecord])

    const store = useMedicineStore()

    const parsed = await store.parseCode('(01)08003423403501(17)270630(10)LOT123')
    expect(parsed.aic).toBe('034234035')

    const aifa = await store.lookupAIFA('tachipirina', '034234035')
    expect(aifa).toHaveLength(1)
    expect(aifa[0].active_ingredient).toBe('Paracetamolo')
  })
})
