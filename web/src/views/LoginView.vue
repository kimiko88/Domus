<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import BaseButton from '../components/ui/BaseButton.vue'
import { Home, User, Lock, AlertCircle } from '@lucide/vue'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const errorMessage = ref<string | null>(null)
const isLoading = ref(false)

async function handleSubmit() {
  if (!username.value || !password.value) {
    errorMessage.value = 'Inserisci nome utente e password'
    return
  }

  isLoading.value = true
  errorMessage.value = null

  try {
    await authStore.login(username.value, password.value)
    if (router) {
      router.push('/')
    }
  } catch (err: any) {
    errorMessage.value = err?.message || 'Credenziali non valide o errore di rete'
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card glass-panel">
      <!-- Header -->
      <div class="card-header">
        <div class="logo-box">
          <Home :size="28" />
        </div>
        <h2 class="title">Domus</h2>
        <p class="subtitle">Accedi al tuo hub locale intelligente</p>
      </div>

      <!-- Error Alert -->
      <div v-if="errorMessage" class="error-alert">
        <AlertCircle :size="18" class="error-icon" />
        <span>{{ errorMessage }}</span>
      </div>

      <!-- Form -->
      <form class="login-form" @submit.prevent="handleSubmit">
        <div class="form-group">
          <label for="username">Nome Utente</label>
          <div class="input-wrapper">
            <User :size="18" class="input-icon" />
            <input
              id="username"
              v-model="username"
              type="text"
              placeholder="admin"
              required
              autocomplete="username"
            />
          </div>
        </div>

        <div class="form-group">
          <label for="password">Password</label>
          <div class="input-wrapper">
            <Lock :size="18" class="input-icon" />
            <input
              id="password"
              v-model="password"
              type="password"
              placeholder="••••••••"
              required
              autocomplete="current-password"
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
          Accedi
        </BaseButton>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: radial-gradient(circle at 50% 30%, rgba(14, 165, 233, 0.12) 0%, transparent 60%),
              var(--bg-primary);
}

.login-card {
  width: 100%;
  max-width: 420px;
  padding: 36px 32px;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border-card);
  box-shadow: var(--shadow-lg);
}

.card-header {
  text-align: center;
  margin-bottom: 28px;
}

.logo-box {
  width: 56px;
  height: 56px;
  margin: 0 auto 16px;
  border-radius: var(--radius-lg);
  background: linear-gradient(135deg, var(--primary) 0%, #0284c7 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 0 24px var(--primary-glow);
}

.title {
  font-size: 24px;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.02em;
}

.subtitle {
  font-size: 14px;
  color: var(--text-muted);
  margin-top: 6px;
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

.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
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
  margin-top: 6px;
}
</style>
