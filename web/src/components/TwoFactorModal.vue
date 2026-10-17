<template>
  <div
    class="modal-backdrop"
    role="dialog"
    aria-modal="true"
    aria-labelledby="modal-2fa-title"
    @keydown.esc="close"
  >
    <div class="modal-card">
      <header class="modal-header">
        <h2 id="modal-2fa-title">Autenticazione a Due Fattori (2FA / TOTP)</h2>
        <button class="btn-close" @click="close" aria-label="Chiudi finestra modale">✕</button>
      </header>

      <!-- Step 1: Scan QR & Secret -->
      <section v-if="step === 'setup'" class="modal-body">
        <p class="step-desc">
          1. Scansiona questo codice QR con la tua app di autenticazione (Google Authenticator, Apple Password, 1Password, Authy).
        </p>

        <div class="qr-container">
          <!-- Render visual OTP code / QR block -->
          <div class="qr-preview" aria-label="Codice QR di configurazione">
            <svg class="qr-svg" viewBox="0 0 120 120" width="120" height="120" role="img" aria-label="Simbolo QR">
              <rect width="120" height="120" fill="#ffffff" rx="6" />
              <!-- Mock QR corner squares -->
              <rect x="10" y="10" width="30" height="30" fill="#000000" />
              <rect x="16" y="16" width="18" height="18" fill="#ffffff" />
              <rect x="20" y="20" width="10" height="10" fill="#000000" />

              <rect x="80" y="10" width="30" height="30" fill="#000000" />
              <rect x="86" y="16" width="18" height="18" fill="#ffffff" />
              <rect x="90" y="20" width="10" height="10" fill="#000000" />

              <rect x="10" y="80" width="30" height="30" fill="#000000" />
              <rect x="16" y="86" width="18" height="18" fill="#ffffff" />
              <rect x="20" y="90" width="10" height="10" fill="#000000" />

              <!-- QR data matrix pattern -->
              <rect x="50" y="20" width="8" height="8" fill="#000000" />
              <rect x="60" y="30" width="8" height="8" fill="#000000" />
              <rect x="50" y="50" width="16" height="16" fill="#000000" />
              <rect x="70" y="70" width="12" height="12" fill="#000000" />
              <rect x="50" y="90" width="10" height="10" fill="#000000" />
              <rect x="85" y="50" width="8" height="14" fill="#000000" />
              <rect x="20" y="55" width="12" height="8" fill="#000000" />
            </svg>
          </div>

          <div class="secret-box">
            <span class="secret-label">Chiave manuale (Base32):</span>
            <code class="secret-code">{{ secret }}</code>
            <button type="button" class="btn btn-sm btn-secondary" @click="copySecret">
              {{ copied ? 'Copiato!' : 'Copia Chiave' }}
            </button>
          </div>
        </div>

        <form @submit.prevent="verifyCode" class="verify-form">
          <label for="totp-code">2. Inserisci il codice a 6 cifre generato dall'app:</label>
          <div class="verify-input-group">
            <input
              id="totp-code"
              name="totp_code"
              v-model="totpCode"
              type="text"
              maxlength="6"
              inputmode="numeric"
              pattern="[0-9]*"
              autocomplete="one-time-code"
              required
              placeholder="123456"
              class="input-totp font-mono"
            />
            <button type="submit" class="btn btn-primary" :disabled="verifying || totpCode.length < 6">
              <span v-if="verifying">Verifica...</span>
              <span v-else>Verifica & Attiva</span>
            </button>
          </div>
          <p v-if="verifyError" class="error-msg" role="alert">{{ verifyError }}</p>
        </form>
      </section>

      <!-- Step 2: Backup Codes Success -->
      <section v-else-if="step === 'success'" class="modal-body success-body">
        <div class="success-icon" aria-hidden="true">🎉</div>
        <h3>2FA Attivata con Successo!</h3>
        <p class="backup-desc">
          Salva questi codici di backup in un luogo sicuro. Ciascun codice può essere utilizzato una sola volta per accedere se perdi il tuo dispositivo:
        </p>

        <div class="backup-codes-grid">
          <code v-for="c in backupCodes" :key="c" class="backup-code-item">{{ c }}</code>
        </div>

        <div class="backup-actions">
          <button type="button" class="btn btn-secondary" @click="downloadBackupCodes">
            💾 Scarica Codici (.txt)
          </button>
          <button type="button" class="btn btn-primary" @click="close">
            Ho salvato i codici, Chiudi
          </button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { apiClient } from '../api/client'

