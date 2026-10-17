package com.domus.app.domain.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import java.util.UUID

enum class UserAccessRole(val displayName: String, val description: String, val icon: String) {
    ADMIN(
        "Amministratore",
        "Accesso completo a tutti i dispositivi, allarmi, impostazioni e backup.",
        "🔑"
    ),
    FAMILY(
        "Membro Famiglia",
        "Controllo di tutta la casa e allarme. Nessun accesso a impostazioni di sistema.",
        "👥"
    ),
    CHILD(
        "Bambino",
        "Controllo limitato alla camera da letto e salotto. Allarme e serrature bloccati.",
        "👶"
    ),
    GUEST(
        "Ospite Temporaneo",
        "Accesso limitato alle stanze selezionate, con scadenza temporale automatica.",
        "🎟️"
    )
}

data class DomusUserAccount(
    val id: String = UUID.randomUUID().toString(),
    var name: String,
    var role: UserAccessRole,
    var allowedRooms: List<String> = emptyList(),
    var canDisarmAlarm: Boolean = false,
    var canViewCameras: Boolean = false,
    var expiresAtMillis: Long? = null
) {
    companion object {
        val sampleAccounts = listOf(
            DomusUserAccount(
                id = "user.fabrizio",
                name = "Fabrizio (Tu)",
                role = UserAccessRole.ADMIN,
                allowedRooms = listOf("Tutte le stanze"),
                canDisarmAlarm = true,
                canViewCameras = true
            ),
            DomusUserAccount(
                id = "user.elena",
                name = "Elena",
                role = UserAccessRole.FAMILY,
                allowedRooms = listOf("Tutte le stanze"),
                canDisarmAlarm = true,
                canViewCameras = true
            ),
            DomusUserAccount(
                id = "user.leo",
                name = "Leo (Bambino)",
                role = UserAccessRole.CHILD,
                allowedRooms = listOf("Cameretta", "Salotto"),
                canDisarmAlarm = false,
                canViewCameras = false
            ),
            DomusUserAccount(
                id = "user.guest_bnb",
                name = "Ospite Weekend",
                role = UserAccessRole.GUEST,
                allowedRooms = listOf("Camera Ospiti", "Bagno Ospiti", "Salotto"),
                canDisarmAlarm = false,
                canViewCameras = false,
                expiresAtMillis = System.currentTimeMillis() + 24 * 60 * 60 * 1000
            )
        )
    }
}

@Serializable
data class FamilyMemberDto(
    val id: String,
    val username: String,
    val email: String? = null,
    val role: String? = "family",
    val enabled: Boolean? = true,
    @SerialName("created_at") val createdAt: String? = null
)

@Serializable
data class FamilyMembersResponseDto(
    val members: List<FamilyMemberDto> = emptyList(),
    val total: Int? = null
)

@Serializable
data class FamilyInviteDto(
    val id: String,
    val token: String,
    val role: String,
    @SerialName("expires_at") val expiresAt: String? = null,
    @SerialName("invite_url") val inviteUrl: String? = null
)

@Serializable
data class FamilyInvitesResponseDto(
    val invites: List<FamilyInviteDto> = emptyList(),
    val total: Int? = null
)

@Serializable
data class CreateInviteRequestDto(
    val role: String,
    @SerialName("expires_in_hours") val expiresInHours: Int
)
