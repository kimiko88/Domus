<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useFamilyStore } from '../stores/family'
import { apiClient } from '../api/client'
import type { UserRole, UserPermissions, Area, Entity } from '../api/types'
import BaseButton from '../components/ui/BaseButton.vue'
import {
  Users,
  UserPlus,
  Shield,
  Sliders,
  Eye,
  Copy,
  Check,
  Trash2,
  Clock,
  X,
  Link,
  KeyRound,
  Baby
} from '@lucide/vue'

const { t } = useI18n()
const authStore = useAuthStore()
const familyStore = useFamilyStore()

const isAdmin = computed(() => authStore.isAdmin)
const currentUserId = computed(() => authStore.user?.id)

// Invite Modal State
const isInviteModalOpen = ref(false)
const selectedRole = ref<UserRole>('operator')
const selectedExpiresHours = ref<number>(48)
const createdInviteUrl = ref<string>('')
const copySuccess = ref(false)
const roleChangeSuccess = ref<string | null>(null)
const errorMsg = ref<string | null>(null)

// Guest & Kiosk Modal State
const isGuestModalOpen = ref(false)
const guestUsername = ref('')
const guestPin = ref('')
const guestHours = ref(24)
const guestKiosk = ref(true)
const guests = computed(() => familyStore.guests)

const availableAreas = ref<Area[]>([])
const availableEntities = ref<Entity[]>([])
const availableDashboards = ref<Array<{ id: string; title: string }>>([])
const isPermissionsModalOpen = ref(false)
const isSavingPermissions = ref(false)
const permissionsMember = ref<{ id: string; username: string } | null>(null)
const allAreas = ref(true)
const allEntities = ref(true)
const allDashboards = ref(true)
const allViews = ref(true)
const selectedAreaIDs = ref<string[]>([])
const selectedEntityIDs = ref<string[]>([])
const selectedDashboardIDs = ref<string[]>([])
const selectedViews = ref<string[]>([])
const canControlDevices = ref(false)
const canManageAutomations = ref(false)
const canViewCameras = ref(false)
const permissionViews = [
  { path: '/', labelKey: 'nav.dashboard' },
  { path: '/areas', labelKey: 'nav.areas' },
  { path: '/devices', labelKey: 'nav.devices' },
  { path: '/integrations', labelKey: 'nav.integrations' },
  { path: '/energy', labelKey: 'nav.energy' },
  { path: '/climate/predictive', labelKey: 'nav.predictiveClimate' },
  { path: '/automations', labelKey: 'nav.automations' },
  { path: '/scenes', labelKey: 'nav.scenes' },
  { path: '/schedules', labelKey: 'nav.schedules' },
  { path: '/history', labelKey: 'nav.history' },
  { path: '/notifications', labelKey: 'family.permissions.notifications' },
  { path: '/cameras', labelKey: 'nav.cameras' },
  { path: '/media', labelKey: 'nav.media' },
  { path: '/chores', labelKey: 'nav.chores' },
  { path: '/shopping', labelKey: 'nav.shopping' },
  { path: '/pantry', labelKey: 'nav.pantry' },
  { path: '/medicines', labelKey: 'nav.medicines' },
  { path: '/finance', labelKey: 'nav.finance' },
  { path: '/family', labelKey: 'nav.family' },
  { path: '/custom-dashboards', labelKey: 'nav.customDashboards' },
  { path: '/guide', labelKey: 'nav.guide' },
  { path: '/logbook', labelKey: 'nav.logbook' }
]

async function loadData() {
  try {
    await familyStore.fetchMembers()
    if (isAdmin.value) {
      const [areas, entities, dashboards] = await Promise.all([
        apiClient.get<Area[]>('/api/v1/areas'),
        apiClient.get<Entity[]>('/api/v1/entities'),
        apiClient.get<Array<{ id: string; title: string }>>('/api/v1/dashboards')
      ])
      availableAreas.value = areas || []
      availableEntities.value = entities || []
      availableDashboards.value = dashboards || []
    }
    if (isAdmin.value) {
      await familyStore.fetchInvites()
      try {
        await familyStore.fetchGuests()
      } catch {}
    }
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.load')
  }
}

function openGuestModal() {
  guestUsername.value = ''
  guestPin.value = ''
  guestHours.value = 24
  guestKiosk.value = true
  isGuestModalOpen.value = true
}

