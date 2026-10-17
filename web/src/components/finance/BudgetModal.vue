<script setup lang="ts">
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { X, Check, PiggyBank } from '@lucide/vue'

const props = defineProps<{
  isOpen: boolean
  currentBudget: number
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'save', limit: number): void
}>()

const limit = ref<number | ''>(props.currentBudget || 1500)

watch(
  () => props.currentBudget,
  (val) => {
    limit.value = val || 1500
  }
)

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && props.isOpen) {
    emit('close')
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleKeydown)
})

function submit() {
  if (limit.value && Number(limit.value) > 0) {
    emit('save', Number(limit.value))
    emit('close')
  }
}
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop" @click="emit('close')" aria-hidden="true">
    <div
      class="modal-dialog glass-panel"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-budget-title"
      @click.stop
    >
      <div class="modal-header">
        <div class="header-left">
          <PiggyBank :size="20" class="piggy-icon" aria-hidden="true" />
          <h3 id="modal-budget-title" class="modal-title">Imposta Budget Mensile</h3>
        </div>
        <button
          class="close-btn"
          aria-label="Chiudi finestra"
          @click="emit('close')"
        >
          <X :size="20" aria-hidden="true" />
        </button>
      </div>

      <form class="modal-form" @submit.prevent="submit">
        <p class="budget-desc">
          Imposta il tetto massimo di spesa per il mese corrente. Domus ti avviserà tempestivamente se i tuoi costi superano l'80% o sforano il limite stabilito.
        </p>

        <div class="form-group">
          <label for="budget-limit" class="form-label">Limite Mensile di Spesa (€) *</label>
          <input
            id="budget-limit"
            v-model="limit"
            type="number"
            step="10"
            min="10"
            required
            class="form-input budget-input"
            placeholder="es. 1500"
          />
        </div>

        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="emit('close')">
            Annulla
          </button>
          <button type="submit" class="btn btn-primary">
            <Check :size="16" aria-hidden="true" />
            <span>Salva Budget</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: var(--bg-overlay);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-dialog {
  width: 100%;
  max-width: 440px;
  border-radius: var(--radius-xl);
  padding: 1.5rem;
  background: var(--bg-secondary);
  border: 1px solid var(--border-card);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--border-subtle);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.piggy-icon {
  color: var(--color-success);
}

.modal-title {
  font-size: 1.15rem;
  font-weight: 700;
  color: var(--text-main);
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 0.4rem;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  min-width: 44px;
  transition: color var(--transition-fast);
}

.close-btn:hover {
  color: var(--text-main);
}

.budget-desc {
  font-size: 0.85rem;
  color: var(--text-muted);
  line-height: 1.45;
  margin: 0 0 1rem 0;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-muted);
}

.form-input {
  width: 100%;
  padding: 0.65rem 0.85rem;
  min-height: 44px;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  color: var(--text-main);
  font-size: 1.1rem;
  font-weight: 700;
  transition: border-color var(--transition-fast);
}

.form-input:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 2px var(--primary-light);
}

.budget-input {
  color: var(--color-success);
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.65rem 1.25rem;
  min-height: 44px;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: opacity var(--transition-fast);
}

.btn-primary {
  background: var(--primary);
  color: #fff;
}

.btn-primary:hover {
  opacity: 0.9;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-muted);
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  color: var(--text-main);
}
</style>
