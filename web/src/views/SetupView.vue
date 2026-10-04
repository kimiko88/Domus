<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import BaseButton from '../components/ui/BaseButton.vue'
import { ShieldCheck, User, Lock, AlertCircle } from '@lucide/vue'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const confirmPassword = ref('')
const errorMessage = ref<string | null>(null)
const isLoading = ref(false)

async function handleSubmit() {
  if (password.value !== confirmPassword.value) {
    errorMessage.value = 'Le password non coincidono'
    return
  }

  if (password.value.length < 8) {
    errorMessage.value = 'La password deve avere almeno 8 caratteri'
    return
  }

  isLoading.value = true
  errorMessage.value = null

  try {
    await authStore.registerFirstAdmin(username.value, password.value)
    if (router) {
      router.push('/')
    }
  } catch (err: any) {
    errorMessage.value = err?.message || 'Errore durante la creazione dell\'amministratore'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="setup-page">
    <div class="setup-card glass-panel">
      <!-- Header -->
      <div class="card-header">
        <div class="icon-box">
          <ShieldCheck :size="32" />
        </div>
        <h2 class="title">Configurazione Iniziale</h2>
        <p class="subtitle">Crea l'account di amministrazione principale per Domus</p>
      </div>

      <!-- Error Alert -->
      <div v-if="errorMessage" class="error-alert">
        <AlertCircle :size="18" class="error-icon" />
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Form -->
      <form class="setup-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="admin-username">Nome Utente Admin</label>
          <div class="input-wrapper">
            <User :size="18" class="input-icon" />
            <input
              id="admin-username"
              v-model="username"
              type="text"
              placeholder="admin"
              required
              autocomplete="username"
            />
          </div>
        </div>

        <div class="form-group">
          <label for="admin-password">Password Amministratore</label>
          <div class="input-wrapper">
            <Lock :size="18" class="input-icon" />
            <input
              id="admin-password"
              v-model="password"
              type="password"
              placeholder="Minimo 8 caratteri"
              required
              autocomplete="new-password"
            />
          </div>
        </div>

        <div class="form-group">
          <label for="admin-password-confirm">Conferma Password</label>
          <div class="input-wrapper">
            <Lock :size="18" class="input-icon" />
            <input
              id="admin-password-confirm"
              v-model="confirmPassword"
              type="password"
              placeholder="Ripeti la password"
              required
              autocomplete="new-password"
            />
          </div>
        </div>

        <BaseButton
          type="submit"
          variant="primary"
          size="lg"
          :loading="isLoading"
          class="submit-btn"
        >
          Crea Amministratore e Inizia
        </BaseButton>
      </form>
    </div>
  </div>
</template>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: radial-gradient(circle at 50% 25%, rgba(16, 185, 129, 0.1) 0%, transparent 60%),
              var(--bg-primary);
}

.setup-card {
  width: 100%;
  max-width: 440px;
  padding: 36px 32px;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
}

.card-header {
  text-align: center;
  margin-bottom: 28px;
}

.icon-box {
  width: 60px;
  height: 60px;
  margin: 0 auto 16px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--color-success) 0%, #059669 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 0 24px var(--color-success-glow);
}

.title {
  font-size: 22px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 6px;
  line-height: 1.4;
}

.error-alert {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  background: var(--color-danger-bg);
  border: 1px solid rgba(244, 63, 94, 0.3);
  border-radius: var(--radius-md);
  color: var(--color-danger);
  font-size: 13px;
  margin-bottom: 20px;
  animation: slide-in-up 0.2s ease-out;
}

.error-icon {
  flex-shrink: 0;
}

.setup-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

label {
  font-size: 13px;
  font-weight: 500;
  color: var(--text-muted);
}

.input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.input-icon {
  position: absolute;
  left: 14px;
  color: var(--text-dim);
  pointer-events: none;
}

input {
  width: 100%;
  padding: 12px 14px 12px 42px;
  background: var(--bg-surface);
  border: 1px solid var(--border-card);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 14px;
  transition: all var(--transition-fast);
}

input:focus {
  border-color: var(--primary);
  box-shadow: 0 0 0 3px var(--primary-light);
  background: var(--bg-surface-hover);
}

.submit-btn {
  width: 100%;
  margin-top: 10px;
}
</style>