const props = defineProps<{
  secret: string
  qrUrl?: string
  initialBackupCodes?: string[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'enabled'): void
}>()

const step = ref<'setup' | 'success'>('setup')
const totpCode = ref('')
const verifying = ref(false)
const verifyError = ref('')
const copied = ref(false)
const backupCodes = ref<string[]>(props.initialBackupCodes || [])

async function copySecret() {
  await navigator.clipboard.writeText(props.secret)
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

async function verifyCode() {
  verifying.value = true
  verifyError.value = ''
  try {
    const res = await apiClient.post<{ status: string; backup_codes?: string[] }>(
      '/api/v1/auth/2fa/verify-and-enable',
      { code: totpCode.value }
    )
    if (res.backup_codes && res.backup_codes.length > 0) {
      backupCodes.value = res.backup_codes
    }
    step.value = 'success'
    emit('enabled')
  } catch (err: any) {
    verifyError.value = err?.message || 'Codice TOTP non valido. Riprova.'
  } finally {
    verifying.value = false
  }
}

function downloadBackupCodes() {
  const content = `=== CODICI DI BACKUP DOMUS 2FA ===\nData: ${new Date().toISOString()}\n\n` +
    backupCodes.value.join('\n') +
    '\n\nCustodisci questi codici con cura.'
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'domus-2fa-backup-codes.txt'
  a.click()
  URL.revokeObjectURL(url)
}

function close() {
  emit('close')
}
</script>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-card {
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.12);
  border-radius: 12px;
  padding: 1.5rem;
  width: 90%;
  max-width: 500px;
  color: #ffffff;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.modal-header h2 {
  font-size: 1.25rem;
  font-weight: 700;
  margin: 0;
}

.btn-close {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
}

.step-desc {
  font-size: 0.95rem;
  color: #cbd5e1;
  margin-bottom: 1rem;
}

.qr-container {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  background: rgba(15, 23, 42, 0.6);
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.qr-preview {
  flex-shrink: 0;
}

.secret-box {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
}

.secret-label {
  font-size: 0.8rem;
  color: #94a3b8;
}

.secret-code {
  font-size: 0.95rem;
  color: #38bdf8;
  word-break: break-all;
  letter-spacing: 0.05em;
}

.verify-form {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.verify-form label {
  font-size: 0.9rem;
  color: #cbd5e1;
}

.verify-input-group {
  display: flex;
  gap: 0.5rem;
}

.input-totp {
  flex: 1;
  padding: 0.6rem 1rem;
  font-size: 1.25rem;
  letter-spacing: 0.25em;
  text-align: center;
  background: #090d16;
  border: 1px solid rgba(255, 255, 255, 0.15);
  border-radius: 6px;
  color: #ffffff;
}

.input-totp:focus-visible {
  outline: 2px solid #38bdf8;
}

.error-msg {
  color: #f87171;
  font-size: 0.85rem;
  margin-top: 0.25rem;
}

.success-body {
  text-align: center;
}

.success-icon {
  font-size: 2.5rem;
  margin-bottom: 0.5rem;
}

.backup-desc {
  font-size: 0.9rem;
  color: #94a3b8;
  margin-bottom: 1rem;
}

.backup-codes-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
  background: #090d16;
  padding: 1rem;
  border-radius: 8px;
  margin-bottom: 1.25rem;
}

.backup-code-item {
  color: #38bdf8;
  font-size: 1rem;
  letter-spacing: 0.05em;
}

.backup-actions {
  display: flex;
  justify-content: center;
  gap: 0.75rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  min-height: 44px;
}

.btn-primary {
  background: #38bdf8;
  color: #0f172a;
  border: none;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
  border: none;
}

.btn-sm {
  min-height: 32px;
  padding: 0.2rem 0.5rem;
  font-size: 0.8rem;
}
</style>
