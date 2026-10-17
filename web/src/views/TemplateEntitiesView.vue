<template>
  <main class="template-entities-view" id="main-content" tabindex="-1">
    <header class="view-header">
      <div class="header-content">
        <h1>Entità Template & Sensori Virtuali</h1>
        <p class="subtitle">Composizione visiva di formule, logica condizionale e sensori calcolati in tempo reale</p>
      </div>
      <div class="header-actions">
        <button class="btn btn-primary" @click="openCreateModal" aria-label="Crea nuovo sensore template">
          + Nuovo Sensore Template
        </button>
      </div>
    </header>

    <!-- Template Live Playground / Sandbox -->
    <section class="playground-section" aria-label="Collaudo espressioni template">
      <div class="playground-card">
        <div class="playground-header">
          <h2>Collaudo Live Espressioni (Jinja2 / Formula Evaluator)</h2>
          <button class="btn btn-sm btn-secondary" @click="testPreview" :disabled="testingPreview">
            <span v-if="testingPreview">Valutazione...</span>
            <span v-else>Valuta Template ▶</span>
          </button>
        </div>
        <div class="playground-body">
          <label for="template-playground-code" class="sr-only">Editor codice template di prova</label>
          <textarea
            id="template-playground-code"
            name="template_playground_code"
            v-model="playgroundCode"
            class="template-editor"
            rows="3"
            placeholder="Es. {{ states('sensor.salotto_temp') | float * 1.8 + 32 }} °F"
            aria-label="Editor codice template di prova"
            autocomplete="off"
          ></textarea>
          <div class="preview-output">
            <span class="preview-label">Risultato:</span>
            <code class="preview-result">{{ previewResult || 'Digita un template e clicca "Valuta Template"' }}</code>
          </div>
        </div>
      </div>
    </section>

    <!-- Configured Template Entities List -->
    <section class="entities-section" aria-label="Sensori template configurati">
      <h2>Sensori Template Attivi ({{ store.templates.length }})</h2>
      <div v-if="store.loading && store.templates.length === 0" class="loading-state">
        <p>Caricamento sensori template...</p>
      </div>
      <div v-else-if="store.templates.length === 0" class="empty-state">
        <p>Nessun sensore template configurato. Crea formule o sensori virtuali personalizzati.</p>
      </div>
      <div v-else class="templates-grid">
        <article v-for="t in store.templates" :key="t.id" class="template-card">
          <div class="card-top">
            <span class="entity-icon" aria-hidden="true">{{ t.icon || 'ƒ(x)' }}</span>
            <div class="entity-title">
              <h3>{{ t.name }}</h3>
              <code class="entity-id">{{ t.entity_id }}</code>
            </div>
            <span class="domain-tag">{{ t.domain }}</span>
          </div>

          <div class="template-code-preview">
            <span class="code-label">Formula / Espressione:</span>
            <code>{{ t.state_template }}</code>
          </div>

          <footer class="card-footer">
            <span v-if="t.unit_of_measurement" class="meta-unit">Unità: {{ t.unit_of_measurement }}</span>
            <span v-if="t.device_class" class="meta-class">Classe: {{ t.device_class }}</span>
            <button
              class="btn-delete"
              @click="handleDelete(t.id)"
              :aria-label="`Elimina sensore template ${t.name}`"
            >
              🗑️
            </button>
          </footer>
        </article>
      </div>
    </section>

    <!-- Create Modal -->
    <div v-if="showModal" class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="modal-template-title">
      <div class="modal-card">
        <h2 id="modal-template-title">Crea Sensore Template</h2>
        <form @submit.prevent="saveTemplate" class="modal-form">
          <div class="form-group">
            <label for="template-name">Nome Sensore</label>
            <input
              id="template-name"
              name="template_name"
              v-model="form.name"
              required
              placeholder="Es. Potenza Totale Casa"
              class="input"
            />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="template-id">Entity ID</label>
              <input
                id="template-id"
                name="template_entity_id"
                v-model="form.entity_id"
                required
                placeholder="sensor.potenza_totale"
                class="input"
              />
            </div>
            <div class="form-group">
              <label for="template-domain">Dominio</label>
              <select id="template-domain" name="template_domain" v-model="form.domain" class="input">
                <option value="sensor">sensor (Numerico / Stringa)</option>
                <option value="binary_sensor">binary_sensor (Booleano on/off)</option>
              </select>
            </div>
          </div>

          <div class="form-group">
            <label for="state-template">Formula Template (State)</label>
            <textarea
              id="state-template"
              name="state_template"
              v-model="form.state_template"
              required
              rows="3"
              placeholder="{{ states('sensor.fase_1') | float + states('sensor.fase_2') | float }}"
              class="input font-mono"
            ></textarea>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label for="unit">Unità di Misura</label>
              <input id="unit" name="unit_of_measurement" v-model="form.unit_of_measurement" placeholder="Es. W, kW, °C, %" class="input" />
            </div>
            <div class="form-group">
              <label for="class">Device Class</label>
              <input id="class" name="device_class" v-model="form.device_class" placeholder="power, energy, temperature" class="input" />
            </div>
          </div>

          <div class="modal-actions">
            <button type="button" class="btn btn-secondary" @click="closeModal">Annulla</button>
            <button type="submit" class="btn btn-primary" :disabled="saving">Crea Sensore</button>
          </div>
        </form>
      </div>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTemplatesStore } from '../stores/templates'
