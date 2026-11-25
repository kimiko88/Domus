<script setup lang="ts">
import { onBeforeUnmount, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Camera,
  FileText,
  FolderOpen,
  ImagePlus,
  Sparkles,
  ShoppingBag,
  Plus,
  Trash2,
  Calendar,
  PackageCheck,
  RotateCw,
  X,
  ScanLine
} from '@lucide/vue'
import type { Receipt, ReceiptItem, PantryCategory, PantryUnit } from '../../api/types'
import { decodeHeicBitmap } from '../../utils/heic'

interface Props {
  parsedReceipt: Receipt | null
  loading?: boolean
}

const props = defineProps<Props>()
const { t } = useI18n()
const emit = defineEmits<{
  (e: 'parse-receipt', rawText: string): void
  (e: 'parse-pdf', file: File): void
  (e: 'parse-image', file: File): void
  (e: 'import-receipt', data: {
    store_name: string
    date: string
    items: ReceiptItem[]
    total_amount: number
    document_type: 'receipt' | 'invoice'
    auto_add_to_finance: boolean
    auto_update_pantry: boolean
  }): void
  (e: 'clear-parsed'): void
}>()

const rawReceiptText = ref('')
const documentType = ref<'receipt' | 'invoice'>('receipt')
const ocrStatus = ref('')
const maxImageBytes = 4 * 1024 * 1024
const maxSourceImageBytes = 32 * 1024 * 1024
const maxPDFBytes = 3 * 1024 * 1024
const autoAddToFinance = ref(true)
const autoUpdatePantry = ref(true)

const fileInput = ref<HTMLInputElement | null>(null)
const cameraFallbackInput = ref<HTMLInputElement | null>(null)
const videoElement = ref<HTMLVideoElement | null>(null)
const cameraOpen = ref(false)
const cameraError = ref('')
const cameraBusy = ref(false)
const cameraStream = ref<MediaStream | null>(null)
const sourceImage = ref<File | null>(null)
const pendingImage = ref<File | null>(null)
const pendingPDF = ref<File | null>(null)
const previewURL = ref('')
const imageRotation = ref(0)
let requestStarted = false
let cameraRequestId = 0
let componentActive = true

const categories: { key: PantryCategory; label: string }[] = [
  { key: 'produce', label: '🍎 Ortofrutta' },
  { key: 'dairy', label: '🧀 Latticini' },
  { key: 'meat_fish', label: '🥩 Carne/Pesce' },
  { key: 'bakery', label: '🥖 Forno' },
  { key: 'pantry', label: '🍝 Dispensa' },
  { key: 'frozen', label: '❄️ Surgelati' },
  { key: 'drinks', label: '🥤 Bevande' },
  { key: 'condiments', label: '🫒 Condimenti' },
  { key: 'snacks', label: '🍪 Snack' },
  { key: 'household', label: '🧼 Casa' },
  { key: 'other', label: '📦 Altro' }
]

const sampleReceiptEsselunga = `ESSELUNGA S.P.A.
VIA PISACANE 12 - MILANO
DATA 18/10/2026 ORA 19:15
PASTA PENNE RIGATE 500G 1.30
RISO CARNAROLI 1KG 2.79
LATTE PARZ SCREM 1L 1.49
MELE GOLDEN KG 1.500 2.99
MACINATO MANZO 450G 4.90
ACQUA NATURALE 6X1.5L 2.10
SUBTOTALE EURO 15.57
TOTALE COMPLESSIVO EURO 15.57
PAGAMENTO POS 15.57`

const sampleReceiptCoop = `COOP LOMBARDIA
DATA 17/10/2026
PANE INTEGRALE 500G 1.80
YOGURT GRECO BIANCO 2PZ 2.20
PASSATA DI POMODORO 700G 0.99
BANANE BIO KG 1.200 2.40
TOTALE EURO 7.39`

function loadSample(sample: string) {
  rawReceiptText.value = sample
  resetSelectedFile()
}

function revokePreview() {
  if (previewURL.value) URL.revokeObjectURL(previewURL.value)
  previewURL.value = ''
}

function resetSelectedFile() {
  revokePreview()
  sourceImage.value = null
  pendingImage.value = null
  pendingPDF.value = null
  imageRotation.value = 0
}

function stopCamera() {
  cameraStream.value?.getTracks().forEach(track => track.stop())
  cameraStream.value = null
  if (videoElement.value) videoElement.value.srcObject = null
  cameraOpen.value = false
  cameraBusy.value = false
}

async function startCamera() {
  cameraError.value = ''
  const requestId = ++cameraRequestId
  if (!navigator.mediaDevices?.getUserMedia) {
    cameraFallbackInput.value?.click()
    return
  }

  cameraBusy.value = true
  try {
    const stream = await navigator.mediaDevices.getUserMedia({
      audio: false,
      video: {
        facingMode: { ideal: 'environment' },
        width: { ideal: 1920 },
        height: { ideal: 1080 }
      }
    })
    if (!componentActive || requestId !== cameraRequestId) {
      stream.getTracks().forEach(track => track.stop())
      return
    }
    cameraStream.value = stream
    cameraOpen.value = true
    await nextTick()
    if (!componentActive || requestId !== cameraRequestId) {
      stream.getTracks().forEach(track => track.stop())
      cameraStream.value = null
      return
    }
    if (!videoElement.value) throw new Error('Camera preview is not available')
    videoElement.value.srcObject = stream
    await videoElement.value.play()
  } catch {
    if (componentActive && requestId === cameraRequestId) {
      stopCamera()
      cameraError.value = t('shopping.view.receiptScanner.cameraUnavailable')
    }
  } finally {
    if (componentActive && requestId === cameraRequestId) cameraBusy.value = false
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    canvas.toBlob(blob => {
      if (blob) resolve(blob)
      else reject(new Error('Image conversion failed'))
    }, 'image/jpeg', quality)
  })
}

