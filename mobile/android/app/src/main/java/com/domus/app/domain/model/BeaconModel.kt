package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
data class DomusRoomBeacon(
    val id: String,
    val roomId: String,
    val roomName: String,
    val proximityUUID: String = "E2C56DB5-DFFB-48D2-B060-D0F5A71096E0",
    val major: Int,
    val minor: Int,
    val rssiThreshold: Int = -68
)
