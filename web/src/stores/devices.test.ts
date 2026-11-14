import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDevicesStore } from './devices'
import { apiClient } from '../api/client'
import type { Device } from '../api/types'

vi.mock('../api/client', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn()
  }
}))

describe('Devices Store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  const mockDevices: Device[] = [
    {
      id: 'dev-1',
      name: 'Smart Bulb 1',
      area_id: 'area-1',
      integration_id: 'int-z2m',
      status: 'online',
      created_at: '',
      updated_at: ''
    },
    {
      id: 'dev-2',
      name: 'Temp Sensor',
      area_id: 'area-2',
      integration_id: 'int-esphome',
      status: 'offline',
      created_at: '',
      updated_at: ''
    }
  ]

  it('fetches devices and filters by area or status', async () => {
    const store = useDevicesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(mockDevices)

    await store.fetchDevices()
    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/devices')
    expect(store.devices.length).toBe(2)
    expect(store.byArea('area-1').length).toBe(1)
    expect(store.byArea('area-1')[0].id).toBe('dev-1')
    expect(store.onlineDevices.length).toBe(1)
    expect(store.onlineDevices[0].id).toBe('dev-1')
  })


  it('normalizes the backend online boolean into the frontend status field', async () => {
    const store = useDevicesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce([
      { id: 'dev-online', name: 'Online Tuya', integration_id: 'tuya_local', online: true, created_at: '', updated_at: '' },
      { id: 'dev-offline', name: 'Offline Tuya', integration_id: 'tuya_local', online: false, created_at: '', updated_at: '' }
    ])

    await store.fetchDevices()

    expect(store.devices.map(device => device.status)).toEqual(['online', 'offline'])
    expect(store.onlineDevices).toHaveLength(1)
    expect(store.offlineDevices).toHaveLength(1)
  })

  it('handles null API response gracefully without crashing', async () => {
    const store = useDevicesStore()
    vi.mocked(apiClient.get).mockResolvedValueOnce(null)

    await store.fetchDevices()
    expect(store.devices).toEqual([])
    expect(store.onlineDevices).toEqual([])
    expect(store.offlineDevices).toEqual([])
  })
})
