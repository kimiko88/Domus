<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from './stores/auth'
import { useWsStore } from './stores/ws'
import { useWebSocket } from './composables/useWebSocket'
import AppSidebar from './components/layout/AppSidebar.vue'
import AppTopbar from './components/layout/AppTopbar.vue'
import BaseToast from './components/ui/BaseToast.vue'
import AppTourModal from './components/modals/AppTourModal.vue'
import { useTourStore } from './stores/tour'
import { WifiOff } from '@lucide/vue'

const route = useRoute()
const authStore = useAuthStore()
const wsStore = useWsStore()
const tourStore = useTourStore()

const isSidebarOpen = ref(false)

// Hide sidebar and topbar on login, setup, and register pages
const isAuthPage = computed(() => {
  return route?.path === '/login' || route?.path === '/setup' || route?.path === '/register'
})

const pageTitle = computed(() => {
  return (route?.meta?.title as string) || 'Dashboard'
})

// Initialize WebSocket - autoConnect is false so unauthenticated users do not flood connection errors
const { isConnected, isConnecting, subscribe, connect, disconnect } = useWebSocket({
  autoConnect: false
})

// Connect WebSocket only when authenticated; disconnect on logout/unauthenticated
watch(
  () => authStore.isAuthenticated,
  (isAuth) => {
    if (isAuth) {
      connect()
    } else {
      disconnect()
    }
  },
  { immediate: true }
)

// Sync WS connection state with store
watch(
  isConnected,
  (connected) => {
    wsStore.setConnected(connected)
  },
  { immediate: true }
)

watch(
  isConnecting,
  (connecting) => {
    wsStore.setConnecting(connecting)
  },
  { immediate: true }
)

watch(isAuthPage, (isAuth) => {
  if (!isAuth) {
    tourStore.checkAutoStart()
  }
})

onMounted(() => {
  if (!isAuthPage.value) {
    tourStore.checkAutoStart()
  }

  // Subscribe to all WS events and route them to wsStore
  subscribe('*', (payload, raw) => {
    const eventType = raw.event_type || raw.type
    wsStore.handleIncomingEvent(eventType, payload)
  })
})
</script>

<template>
  <div class="app-layout">
    <BaseToast />
    <AppTourModal />

    <!-- Fullscreen Auth / Setup View -->
    <div v-if="isAuthPage" class="auth-wrapper">
      <RouterView />
    </div>

    <!-- Main Dashboard Layout -->
    <div v-else class="main-layout">
      <a href="#main-content" class="skip-link">Salta al contenuto principale</a>
      <AppSidebar
        :is-open="isSidebarOpen"
        :current-path="route?.path || '/'"
        @close="isSidebarOpen = false"
      />

      <div class="content-wrapper">
        <AppTopbar
          :title="pageTitle"
          @toggle-menu="isSidebarOpen = !isSidebarOpen"
        />

        <Transition name="fade-slide">
          <div
            v-if="!isAuthPage && authStore.isAuthenticated && !wsStore.isConnected"
            class="offline-banner"
            role="status"
            aria-live="polite"
          >
            <WifiOff :size="16" class="offline-icon" />
            <span class="offline-text">Connessione all'hub locale interrotta. Riconnessione automatica in corso...</span>
          </div>
        </Transition>

        <main id="main-content" class="page-content" tabindex="-1">
          <RouterView />
        </main>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app-layout {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.auth-wrapper {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-layout {
  display: flex;
  min-height: 100vh;
}

.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin-left: var(--sidebar-width);
  transition: margin-left var(--transition-normal);
}

@media (max-width: 768px) {
  .content-wrapper {
    margin-left: 0;
  }
}

.offline-banner {
  background: linear-gradient(90deg, rgba(245, 158, 11, 0.2) 0%, rgba(244, 63, 94, 0.2) 100%);
  border-bottom: 1px solid rgba(245, 158, 11, 0.4);
  color: #fcd34d;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px 16px;
  font-size: 13px;
  font-weight: 600;
  backdrop-filter: var(--blur-card);
  -webkit-backdrop-filter: var(--blur-card);
}

.offline-icon {
  flex-shrink: 0;
  animation: pulse-subtle 2s infinite;
}

.page-content {
  flex: 1;
  padding: 24px;
  max-width: 1400px;
  width: 100%;
  margin: 0 auto;
}

@media (max-width: 640px) {
  .page-content {
    padding: 16px;
  }
}
</style>
