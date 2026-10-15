import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiClient } from '../api/client'
import type { AlarmControlPanel, AlarmState } from '../api/types'

export const useAlarmStore = defineStore('alarm', () => {
  const panel = ref<AlarmControlPanel | null>({
    id: 'alarm_main',
    name: 'Sistema Antifurto Domus',
    state: 'disarmed',
    entry_delay_seconds: 30,
    exit_delay_seconds: 45,
    trigger_delay_seconds: 0,
    perimeter_sensors: ['binary_sensor.porta_ingresso', 'binary_sensor.finestra_salotto'],
    interior_sensors: ['sensor.radar_mmwave_corridoio', 'binary_sensor.pir_salotto'],
    siren_entities: ['switch.sirena_esterna'],
    updated_at: new Date().toISOString()
  })

  const loading = ref<boolean>(false)
  const error = ref<string | null>(null)

  const isArmed = computed(() => {
    return panel.value?.state === 'armed_home' ||
      panel.value?.state === 'armed_away' ||
      panel.value?.state === 'armed_night'
  })

  const isTriggered = computed(() => panel.value?.state === 'triggered')
  const isPending = computed(() => panel.value?.state === 'pending')

  async function fetchPanel(): Promise<AlarmControlPanel | null> {
    loading.value = true
    error.value = null
    try {
      const data = await apiClient.get<AlarmControlPanel>('/api/v1/alarm')
      if (data) {
        panel.value = data
      }
      return panel.value
    } catch (err: any) {
      error.value = err?.message || 'Errore recupero centrale allarme'
      return panel.value
    } finally {
      loading.value = false
    }
  }

  async function arm(mode: 'armed_home' | 'armed_away' | 'armed_night', pin: string): Promise<AlarmControlPanel> {
    loading.value = true
    error.value = null
    try {
      const res = await apiClient.post<AlarmControlPanel>('/api/v1/alarm/arm', { mode, pin })
      panel.value = res
      return res
    } catch (err: any) {
      error.value = err?.message || 'Impossibile armare allarme'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function disarm(pin: string): Promise<AlarmControlPanel> {
    loading.value = true
    error.value = null
    try {
      const res = await apiClient.post<AlarmControlPanel>('/api/v1/alarm/disarm', { pin })
      panel.value = res
      return res
    } catch (err: any) {
      error.value = err?.message || 'PIN non valido'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function trigger(reason: string = 'SOS Manuale'): Promise<AlarmControlPanel> {
    loading.value = true
    error.value = null
    try {
      const res = await apiClient.post<AlarmControlPanel>('/api/v1/alarm/trigger', { reason })
      panel.value = res
      return res
    } catch (err: any) {
      error.value = err?.message || 'Impossibile innescare allarme'
      throw err
    } finally {
      loading.value = false
    }
  }

  async function updateConfig(config: Partial<AlarmControlPanel> & { pin?: string }): Promise<AlarmControlPanel> {
    loading.value = true
    error.value = null
    try {
      const res = await apiClient.put<AlarmControlPanel>('/api/v1/alarm/config', config)
      panel.value = res
      return res
    } catch (err: any) {
      error.value = err?.message || 'Errore salvataggio configurazione allarme'
      throw err
    } finally {
      loading.value = false
    }
  }

  function handleAlarmEvent(eventType: string, payload: any) {
    if (!panel.value) return
    if (eventType === 'alarm.state_changed' || eventType === 'alarm.pending_entry') {
      if (payload.state) panel.value.state = payload.state
      if (payload.last_changed_by) panel.value.last_changed_by = payload.last_changed_by
    } else if (eventType === 'alarm.triggered') {
      panel.value.state = 'triggered'
      if (payload.reason) panel.value.last_triggered_reason = payload.reason
      if (payload.triggered_at) panel.value.triggered_at = payload.triggered_at
    } else if (eventType === 'alarm.disarmed') {
      panel.value.state = 'disarmed'
      if (payload.user) panel.value.last_changed_by = payload.user
    }
    panel.value.updated_at = new Date().toISOString()
  }

  return {
    panel,
    loading,
    error,
    isArmed,
    isTriggered,
    isPending,
    fetchPanel,
    arm,
    disarm,
    trigger,
    updateConfig,
    handleAlarmEvent
  }
})