import type { TemplateEntity } from '../api/types'

const store = useTemplatesStore()
const playgroundCode = ref("{{ states('sensor.temperatura') | float + 2.5 }}")
const previewResult = ref('')
const testingPreview = ref(false)
const showModal = ref(false)
const saving = ref(false)

const form = ref<Partial<TemplateEntity>>({
  name: '',
  entity_id: 'sensor.',
  domain: 'sensor',
  state_template: '',
  unit_of_measurement: '',
  device_class: '',
  icon: 'ƒ(x)',
})

onMounted(async () => {
  await store.fetchTemplates()
})

async function testPreview() {
  if (!playgroundCode.value) return
  testingPreview.value = true
  try {
    previewResult.value = await store.previewTemplate(playgroundCode.value)
  } catch (err: any) {
    previewResult.value = 'Errore: ' + err.message
  } finally {
    testingPreview.value = false
  }
}

function openCreateModal() {
  form.value = {
    name: '',
    entity_id: 'sensor.',
    domain: 'sensor',
    state_template: '',
    unit_of_measurement: '',
    device_class: '',
    icon: 'ƒ(x)',
  }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

async function saveTemplate() {
  saving.value = true
  try {
    await store.createTemplate(form.value)
    closeModal()
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: string) {
  if (confirm('Eliminare questo sensore virtuale template?')) {
    await store.deleteTemplate(id)
  }
}
</script>

<style scoped>
.template-entities-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6, 1.5rem);
  padding: var(--space-6, 1.5rem);
  max-width: 1200px;
  margin: 0 auto;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

h1 {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--text-primary, #ffffff);
  margin: 0;
}

.subtitle {
  color: var(--text-secondary, #94a3b8);
  margin-top: 0.25rem;
  font-size: 0.95rem;
}

.playground-card {
  background: var(--surface-card, rgba(30, 41, 59, 0.7));
  border: 1px solid var(--border-subtle, rgba(255, 255, 255, 0.08));
  border-radius: var(--radius-lg, 12px);
  padding: 1.25rem;
  backdrop-filter: blur(12px);
}

.playground-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.playground-header h2 {
  font-size: 1.1rem;
  color: var(--text-primary, #ffffff);
  margin: 0;
}

.template-editor {
  width: 100%;
  font-family: monospace;
  font-size: 0.95rem;
  padding: 0.75rem;
  background: #090d16;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  color: #38bdf8;
  resize: vertical;
}

.preview-output {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.75rem;
  padding: 0.5rem 0.75rem;
  background: var(--bg-surface-hover);
  border: 1px solid var(--border-color);
  border-radius: 6px;
}

.preview-label {
  font-size: 0.85rem;
  color: var(--text-secondary, #94a3b8);
}

.preview-result {
  font-weight: 600;
  color: #4ade80;
  font-size: 1rem;
}

.templates-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 1rem;
  margin-top: 1rem;
}

.template-card {
  background: var(--surface-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md, 10px);
  padding: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  backdrop-filter: blur(8px);
}

.card-top {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.entity-icon {
  font-size: 1.25rem;
  font-weight: bold;
  color: var(--primary, #38bdf8);
}

.entity-title h3 {
  margin: 0;
  font-size: 1rem;
  color: var(--text-main);
}

.entity-id {
  font-size: 0.75rem;
  color: #94a3b8;
}

.domain-tag {
  margin-left: auto;
  font-size: 0.75rem;
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  background: rgba(56, 189, 248, 0.15);
  color: #38bdf8;
}

.template-code-preview {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  background: rgba(0, 0, 0, 0.3);
  padding: 0.5rem;
  border-radius: 6px;
  font-size: 0.85rem;
  color: #e2e8f0;
}

.code-label {
  font-size: 0.7rem;
  color: #94a3b8;
  text-transform: uppercase;
}

.card-footer {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.8rem;
  color: #94a3b8;
  margin-top: auto;
}

.btn-delete {
  margin-left: auto;
  background: transparent;
  border: none;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.btn-delete:hover {
  background: rgba(239, 68, 68, 0.15);
}

.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal-card {
  background: #1e293b;
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 1.5rem;
  width: 90%;
  max-width: 520px;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-top: 1rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
}

.input {
  padding: 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border-color);
  background: var(--bg-surface);
  color: var(--text-main);
}

.font-mono {
  font-family: monospace;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 1rem;
}

.btn {
  padding: 0.5rem 1rem;
  border-radius: 6px;
  cursor: pointer;
  font-weight: 600;
  min-height: 44px;
}

.btn-primary {
  background: var(--primary, #38bdf8);
  color: #0f172a;
  border: none;
}

.btn-secondary {
  background: var(--bg-surface-hover);
  color: var(--text-main);
  border: 1px solid var(--border-color);
}

.btn-sm {
  min-height: 36px;
  padding: 0.25rem 0.75rem;
  font-size: 0.85rem;
}
</style>
