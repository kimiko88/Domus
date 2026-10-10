import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAlarmStore } from './alarm'
import { apiClient } from '../api/client'

describe('useAlarmStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('initializes with default disarmed state', () => {
    const store = useAlarmStore()
    expect(store.panel?.state).toBe('disarmed')
    expect(store.isArmed).toBe(false)
    expect(store.isTriggered).toBe(false)
    expect(store.isPending).toBe(false)
  })

  it('fetches alarm panel from backend API', async () => {
    const mockPanel = {
      id: 'alarm_main',
      name: 'Sistema Antifurto Domus',
      state: 'armed_away',
      entry_delay_seconds: 30,
      exit_delay_seconds: 45,
      trigger_delay_seconds: 0,
      perimeter_sensors: ['binary_sensor.porta'],
      interior_sensors: ['sensor.radar'],
      siren_entities: ['switch.sirena'],
      updated_at: '2026-10-10T12:00:00Z'
    }

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockPanel as any)

    const store = useAlarmStore()
    const result = await store.fetchPanel()

    expect(result?.state).toBe('armed_away')
    expect(store.isArmed).toBe(true)
  })

  it('arms alarm in armed_home mode with PIN', async () => {
    const mockRes = {
      id: 'alarm_main',
      name: 'Sistema Antifurto Domus',
      state: 'armed_home',
      updated_at: '2026-10-10T12:00:00Z'
    }

    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(mockRes as any)

    const store = useAlarmStore()
    const result = await store.arm('armed_home', '1234')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/alarm/arm', { mode: 'armed_home', pin: '1234' })
    expect(result.state).toBe('armed_home')
    expect(store.isArmed).toBe(true)
  })

  it('disarms alarm with valid PIN', async () => {
    const mockRes = {
      id: 'alarm_main',
      name: 'Sistema Antifurto Domus',
      state: 'disarmed',
      updated_at: '2026-10-10T12:00:00Z'
    }

    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(mockRes as any)

    const store = useAlarmStore()
    const result = await store.disarm('1234')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/alarm/disarm', { pin: '1234' })
    expect(result.state).toBe('disarmed')
    expect(store.isArmed).toBe(false)
  })

  it('triggers panic alarm and sets triggered state', async () => {
    const mockRes = {
      id: 'alarm_main',
      name: 'Sistema Antifurto Domus',
      state: 'triggered',
      last_triggered_reason: 'SOS Panico',
      updated_at: '2026-10-10T12:00:00Z'
    }

    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(mockRes as any)

    const store = useAlarmStore()
    const result = await store.trigger('SOS Panico')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/alarm/trigger', { reason: 'SOS Panico' })
    expect(result.state).toBe('triggered')
    expect(store.isTriggered).toBe(true)
  })
})
