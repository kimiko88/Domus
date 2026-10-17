package com.domus.app.services.nfc

import android.content.Context
import android.net.Uri
import android.nfc.NfcAdapter
import com.domus.app.domain.model.DomusNfcTag
import com.domus.app.domain.model.NfcSessionMode
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import java.util.UUID

/**
 * Gestore Scansione e Scrittura Tag NFC nativi per Android.
 * Supporta la lettura di tag NDEF con URI domus://, la simulazione in ambiente di test,
 * e il dispatch delle azioni su scene, entità o allarmi.
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

    private val _lastReadTag = MutableStateFlow<DomusNfcTag?>(null)
    val lastReadTag: StateFlow<DomusNfcTag?> = _lastReadTag.asStateFlow()

    private val _lastStatusMessage = MutableStateFlow("")
    val lastStatusMessage: StateFlow<String> = _lastStatusMessage.asStateFlow()

    var onTagScanned: ((DomusNfcTag) -> Unit)? = null

    init {
        try {
            val nfcAdapter = context?.let { NfcAdapter.getDefaultAdapter(it) } ?: NfcAdapter.getDefaultAdapter(null)
            _isNfcAvailable.value = nfcAdapter != null
        } catch (e: Exception) {
            _isNfcAvailable.value = false
        }
    }

    fun startReading() {
        if (!_isNfcAvailable.value) {
            _lastStatusMessage.value = "NFC non disponibile o disattivato su questo dispositivo"
            return
        }
        _isSessionActive.value = true
        _lastStatusMessage.value = "Avvicina il retro dello smartphone al Tag NFC Domus..."
    }

    fun stopReading() {
        _isSessionActive.value = false
    }

    fun writeTag(uri: String) {
        if (!_isNfcAvailable.value) {
            _lastStatusMessage.value = "NFC non supportato su questo dispositivo"
            return
        }
        _isSessionActive.value = true
        _lastStatusMessage.value = "Avvicina il Tag NFC vuoto da programmare..."
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
            val uri = Uri.parse(uriString)
            val scheme = uri.scheme ?: "domus"
            val host = uri.host ?: "scene"
            val path = uri.path?.trimStart('/') ?: "generic"

            val actionType = when (host) {
                "scene" -> "scene"
                "toggle" -> "toggle"
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
