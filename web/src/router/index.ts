import { createRouter, createMemoryHistory, createWebHistory, type Router, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '../stores/auth'

import DashboardView from '../views/DashboardView.vue'
import AreasView from '../views/AreasView.vue'
import DevicesView from '../views/DevicesView.vue'
import IntegrationsView from '../views/IntegrationsView.vue'
import AutomationsView from '../views/AutomationsView.vue'
import ScenesView from '../views/ScenesView.vue'
import HistoryView from '../views/HistoryView.vue'
import AuditView from '../views/AuditView.vue'
import FamilyView from '../views/FamilyView.vue'
import SettingsView from '../views/SettingsView.vue'
import LoginView from '../views/LoginView.vue'
import RegisterView from '../views/RegisterView.vue'
import SetupView from '../views/SetupView.vue'
import FloorplanView from '../views/FloorplanView.vue'
import ChoresView from '../views/ChoresView.vue'
import CustomDashboardView from '../views/CustomDashboardView.vue'
import EnergyDashboardView from '../views/EnergyDashboardView.vue'
import HelpersView from '../views/HelpersView.vue'
import CamerasView from '../views/CamerasView.vue'
import ScheduleView from '../views/ScheduleView.vue'
import MediaPlayerView from '../views/MediaPlayerView.vue'
import MeshMapView from '../views/MeshMapView.vue'
import StoreView from '../views/StoreView.vue'
import ShoppingListView from '../views/ShoppingListView.vue'
import GuideView from '../views/GuideView.vue'
import LogbookView from '../views/LogbookView.vue'
import ZonesView from '../views/ZonesView.vue'
import TemplateEntitiesView from '../views/TemplateEntitiesView.vue'
import SharedDashboardView from '../views/SharedDashboardView.vue'

export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'dashboard',
    component: DashboardView,
    meta: { requiresAuth: true, title: 'Dashboard' }
  },
  {
    path: '/custom-dashboards',
    name: 'custom-dashboards',
    component: CustomDashboardView,
    meta: { requiresAuth: true, title: 'Dashboard Tablet' }
  },
  {
    path: '/share/:token',
    name: 'shared-dashboard',
    component: SharedDashboardView,
    meta: { requiresAuth: false, title: 'Dashboard Condivisa' }
  },
  {
    path: '/energy',
    name: 'energy',
    component: EnergyDashboardView,
    meta: { requiresAuth: true, title: 'Energia & Solare' }
  },
  {
    path: '/helpers',
    name: 'helpers',
    component: HelpersView,
    meta: { requiresAuth: true, title: 'Helper Virtuali' }
  },
  {
    path: '/cameras',
    name: 'cameras',
    component: CamerasView,
    meta: { requiresAuth: true, title: 'Telecamere' }
  },
  {
    path: '/schedules',
    name: 'schedules',
    component: ScheduleView,
    meta: { requiresAuth: true, title: 'Programmazione' }
  },
  {
    path: '/media',
    name: 'media',
    component: MediaPlayerView,
    meta: { requiresAuth: true, title: 'Audio Multi-room' }
  },
  {
    path: '/mesh',
    name: 'mesh',
    component: MeshMapView,
    meta: { requiresAuth: true, title: 'Mappa Mesh' }
  },
  {
    path: '/store',
    name: 'store',
    component: StoreView,
    meta: { requiresAuth: true, title: 'Domus Store' }
  },
  {
    path: '/floorplan',
    name: 'floorplan',
    component: FloorplanView,
    meta: { requiresAuth: true, title: 'Mappa Piani' }
  },
  {
    path: '/chores',
    name: 'chores',
    component: ChoresView,
    meta: { requiresAuth: true, title: 'Faccende' }
  },
  {
    path: '/shopping',
    name: 'shopping',
    component: ShoppingListView,
    meta: { requiresAuth: true, title: 'Lista Spesa' }
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
    path: '/audit',
    name: 'audit',
    component: AuditView,
    meta: { requiresAuth: true, title: 'Audit' }
  },
  {
    path: '/family',
    name: 'family',
    component: FamilyView,
    meta: { requiresAuth: true, title: 'Famiglia' }
  },
  {
    path: '/guide',
    name: 'guide',
    component: GuideView,
    meta: { requiresAuth: true, title: 'Guida & Manuale' }
  },
  {
    path: '/logbook',
    name: 'logbook',
    component: LogbookView,
    meta: { requiresAuth: true, title: 'Registro Eventi' }
  },
  {
    path: '/zones',
    name: 'zones',
    component: ZonesView,
    meta: { requiresAuth: true, title: 'Zone & Geofencing' }
  },
  {
    path: '/templates',
    name: 'templates',
    component: TemplateEntitiesView,
    meta: { requiresAuth: true, title: 'Entità Template' }
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
    path: '/register',
    name: 'register',
    component: RegisterView,
    meta: { guestOnly: true, title: 'Registrazione' }
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