type LoadedReceiptImage = {
  source: CanvasImageSource
  width: number
  height: number
  dispose: () => void
}

function hasHeicHint(file: File): boolean {
  return /\.(heic|heif)$/i.test(file.name) ||
    ['image/heic', 'image/heif', 'image/heic-sequence', 'image/heif-sequence'].includes(file.type.toLowerCase())
}

async function loadHeicImage(file: File): Promise<LoadedReceiptImage | null> {
  // Load the decoder only for files that need it; normal JPEG/PNG uploads do
  // not pay the WASM download or initialization cost.
  const bitmap = await decodeHeicBitmap(file)
  if (!bitmap) return null
  if (!bitmap.width || !bitmap.height) {
    bitmap.close()
    throw new Error('HEIC image has invalid dimensions')
  }

  return {
    source: bitmap,
    width: bitmap.width,
    height: bitmap.height,
    dispose: () => bitmap.close()
  }
}

async function waitForBrowserImage(image: HTMLImageElement): Promise<void> {
  if (image.complete) {
    if (image.naturalWidth > 0 && image.naturalHeight > 0) return
    throw new Error('Unsupported or damaged image format')
  }

  await new Promise<void>((resolve, reject) => {
    image.addEventListener('load', () => resolve(), { once: true })
    image.addEventListener('error', () => reject(new Error('Unsupported or damaged image format')), { once: true })
  })

  if (!image.naturalWidth || !image.naturalHeight) {
    throw new Error('Unsupported or damaged image format')
  }
}

async function loadBrowserImage(file: File): Promise<LoadedReceiptImage> {
  // iOS Photos commonly supplies HEIC/HEIF files. Safari's HTMLImageElement
  // decoder does not handle every such file, so use the local libheif/WASM
  // decoder before falling back to the browser's native image pipeline.
  if (hasHeicHint(file)) {
    try {
      const heicImage = await loadHeicImage(file)
      if (heicImage) return heicImage
    } catch {
      // Some iOS providers use a generic MIME type or a renamed file. Try
      // the normal browser decoder below before reporting a useful error.
    }
  }

  const objectURL = URL.createObjectURL(file)
  try {
    const image = new Image()
    image.decoding = 'async'
    image.src = objectURL
    if (typeof image.decode === 'function') {
      try {
        await image.decode()
      } catch (decodeError) {
        // Safari can reject decode() even when the image's load event succeeds.
        // Fall back to the actual load state before treating it as unreadable.
        try {
          await waitForBrowserImage(image)
        } catch {
          throw decodeError
        }
      }
    } else {
      await waitForBrowserImage(image)
    }

    if (!image.naturalWidth || !image.naturalHeight) {
      throw new Error('Unsupported or damaged image format')
    }

    return {
      source: image,
      width: image.naturalWidth,
      height: image.naturalHeight,
      dispose: () => URL.revokeObjectURL(objectURL)
    }
  } catch (browserError) {
    URL.revokeObjectURL(objectURL)

    // Last-chance signature detection handles HEIC files returned by iOS
    // without a useful filename extension or MIME type.
    if (!hasHeicHint(file)) {
      try {
        const heicImage = await loadHeicImage(file)
        if (heicImage) return heicImage
      } catch {
        // Preserve the original browser error for unsupported non-HEIC files.
      }
    }
    throw browserError
  }
}

async function normalizeReceiptImage(file: File, rotation = 0): Promise<File> {
  if (file.size > maxSourceImageBytes) {
    throw new Error(t('shopping.view.receiptScanner.sourceImageTooLarge'))
  }

  const loaded = await loadBrowserImage(file)
  try {
  const sourceWidth = loaded.width
  const sourceHeight = loaded.height
  if (!sourceWidth || !sourceHeight || sourceWidth * sourceHeight > 60_000_000) {
    throw new Error(t('shopping.view.receiptScanner.imageDimensionsTooLarge'))
  }

  const maxSide = 2600
  const scale = Math.min(1, maxSide / Math.max(sourceWidth, sourceHeight))
  const drawWidth = Math.max(1, Math.round(sourceWidth * scale))
  const drawHeight = Math.max(1, Math.round(sourceHeight * scale))
  const angle = ((rotation % 360) + 360) % 360
  const swapped = angle === 90 || angle === 270
  const width = swapped ? drawHeight : drawWidth
  const height = swapped ? drawWidth : drawHeight

  for (const factor of [1, 0.82, 0.66, 0.52]) {
    const canvas = document.createElement('canvas')
    canvas.width = Math.max(1, Math.round(width * factor))
    canvas.height = Math.max(1, Math.round(height * factor))
    const context = canvas.getContext('2d', { alpha: false })
    if (!context) throw new Error('Canvas is not available')
    context.fillStyle = '#fff'
    context.fillRect(0, 0, canvas.width, canvas.height)
    context.translate(canvas.width / 2, canvas.height / 2)
    context.rotate(angle * Math.PI / 180)
    context.drawImage(
      loaded.source,
      -drawWidth * factor / 2,
      -drawHeight * factor / 2,
      drawWidth * factor,
      drawHeight * factor
    )
    const blob = await canvasToBlob(canvas, 0.86)
    canvas.width = 0
    canvas.height = 0
    if (blob.size <= maxImageBytes) {
      return new File([blob], 'domus-receipt-scan.jpg', {
        type: 'image/jpeg',
        lastModified: Date.now()
      })
    }
  }

  throw new Error(t('shopping.view.receiptScanner.imageStillTooLarge'))
  } finally {
    loaded.dispose()
  }
}

