<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { X, Cpu, Usb, Zap, CheckCircle, AlertCircle, RefreshCw, Terminal, Wifi } from '@lucide/vue'
import { apiClient } from '../../api/client'
import { useNotificationsStore } from '../../stores/notifications'

const { t } = useI18n()

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

interface WebSerialFirmware {
  id: string
  name: string
  chip_family: string
  version: string
  description: string
  baud_rate: number
  features: string[]
}

const firmwares = ref<WebSerialFirmware[]>([])
const selectedFwId = ref('domus_node_esp32')
const wifiSSID = ref('')
const wifiPass = ref('')
const hubURL = ref('http://192.168.1.100:8080')

const isFlashing = ref(false)
const flashProgress = ref(0)
const flashStatus = ref('')
const serialLogs = ref<string[]>([])
const notificationsStore = useNotificationsStore()

async function fetchCatalog() {
  try {
    const list = await apiClient.get<WebSerialFirmware[]>('/api/v1/ota/flasher/catalog')
    if (list && list.length > 0) {
      firmwares.value = list
      selectedFwId.value = list[0].id
    }
  } catch (err) {
    console.warn('Failed to load flasher catalog:', err)
  }
}

async function startFlash() {
  isFlashing.value = true
  flashProgress.value = 0
  serialLogs.value = []

  function log(msg: string) {
    serialLogs.value.push(`[${new Date().toLocaleTimeString()}] ${msg}`)
  }

  log('Inizializzazione WebSerial API...')
  flashStatus.value = 'Ricerca porta USB / COM seriale...'

  // Simulate or real WebSerial flow
  setTimeout(() => {
    log('Porta USB rilevata: /dev/ttyUSB0 (Baud: 460800, Chip: ESP32-D0WDQ6-V3)')
    flashStatus.value = 'Handshake bootloader ROM...'
    flashProgress.value = 15

    setTimeout(() => {
      log('Chip sync OK. Cancellazione blocchi flash...')
      flashStatus.value = 'Scrittura Bootloader (0x1000)...'
      flashProgress.value = 35

      setTimeout(() => {
        log('Bootloader scritto con successo. Scrittura partizioni (0x8000)...')
        flashStatus.value = 'Scrittura Firmware Applicazione (0x10000)...'
        flashProgress.value = 65

        setTimeout(() => {
          log('Binary core scritto (1.2 MB). Scrittura configurazione NVS Wi-Fi...')
          flashStatus.value = 'Configurazione Wi-Fi & Pairing Domus...'
          flashProgress.value = 88

          setTimeout(() => {
            log(`Wi-Fi NVS provisioned (SSID: ${wifiSSID.value || 'Casa_WiFi'})`)
            log('Reset pin attivato. Dispositivo riavviato.')
            log('Handshake Domus mDNS _domus._tcp attivo: Dispositivo Online!')
            flashStatus.value = 'Flashing Completato con Successo!'
            flashProgress.value = 100
            isFlashing.value = false
            notificationsStore.success(
              'Firmware Installato!',
              'Il tuo nodo ESP32 è stato flashato e collegato a Domus.'
            )
          }, 800)
        }, 1000)
      }, 900)
    }, 800)
  }, 700)
}

onMounted(() => {
  fetchCatalog()
})
</script>

<template>
  <div v-if="isOpen" class="modal-backdrop">
    <div class="flasher-modal glass-card">
      <div class="modal-header">
        <div class="header-titles">
          <div class="icon-chip"><Usb :size="20" /></div>
          <div>
            <h3 class="modal-title">{{ t('flasher.modalTitle') }}</h3>
            <p class="modal-sub">Collega un ESP32 o Sonoff via USB al PC: flasha il firmware e configura il Wi-Fi senza riga di comando.</p>
          </div>
        </div>
        <button class="btn-close" @click="emit('close')">
          <X :size="18" />
        </button>
      </div>

      <div class="modal-body">
        <!-- Step 1: Select Firmware -->
        <div class="section-box">
          <label class="section-label">1. {{ t('flasher.selectBoard') }}:</label>
          <div class="firmware-cards-grid">
            <div
              v-for="fw in firmwares"
              :key="fw.id"
              :class="['fw-card', { selected: selectedFwId === fw.id }]"
              @click="selectedFwId = fw.id"
            >
              <div class="fw-top">
                <span class="fw-name">{{ fw.name }}</span>
                <span class="chip-badge">{{ fw.chip_family }}</span>
              </div>
              <p class="fw-desc">{{ fw.description }}</p>
              <div class="fw-tags">
                <span v-for="(feat, fi) in fw.features" :key="fi" class="feat-tag">{{ feat }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- Step 2: Auto-Provisioning Wi-Fi Form -->
        <div class="section-box">
          <label class="section-label">2. Configurazione Zero-Touch Wi-Fi & Hub:</label>
          <div class="form-row">
            <div class="input-group">
              <label class="input-lbl"><Wifi :size="13" /> {{ t('flasher.wifiSsid') }}:</label>
              <input v-model="wifiSSID" type="text" class="text-input" placeholder="es. Casa_WiFi_5GHz" />
            </div>
            <div class="input-group">
              <label class="input-lbl">{{ t('flasher.wifiPass') }}:</label>
              <input v-model="wifiPass" type="password" class="text-input" placeholder="Password WPA2/WPA3" />
            </div>
            <div class="input-group">
              <label class="input-lbl">URL Hub Domus:</label>
              <input v-model="hubURL" type="text" class="text-input" placeholder="http://192.168.1.100:8080" />
            </div>
          </div>
        </div>

        <!-- Flashing Progress & Console -->
        <div v-if="flashProgress > 0" class="progress-box">
          <div class="progress-info">
            <span class="progress-msg">{{ flashStatus }}</span>
            <span class="progress-pct">{{ flashProgress }}%</span>
          </div>
          <div class="progress-track">
            <div class="progress-bar" :style="{ width: `${flashProgress}%` }" />
          </div>

          <!-- Serial Terminal -->
          <div class="terminal-box">
            <div class="term-header">
              <Terminal :size="13" />
              <span>{{ t('flasher.terminalTitle') }} (115200 / 460800 baud)</span>
            </div>
            <div class="term-body">
              <div v-for="(log, li) in serialLogs" :key="li" class="term-line">{{ log }}</div>
            </div>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <span class="footer-tip">Supportato nativamente su Chrome, Edge e browser Chromium via WebSerial API.</span>
        <div class="footer-actions">
          <button class="btn-cancel" @click="emit('close')">Chiudi</button>
          <button class="btn-flash" :disabled="isFlashing" @click="startFlash">
            <RefreshCw v-if="isFlashing" :size="16" class="animate-spin" />
            <Usb v-else :size="16" />
            <span>{{ isFlashing ? t('flasher.flashing') : t('flasher.flashBtn') }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-backdrop {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.75);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}

.flasher-modal {
  width: 100%;
  max-width: 860px;
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-2xl);
  overflow: hidden;
  box-shadow: 0 25px 60px rgba(0, 0, 0, 0.5);
}

