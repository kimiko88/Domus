import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAutomationsStore } from './automations'
import { apiClient } from '../api/client'
import type { Automation, AutomationListResponse, AutomationRunsResponse } from '../api/types'

describe('useAutomationsStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('fetches automations and populates list', async () => {
    const mockResponse: AutomationListResponse = {
      automations: [
        {
          id: 'auto_1',
          name: 'Night Lights Off',
          enabled: true,
          execution_mode: 'single',
          cooldown_seconds: 0,
          triggers: [{ platform: 'time', at: '23:00:00' }],
          conditions: [],
          actions: [{ type: 'service', service: 'turn_off', target_entity_key: 'light.living_room' }],
          created_at: '2026-10-04T00:00:00Z',
          updated_at: '2026-10-04T00:00:00Z'
        }
      ],
      total: 1
    }

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockResponse)

    const store = useAutomationsStore()
    const result = await store.fetchAutomations()

    expect(result).toHaveLength(1)
    expect(store.automations).toHaveLength(1)
    expect(store.automations[0].name).toBe('Night Lights Off')
    expect(store.loading).toBe(false)
  })

  it('creates an automation and adds it to store', async () => {
    const newAuto: Partial<Automation> = {
      name: 'Motion Light',
      enabled: true,
      execution_mode: 'single',
      cooldown_seconds: 10,
      triggers: [{ platform: 'state', entity_key: 'binary_sensor.hallway', to: 'on' }],
      conditions: [],
      actions: [{ type: 'service', service: 'turn_on', target_entity_key: 'light.hallway' }]
    }

    const created: Automation = {
      ...(newAuto as any),
      id: 'auto_created_1',
      created_at: '2026-10-04T01:00:00Z',
      updated_at: '2026-10-04T01:00:00Z'
    }

    vi.spyOn(apiClient, 'post').mockResolvedValue(created)

    const store = useAutomationsStore()
    const result = await store.createAutomation(newAuto)

    expect(result.id).toBe('auto_created_1')
    expect(store.automations).toContainEqual(created)
  })

  it('updates an existing automation in place', async () => {
    const store = useAutomationsStore()
    store.automations = [
      {
        id: 'auto_1',
        name: 'Old Name',
        enabled: true,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [{ platform: 'time', at: '22:00:00' }],
        conditions: [],
        actions: [{ type: 'service', service: 'turn_off', target_entity_key: 'light.all' }],
        created_at: '',
        updated_at: ''
      }
    ]

    const updated = { ...store.automations[0], name: 'Updated Name' }
    vi.spyOn(apiClient, 'put').mockResolvedValue(updated)

    await store.updateAutomation('auto_1', { name: 'Updated Name' })

    expect(store.automations[0].name).toBe('Updated Name')
  })

  it('toggles automation enabled state', async () => {
    const store = useAutomationsStore()
    store.automations = [
      {
        id: 'auto_toggle',
        name: 'Auto Toggle',
        enabled: true,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [{ platform: 'time', at: '20:00:00' }],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]

    const putSpy = vi.spyOn(apiClient, 'put').mockImplementation(async (_path, body) => {
      return { ...store.automations[0], ...body }
    })

    await store.toggleAutomation('auto_toggle')

    expect(store.automations[0].enabled).toBe(false)
    expect(putSpy).toHaveBeenCalledWith('/api/v1/automations/auto_toggle', { enabled: false })
  })

  it('deletes automation from store', async () => {
    const store = useAutomationsStore()
    store.automations = [
      {
        id: 'auto_del',
        name: 'To Delete',
        enabled: false,
        execution_mode: 'single',
        cooldown_seconds: 0,
        triggers: [],
        conditions: [],
        actions: [],
        created_at: '',
        updated_at: ''
      }
    ]

    vi.spyOn(apiClient, 'delete').mockResolvedValue({})

    await store.deleteAutomation('auto_del')

    expect(store.automations).toHaveLength(0)
  })

  it('triggers an automation manually', async () => {
    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue({ status: 'accepted' })

    const store = useAutomationsStore()
    await store.triggerAutomation('auto_1')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/automations/auto_1/trigger')
  })

  it('fetches run history for an automation', async () => {
    const mockRunsResponse: AutomationRunsResponse = {
      runs: [
        {
          id: 'run_1',
          automation_id: 'auto_1',
          started_at: '2026-10-04T02:00:00Z',
          finished_at: '2026-10-04T02:00:00Z',
          status: 'completed',
          trigger_context: '{"entity_key":"light.living_room"}',
          log: 'Success',
          duration_ms: 45
        }
      ],
      total: 1
    }

    vi.spyOn(apiClient, 'get').mockResolvedValue(mockRunsResponse)

    const store = useAutomationsStore()
    const runs = await store.fetchRuns('auto_1')

    expect(runs).toHaveLength(1)
    expect(store.runs['auto_1']).toHaveLength(1)
    expect(store.runs['auto_1'][0].status).toBe('completed')
  })
})
