<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import BaseButton from '../components/ui/BaseButton.vue'
import { User, Shield, Info, LogOut, Moon, HardDrive } from '@lucide/vue'

const authStore = useAuthStore()
const currentUser = computed(() => authStore.user)

async function handleLogout() {
  await authStore.logout()
}
</script>

<template>
  <div class="settings-view">
    <div class="header-row">
      <div>
        <h2 class="section-title">Impostazioni</h2>
        <p class="section-subtitle">Gestione profilo, sistema e preferenze generali</p>
      </div>
    </div>

    <div class="settings-grid">
      <!-- Profile Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <User :size="20" />
          </div>
          <h3 class="card-title">Profilo Utente</h3>
        </div>

        <div v-if="currentUser" class="profile-details">
          <div class="detail-row">
            <span class="detail-label">Nome Utente</span>
            <span class="detail-value font-mono">{{ currentUser.username }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Ruolo</span>
            <span class="role-badge">{{ currentUser.role }}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">ID Utente</span>
            <span class="detail-value font-mono text-dim">{{ currentUser.id }}</span>
          </div>

          <div class="action-row">
            <BaseButton
              variant="danger"
              class="btn-logout-settings"
              @click="handleLogout"
            >
              <LogOut :size="16" />
              <span>Termina Sessione (Logout)</span>
            </BaseButton>
          </div>
        </div>
      </div>

      <!-- System Info Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <HardDrive :size="20" />
          </div>
          <h3 class="card-title">Informazioni Sistema</h3>
        </div>

        <div class="profile-details">
          <div class="detail-row">
            <span class="detail-label">Versione Core</span>
            <span class="detail-value font-mono">v1.0.0 (Go modular monolith)</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Versione Web UI</span>
            <span class="detail-value font-mono">v0.6.0 (Vue 3 + Vite PWA)</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Database</span>
            <span class="detail-value font-mono">SQLite WAL mode</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Architettura target</span>
            <span class="detail-value font-mono">ARM64 / Raspberry Pi 4</span>
          </div>
        </div>
      </div>

      <!-- Appearance Card -->
      <div class="settings-card glass-card">
        <div class="card-header">
          <div class="icon-box">
            <Moon :size="20" />
          </div>
          <h3 class="card-title">Aspetto & Tema</h3>
        </div>

        <div class="profile-details">
          <div class="detail-row">
            <span class="detail-label">Tema attivo</span>
            <span class="detail-value">Dark Slate (Glassmorphism)</span>
          </div>
          <p class="theme-desc">
            Interfaccia ottimizzata per consumi ridotti su schermi OLED e contrasto visivo elevato in ambienti domestici.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.settings-view {
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

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
  gap: 20px;
}

.settings-card {
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-glass-card);
}

.card-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.icon-box {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--primary);
  flex-shrink: 0;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-main);
}

.profile-details {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-subtle);
}

.detail-label {
  font-size: 13px;
  color: var(--text-muted);
}

.detail-value {
  font-size: 13px;
  color: var(--text-main);
}

.font-mono {
  font-family: var(--font-mono);
}

.text-dim {
  color: var(--text-dim);
  font-size: 12px;
}

.role-badge {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  border: 1px solid rgba(14, 165, 233, 0.3);
}

.action-row {
  margin-top: 8px;
}

.theme-desc {
  font-size: 13px;
  color: var(--text-dim);
  line-height: 1.5;
}
</style>
