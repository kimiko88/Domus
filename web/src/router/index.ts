import { createRouter, createMemoryHistory, createWebHistory, type Router, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

import DashboardView from '../views/DashboardView.vue'
import AreasView from '../views/AreasView.vue'
import DevicesView from '../views/DevicesView.vue'
import IntegrationsView from '../views/IntegrationsView.vue'
import AutomationsView from '../views/AutomationsView.vue'
import ScenesView from '../views/ScenesView.vue'
import HistoryView from '../views/HistoryView.vue'
import SettingsView from '../views/SettingsView.vue'
import LoginView from '../views/LoginView.vue'
import SetupView from '../views/SetupView.vue'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'dashboard',
    component: DashboardView,
    meta: { requiresAuth: true, title: 'Dashboard' }
  },
  {
    path: '/areas',
    name: 'areas',
    component: AreasView,
    meta: { requiresAuth: true, title: 'Aree' }
  },
  {
    path: '/devices',
    name: 'devices',
    component: DevicesView,
    meta: { requiresAuth: true, title: 'Dispositivi' }
  },
  {
    path: '/integrations',
    name: 'integrations',
    component: IntegrationsView,
    meta: { requiresAuth: true, title: 'Integrazioni' }
  },
  {
    path: '/automations',
    name: 'automations',
    component: AutomationsView,
    meta: { requiresAuth: true, title: 'Automazioni' }
  },
  {
    path: '/scenes',
    name: 'scenes',
    component: ScenesView,
    meta: { requiresAuth: true, title: 'Scene' }
  },
  {
    path: '/history',
    name: 'history',
    component: HistoryView,
    meta: { requiresAuth: true, title: 'Storico' }
  },
  {
    path: '/settings',
    name: 'settings',
    component: SettingsView,
    meta: { requiresAuth: true, title: 'Impostazioni' }
  },
  {
    path: '/login',
    name: 'login',
    component: LoginView,
    meta: { guestOnly: true, title: 'Accedi' }
  },
  {
    path: '/setup',
    name: 'setup',
    component: SetupView,
    meta: { guestOnly: true, title: 'Primo Avvio' }
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

export function createRouterInstance(isTest: boolean = false): Router {
  const router = createRouter({
    history: isTest || typeof window === 'undefined' ? createMemoryHistory() : createWebHistory(),
    routes
  })

  router.beforeEach(async (to, from, next) => {
    const authStore = useAuthStore()

    // 1. Check if setup is required
    let setupNeeded = authStore.isSetupRequired
    if (!setupNeeded && authStore.user === null) {
      setupNeeded = await authStore.checkSetup()
    }

    if (setupNeeded) {
      if (to.path !== '/setup') {
        return next('/setup')
      }
      return next()
    }

    // 2. Fetch current user if not already loaded and not in setup
    if (!authStore.user && to.meta.requiresAuth) {
      await authStore.fetchMe()
    }

    const isAuthenticated = authStore.isAuthenticated

    // 3. Protected route check
    if (to.meta.requiresAuth && !isAuthenticated) {
      return next('/login')
    }

    // 4. Guest-only route check (e.g. login/setup when already authenticated)
    if (to.meta.guestOnly && isAuthenticated && !setupNeeded) {
      return next('/')
    }

    next()
  })

  router.afterEach((to) => {
    if (typeof document !== 'undefined' && to.meta.title) {
      document.title = `${to.meta.title} — Domus Smart Home`
    }
  })

  return router
}

export const router = createRouterInstance(typeof window === 'undefined')