async function setReceiptImage(file: File) {
  ocrStatus.value = t('shopping.view.receiptScanner.preparingImage')
  pendingPDF.value = null
  sourceImage.value = file
  imageRotation.value = 0
  try {
    const normalized = await normalizeReceiptImage(file)
    revokePreview()
    pendingImage.value = normalized
    previewURL.value = URL.createObjectURL(normalized)
    ocrStatus.value = t('shopping.view.receiptScanner.imageReady')
  } catch (error) {
    sourceImage.value = null
    pendingImage.value = null
    revokePreview()
    ocrStatus.value = error instanceof Error ? error.message : t('shopping.view.receiptScanner.imagePrepareFailed')
  }
}

async function handleFile(file: File) {
  stopCamera()
  cameraError.value = ''
  rawReceiptText.value = ''
  revokePreview()
  sourceImage.value = null
  pendingImage.value = null
  pendingPDF.value = null

  const isPDF = file.type === 'application/pdf' || /\.pdf$/i.test(file.name)
  if (isPDF) {
    if (file.size > maxPDFBytes) {
      ocrStatus.value = t('shopping.view.receiptScanner.pdfTooLarge')
      return
    }
    pendingPDF.value = file
    ocrStatus.value = t('shopping.view.receiptScanner.pdfReady')
    return
  }

  const isKnownImage = file.type.startsWith('image/') || /\.(jpe?g|png|webp|gif|heic|heif|tiff?)$/i.test(file.name)
  if (!isKnownImage) {
    ocrStatus.value = t('shopping.view.receiptScanner.unsupportedFile')
    return
  }
  await setReceiptImage(file)
}

async function handleFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (file) await handleFile(file)
}

async function captureCameraFrame() {
  const video = videoElement.value
  if (!video || !video.videoWidth || !video.videoHeight) {
    cameraError.value = t('shopping.view.receiptScanner.cameraNotReady')
    return
  }

  const videoWidth = video.videoWidth
  const videoHeight = video.videoHeight
  // Match the 3:4 camera viewport (object-fit: cover), then leave a small
  // safety margin so the receipt border is not cut by the crop.
  const frameRatio = 3 / 4
  let cropWidth = videoWidth
  let cropHeight = videoHeight
  if (videoWidth / videoHeight > frameRatio) {
    cropWidth = videoHeight * frameRatio
  } else {
    cropHeight = videoWidth / frameRatio
  }
  const cropX = (videoWidth - cropWidth) / 2 + cropWidth * 0.025
  const cropY = (videoHeight - cropHeight) / 2 + cropHeight * 0.025
  cropWidth *= 0.95
  cropHeight *= 0.95

  const canvas = document.createElement('canvas')
  const scale = Math.min(1, 2600 / Math.max(cropWidth, cropHeight))
  canvas.width = Math.round(cropWidth * scale)
  canvas.height = Math.round(cropHeight * scale)
  const context = canvas.getContext('2d', { alpha: false })
  if (!context) return
  context.fillStyle = '#fff'
  context.fillRect(0, 0, canvas.width, canvas.height)
  context.drawImage(
    video,
    cropX, cropY, cropWidth, cropHeight,
    0, 0, canvas.width, canvas.height
  )

  try {
    const blob = await canvasToBlob(canvas, 0.88)
    const file = new File([blob], 'domus-camera-scan.jpg', { type: 'image/jpeg', lastModified: Date.now() })
    stopCamera()
    await setReceiptImage(file)
  } catch {
    cameraError.value = t('shopping.view.receiptScanner.captureFailed')
  } finally {
    canvas.width = 0
    canvas.height = 0
  }
}

async function rotateReceiptImage() {
  if (!sourceImage.value || props.loading) return
  imageRotation.value = (imageRotation.value + 90) % 360
  ocrStatus.value = t('shopping.view.receiptScanner.preparingImage')
  try {
    const normalized = await normalizeReceiptImage(sourceImage.value, imageRotation.value)
    revokePreview()
    pendingImage.value = normalized
    previewURL.value = URL.createObjectURL(normalized)
    ocrStatus.value = t('shopping.view.receiptScanner.imageReady')
  } catch {
    ocrStatus.value = t('shopping.view.receiptScanner.imagePrepareFailed')
  }
}

function analyzeSelectedImage() {
  if (!pendingImage.value || props.loading) return
  ocrStatus.value = t('shopping.view.receiptScanner.ocrInProgress')
  emit('parse-image', pendingImage.value)
}

function analyzeSelectedPDF() {
  if (!pendingPDF.value || props.loading) return
  ocrStatus.value = t('shopping.view.receiptScanner.pdfInProgress')
  emit('parse-pdf', pendingPDF.value)
}

function handleParse() {
  if (!rawReceiptText.value.trim() || props.loading) return
  ocrStatus.value = t('shopping.view.receiptScanner.analysisInProgress')
  emit('parse-receipt', rawReceiptText.value.trim())
}

function removeReceiptItem(index: number) {
  if (!props.parsedReceipt) return
  props.parsedReceipt.items.splice(index, 1)
  recalculateTotal()
}

function addReceiptItem() {
  if (!props.parsedReceipt) return
  props.parsedReceipt.items.push({
    raw_line: '',
    name: t('shopping.view.receiptScanner.manualItem'),
    quantity: 1,
    unit: 'pz' as PantryUnit,
    unit_price: 0,
    total_price: 0,
    category: 'other'
  })
}

