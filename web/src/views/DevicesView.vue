<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDevicesStore } from '../stores/devices'
import { Cpu, Wifi, WifiOff } from '@lucide/vue'

const { t } = useI18n()
const devicesStore = useDevicesStore()

const filterStatus = ref<'all' | 'online' | 'offline'>('all')

onMounted(async () => {
  try {
    await devicesStore.fetchDevices()
  } catch (err) {
    console.warn('DevicesView fetchDevices warning:', err)
  }
})

const filteredDevices = computed(() => {
  if (filterStatus.value === 'online') {
    return devicesStore.onlineDevices || []
  }
  if (filterStatus.value === 'offline') {
    return devicesStore.offlineDevices || []
  }
  return devicesStore.devices || []
})

const totalDevicesCount = computed(() => (devicesStore.devices || []).length)
const onlineDevicesCount = computed(() => (devicesStore.onlineDevices || []).length)
const offlineDevicesCount = computed(() => (devicesStore.offlineDevices || []).length)
</script>

<template>
  <div class="devices-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">{{ t('devices.title') }}</h2>
        <p class="section-subtitle">{{ t('devices.subtitle') }}</p>
      </div>
    </div>

    <!-- Filters Row -->
    <div class="filters-row">
      <button
        :class="['filter-pill', { active: filterStatus === 'all' }]"
        @click="filterStatus = 'all'"
      >
        {{ t('devices.filterAll') }} ({{ totalDevicesCount }})
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'online' }]"
        @click="filterStatus = 'online'"
      >
        <Wifi :size="15" />
        {{ t('devices.filterOnline') }} ({{ onlineDevicesCount }})
      </button>
      <button
        :class="['filter-pill', { active: filterStatus === 'offline' }]"
        @click="filterStatus = 'offline'"
      >
        <WifiOff :size="15" />
        {{ t('devices.filterOffline') }} ({{ offlineDevicesCount }})
      </button>
    </div>

    <!-- Devices Grid -->
    <div v-if="filteredDevices.length > 0" class="devices-grid">
      <div
        v-for="device in filteredDevices"
        :key="device.id"
        :class="['device-card glass-card', { 'is-online': device.status === 'online' }]"
      >
        <div class="device-card-top">
          <div class="device-icon-box">
            <Cpu :size="22" />
          </div>
          <div class="device-info">
            <h3 class="device-name" :title="device.name">{{ device.name }}</h3>
            <div class="device-meta">
              <span v-if="device.manufacturer" class="meta-tag">{{ device.manufacturer }}</span>
              <span v-if="device.model" class="meta-tag">{{ device.model }}</span>
            </div>
          </div>
        </div>

        <div class="device-card-footer">
          <span class="integration-badge">{{ device.integration_id }}</span>
          <div class="status-indicator">
            <span class="status-dot" :class="{ online: device.status === 'online' }" />
            <span class="status-text">{{ device.status }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else class="empty-state glass-card">
      <Cpu :size="36" class="empty-icon" />
      <h3 class="empty-title">{{ t('devices.emptyTitle') }}</h3>
      <p class="empty-desc">
        {{ t('devices.emptyDesc') }}
      </p>
    </div>
  </div>
</template>

<style scoped>
.devices-view {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.section-subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 4px;
}

.filters-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-muted);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
}

.filter-pill:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.filter-pill.active {
  background: var(--primary);
  color: white;
  border-color: var(--primary);
  box-shadow: 0 0 12px var(--primary-glow);
}

.devices-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 18px;
}

.device-card {
  padding: 18px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
  transition: all var(--transition-normal);
  gap: 16px;
}

.device-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-card);
  box-shadow: var(--shadow-md);
}

.device-card-top {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.device-icon-box {
  width: 44px;
  height: 44px;
  border-radius: var(--radius-md);
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-dim);
  flex-shrink: 0;
}

.is-online .device-icon-box {
  color: var(--primary);
  background: rgba(14, 165, 233, 0.15);
}

.device-info {
  flex: 1;
  min-width: 0;
}

.device-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.device-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 6px;
}

.meta-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: var(--bg-surface);
  color: var(--text-dim);
  border: 1px solid var(--border-card);
}

.device-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-top: 12px;
  border-top: 1px solid var(--border-subtle);
}

.integration-badge {
  font-family: var(--font-mono);
  font-size: 11px;
  color: var(--text-dim);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  text-transform: capitalize;
  color: var(--text-dim);
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: var(--color-danger);
}

.status-dot.online {
  background-color: var(--color-success);
  box-shadow: 0 0 6px var(--color-success-glow);
}

.empty-state {
  padding: 48px 24px;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  max-width: 500px;
  margin: 30px auto;
  border-radius: var(--radius-xl);
}

.empty-icon {
  color: var(--primary);
  margin-bottom: 16px;
  opacity: 0.8;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-main);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 8px;
  line-height: 1.5;
}
</style>
