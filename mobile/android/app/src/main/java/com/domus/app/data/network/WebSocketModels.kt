package com.domus.app.data.network

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement
import kotlinx.serialization.json.JsonObject

@Serializable
data class DomusWsMessage(
    val type: String,
    val id: Long? = null,
    @SerialName("access_token") val accessToken: String? = null,
    @SerialName("event_type") val eventType: String? = null,
    val success: Boolean? = null,
    val event: JsonObject? = null,
    val result: JsonElement? = null,
    val error: WsError? = null
)

@Serializable
data class WsError(
    val code: String,
    val message: String
)

@Serializable
data class EntityStateChangedEvent(
    @SerialName("entity_id") val entityId: String,
    @SerialName("old_state") val oldState: EntitySnapshot? = null,
    @SerialName("new_state") val newState: EntitySnapshot
)

@Serializable
data class EntitySnapshot(
    @SerialName("entity_id") val entityId: String,
    val state: String,
    val attributes: Map<String, JsonElement> = emptyMap(),
    @SerialName("last_changed") val lastChanged: String = "",
    @SerialName("last_updated") val lastUpdated: String = ""
)