function recalculateTotal() {
  if (!props.parsedReceipt) return
  const sum = props.parsedReceipt.items.reduce((acc, curr) => acc + (Number(curr.total_price) || 0), 0)
  props.parsedReceipt.total_amount = Math.round(sum * 100) / 100
}

function handleConfirmImport() {
  if (!props.parsedReceipt) return
  for (const item of props.parsedReceipt.items) {
    const quantity = Number(item.quantity) || 0
    item.quantity = quantity
    item.total_price = Number(item.total_price) || 0
    item.unit_price = quantity > 0 ? item.total_price / quantity : 0
  }
  emit('import-receipt', {
    store_name: props.parsedReceipt.store_name,
    date: props.parsedReceipt.date || new Date().toISOString().slice(0, 10),
    items: props.parsedReceipt.items,
    total_amount: Number(props.parsedReceipt.total_amount) || 0,
    document_type: documentType.value,
    auto_add_to_finance: autoAddToFinance.value,
    auto_update_pantry: documentType.value === 'receipt' && autoUpdatePantry.value
  })
}

function clearAndRestart() {
  resetSelectedFile()
  rawReceiptText.value = ''
  ocrStatus.value = ''
  emit('clear-parsed')
}

watch(() => props.loading, value => {
  if (value) {
    requestStarted = true
  } else if (requestStarted) {
    requestStarted = false
    if (!props.parsedReceipt) ocrStatus.value = t('shopping.view.receiptScanner.ocrFailed')
  }
})

watch(() => props.parsedReceipt, receipt => {
  if (receipt) resetSelectedFile()
})

onBeforeUnmount(() => {
  componentActive = false
  cameraRequestId++
  stopCamera()
  revokePreview()
})
</script>

