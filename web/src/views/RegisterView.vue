<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'
import BaseButton from '../components/ui/BaseButton.vue'
import type { UserRole } from '../api/types'
import {
  Home,
  User,
  Lock,
  Mail,
  AlertCircle,
  Shield,
  Sliders,
  Eye,
  CheckCircle2
} from '@lucide/vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const token = ref<string>('')
const isValidating = ref(true)
const isInviteValid = ref(false)
const inviteRole = ref<UserRole>('viewer')
const inviteExpiresAt = ref<string>('')

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const errorMessage = ref<string | null>(null)
const isSubmitting = ref(false)
const isSuccess = ref(false)

async function validateToken() {
  const qToken = route.query?.token as string
  if (!qToken) {
    isValidating.value = false
    isInviteValid.value = false
    errorMessage.value = t('register.errorInvalidInvite')
    return
  }

  token.value = qToken
  isValidating.value = true
  errorMessage.value = null

  try {
    const res = await authStore.validateInvite(qToken)
    if (res.valid) {
      isInviteValid.value = true
      inviteRole.value = res.role
      inviteExpiresAt.value = res.expires_at
    } else {
      isInviteValid.value = false
      errorMessage.value = t('register.errorInvalidInvite')
    }
  } catch (err: any) {
    isInviteValid.value = false
    errorMessage.value = err?.message || t('register.errorInvalidInvite')
  } finally {
    isValidating.value = false
  }
}

async function handleSubmit() {
  if (!username.value.trim() || !password.value) {
    errorMessage.value = 'Compila tutti i campi obbligatori'
    return
  }

  if (password.value !== confirmPassword.value) {
    errorMessage.value = t('register.errorPasswordMismatch')
    return
  }

  if (password.value.length < 8) {
    errorMessage.value = 'La password deve contenere almeno 8 caratteri'
    return
  }

  isSubmitting.value = true
  errorMessage.value = null

  try {
    await authStore.registerWithInvite({
      token: token.value,
      username: username.value.trim(),
      password: password.value,
      email: email.value.trim() || undefined
    })

    isSuccess.value = true
    setTimeout(() => {
      router.push('/')
    }, 1200)
  } catch (err: any) {
    errorMessage.value = err?.message || 'Registrazione fallita. Riprova.'
  } finally {
    isSubmitting.value = false
  }
}

onMounted(() => {
  validateToken()
})
</script>

<template>
  <div class="register-page">
    <div class="register-card glass-panel">
      <!-- Header -->
      <div class="card-header">
        <div class="logo-box">
          <Home :size="28" />
        </div>
        <h2 class="title">{{ t('register.title') }}</h2>
        <p class="subtitle">{{ t('register.subtitle') }}</p>
      </div>

      <!-- Validating State -->
      <div v-if="isValidating" class="status-box loading-box">
        <div class="spinner"></div>
        <span>{{ t('register.loading') }}</span>
      </div>

      <!-- Invalid Invite Alert -->
      <div v-else-if="!isInviteValid" class="status-box error-box">
        <AlertCircle :size="32" class="error-icon" />
        <p class="error-text">{{ errorMessage || t('register.errorInvalidInvite') }}</p>
        <router-link to="/login" class="back-link">
          {{ t('register.backToLogin') }}
        </router-link>
      </div>

      <!-- Success State -->
      <div v-else-if="isSuccess" class="status-box success-box">
        <CheckCircle2 :size="40" class="success-icon" />
        <p class="success-text">{{ t('register.success') }}</p>
      </div>

      <!-- Registration Form -->
      <div v-else class="form-container">
        <!-- Role Notice Banner -->
        <div class="role-banner" :class="`role-${inviteRole}`">
          <div class="role-banner-icon">
            <Shield v-if="inviteRole === 'admin'" :size="20" />
            <Sliders v-else-if="inviteRole === 'operator'" :size="20" />
            <Eye v-else :size="20" />
          </div>
          <div class="role-banner-text">
            <span class="role-notice-label">{{ t('register.roleNotice') }}</span>
            <span class="role-notice-val">{{ t(`family.roles.${inviteRole}`) }}</span>
          </div>
        </div>

        <!-- Form Error Alert -->
        <div v-if="errorMessage" class="error-alert">
          <AlertCircle :size="18" class="error-icon-small" />
          <span>{{ errorMessage }}</span>
        </div>

        <form class="register-form" @submit.prevent="handleSubmit">
          <div class="form-group">
            <label for="username">{{ t('register.username') }} *</label>
            <div class="input-wrapper">
              <User :size="18" class="input-icon" />
              <input
                id="username"
                name="username"
                v-model="username"
                type="text"
                autocomplete="username"
                required
                placeholder="es. marco_rossi"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="email">{{ t('register.email') }}</label>
            <div class="input-wrapper">
              <Mail :size="18" class="input-icon" />
              <input
                id="email"
                name="email"
                v-model="email"
                type="email"
                autocomplete="email"
                placeholder="es. marco@email.it"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="password">{{ t('register.password') }} *</label>
            <div class="input-wrapper">
              <Lock :size="18" class="input-icon" />
              <input
                id="password"
                name="password"
                v-model="password"
                type="password"
                autocomplete="new-password"
                required
                minlength="8"
                placeholder="••••••••"
              />
            </div>
          </div>

          <div class="form-group">
            <label for="confirmPassword">{{ t('register.confirmPassword') }} *</label>
            <div class="input-wrapper">
              <Lock :size="18" class="input-icon" />
              <input
                id="confirmPassword"
                name="confirm_password"
                v-model="confirmPassword"
                type="password"
                autocomplete="new-password"
                required
                minlength="8"
                placeholder="••••••••"
              />
            </div>
          </div>

          <BaseButton
            type="submit"
            variant="primary"
            class="submit-btn"
            :loading="isSubmitting"
          >
            {{ t('register.submit') }}
          </BaseButton>
        </form>

        <div class="form-footer">
          <router-link to="/login" class="back-link">
            {{ t('register.backToLogin') }}
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.register-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1.5rem;
  background: radial-gradient(circle at top right, #1e1b4b 0%, #0f172a 100%);
}

