import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useSystemStore } from './system'
import { apiClient } from '@/api/client'

describe('useSystemStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('fetches system metrics and formats uptime correctly', async () => {
    const mockMetrics = {
      status: 'ok',
      uptime_s: 3665, // 1h 1m 5s
      entity_count: 14,
      device_count: 6,
      mqtt_connected: true,
      ws_clients: 2,
      db_size_bytes: 524288
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockMetrics)

    const store = useSystemStore()
    const res = await store.fetchMetrics()

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/system/metrics')
    expect(res.uptime_s).toBe(3665)
    expect(store.metrics).toEqual(mockMetrics)
    expect(store.formattedUptime).toBe('1h 1m 5s')
  })

  it('formats shorter uptime properly', async () => {
    const mockMetrics = {
      status: 'ok',
      uptime_s: 42,
      entity_count: 0,
      device_count: 0,
      mqtt_connected: false,
      ws_clients: 0,
      db_size_bytes: 0
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockMetrics)

    const store = useSystemStore()
    await store.fetchMetrics()

    expect(store.formattedUptime).toBe('42s')
  })

  it('fetches OS diagnostics and formats telemetry accurately', async () => {
    const mockDiag = {
      soc_temp_c: 48.5,
      cpu_load_1m: 0.25,
      cpu_load_5m: 0.18,
      cpu_load_15m: 0.12,
      cpu_cores: 4,
      memory_total_bytes: 4 * 1024 * 1024 * 1024, // 4096 MB
      memory_available_bytes: 2.5 * 1024 * 1024 * 1024,
      memory_used_bytes: 1.5 * 1024 * 1024 * 1024, // 1536 MB
      memory_used_percent: 37.5,
      disk_total_bytes: 32 * 1024 * 1024 * 1024, // 32 GB
      disk_free_bytes: 20 * 1024 * 1024 * 1024,
      disk_used_bytes: 12 * 1024 * 1024 * 1024, // 12 GB
      disk_used_percent: 37.5,
      hostname: 'domus',
      os_version: 'Domus OS 1.0.0',
      kernel_version: '6.6.28-v8',
      architecture: 'arm64',
      uptime_seconds: 12000,
      active_slot: 'domus-root-a',
      booted_slot: 'domus-root-a',
      slot_confirmed: true,
      rauc_status: 'good',
      core_healthy: true,
      running_containers: 2,
      total_containers: 2,
      installed_apps: 1,
      maintenance_mode: false,
      timestamp: '2026-10-04T12:00:00Z'
    }

    vi.spyOn(apiClient, 'get').mockResolvedValueOnce(mockDiag)

    const store = useSystemStore()
    await store.fetchOSDiagnostics()

    expect(apiClient.get).toHaveBeenCalledWith('/api/v1/system/diagnostics')
    expect(store.osDiagnostics).toEqual(mockDiag)
    expect(store.formattedMemory).toContain('MB')
    expect(store.formattedDisk).toContain('GB')
    expect(store.tempStatusClass).toBe('temp-normal')
  })

  it('manages maintenance mode lifecycle (enable and disable)', async () => {
    const activeStatus = {
      active: true,
      expires_at: '2026-10-04T13:00:00Z',
      remaining_seconds: 1800,
      ssh_port: 22,
      key_fingerprint: 'SHA256:abc123'
    }

    const inactiveStatus = {
      active: false,
      remaining_seconds: 0,
      ssh_port: 22
    }

    vi.spyOn(apiClient, 'post').mockResolvedValueOnce(activeStatus)

    const store = useSystemStore()
    await store.enableMaintenance({ duration_minutes: 30, authorized_key: 'ssh-ed25519 AAA...' })

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/system/maintenance/enable', {
      duration_minutes: 30,
      authorized_key: 'ssh-ed25519 AAA...'
    })
    expect(store.maintenanceStatus?.active).toBe(true)
    expect(store.formattedMaintenanceRemaining).toBe('30m 0s')

    // Disable
    vi.spyOn(apiClient, 'post').mockResolvedValueOnce(inactiveStatus)
    await store.disableMaintenance()

    expect(apiClient.post).toHaveBeenCalledWith('/api/v1/system/maintenance/disable', {})
    expect(store.maintenanceStatus?.active).toBe(false)
  })
})