function closeGuestModal() {
  isGuestModalOpen.value = false
}

async function handleCreateGuest() {
  if (!guestUsername.value || !guestPin.value) {
    errorMsg.value = t('family.errors.guestRequired')
    return
  }
  try {
    errorMsg.value = null
    await familyStore.createGuest(guestUsername.value, guestPin.value, guestHours.value, guestKiosk.value)
    closeGuestModal()
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.createGuest')
  }
}

function openInviteModal() {
  selectedRole.value = 'operator'
  selectedExpiresHours.value = 48
  createdInviteUrl.value = ''
  copySuccess.value = false
  isInviteModalOpen.value = true
}

function closeInviteModal() {
  isInviteModalOpen.value = false
  createdInviteUrl.value = ''
}

async function handleGenerateInvite() {
  try {
    errorMsg.value = null
    const res = await familyStore.createInvite(selectedRole.value, selectedExpiresHours.value)
    const base = typeof window !== 'undefined' ? window.location.origin : ''
    createdInviteUrl.value = `${base}${res.invite_url}`
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.generateInvite')
  }
}

async function handleCopy(url: string) {
  try {
    if (navigator?.clipboard) {
      await navigator.clipboard.writeText(url)
    }
    copySuccess.value = true
    setTimeout(() => {
      copySuccess.value = false
    }, 2500)
  } catch (err) {
    console.warn('Clipboard copy failed:', err)
  }
}

function openPermissions(member: { id: string; username: string; permissions?: UserPermissions }) {
  const permissions = member.permissions
  permissionsMember.value = { id: member.id, username: member.username }
  allAreas.value = permissions?.allowed_area_ids == null
  allEntities.value = permissions?.allowed_entity_ids == null
  allDashboards.value = permissions?.allowed_dashboard_ids == null
  allViews.value = permissions?.allowed_views == null
  selectedAreaIDs.value = [...(permissions?.allowed_area_ids || [])]
  selectedEntityIDs.value = [...(permissions?.allowed_entity_ids || [])]
  selectedDashboardIDs.value = [...(permissions?.allowed_dashboard_ids || [])]
  selectedViews.value = [...(permissions?.allowed_views || [])]
  canControlDevices.value = permissions?.can_control_devices ?? false
  canManageAutomations.value = (permissions?.can_manage_automations ?? false) && canControlDevices.value && allAreas.value && allEntities.value
  canViewCameras.value = permissions?.can_view_cameras ?? false
  isPermissionsModalOpen.value = true
}

function closePermissions() {
  isPermissionsModalOpen.value = false
  permissionsMember.value = null
}

async function savePermissions() {
  if (!permissionsMember.value) return
  isSavingPermissions.value = true
  errorMsg.value = null
  const payload: UserPermissions = {
    role: familyStore.members.find(member => member.id === permissionsMember.value?.id)?.role || 'viewer',
    allowed_area_ids: allAreas.value ? null : [...selectedAreaIDs.value],
    allowed_entity_ids: allEntities.value ? null : [...selectedEntityIDs.value],
    allowed_dashboard_ids: allDashboards.value ? null : [...selectedDashboardIDs.value],
    allowed_views: allViews.value ? null : [...selectedViews.value],
    can_control_devices: canControlDevices.value,
    can_manage_automations: canManageAutomations.value && canControlDevices.value && allAreas.value && allEntities.value,
    can_manage_settings: false,
    can_view_cameras: canViewCameras.value
  }
  try {
    await familyStore.updateMemberPermissions(permissionsMember.value.id, payload)
    closePermissions()
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.updatePermissions')
  } finally {
    isSavingPermissions.value = false
  }
}

async function handleRoleChange(memberId: string, newRole: UserRole) {
  try {
    errorMsg.value = null
    roleChangeSuccess.value = null
    await familyStore.updateMemberRole(memberId, newRole)
    roleChangeSuccess.value = memberId
    setTimeout(() => {
      roleChangeSuccess.value = null
    }, 2500)
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.updateRole')
  }
}

async function handleRemoveMember(memberId: string, username: string) {
  if (!confirm(t('family.confirmRemove') + ` (${username})`)) {
    return
  }
  try {
    errorMsg.value = null
    await familyStore.removeMember(memberId)
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.removeMember')
  }
}

