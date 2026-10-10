<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import { useFamilyStore } from '../stores/family'
import BaseButton from '../components/ui/BaseButton.vue'
import type { UserRole } from '../api/types'
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
  KeyRound
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

async function loadData() {
  try {
    await familyStore.fetchMembers()
    if (isAdmin.value) {
      await familyStore.fetchInvites()
      try {
        await familyStore.fetchGuests()
      } catch {}
    }
  } catch (err: any) {
    errorMsg.value = err?.message || 'Failed to load family data'
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
    errorMsg.value = 'Inserisci nome ospite e PIN'
    return
  }
  try {
    errorMsg.value = null
    await familyStore.createGuest(guestUsername.value, guestPin.value, guestHours.value, guestKiosk.value)
    closeGuestModal()
  } catch (err: any) {
    errorMsg.value = err?.message || 'Impossibile creare profilo ospite'
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
    errorMsg.value = err?.message || 'Failed to generate invite'
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
    errorMsg.value = err?.message || 'Failed to update member role'
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
    errorMsg.value = err?.message || 'Failed to remove member'
  }
}

async function handleRevokeInvite(inviteId: string) {
  try {
    errorMsg.value = null
    await familyStore.revokeInvite(inviteId)
  } catch (err: any) {
    errorMsg.value = err?.message || 'Failed to revoke invite'
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
                <span v-if="member.id === currentUserId" class="self-tag">(Tu)</span>
              </div>
              <span class="email">{{ member.email || '—' }}</span>
            </div>
            <div class="role-badge" :class="`role-${member.role}`">
              <Shield v-if="member.role === 'admin'" :size="14" />
              <Sliders v-else-if="member.role === 'operator'" :size="14" />
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
              </select>

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
              <Eye v-else :size="14" />
              <span>{{ t(`family.roles.${inv.role}`) }}</span>
            </div>
            <div class="invite-details">
              <span class="invite-token">Token: {{ inv.token.slice(0, 12) }}...</span>
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
            <h3 class="section-subtitle">Profili Ospiti & PIN Kiosk Temporaneo</h3>
            <p class="section-desc">Crea account ad accesso limitato con codice PIN o sblocco Kiosk temporaneo (es. babysitter, affitti brevi)</p>
          </div>
        </div>
        <BaseButton
          variant="secondary"
          size="sm"
          class="guest-btn"
          @click="openGuestModal"
        >
          <UserPlus :size="16" />
          Nuovo Ospite / PIN Kiosk
        </BaseButton>
      </div>

      <div v-if="guests.length === 0" class="empty-state">
        <KeyRound :size="32" class="empty-icon" />
        <p class="empty-text">Nessun profilo ospite attivo. Crea un PIN temporaneo per i tuoi visitatori.</p>
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
              <span>Ospite {{ guest.kiosk_mode ? '(Kiosk)' : '' }}</span>
            </div>
            <div class="invite-meta">
              <span class="invite-user"><strong>{{ guest.username }}</strong></span>
              <span v-if="guest.pin_code" class="invite-pin">PIN: <code>{{ guest.pin_code }}</code></span>
              <span v-if="guest.guest_expires_at" class="invite-expires">
                Scadenza: {{ formatDate(guest.guest_expires_at) }}
              </span>
            </div>
          </div>

          <div class="invite-actions">
            <button
              class="revoke-btn action-icon-btn"
              title="Rimuovi ospite"
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
            <h3 class="modal-title">Crea Profilo Ospite / Kiosk</h3>
          </div>
          <button class="modal-close" @click="closeGuestModal">
            <X :size="18" />
          </button>
        </div>

        <div class="modal-body">
          <div class="form-group">
            <label class="form-label">Nome Profilo / Visitatore</label>
            <input
              v-model="guestUsername"
              type="text"
              class="text-input"
              placeholder="es. Babysitter, Ospite Salotto"
            />
          </div>

          <div class="form-group">
            <label class="form-label">Codice PIN Numerico (4-8 cifre)</label>
            <input
              v-model="guestPin"
              type="password"
              maxlength="8"
              class="text-input"
              placeholder="1234"
            />
          </div>

          <div class="form-group">
            <label class="form-label">Validità Temporanea (Ore)</label>
            <select v-model="guestHours" class="role-select">
              <option :value="4">4 Ore</option>
              <option :value="12">12 Ore</option>
              <option :value="24">24 Ore (1 Giorno)</option>
              <option :value="72">72 Ore (3 Giorni)</option>
              <option :value="168">7 Giorni (1 Settimana)</option>
            </select>
          </div>

          <div class="form-group">
            <label class="checkbox-label">
              <input type="checkbox" v-model="guestKiosk" />
              <span>Abilita Modalità Kiosk a Schermo Intero per questo PIN</span>
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
  color: var(--color-text, #f1f5f9);
  margin: 0;
}

.section-subtitle {
  font-size: 0.875rem;
  color: var(--color-text-muted, #94a3b8);
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
  background: var(--card-bg, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
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
  color: var(--color-text, #f1f5f9);
  margin: 0;
}

.count-badge {
  background: rgba(255, 255, 255, 0.08);
  padding: 0.25rem 0.625rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--color-text-muted, #94a3b8);
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
  color: var(--color-text, #f1f5f9);
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
  color: var(--color-text-muted, #94a3b8);
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
  color: #a5b4fc;
  border: 1px solid rgba(99, 102, 241, 0.3);
}

.role-operator {
  background: rgba(16, 185, 129, 0.15);
  color: #6ee7b7;
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.role-viewer {
  background: rgba(245, 158, 11, 0.15);
  color: #fcd34d;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.member-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 0.5rem;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  font-size: 0.75rem;
}

.joined-date {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--color-text-muted, #94a3b8);
}

.member-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.role-select {
  background: rgba(30, 41, 59, 0.8);
  color: var(--color-text, #f1f5f9);
  border: 1px solid rgba(255, 255, 255, 0.1);
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
  color: var(--color-text, #f1f5f9);
}

.invite-expiry {
  font-size: 0.75rem;
  color: var(--color-text-muted, #94a3b8);
}

.invite-actions {
  display: flex;
  gap: 0.5rem;
}

.action-icon-btn {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
  color: var(--color-text-muted, #cbd5e1);
  padding: 0.4rem;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
}

.action-icon-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--color-text, #fff);
}

.revoke-btn:hover {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
}

.empty-state {
  text-align: center;
  padding: 2.5rem 1rem;
  color: var(--color-text-muted, #94a3b8);
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
  background: var(--card-bg, #1e293b);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.5);
}

.modal-header {
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
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
  color: var(--color-text, #f1f5f9);
  margin: 0;
}

.modal-close {
  background: none;
  border: none;
  color: var(--color-text-muted, #94a3b8);
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
  color: var(--color-text, #e2e8f0);
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