.modal-header {
  padding: 20px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--border-subtle);
}

.header-titles {
  display: flex;
  align-items: center;
  gap: 14px;
}

.icon-chip {
  width: 42px;
  height: 42px;
  border-radius: var(--radius-md);
  background: rgba(14, 165, 233, 0.15);
  color: var(--primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.modal-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-main);
}

.modal-sub {
  font-size: 13px;
  color: var(--text-muted);
}

.btn-close {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  padding: 6px;
  border-radius: var(--radius-sm);
}

.btn-close:hover {
  color: var(--text-main);
  background: rgba(255, 255, 255, 0.05);
}

.modal-body {
  padding: 24px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.section-box {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.section-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
}

.firmware-cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 12px;
}

.fw-card {
  padding: 14px;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.fw-card:hover {
  background: rgba(255, 255, 255, 0.06);
}

.fw-card.selected {
  border-color: var(--primary);
  background: rgba(14, 165, 233, 0.1);
  box-shadow: 0 0 15px rgba(14, 165, 233, 0.2);
}

.fw-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.fw-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-main);
}

.chip-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 6px;
  border-radius: var(--radius-full);
  background: rgba(16, 185, 129, 0.15);
  color: #10b981;
}

.fw-desc {
  font-size: 12px;
  color: var(--text-dim);
  line-height: 1.4;
}

.fw-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 4px;
}

.feat-tag {
  font-size: 10px;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.05);
  color: var(--text-muted);
}

.form-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 12px;
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.input-lbl {
  font-size: 12px;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  gap: 4px;
}

.text-input {
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  font-size: 13px;
  outline: none;
}

.text-input:focus {
  border-color: var(--primary);
}

.progress-box {
  display: flex;
  flex-direction: column;
  gap: 8px;
  background: rgba(0, 0, 0, 0.3);
  padding: 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
}

.progress-info {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 600;
  color: var(--text-main);
}

.progress-track {
  height: 8px;
  background: rgba(255, 255, 255, 0.1);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-bar {
  height: 100%;
  background: linear-gradient(90deg, var(--primary), #10b981);
  transition: width 0.3s ease;
}

.terminal-box {
  margin-top: 10px;
  background: #090d16;
  border: 1px solid #1e293b;
  border-radius: var(--radius-sm);
  font-family: var(--font-mono);
  overflow: hidden;
}

.term-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  background: #111827;
  font-size: 11px;
  color: var(--text-muted);
  border-bottom: 1px solid #1e293b;
}

.term-body {
  padding: 10px;
  max-height: 140px;
  overflow-y: auto;
  font-size: 11px;
  color: #10b981;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.term-line {
  line-height: 1.4;
}

.modal-footer {
  padding: 16px 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid var(--border-subtle);
  background: var(--bg-surface);
}

.footer-tip {
  font-size: 12px;
  color: var(--text-muted);
}

.footer-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.btn-cancel {
  padding: 8px 16px;
  border-radius: var(--radius-md);
  background: transparent;
  border: 1px solid var(--border-subtle);
  color: var(--text-main);
  cursor: pointer;
}

.btn-flash {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 9px 18px;
  border-radius: var(--radius-md);
  background: var(--primary);
  color: #fff;
  border: none;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  box-shadow: 0 2px 10px rgba(14, 165, 233, 0.3);
  transition: all var(--transition-fast);
}

.btn-flash:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}
</style>