async function handleRevokeInvite(inviteId: string) {
  try {
    errorMsg.value = null
    await familyStore.revokeInvite(inviteId)
  } catch (err: any) {
    errorMsg.value = err?.message || t('family.errors.revokeInvite')
  }
}

function formatDate(iso: string): string {
  if (!iso) return '-'
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    })
  } catch {
    return iso
  }
}

function getInitials(name: string): string {
  if (!name) return '?'
  return name.slice(0, 2).toUpperCase()
}

function getInviteLink(token: string): string {
  const base = typeof window !== 'undefined' ? window.location.origin : ''
  return `${base}/register?token=${token}`
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="family-view">
    <!-- Header -->
    <div class="header-row">
      <div>
        <div class="title-with-icon">
          <div class="icon-bubble">
            <Users :size="24" />
          </div>
          <div>
            <h2 class="section-title">{{ t('family.title') }}</h2>
            <p class="section-subtitle">{{ t('family.subtitle') }}</p>
          </div>
        </div>
      </div>
      <div v-if="isAdmin">
        <BaseButton
          variant="primary"
          class="invite-btn"
          @click="openInviteModal"
        >
          <UserPlus :size="18" />
          <span>{{ t('family.inviteButton') }}</span>
        </BaseButton>
      </div>
    </div>

    <!-- Error Alert -->
    <div v-if="errorMsg" class="alert error-alert">
      <span>{{ errorMsg }}</span>
      <button class="close-alert-btn" @click="errorMsg = null">
        <X :size="16" />
      </button>
    </div>

    <!-- Members Section -->
    <div class="section-card glass-card">
      <div class="card-header">
        <h3 class="card-title">{{ t('family.membersTitle') }}</h3>
        <span class="count-badge">{{ familyStore.members.length }}</span>
      </div>

      <div v-if="familyStore.members.length === 0" class="empty-state">
        <Users :size="48" class="empty-icon" />
        <p>{{ t('family.noMembers') }}</p>
      </div>

      <div v-else class="members-grid">
        <div
          v-for="member in familyStore.members"
          :key="member.id"
          class="member-card"
        >
          <div class="member-top">
            <div class="avatar-circle">
              {{ getInitials(member.username) }}
            </div>
            <div class="member-meta">
              <div class="username-row">
                <span class="username">{{ member.username }}</span>
                <span v-if="member.id === currentUserId" class="self-tag">{{ t('family.view.you') }}</span>
              </div>
              <span class="email">{{ member.email || '—' }}</span>
            </div>
            <div class="role-badge" :class="`role-${member.role}`">
              <Shield v-if="member.role === 'admin'" :size="14" />
              <Sliders v-else-if="member.role === 'operator'" :size="14" />
              <Baby v-else-if="member.role === 'kid'" :size="14" />
              <Eye v-else :size="14" />
              <span>{{ t(`family.roles.${member.role}`) }}</span>
            </div>
          </div>

          <div class="member-footer">
            <span class="joined-date">
              <Clock :size="13" />
              {{ t('family.joinedAt') }} {{ formatDate(member.created_at) }}
            </span>

            <!-- Admin actions -->
            <div v-if="isAdmin" class="member-actions">
              <select
                class="role-select"
                :value="member.role"
                :disabled="member.id === currentUserId"
                @change="handleRoleChange(member.id, ($event.target as HTMLSelectElement).value as UserRole)"
              >
                <option value="admin">{{ t('family.roles.admin') }}</option>
                <option value="operator">{{ t('family.roles.operator') }}</option>
                <option value="viewer">{{ t('family.roles.viewer') }}</option>
                <option value="guest">{{ t('family.roles.guest') }}</option>
                <option value="kid">{{ t('family.roles.kid') }}</option>
              </select>

              <button
                v-if="member.id !== currentUserId && member.role !== 'admin'"
                class="permissions-btn"
                :title="t('family.permissions.manage')"
                @click="openPermissions(member)"
              >
                <Sliders :size="16" />
              </button>
              <button
                v-if="member.id !== currentUserId"
                class="remove-btn"
                :title="t('family.removeMember')"
                @click="handleRemoveMember(member.id, member.username)"
              >
                <Trash2 :size="16" />
              </button>
            </div>
          </div>

          <div v-if="roleChangeSuccess === member.id" class="toast-inline">
            <Check :size="14" /> Ruolo aggiornato
          </div>
        </div>
      </div>
    </div>

    <!-- Pending Invites Section (Admin only) -->
    <div v-if="isAdmin" class="section-card glass-card invites-section">
      <div class="card-header">
        <h3 class="card-title">{{ t('family.invitesTitle') }}</h3>
        <span class="count-badge">{{ familyStore.invites.length }}</span>
      </div>

      <div v-if="familyStore.invites.length === 0" class="empty-state">
        <Clock :size="40" class="empty-icon" />
        <p>{{ t('family.noPendingInvites') }}</p>
      </div>

      <div v-else class="invites-list">
        <div
          v-for="inv in familyStore.invites"
          :key="inv.id"
          class="invite-row"
        >
          <div class="invite-info">
            <div class="role-badge" :class="`role-${inv.role}`">
              <Shield v-if="inv.role === 'admin'" :size="14" />
              <Sliders v-else-if="inv.role === 'operator'" :size="14" />
              <Baby v-else-if="inv.role === 'kid'" :size="14" />
              <Eye v-else :size="14" />
              <span>{{ t(`family.roles.${inv.role}`) }}</span>
            </div>
            <div class="invite-details">
              <span class="invite-token">{{ t('family.tokenLabel') }}: {{ inv.token.slice(0, 12) }}...</span>
              <span class="invite-expiry">
                {{ t('family.expiresAt') }}: {{ formatDate(inv.expires_at) }}
              </span>
            </div>
          </div>

          <div class="invite-actions">
            <button
              class="copy-btn action-icon-btn"
              :title="t('family.copyLink')"
              @click="handleCopy(getInviteLink(inv.token))"
            >
              <Copy :size="16" />
            </button>
            <button
              class="revoke-btn action-icon-btn"
              :title="t('family.revoke')"
              @click="handleRevokeInvite(inv.id)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Guest Profiles & Kiosk PIN Section (Admin only) -->
    <div v-if="isAdmin" class="section-card glass-card guests-section">
      <div class="section-header">
        <div class="section-header-left">
          <KeyRound :size="18" class="section-icon" />
          <div>
            <h3 class="section-subtitle">{{ t('family.view.guestProfilesTitle') }}</h3>
            <p class="section-desc">{{ t('family.view.guestProfilesDesc') }}</p>
          </div>
        </div>
        <BaseButton
          variant="secondary"
          size="sm"
          class="guest-btn"
          @click="openGuestModal"
        >
          <UserPlus :size="16" />
          {{ t('family.view.createGuestButton') }}
        </BaseButton>
      </div>

      <div v-if="guests.length === 0" class="empty-state">
        <KeyRound :size="32" class="empty-icon" />
        <p class="empty-text">{{ t('family.view.guestProfilesEmpty') }}</p>
      </div>

      <div v-else class="invites-list">
        <div
          v-for="guest in guests"
          :key="guest.id"
          class="invite-row"
        >
          <div class="invite-info">
            <div class="invite-role-badge role-viewer">
              <KeyRound :size="14" />
              <span>{{ t(guest.kiosk_mode ? 'family.view.guestKioskLabel' : 'family.view.guestLabel') }}</span>
            </div>
            <div class="invite-meta">
              <span class="invite-user"><strong>{{ guest.username }}</strong></span>
              <span v-if="guest.pin_code" class="invite-pin">{{ t('family.view.pinLabel') }}: <code>{{ guest.pin_code }}</code></span>
              <span v-if="guest.guest_expires_at" class="invite-expires">
                {{ t('family.view.expiresLabel') }}: {{ formatDate(guest.guest_expires_at) }}
              </span>
            </div>
          </div>

          <div class="invite-actions">
            <button
              class="revoke-btn action-icon-btn"
              :title="t('family.view.removeGuest')"
              @click="handleRemoveMember(guest.id, guest.username)"
            >
              <Trash2 :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Guest Modal -->
    <div v-if="isGuestModalOpen" class="modal-backdrop" @click.self="closeGuestModal">
      <div class="modal-card glass-card">
        <div class="modal-header">
          <div class="modal-title-box">
            <KeyRound :size="20" />
            <h3 class="modal-title">{{ t('family.view.createGuestProfileTitle') }}</h3>
          </div>
          <button class="modal-close" @click="closeGuestModal">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">{{ t('family.view.profileName') }}</label>
            <input
              v-model="guestUsername"
              type="text"
              class="text-input"
              :placeholder="t('family.view.profileNamePlaceholder')"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('family.view.numericPin') }}</label>
            <input
              v-model="guestPin"
              type="password"
              maxlength="8"
              class="text-input"
              placeholder="1234"
            />
          </div>

          <div class="form-group">
            <label class="form-label">{{ t('family.view.temporaryValidity') }}</label>
            <select v-model="guestHours" class="role-select">
              <option :value="4">{{ t('family.view.hours', { value: 4 }) }}</option>
              <option :value="12">{{ t('family.view.hours', { value: 12 }) }}</option>
              <option :value="24">{{ t('family.view.dayOne') }}</option>
              <option :value="72">{{ t('family.view.days', { value: 3 }) }}</option>
              <option :value="168">{{ t('family.view.oneWeek') }}</option>
            </select>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="guestKiosk" />
              <span>{{ t('family.view.enableKiosk') }}</span>
            </label>
          </div>
        </div>

        <div class="modal-footer">
          <BaseButton variant="ghost" size="md" @click="closeGuestModal">
            Annulla
          </BaseButton>
          <BaseButton variant="primary" size="md" @click="handleCreateGuest">
            Crea Profilo Ospite
          </BaseButton>
        </div>
      </div>
    </div>

    <!-- Invite Modal -->
    <div v-if="isInviteModalOpen" class="modal-backdrop" @click.self="closeInviteModal">
      <div class="modal-card glass-card">
        <div class="modal-header">
          <div class="modal-title-box">
            <UserPlus :size="20" />
            <h3 class="modal-title">{{ t('family.inviteModalTitle') }}</h3>
          </div>
          <button class="modal-close" @click="closeInviteModal">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <!-- Role Selection with descriptions -->
          <div class="form-group">
            <label class="form-label">{{ t('family.selectRole') }}</label>
            <div class="role-options">
              <label
                class="role-card"
                :class="{ active: selectedRole === 'operator' }"
              >
                <input
                  type="radio"
                  name="inviteRole"
                  value="operator"
                  v-model="selectedRole"
                />
                <div class="role-card-content">
                  <div class="role-card-header">
                    <Sliders :size="16" class="role-icon operator-icon" />
                    <span class="role-name">{{ t('family.roles.operator') }}</span>
                  </div>
                  <p class="role-desc">{{ t('family.rolesDesc.operator') }}</p>
                </div>
              </label>

              <label
                class="role-card"
                :class="{ active: selectedRole === 'viewer' }"
              >
                <input
                  type="radio"
                  name="inviteRole"
                  value="viewer"
                  v-model="selectedRole"
                />
                <div class="role-card-content">
                  <div class="role-card-header">
                    <Eye :size="16" class="role-icon viewer-icon" />
                    <span class="role-name">{{ t('family.roles.viewer') }}</span>
                  </div>
                  <p class="role-desc">{{ t('family.rolesDesc.viewer') }}</p>
                </div>
              </label>

              <label
                class="role-card"
                :class="{ active: selectedRole === 'kid' }"
              >
                <input
                  type="radio"
                  name="inviteRole"
                  value="kid"
                  v-model="selectedRole"
                />
                <div class="role-card-content">
                  <div class="role-card-header">
                    <Baby :size="16" class="role-icon viewer-icon" />
                    <span class="role-name">{{ t('family.roles.kid') }}</span>
                  </div>
                  <p class="role-desc">{{ t('family.rolesDesc.kid') }}</p>
                </div>
              </label>

              <label
                class="role-card"
                :class="{ active: selectedRole === 'admin' }"
              >
                <input
                  type="radio"
                  name="inviteRole"
                  value="admin"
                  v-model="selectedRole"
                />
                <div class="role-card-content">
                  <div class="role-card-header">
                    <Shield :size="16" class="role-icon admin-icon" />
                    <span class="role-name">{{ t('family.roles.admin') }}</span>
                  </div>
                  <p class="role-desc">{{ t('family.rolesDesc.admin') }}</p>
                </div>
              </label>
            </div>
          </div>

          <!-- Expiration options -->
          <div class="form-group">
            <label class="form-label">{{ t('family.expiration') }}</label>
            <div class="expiry-chips">
              <button
                type="button"
                class="chip"
                :class="{ active: selectedExpiresHours === 24 }"
                @click="selectedExpiresHours = 24"
              >
                {{ t('family.hours24') }}
              </button>
              <button
                type="button"
                class="chip"
                :class="{ active: selectedExpiresHours === 48 }"
                @click="selectedExpiresHours = 48"
              >
                {{ t('family.hours48') }}
              </button>
              <button
                type="button"
                class="chip"
                :class="{ active: selectedExpiresHours === 168 }"
                @click="selectedExpiresHours = 168"
              >
                {{ t('family.days7') }}
              </button>
            </div>
          </div>

          <!-- Generated Link Display -->
          <div v-if="createdInviteUrl" class="generated-box">
            <label class="form-label">{{ t('family.copyLink') }}</label>
            <div class="copy-input-row">
              <input
                type="text"
                readonly
                :value="createdInviteUrl"
                class="link-input"
              />
              <BaseButton
                variant="primary"
                class="copy-action-btn"
                @click="handleCopy(createdInviteUrl)"
              >
                <Check v-if="copySuccess" :size="16" />
                <Copy v-else :size="16" />
                <span>{{ copySuccess ? t('family.linkCopied') : t('family.copyLink') }}</span>
              </BaseButton>
            </div>
          </div>
        </div>

        <div class="modal-footer">
          <BaseButton variant="secondary" @click="closeInviteModal">
            {{ t('common.cancel') }}
          </BaseButton>
          <BaseButton
            v-if="!createdInviteUrl"
            variant="primary"
            :loading="familyStore.loading"
            @click="handleGenerateInvite"
          >
            <Link :size="16" />
            <span>{{ t('family.generateInvite') }}</span>
          </BaseButton>
        </div>
      </div>
    </div>
    <!-- Granular permissions modal -->
    <div v-if="isPermissionsModalOpen && permissionsMember" class="modal-overlay" @click.self="closePermissions">
      <div class="modal-card permission-modal-card">
        <div class="modal-header">
          <div class="modal-title-group">
            <Sliders :size="20" />
            <h3 class="modal-title">{{ t('family.permissions.title', { name: permissionsMember.username }) }}</h3>
          </div>
          <button class="modal-close" @click="closePermissions"><X :size="18" /></button>
        </div>
        <div class="modal-body permission-modal-body">
          <section class="permission-section">
            <h4>{{ t('family.permissions.areasTitle') }}</h4>
            <label class="permission-check">
              <input v-model="allAreas" type="checkbox" />
              <span>{{ t('family.permissions.allAreas') }}</span>
            </label>
            <div v-if="!allAreas" class="permission-options">
              <label v-for="area in availableAreas" :key="area.id" class="permission-check">
                <input v-model="selectedAreaIDs" type="checkbox" :value="area.id" />
                <span>{{ area.name }}</span>
              </label>
              <p v-if="availableAreas.length === 0" class="permission-hint">{{ t('family.permissions.noAreas') }}</p>
            </div>
          </section>

          <section class="permission-section">
            <h4>{{ t('family.permissions.entitiesTitle') }}</h4>
            <label class="permission-check">
              <input v-model="allEntities" type="checkbox" />
              <span>{{ t('family.permissions.allEntities') }}</span>
            </label>
            <div v-if="!allEntities" class="permission-options permission-options-scroll">
              <label v-for="entity in availableEntities" :key="entity.id" class="permission-check">
                <input v-model="selectedEntityIDs" type="checkbox" :value="entity.id" />
                <span>{{ entity.name || entity.id }}</span>
                <small>{{ entity.domain }}</small>
              </label>
              <p v-if="availableEntities.length === 0" class="permission-hint">{{ t('family.permissions.noEntities') }}</p>
            </div>
          </section>

          <section class="permission-section">
            <h4>{{ t('family.permissions.dashboardsTitle') }}</h4>
            <label class="permission-check">
              <input v-model="allDashboards" type="checkbox" />
              <span>{{ t('family.permissions.allDashboards') }}</span>
            </label>
            <div v-if="!allDashboards" class="permission-options">
              <label v-for="dashboard in availableDashboards" :key="dashboard.id" class="permission-check">
                <input v-model="selectedDashboardIDs" type="checkbox" :value="dashboard.id" />
                <span>{{ dashboard.title }}</span>
              </label>
              <p v-if="availableDashboards.length === 0" class="permission-hint">{{ t('family.permissions.noDashboards') }}</p>
            </div>
          </section>

          <section class="permission-section">
            <h4>{{ t('family.permissions.viewsTitle') }}</h4>
            <label class="permission-check">
              <input v-model="allViews" type="checkbox" />
              <span>{{ t('family.permissions.allViews') }}</span>
            </label>
            <div v-if="!allViews" class="permission-options permission-options-grid">
              <label v-for="view in permissionViews" :key="view.path" class="permission-check">
                <input v-model="selectedViews" type="checkbox" :value="view.path" />
                <span>{{ t(view.labelKey) }}</span>
              </label>
            </div>
          </section>

          <section class="permission-section">
            <h4>{{ t('family.permissions.actionsTitle') }}</h4>
            <label class="permission-check">
              <input v-model="canControlDevices" type="checkbox" />
              <span>{{ t('family.permissions.controlDevices') }}</span>
            </label>
            <label class="permission-check">
              <input v-model="canManageAutomations" type="checkbox" :disabled="!canControlDevices || !allAreas || !allEntities" />
              <span>{{ t('family.permissions.manageAutomations') }}</span>
            </label>
            <p v-if="!canControlDevices || !allAreas || !allEntities" class="permission-hint">{{ t('family.permissions.automationScopeHint') }}</p>
            <label class="permission-check">
              <input v-model="canViewCameras" type="checkbox" />
              <span>{{ t('family.permissions.viewCameras') }}</span>
            </label>
          </section>
          <p class="permission-hint">{{ t('family.permissions.hint') }}</p>
        </div>
        <div class="modal-footer">
          <BaseButton variant="secondary" @click="closePermissions">{{ t('family.permissions.cancel') }}</BaseButton>
          <BaseButton variant="primary" :loading="isSavingPermissions" @click="savePermissions">{{ t('family.permissions.save') }}</BaseButton>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.family-view {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
  max-width: 1200px;
  margin: 0 auto;
}