<template>
  <div class="scanner-tab-content">
    <div v-if="!parsedReceipt" class="scanner-input-card glass-panel">
      <div class="document-type-control">
        <label for="document-type">{{ t('shopping.view.receiptScanner.documentType') }}</label>
        <select id="document-type" v-model="documentType" class="table-select">
          <option value="receipt">{{ t('shopping.view.receiptScanner.receipt') }}</option>
          <option value="invoice">{{ t('shopping.view.receiptScanner.invoice') }}</option>
        </select>
      </div>

      <div class="card-intro">
        <div class="intro-icon bg-primary-soft">
          <ScanLine :size="24" class="text-primary" />
        </div>
        <div>
          <h3 class="intro-title">{{ t('shopping.view.receiptScanner.title') }}</h3>
          <p class="intro-desc">{{ t('shopping.view.receiptScanner.subtitle') }}</p>
        </div>
      </div>

      <div class="scan-methods">
        <button class="scan-method scan-method-primary" type="button" :disabled="loading || cameraBusy" @click="startCamera">
          <Camera :size="22" />
          <span>
            <strong>{{ cameraBusy ? t('shopping.view.receiptScanner.openingCamera') : t('shopping.view.receiptScanner.scanWithCamera') }}</strong>
            <small>{{ t('shopping.view.receiptScanner.cameraHint') }}</small>
          </span>
        </button>
        <button class="scan-method" type="button" :disabled="loading" @click="fileInput?.click()">
          <FolderOpen :size="22" />
          <span>
            <strong>{{ t('shopping.view.receiptScanner.chooseFile') }}</strong>
            <small>{{ t('shopping.view.receiptScanner.fileHint') }}</small>
          </span>
        </button>
        <input
          ref="fileInput"
          class="visually-hidden"
          type="file"
          accept="image/*,application/pdf,.jpg,.jpeg,.png,.webp,.heic,.heif,.pdf"
          @change="handleFileSelected"
        />
        <input
          id="receipt-image-input"
          ref="cameraFallbackInput"
          class="visually-hidden"
          type="file"
          accept="image/*"
          capture="environment"
          @change="handleFileSelected"
        />
      </div>

      <div v-if="cameraError" class="camera-fallback-actions">
        <p class="ocr-status ocr-error" role="alert">{{ cameraError }}</p>
        <button type="button" class="btn btn-secondary" @click="cameraFallbackInput?.click()">
          {{ t('shopping.view.receiptScanner.cameraFallbackButton') }}
        </button>
      </div>

      <div v-if="pendingImage && previewURL" class="selected-document-card">
        <div class="selected-document-heading">
          <div class="selected-document-label">
            <ImagePlus :size="18" />
            <div>
              <strong>{{ t('shopping.view.receiptScanner.imageReadyTitle') }}</strong>
              <small>{{ sourceImage?.name || pendingImage.name }} · {{ (pendingImage.size / 1024).toFixed(0) }} KB</small>
            </div>
          </div>
          <button class="icon-action" type="button" :aria-label="t('shopping.view.receiptScanner.removeFile')" @click="resetSelectedFile">
            <X :size="18" />
          </button>
        </div>
        <div class="receipt-image-preview">
          <img :src="previewURL" :alt="t('shopping.view.receiptScanner.imagePreviewAlt')" />
        </div>
        <div class="selected-document-actions">
          <button class="btn btn-secondary" type="button" :disabled="loading" @click="rotateReceiptImage">
            <RotateCw :size="16" /> {{ t('shopping.view.receiptScanner.rotateImage') }}
          </button>
          <button class="btn btn-primary" type="button" :disabled="loading" @click="analyzeSelectedImage">
            <Sparkles :size="18" /> {{ loading ? t('shopping.view.receiptScanner.ocrInProgress') : t('shopping.view.receiptScanner.readReceipt') }}
          </button>
        </div>
      </div>

      <div v-else-if="pendingPDF" class="selected-document-card selected-pdf-card">
        <div class="selected-document-label">
          <FileText :size="22" />
          <div>
            <strong>{{ pendingPDF.name }}</strong>
            <small>{{ (pendingPDF.size / 1024).toFixed(0) }} KB · {{ t('shopping.view.receiptScanner.pdfReady') }}</small>
          </div>
        </div>
        <button class="icon-action" type="button" :aria-label="t('shopping.view.receiptScanner.removeFile')" @click="resetSelectedFile">
          <X :size="18" />
        </button>
        <button class="btn btn-primary" type="button" :disabled="loading" @click="analyzeSelectedPDF">
          <Sparkles :size="18" /> {{ loading ? t('shopping.view.receiptScanner.ocrInProgress') : t('shopping.view.receiptScanner.readPDF') }}
        </button>
      </div>

      <p v-if="ocrStatus" class="ocr-status" role="status" aria-live="polite">{{ ocrStatus }}</p>
      <p class="ocr-privacy-note">{{ t('shopping.view.receiptScanner.privacy') }}</p>

      <div v-if="cameraOpen" class="camera-scan-panel" role="dialog" aria-modal="true" :aria-label="t('shopping.view.receiptScanner.cameraDialogTitle')">
        <div class="camera-scan-header">
          <div>
            <strong>{{ t('shopping.view.receiptScanner.cameraDialogTitle') }}</strong>
            <p>{{ t('shopping.view.receiptScanner.cameraDialogHint') }}</p>
          </div>
          <button class="icon-action" type="button" :aria-label="t('shopping.view.receiptScanner.closeCamera')" @click="stopCamera">
            <X :size="20" />
          </button>
        </div>
        <div class="camera-feed">
          <video ref="videoElement" autoplay muted playsinline />
          <div class="document-frame" aria-hidden="true">
            <i class="corner corner-tl"></i><i class="corner corner-tr"></i>
            <i class="corner corner-bl"></i><i class="corner corner-br"></i>
            <span>{{ t('shopping.view.receiptScanner.alignReceipt') }}</span>
          </div>
        </div>
        <p class="camera-scan-tip">{{ t('shopping.view.receiptScanner.cameraTip') }}</p>
        <div class="camera-scan-actions">
          <button class="btn btn-secondary" type="button" @click="stopCamera">{{ t('shopping.view.receiptScanner.cancel') }}</button>
          <button class="btn btn-primary" type="button" @click="captureCameraFrame">
            <Camera :size="18" /> {{ t('shopping.view.receiptScanner.takePhoto') }}
          </button>
        </div>
      </div>

      <div class="quick-samples">
        <span class="samples-label">{{ t('shopping.view.receiptScanner.sampleHint') }}</span>
        <button type="button" class="sample-btn" @click="loadSample(sampleReceiptEsselunga)">🛒 Scontrino Esselunga</button>
        <button type="button" class="sample-btn" @click="loadSample(sampleReceiptCoop)">🏪 Scontrino Coop</button>
      </div>

      <div class="textarea-wrap">
        <label class="samples-label" for="receipt-raw-text">{{ t('shopping.view.receiptScanner.pasteText') }}</label>
        <textarea
          id="receipt-raw-text"
          v-model="rawReceiptText"
          rows="7"
          class="receipt-textarea"
          :placeholder="t('shopping.view.receiptScanner.pastePlaceholder')"
          :aria-label="t('shopping.view.receiptScanner.pasteText')"
        ></textarea>
      </div>
      <div class="input-actions">
        <button type="button" class="btn btn-primary" :disabled="!rawReceiptText.trim() || loading" @click="handleParse">
          <Sparkles :size="18" />
          <span>{{ loading ? t('shopping.view.receiptScanner.analysisInProgress') : t('shopping.view.receiptScanner.analyzeText') }}</span>
        </button>
      </div>
    </div>

    <div v-else class="scanner-preview-card glass-panel">
      <div class="preview-header">
        <div class="header-left">
          <div class="store-badge">
            <ShoppingBag :size="20" class="text-primary" />
            <input v-model="parsedReceipt.store_name" class="store-name-input" :aria-label="t('shopping.view.receiptScanner.storeName')" />
          </div>
          <div class="date-badge">
            <Calendar :size="16" class="text-muted" />
            <input v-model="parsedReceipt.date" type="date" class="date-input" :aria-label="t('shopping.view.receiptScanner.documentDate')" />
          </div>
        </div>
        <div class="header-right">
          <div class="total-badge">
            <label class="total-label" for="receipt-total">{{ t('shopping.view.receiptScanner.total') }}</label>
            <input id="receipt-total" v-model.number="parsedReceipt.total_amount" class="total-amount total-input" type="number" step="0.01" min="0" />
            <span class="total-label">€</span>
          </div>
          <button type="button" class="btn-ghost" :title="t('shopping.view.receiptScanner.rescan')" @click="clearAndRestart">
            {{ t('shopping.view.receiptScanner.rescan') }}
          </button>
        </div>
      </div>

      <div class="receipt-review-meta">
        <span>{{ t('shopping.view.receiptScanner.itemsFound', { count: parsedReceipt.items.length }) }}</span>
        <span>{{ t('shopping.view.receiptScanner.confidence', { value: Math.round((parsedReceipt.confidence || 0) * 100) }) }}</span>
        <div class="confidence-track"><span :style="{ width: Math.max(0, Math.min(100, (parsedReceipt.confidence || 0) * 100)) + '%' }"></span></div>
      </div>

      <details v-if="parsedReceipt.raw_text" class="ocr-raw-details">
        <summary>{{ t('shopping.view.receiptScanner.showOriginalText') }}</summary>
        <pre>{{ parsedReceipt.raw_text }}</pre>
      </details>

      <div class="items-table-wrap">
        <table class="receipt-table" role="table">
          <thead>
            <tr>
              <th>{{ t('shopping.view.receiptScanner.product') }}</th>
              <th>{{ t('shopping.view.receiptScanner.quantity') }}</th>
              <th>{{ t('shopping.view.receiptScanner.unit') }}</th>
              <th>{{ t('shopping.view.receiptScanner.price') }}</th>
              <th>{{ t('shopping.view.receiptScanner.category') }}</th>
              <th :aria-label="t('shopping.view.receiptScanner.actions')"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, idx) in parsedReceipt.items" :key="idx">
              <td><input v-model="it.name" class="table-input item-name" :aria-label="t('shopping.view.receiptScanner.product')" /></td>
              <td class="qty-col">
                <input v-model.number="it.quantity" type="number" step="any" min="0.001" class="table-input item-qty" :aria-label="t('shopping.view.receiptScanner.quantity')" />
              </td>
              <td class="unit-col">
                <select v-model="it.unit" class="table-select" :aria-label="t('shopping.view.receiptScanner.unit')">
                  <option value="pz">pz</option><option value="g">g</option><option value="kg">kg</option>
                  <option value="ml">ml</option><option value="l">L</option><option value="pack">pack</option>
                </select>
              </td>
              <td class="price-col">
                <input v-model.number="it.total_price" type="number" step="0.01" min="0" class="table-input item-price" :aria-label="t('shopping.view.receiptScanner.price')" />
              </td>
              <td>
                <select v-model="it.category" class="table-select cat-select" :aria-label="t('shopping.view.receiptScanner.category')">
                  <option v-for="category in categories" :key="category.key" :value="category.key">{{ category.label }}</option>
                </select>
              </td>
              <td class="action-col">
                <button type="button" class="table-del-btn" :title="t('shopping.view.receiptScanner.removeProduct')" :aria-label="t('shopping.view.receiptScanner.removeProduct')" @click="removeReceiptItem(idx)">
                  <Trash2 :size="15" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <button type="button" class="btn btn-secondary add-manual-item" @click="addReceiptItem">
        <Plus :size="16" /> {{ t('shopping.view.receiptScanner.addManualItem') }}
      </button>

      <div class="integration-switches-box">
        <label class="switch-row" :class="{ 'switch-disabled': documentType === 'invoice' }">
          <input v-model="autoUpdatePantry" type="checkbox" :disabled="documentType === 'invoice'" />
          <div class="switch-info">
            <span class="switch-title">📦 {{ t('shopping.view.receiptScanner.updatePantry') }}</span>
            <span class="switch-desc">{{ documentType === 'invoice' ? t('shopping.view.receiptScanner.invoiceNoPantry') : t('shopping.view.receiptScanner.updatePantryHint') }}</span>
          </div>
        </label>
        <label class="switch-row">
          <input v-model="autoAddToFinance" type="checkbox" />
          <div class="switch-info">
            <span class="switch-title">💰 {{ t('shopping.view.receiptScanner.addFinance') }}</span>
            <span class="switch-desc">{{ t('shopping.view.receiptScanner.addFinanceHint') }}</span>
          </div>
        </label>
      </div>

      <div class="confirm-bar">
        <button type="button" class="btn btn-secondary" @click="clearAndRestart">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary btn-large" :disabled="parsedReceipt.items.length === 0 || parsedReceipt.items.some(item => !item.name.trim() || !Number.isFinite(item.quantity) || item.quantity <= 0 || !Number.isFinite(item.total_price) || item.total_price < 0) || !Number.isFinite(parsedReceipt.total_amount) || parsedReceipt.total_amount < 0 || loading" @click="handleConfirmImport">
          <PackageCheck :size="20" />
          <span>{{ loading ? t('shopping.view.receiptScanner.importing') : t('shopping.view.receiptScanner.confirmImport') }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.document-type-control { display: flex; align-items: center; gap: 10px; }
.document-type-control label { color: var(--text-muted, #94a3b8); font-size: 0.88rem; }
.switch-disabled { opacity: 0.65; }
.image-scan-controls {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 8px;
}
.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}
.ocr-status, .ocr-privacy-note {
  margin: 0;
  color: var(--text-muted, #94a3b8);
  font-size: 0.82rem;
}
.ocr-status { color: var(--text-main, #f8fafc); }

.scanner-tab-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.scanner-input-card,
.scanner-preview-card {
  padding: var(--space-5, 20px);
  border-radius: var(--radius-xl, 20px);
  background: var(--surface-card, rgba(255, 255, 255, 0.03));
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.card-intro {
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
}

.intro-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.bg-primary-soft {
  background: rgba(59, 130, 246, 0.15);
}
.text-primary {
  color: var(--primary, #3b82f6);
}

.intro-title {
  font-size: 1.15rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
  margin: 0 0 4px 0;
}

.intro-desc {
  font-size: 0.88rem;
  color: var(--text-muted, #94a3b8);
  margin: 0;
  line-height: 1.4;
}

.quick-samples {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.samples-label {
  font-size: 0.8rem;
  color: var(--text-muted, #94a3b8);
}

.sample-btn {
  padding: 4px 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.78rem;
  cursor: pointer;
  transition: all 0.2s;
}
.sample-btn:hover {
  background: rgba(255, 255, 255, 0.12);
}

.textarea-wrap {
  position: relative;
}

.receipt-textarea {
  width: 100%;
  border-radius: var(--radius-md, 12px);
  background: rgba(0, 0, 0, 0.25);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  padding: 14px;
  color: var(--text-main, #f8fafc);
  font-family: monospace;
  font-size: 0.9rem;
  line-height: 1.5;
  resize: vertical;
}
.receipt-textarea:focus {
  outline: none;
  border-color: var(--primary, #3b82f6);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.25);
}

.input-actions {
  display: flex;
  justify-content: flex-end;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
  flex-wrap: wrap;
  gap: 12px;
}

.header-left,
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.store-badge,
.date-badge {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 12px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
}

.store-name-input {
  background: transparent;
  border: none;
  color: var(--text-main, #f8fafc);
  font-size: 1rem;
  font-weight: 600;
  width: 140px;
}

.date-input {
  background: transparent;
  border: none;
  color: var(--text-main, #f8fafc);
  font-size: 0.9rem;
}

.total-badge {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 6px 14px;
  border-radius: 8px;
  background: rgba(16, 185, 129, 0.12);
  border: 1px solid rgba(16, 185, 129, 0.3);
}

.total-label {
  font-size: 0.85rem;
  color: var(--text-muted, #94a3b8);
}

.total-amount {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--success, #10b981);
}

.items-table-wrap {
  overflow-x: auto;
}

.receipt-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}

.receipt-table th {
  padding: 10px 12px;
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--text-muted, #94a3b8);
  border-bottom: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.receipt-table td {
  padding: 8px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
}

.table-input {
  height: 36px;
  padding: 0 10px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.88rem;
  width: 100%;
}

.item-name {
  min-width: 180px;
}
.item-qty {
  width: 80px;
}
.item-price {
  width: 90px;
}

.table-select {
  height: 36px;
  padding: 0 8px;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  color: var(--text-main, #f8fafc);
  font-size: 0.85rem;
  color-scheme: dark;
}

.table-select option {
  background-color: #1e293b !important;
  color: #f8fafc !important;
}

.table-del-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  background: transparent;
  border: none;
  color: var(--danger, #ef4444);
  cursor: pointer;
  border-radius: 6px;
}
.table-del-btn:hover {
  background: rgba(239, 68, 68, 0.15);
}

.integration-switches-box {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.08));
}

.switch-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  cursor: pointer;
}

.switch-row input {
  margin-top: 3px;
  width: 18px;
  height: 18px;
}

.switch-info {
  display: flex;
  flex-direction: column;
}

.switch-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main, #f8fafc);
}

.switch-desc {
  font-size: 0.78rem;
  color: var(--text-muted, #94a3b8);
}

.confirm-bar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 20px;
  border-radius: var(--radius-md, 10px);
  font-size: 0.92rem;
  font-weight: 600;
  cursor: pointer;
  border: none;
}

.btn-large {
  padding: 0 28px;
  font-size: 1rem;
}

.btn-secondary {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-main, #f8fafc);
}
.btn-primary {
  background: var(--primary, #3b82f6);
  color: #fff;
}
.btn-primary:hover {
  background: #2563eb;
}
.btn-ghost {
  background: transparent;
  border: none;
  color: var(--text-muted, #94a3b8);
  cursor: pointer;
  font-size: 0.85rem;
}

@media (max-width: 640px) {
  .scanner-input-card,
  .scanner-preview-card { padding: 12px; border-radius: 14px; }
  .card-intro { align-items: flex-start; gap: 10px; }
  .intro-icon { width: 42px; height: 42px; flex: 0 0 42px; }
  .intro-title { font-size: 1rem; }
  .image-scan-controls,
  .image-scan-controls .btn { width: 100%; }
  .image-scan-controls .btn { justify-content: center; min-height: 44px; text-align: center; }
  .input-actions,
  .input-actions .btn { width: 100%; }
  .input-actions .btn { justify-content: center; min-height: 44px; }
  .preview-header,
  .header-left,
  .header-right { width: 100%; align-items: stretch; flex-direction: column; }
  .store-badge,
  .date-badge,
  .total-badge { width: 100%; min-width: 0; }
  .store-name-input,
  .date-input { min-width: 0; max-width: 100%; }
  .items-table-wrap { max-width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .receipt-table { min-width: 700px; }
  .integration-switches-box { padding: 10px; }
  .confirm-bar { flex-direction: column-reverse; align-items: stretch; }
  .confirm-bar .btn { width: 100%; justify-content: center; min-height: 44px; }
  .modal-footer-actions { flex-direction: column-reverse; }
  .modal-footer-actions > button { width: 100%; min-height: 44px; }
}

@media (max-width: 420px) {
  .scanner-input-card,
  .scanner-preview-card { padding: 10px; }
  .quick-samples { align-items: stretch; }
  .sample-btn { min-height: 40px; }
  .receipt-textarea { font-size: 16px; }
  .preview-header { gap: 8px; }
}

.scan-methods {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.scan-method {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  min-height: 76px;
  text-align: left;
  color: var(--text-main, #f8fafc);
  background: rgba(255, 255, 255, 0.045);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-radius: 12px;
  cursor: pointer;
  transition: border-color 0.18s ease, background 0.18s ease;
}
.scan-method:hover { border-color: var(--primary, #3b82f6); background: rgba(59, 130, 246, 0.08); }
.scan-method:disabled { opacity: 0.6; cursor: wait; }
.scan-method-primary {
  background: rgba(59, 130, 246, 0.11);
  border-color: rgba(59, 130, 246, 0.35);
}
.scan-method > svg { flex: 0 0 auto; color: var(--primary, #3b82f6); }
.scan-method span { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.scan-method strong { font-size: 0.9rem; font-weight: 650; }
.scan-method small { color: var(--text-muted, #94a3b8); font-size: 0.76rem; line-height: 1.35; }
.selected-document-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  background: rgba(255, 255, 255, 0.035);
}
.selected-document-heading,
.selected-document-label {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.selected-document-heading { justify-content: space-between; }
.selected-document-label > svg { color: var(--primary, #3b82f6); flex: 0 0 auto; }
.selected-document-label div { min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.selected-document-label strong { color: var(--text-main, #f8fafc); font-size: 0.9rem; overflow-wrap: anywhere; }
.selected-document-label small { color: var(--text-muted, #94a3b8); font-size: 0.75rem; overflow-wrap: anywhere; }
.icon-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex: 0 0 40px;
  border-radius: 10px;
  color: var(--text-muted, #94a3b8);
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  cursor: pointer;
}
.icon-action:hover { color: var(--text-main, #f8fafc); background: rgba(255, 255, 255, 0.1); }
.receipt-image-preview {
  display: flex;
  align-items: center;
  justify-content: center;
  max-height: 360px;
  min-height: 110px;
  overflow: hidden;
  background: #0b0f16;
  border-radius: 10px;
}
.receipt-image-preview img {
  display: block;
  max-width: 100%;
  max-height: 360px;
  width: auto;
  height: auto;
  object-fit: contain;
}
.selected-document-actions,
.camera-scan-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex-wrap: wrap;
}
.selected-pdf-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
}
.selected-pdf-card .btn { grid-column: 1 / -1; justify-content: center; }
.camera-scan-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  background: rgba(0, 0, 0, 0.22);
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.12));
  border-radius: 14px;
}
.camera-scan-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.camera-scan-header strong { color: var(--text-main, #f8fafc); font-size: 1rem; }
.camera-scan-header p, .camera-scan-tip {
  margin: 4px 0 0;
  color: var(--text-muted, #94a3b8);
  font-size: 0.8rem;
  line-height: 1.45;
}
.camera-feed {
  position: relative;
  width: min(100%, 440px);
  aspect-ratio: 3 / 4;
  margin: 0 auto;
  overflow: hidden;
  border-radius: 12px;
  background: #05070b;
  touch-action: manipulation;
}
.camera-feed video {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.document-frame {
  position: absolute;
  inset: 4%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  padding-bottom: 14px;
  border: 1px dashed rgba(255, 255, 255, 0.7);
  border-radius: 10px;
  pointer-events: none;
}
.document-frame span {
  padding: 5px 9px;
  border-radius: 999px;
  color: white;
  background: rgba(0, 0, 0, 0.6);
  font-size: 0.72rem;
  text-align: center;
}
.corner {
  position: absolute;
  width: 24px;
  height: 24px;
  border-color: #fff;
  border-style: solid;
}
.corner-tl { top: -1px; left: -1px; border-width: 3px 0 0 3px; border-top-left-radius: 8px; }
.corner-tr { top: -1px; right: -1px; border-width: 3px 3px 0 0; border-top-right-radius: 8px; }
.corner-bl { bottom: -1px; left: -1px; border-width: 0 0 3px 3px; border-bottom-left-radius: 8px; }
.corner-br { bottom: -1px; right: -1px; border-width: 0 3px 3px 0; border-bottom-right-radius: 8px; }
.camera-scan-tip { text-align: center; }
.camera-scan-actions .btn { justify-content: center; }
.ocr-error { color: var(--danger, #ef4444); }
.receipt-review-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 14px;
  color: var(--text-muted, #94a3b8);
  font-size: 0.78rem;
}
.confidence-track {
  width: 100%;
  height: 5px;
  overflow: hidden;
  border-radius: 99px;
  background: rgba(255, 255, 255, 0.1);
}
.confidence-track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--primary, #3b82f6);
}
.ocr-raw-details {
  padding: 10px 12px;
  border: 1px solid var(--border-color, rgba(255, 255, 255, 0.1));
  border-radius: 9px;
  color: var(--text-muted, #94a3b8);
  font-size: 0.8rem;
}
.ocr-raw-details summary { cursor: pointer; color: var(--text-main, #f8fafc); }
.ocr-raw-details pre {
  max-height: 220px;
  overflow: auto;
  margin: 10px 0 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font: 0.76rem/1.45 ui-monospace, SFMono-Regular, Menlo, monospace;
}
.total-input {
  width: 112px;
  min-width: 0;
  padding: 0;
  border: 0;
  outline: 0;
  background: transparent;
  font: inherit;
  color: inherit;
}
.add-manual-item { align-self: flex-start; min-height: 40px; }
@media (max-width: 640px) {
  .scan-methods { grid-template-columns: 1fr; }
  .scan-method { min-height: 66px; padding: 12px; }
  .selected-document-actions { display: grid; grid-template-columns: 1fr; }
  .selected-document-actions .btn { justify-content: center; min-height: 44px; }
  .camera-scan-panel {
    padding: 10px;
    padding-bottom: calc(10px + env(safe-area-inset-bottom));
  }
  .camera-scan-actions { display: grid; grid-template-columns: 1fr 1fr; }
  .camera-scan-actions .btn { min-width: 0; padding: 0 10px; min-height: 44px; }
  .receipt-image-preview { max-height: 280px; }
  .receipt-image-preview img { max-height: 280px; }
  .add-manual-item { width: 100%; justify-content: center; }
}

</style>
