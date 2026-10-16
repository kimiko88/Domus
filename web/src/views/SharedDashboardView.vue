<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiClient } from '../api/client'

interface Widget {
  id: string
  type: string
  title: string
  entity_key?: string
  col_span?: number
  row_span?: number
  config?: Record<string, any>
}

interface KioskConfig {
  enabled: boolean
  screensaver_timeout_sec: number
  wake_on_presence: boolean
  show_clock: boolean
  show_weather: boolean
}

interface Dashboard {
  id: string
  title: string
  icon: string
  layout_json: string
  kiosk_config_json: string
}

interface ShareInfo {
  dashboard_id: string
  title: string
  read_only: boolean
  expires_at: string
  created_at: string
}

const route = useRoute()
const token = route.params.token as string

const loading = ref(true)
const error = ref<string | null>(null)
const dashboard = ref<Dashboard | null>(null)
const shareInfo = ref<ShareInfo | null>(null)
const widgets = ref<Widget[]>([])
const kiosk = ref<KioskConfig | null>(null)

async function loadSharedDashboard() {
  loading.value = true
  error.value = null
  try {
    const res = await apiClient.get<{ dashboard: Dashboard; share_info: ShareInfo }>(
      `/api/v1/dashboards/shared/${token}`
    )
    dashboard.value = res.dashboard
    shareInfo.value = res.share_info

    if (res.dashboard.layout_json) {
      try {
        widgets.value = JSON.parse(res.dashboard.layout_json)
      } catch {
        widgets.value = []
      }
    }
    if (res.dashboard.kiosk_config_json) {
      try {
        kiosk.value = JSON.parse(res.dashboard.kiosk_config_json)
      } catch {
        kiosk.value = null
      }
    }
  } catch (err: any) {
    error.value = err?.message || 'Token di condivisione non valido o scaduto.'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadSharedDashboard()
})
</script>

<template>
  <main class="shared-dashboard-container" id="main-content">
    <!-- Loading State -->
    <div v-if="loading" class="state-box">
      <div class="spinner"></div>
      <p>Caricamento dashboard condivisa...</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="state-box error-box">
      <div class="icon-warning">⚠️</div>
      <h2>Accesso Negato o Scaduto</h2>
      <p>{{ error }}</p>
      <router-link to="/login" class="btn-return">Accedi a Domus</router-link>
    </div>

    <!-- Dashboard Content -->
    <div v-else-if="dashboard" class="dashboard-content">
      <header class="shared-header">
        <div class="header-titles">
          <h1>{{ dashboard.title }}</h1>
          <span v-if="shareInfo?.read_only" class="badge-readonly">Sola Lettura</span>
          <span v-else class="badge-guest">Accesso Ospite</span>
        </div>
        <div class="expiry-indicator" v-if="shareInfo?.expires_at">
          <span>Scade il: {{ new Date(shareInfo.expires_at).toLocaleDateString() }}</span>
        </div>
      </header>

      <!-- Widgets Grid -->
      <section class="widgets-grid" aria-label="Widget Dashboard">
        <div
          v-for="w in widgets"
          :key="w.id"
          class="widget-card"
          :style="{
            gridColumn: `span ${Math.min(w.col_span || 1, 4)}`,
            gridRow: `span ${Math.min(w.row_span || 1, 3)}`
          }"
        >
          <div class="widget-header">
            <h3>{{ w.title }}</h3>
            <span class="widget-type">{{ w.type }}</span>
          </div>
          <div class="widget-body">
            <p v-if="w.entity_key" class="entity-key"><code>{{ w.entity_key }}</code></p>
            <div class="widget-placeholder">
              <span>Attivo</span>
            </div>
          </div>
        </div>

        <div v-if="widgets.length === 0" class="empty-state">
          <p>Nessun widget configurato in questa dashboard.</p>
        </div>
      </section>
    </div>
  </main>
</template>

<style scoped>
.shared-dashboard-container {
  min-height: 100vh;
  background: var(--bg-primary, #090d16);
  color: var(--text-primary, #f1f5f9);
  padding: 1.5rem;
  font-family: inherit;
}

.state-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  gap: 1rem;
  text-align: center;
}

.error-box {
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.25);
  border-radius: 16px;
  padding: 2.5rem;
  max-width: 480px;
  margin: 10vh auto;
}

.icon-warning {
  font-size: 2.5rem;
}

.spinner {
  width: 40px;
  height: 40px;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top-color: var(--primary, #06b6d4);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.btn-return {
  display: inline-block;
  margin-top: 1rem;
  padding: 0.6rem 1.4rem;
  background: var(--primary, #06b6d4);
  color: #000;
  font-weight: 600;
  border-radius: 8px;
  text-decoration: none;
}

.shared-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 1.25rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  margin-bottom: 1.5rem;
}

.header-titles {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header-titles h1 {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
}

.badge-readonly, .badge-guest {
  font-size: 0.75rem;
  padding: 0.25rem 0.6rem;
  border-radius: 20px;
  font-weight: 600;
}

.badge-readonly {
  background: rgba(245, 158, 11, 0.15);
  color: #fbbf24;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.badge-guest {
  background: rgba(6, 182, 212, 0.15);
  color: #22d3ee;
  border: 1px solid rgba(6, 182, 212, 0.3);
}

.expiry-indicator {
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.6);
}

.widgets-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}

@media (max-width: 900px) {
  .widgets-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 600px) {
  .widgets-grid {
    grid-template-columns: 1fr;
  }
}

.widget-card {
  background: rgba(30, 41, 59, 0.7);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 14px;
  padding: 1.2rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.widget-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.widget-header h3 {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.widget-type {
  font-size: 0.7rem;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.4);
}

.entity-key code {
  font-size: 0.8rem;
  color: var(--primary, #06b6d4);
}

.widget-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  color: #10b981;
  font-weight: 600;
}

.empty-state {
  grid-column: 1 / -1;
  text-align: center;
  padding: 3rem;
  color: rgba(255, 255, 255, 0.5);
}
</style>