.header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 1rem;
}

.title-with-icon {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.icon-bubble {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.2), rgba(168, 85, 247, 0.2));
  color: #818cf8;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(129, 140, 248, 0.3);
}

.section-title {
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.section-subtitle {
  font-size: 0.875rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
}

.invite-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 600;
}

.section-card {
  border-radius: 16px;
  padding: 1.5rem;
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  backdrop-filter: blur(12px);
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.25rem;
}

.card-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.count-badge {
  background: var(--bg-surface-hover);
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--text-muted);
}

.members-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
}

.member-card {
  background: var(--bg-glass-card);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  box-shadow: var(--shadow-sm);
  transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
}

.member-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-card);
  box-shadow: var(--shadow-md);
}

.member-top {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.avatar-circle {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  background: linear-gradient(135deg, #4f46e5, #06b6d4);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 0.95rem;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
}

.member-meta {
  flex: 1;
  min-width: 0;
}

.username-row {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}

.username {
  font-weight: 600;
  color: var(--text-main);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.self-tag {
  font-size: 0.75rem;
  color: #818cf8;
  font-weight: 500;
}

.email {
  font-size: 0.8rem;
  color: var(--text-muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  display: block;
}

.role-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 600;
}

.role-admin {
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.role-operator {
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.role-viewer {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.member-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border-subtle);
  font-size: 0.75rem;
}

.joined-date {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--text-muted);
}

.member-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.role-select {
  background: var(--bg-surface);
  color: var(--text-main);
  border: 1px solid var(--border-card);
  padding: 0.25rem 0.5rem;
  border-radius: 6px;
  font-size: 0.75rem;
  outline: none;
  cursor: pointer;
}

.remove-btn {
  background: none;
  border: none;
  color: #ef4444;
  cursor: pointer;
  padding: 0.25rem;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background-color 0.2s;
}

.remove-btn:hover {
  background: rgba(239, 68, 68, 0.15);
}

.toast-inline {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  color: #10b981;
  font-size: 0.75rem;
  margin-top: 0.25rem;
}

.invites-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.invite-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-surface);
  padding: 0.75rem 1rem;
  border-radius: 8px;
  border: 1px solid var(--border-subtle);
}

.invite-info {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.invite-details {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.invite-token {
  font-family: monospace;
  font-size: 0.85rem;
  color: var(--text-main);
}

.invite-expiry {
  font-size: 0.75rem;
  color: var(--text-muted);
}

.invite-actions {
  display: flex;
  gap: 0.5rem;
}

.action-icon-btn {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  padding: 0.4rem;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.action-icon-btn:hover {
  background: var(--bg-surface-hover);
  color: var(--text-main);
}

.revoke-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.empty-state {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--text-muted);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.empty-icon {
  opacity: 0.35;
}

/* Modal */
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.65);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-card {
  width: 100%;
  max-width: 520px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: 16px;
  overflow: hidden;
  box-shadow: var(--shadow-lg);
}

.modal-header {
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-title-box {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: #818cf8;
}

.modal-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-main);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.25rem;
}

.modal-body {
  padding: 1.5rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--text-main);
}

.permissions-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--border-subtle);
  border-radius: 9px;
  color: var(--text-muted);
  background: var(--bg-card);
  cursor: pointer;
}
.permissions-btn:hover { color: var(--text-main); border-color: var(--accent-primary, #6366f1); }
.permission-modal-card { width: min(760px, calc(100vw - 28px)); max-height: min(88vh, 900px); }
.permission-modal-body { overflow-y: auto; max-height: calc(88vh - 150px); }
.permission-section { padding: 0.9rem 0; border-bottom: 1px solid var(--border-subtle); }
.permission-section h4 { margin: 0 0 0.65rem; font-size: 0.92rem; color: var(--text-main); }
.permission-check { display: flex; align-items: center; gap: 0.55rem; min-height: 34px; font-size: 0.86rem; color: var(--text-main); cursor: pointer; }
.permission-check input { accent-color: var(--accent-primary, #6366f1); }
.permission-check small { margin-left: auto; color: var(--text-muted); }
.permission-options { display: grid; grid-template-columns: repeat(auto-fit, minmax(190px, 1fr)); gap: 0.2rem 0.8rem; margin: 0.55rem 0 0 1.3rem; }
.permission-options-scroll { max-height: 190px; overflow: auto; }
.permission-options-grid { grid-template-columns: repeat(auto-fit, minmax(155px, 1fr)); }
.permission-hint { margin: 0.6rem 0 0; font-size: 0.78rem; color: var(--text-muted); line-height: 1.45; }
@media (max-width: 600px) {
  .permission-modal-card { width: calc(100vw - 16px); max-height: 94vh; }
  .permission-modal-body { max-height: calc(94vh - 140px); }
  .permission-options { grid-template-columns: 1fr; margin-left: 0.5rem; }
}
.role-options {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.role-card {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.75rem;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.role-card:hover {
  background: var(--bg-surface-hover);
  border-color: rgba(99, 102, 241, 0.3);
}

.role-card.active {
  background: rgba(99, 102, 241, 0.1);
  border-color: #6366f1;
}

.role-card input[type="radio"] {
  margin-top: 0.25rem;
}

.role-card-content {
  flex: 1;
}

.role-card-header {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-weight: 600;
  font-size: 0.9rem;
  color: var(--text-main);
}

.admin-icon { color: #a5b4fc; }
.operator-icon { color: #6ee7b7; }
.viewer-icon { color: #fcd34d; }

.role-desc {
  font-size: 0.78rem;
  color: var(--text-muted);
  margin-top: 0.25rem;
  line-height: 1.35;
}

.expiry-chips {
  display: flex;
  gap: 0.5rem;
}

.chip {
  flex: 1;
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  color: var(--text-muted);
  padding: 0.5rem;
  border-radius: 8px;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.chip:hover {
  background: var(--bg-surface-hover);
}

.chip.active {
  background: rgba(99, 102, 241, 0.15);
  color: #6366f1;
  border-color: #6366f1;
}

.generated-box {
  background: rgba(16, 185, 129, 0.1);
  border: 1px solid rgba(16, 185, 129, 0.3);
  border-radius: 8px;
  padding: 0.75rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.copy-input-row {
  display: flex;
  gap: 0.5rem;
}

.link-input {
  flex: 1;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  color: var(--text-main);
  font-family: monospace;
  font-size: 0.85rem;
  padding: 0.5rem;
  border-radius: 6px;
  outline: none;
}

.copy-action-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  white-space: nowrap;
}

.modal-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.alert {
  padding: 0.75rem 1rem;
  border-radius: 8px;
  font-size: 0.875rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.error-alert {
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
}

.close-alert-btn {
  background: none;
  border: none;
  color: inherit;
  cursor: pointer;
  padding: 0.2rem;
}
</style>
