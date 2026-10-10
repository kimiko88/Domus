package com.domus.app.domain.model

import kotlinx.serialization.Serializable

@Serializable
data class Area(
    val id: String,
    val name: String,
    val icon: String = "home"
)

@Serializable
data class Device(
    val id: String,
    val name: String,
    val integration: String = "",
    val manufacturer: String = "",
    val model: String = "",
    val areaId: String? = null,
    val swVersion: String = ""
)
