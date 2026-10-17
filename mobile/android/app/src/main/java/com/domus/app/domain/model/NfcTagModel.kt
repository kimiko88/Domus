package com.domus.app.domain.model

import kotlinx.serialization.Serializable
import java.util.UUID

@Serializable
data class DomusNfcTag(
    val id: String = UUID.randomUUID().toString(),
    val uri: String,
    val actionType: String, // "scene", "toggle", "alarm", "custom"
    val targetId: String,
    val createdAtMillis: Long = System.currentTimeMillis()
)

enum class NfcSessionMode {
    READ,
    WRITE
}
