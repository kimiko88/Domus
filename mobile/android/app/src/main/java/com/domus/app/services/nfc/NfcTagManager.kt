package com.domus.app.services.nfc

import android.content.Context
import android.net.Uri
import android.nfc.NdefMessage
import android.nfc.NdefRecord
import android.nfc.NfcAdapter
import android.nfc.Tag
import android.nfc.tech.Ndef
import android.nfc.tech.NdefFormatable
import com.domus.app.domain.model.DomusNfcTag
import com.domus.app.domain.model.NfcSessionMode
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.nio.charset.StandardCharsets
import java.util.UUID

enum class NfcManagerMode {
    READ,
    WRITE
}

/**
 * Gestore Scansione e Scrittura Tag NFC nativi per Android.
 * Supporta la lettura e scrittura di tag NDEF NTAG213/215 con URI domus://,
 * simulazione in ambiente di test, e dispatch delle azioni su scene o entità.
 */
class NfcTagManager(
    private val context: Context? = null
) {
    companion object {
        val shared by lazy { NfcTagManager() }
    }

    private val _isNfcAvailable = MutableStateFlow(false)
    val isNfcAvailable: StateFlow<Boolean> = _isNfcAvailable.asStateFlow()

    private val _isSessionActive = MutableStateFlow(false)
    val isSessionActive: StateFlow<Boolean> = _isSessionActive.asStateFlow()

    private val _currentMode = MutableStateFlow(NfcManagerMode.READ)
    val currentMode: StateFlow<NfcManagerMode> = _currentMode.asStateFlow()

    private val _pendingWritePayload = MutableStateFlow<String?>(null)
    val pendingWritePayload: StateFlow<String?> = _pendingWritePayload.asStateFlow()

    private val _lastReadTag = MutableStateFlow<DomusNfcTag?>(null)
    val lastReadTag: StateFlow<DomusNfcTag?> = _lastReadTag.asStateFlow()

    private val _lastStatusMessage = MutableStateFlow("")
    val lastStatusMessage: StateFlow<String> = _lastStatusMessage.asStateFlow()

    var onTagScanned: ((DomusNfcTag) -> Unit)? = null
    var onTagWritten: ((String) -> Unit)? = null

    init {
        try {
            val nfcAdapter = context?.let { NfcAdapter.getDefaultAdapter(it) } ?: NfcAdapter.getDefaultAdapter(null)
            _isNfcAvailable.value = nfcAdapter != null
        } catch (e: Throwable) {
            _isNfcAvailable.value = false
        }
    }

    fun setNfcAvailableForTest(available: Boolean) {
        _isNfcAvailable.value = available
    }

    fun startReading() {
        if (!_isNfcAvailable.value) {
            _lastStatusMessage.value = "NFC non disponibile o disattivato su questo dispositivo"
            return
        }
        _currentMode.value = NfcManagerMode.READ
        _isSessionActive.value = true
        _lastStatusMessage.value = "Avvicina il retro dello smartphone al Tag NFC Domus..."
    }

    fun stopSession() {
        _isSessionActive.value = false
        _pendingWritePayload.value = null
        _currentMode.value = NfcManagerMode.READ
    }

    fun stopReading() = stopSession()

    /**
     * Prepara l'applicazione per scrivere un URI domus:// su un tag NFC fisico (NTAG213/NTAG215).
     */
    fun prepareTagWriting(actionUri: String) {
        if (!_isNfcAvailable.value) {
            _lastStatusMessage.value = "NFC non supportato su questo dispositivo"
            return
        }
        _currentMode.value = NfcManagerMode.WRITE
        _pendingWritePayload.value = actionUri
        _isSessionActive.value = true
        _lastStatusMessage.value = "Avvicina un tag NTAG213/215 vergine per scrivere: $actionUri"
    }

    /**
     * Scrive il record NDEF su un tag NFC hardware rilevato da Android.
     */
    fun writeNdefToTag(tag: Tag, customUri: String? = null): Result<String> {
        val targetUri = customUri ?: _pendingWritePayload.value ?: "domus://action/toggle"

        return try {
            val uriRecord = NdefRecord.createUri(targetUri)
            val message = NdefMessage(arrayOf(uriRecord))
            val messageSize = message.byteArrayLength

            val ndef = Ndef.get(tag)
            if (ndef != null) {
                ndef.connect()
                if (!ndef.isWritable) {
                    ndef.close()
                    return Result.failure(IllegalStateException("Il tag NFC è bloccato in sola lettura"))
                }
                if (ndef.maxSize < messageSize) {
                    ndef.close()
                    return Result.failure(IllegalStateException("Dimensione payload ($messageSize byte) eccede capacità tag (${ndef.maxSize} byte)"))
                }
                ndef.writeNdefMessage(message)
                ndef.close()
            } else {
                val formatable = NdefFormatable.get(tag)
                    ?: return Result.failure(IllegalStateException("Il tag non supporta la formattazione NDEF"))
                formatable.connect()
                formatable.format(message)
                formatable.close()
            }

            _lastStatusMessage.value = "Tag programmato con successo: $targetUri"
            _isSessionActive.value = false
            _currentMode.value = NfcManagerMode.READ
            _pendingWritePayload.value = null
            onTagWritten?.invoke(targetUri)
            Result.success(targetUri)
        } catch (e: Exception) {
            _lastStatusMessage.value = "Errore durante la scrittura del Tag: ${e.message}"
            Result.failure(e)
        }
    }

    fun writeTag(uri: String) {
        prepareTagWriting(uri)
    }

    /**
     * Simula la scrittura del tag in ambiente di collaudo o anteprima.
     */
    fun simulateTagWrite(actionUri: String = "domus://action/cover.gate"): String {
        _lastStatusMessage.value = "Tag programmato con successo (simulato): $actionUri"
        _isSessionActive.value = false
        _currentMode.value = NfcManagerMode.READ
        _pendingWritePayload.value = null
        onTagWritten?.invoke(actionUri)
        return actionUri
    }

    fun simulateTagScan(uri: String = "domus://scene/relax_evening") {
        val tag = parseDomusUri(uri)
        _lastReadTag.value = tag
        _lastStatusMessage.value = "Tag letto con successo: ${tag.actionType} → ${tag.targetId}"
        _isSessionActive.value = false
        onTagScanned?.invoke(tag)
    }

    fun parseDomusUri(uriString: String): DomusNfcTag {
        try {
            val clean = uriString.trim()
            val scheme = if (clean.contains("://")) clean.substringBefore("://") else "domus"
            val remainder = if (clean.contains("://")) clean.substringAfter("://") else clean
            val host = if (remainder.contains("/")) remainder.substringBefore("/") else remainder
            val path = if (remainder.contains("/")) remainder.substringAfter("/").trimStart('/') else "generic"

            val actionType = when (host) {
                "scene" -> "scene"
                "toggle", "action" -> "toggle"
                "alarm" -> "alarm"
                else -> host
            }

            return DomusNfcTag(
                id = UUID.randomUUID().toString(),
                uri = uriString,
                actionType = actionType,
                targetId = path
            )
        } catch (e: Exception) {
            return DomusNfcTag(
                id = UUID.randomUUID().toString(),
                uri = uriString,
                actionType = "custom",
                targetId = uriString
            )
        }
    }
}