.register-card {
  width: 100%;
  max-width: 440px;
  background: rgba(30, 41, 59, 0.7);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  padding: 2.25rem;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.card-header {
  text-align: center;
  margin-bottom: 1.75rem;
}

.logo-box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #6366f1, #a855f7);
  color: white;
  border-radius: 16px;
  margin-bottom: 1rem;
  box-shadow: 0 8px 16px -4px rgba(99, 102, 241, 0.4);
}

.title {
  font-size: 1.35rem;
  font-weight: 700;
  color: #f1f5f9;
  margin: 0;
}

.subtitle {
  font-size: 0.85rem;
  color: #94a3b8;
  margin-top: 0.35rem;
  line-height: 1.4;
}

.status-box {
  padding: 2rem 1rem;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid rgba(99, 102, 241, 0.2);
  border-top-color: #6366f1;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.error-icon {
  color: #ef4444;
}

.error-text {
  color: #fca5a5;
  font-size: 0.9rem;
  line-height: 1.4;
  margin: 0;
}

.success-icon {
  color: #10b981;
}

.success-text {
  color: #6ee7b7;
  font-weight: 600;
  font-size: 1rem;
  margin: 0;
}

.role-banner {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  border-radius: 10px;
  margin-bottom: 1.25rem;
}

.role-banner.role-admin {
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid rgba(99, 102, 241, 0.3);
  color: #a5b4fc;
}

.role-banner.role-operator {
  background: rgba(16, 185, 129, 0.15);
  border: 1px solid rgba(16, 185, 129, 0.3);
  color: #6ee7b7;
}

.role-banner.role-viewer {
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: #fcd34d;
}

.role-notice-label {
  font-size: 0.8rem;
  color: var(--color-text-muted, #94a3b8);
  display: block;
}

.role-notice-val {
  font-weight: 700;
  font-size: 0.95rem;
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(239, 68, 68, 0.15);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: #fca5a5;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
  font-size: 0.825rem;
  margin-bottom: 1rem;
}

.error-icon-small {
  flex-shrink: 0;
}

.register-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-group label {
  font-size: 0.8rem;
  font-weight: 600;
  color: #cbd5e1;
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 0.85rem;
  color: #64748b;
  pointer-events: none;
}

.input-wrapper input {
  width: 100%;
  background: rgba(15, 23, 42, 0.6);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  padding: 0.65rem 0.85rem 0.65rem 2.5rem;
  color: #f1f5f9;
  font-size: 0.9rem;
  outline: none;
  transition: all 0.2s ease;
}

.input-wrapper input:focus {
  border-color: #6366f1;
  box-shadow: 0 0 0 2px rgba(99, 102, 241, 0.2);
  background: rgba(15, 23, 42, 0.8);
}

.submit-btn {
  width: 100%;
  margin-top: 0.5rem;
  padding: 0.75rem;
  font-weight: 600;
}

.form-footer {
  margin-top: 1.25rem;
  text-align: center;
}

.back-link {
  color: #818cf8;
  font-size: 0.85rem;
  text-decoration: none;
  transition: color 0.2s;
}

.back-link:hover {
  color: #a5b4fc;
  text-decoration: underline;
}
</style>
