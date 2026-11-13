package com.domus.app.data.network

import com.domus.app.domain.model.Area
import com.domus.app.domain.model.AuthSession
import com.domus.app.domain.model.Entity
import com.domus.app.domain.model.AutomationRule
import com.domus.app.domain.model.TimelineEvent
import com.domus.app.domain.model.MeshTopologyDto
import com.domus.app.domain.model.SystemHealthDto
import com.domus.app.domain.model.FamilyMemberDto
import com.domus.app.domain.model.FamilyMembersResponseDto
import com.domus.app.domain.model.FamilyInviteDto
import com.domus.app.domain.model.FamilyInvitesResponseDto
import com.domus.app.domain.model.CreateInviteRequestDto
import com.domus.app.domain.model.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import java.io.IOException

class ApiHttpException(val statusCode: Int, message: String) : IOException(message)

@Serializable
data class AutomationsResponseDto(
    val automations: List<AutomationRule> = emptyList(),
    val total: Int? = null
)

@Serializable
data class LogbookResponseDto(
    val entries: List<TimelineEvent> = emptyList(),
    val total: Int? = null
)

@Serializable
data class LoginRequest(
    val username: String,
    val password: String,
    @kotlinx.serialization.SerialName("two_factor_code") val twoFactorCode: String? = null
)

@Serializable
data class LoginResponse(
    val token: String = "",
    val user: UserDto? = null,
    @kotlinx.serialization.SerialName("requires_2fa") val requiresTwoFactor: Boolean = false
)

@Serializable
data class UserDto(
    val id: String = "",
    val username: String = "",
    val role: String = "user"
)

@Serializable
data class AssistantChatRequestDto(val prompt: String)

@Serializable
data class AssistantActionDto(
    val entity_key: String = "",
    val service: String = "",
    val success: Boolean = false,
    val message: String = ""
)

@Serializable
data class AssistantChatResponseDto(
    val user_prompt: String = "",
    val response_text: String = "",
    val actions_executed: List<AssistantActionDto> = emptyList(),
    val engine_used: String = "",
    val execution_time_ms: Long = 0
)

@Serializable
data class ChoreDto(
    val id: String = "",
    val title: String = "",
    val description: String = "",
    val area_id: String = "",
    val assigned_user_id: String = "",
    val priority: String = "medium",
    val due_date: String? = null,
    val recurrence_type: String = "none",
    val recurrence_rule: String = "",
    val status: String = "pending"
)

@Serializable
data class CreateChoreRequestDto(
    val title: String,
    val description: String = "",
    val area_id: String = "",
    val assigned_user_id: String = "",
    val priority: String = "medium"
)

@Serializable
data class ShoppingItemDto(
    val id: String = "",
    val name: String = "",
    val quantity: String = "1",
    val category: String = "Dispensa",
    val completed: Boolean = false,
    val added_by: String = "",
    val completed_by: String = ""
)

@Serializable
data class CreateShoppingItemDto(
    val name: String,
    val quantity: String = "1",
    val category: String = "Dispensa",
    val added_by: String = "App"
)

@Serializable
data class EnergyFlowDto(
    val solar_watts: Double = 0.0,
    val grid_watts: Double = 0.0,
    val battery_watts: Double = 0.0,
    val battery_soc: Double = 0.0,
    val house_consumption_watts: Double = 0.0,
    val solar_to_house_watts: Double = 0.0,
    val solar_to_battery_watts: Double = 0.0,
    val solar_to_grid_watts: Double = 0.0,
    val battery_to_house_watts: Double = 0.0,
    val grid_to_house_watts: Double = 0.0,
    val grid_to_battery_watts: Double = 0.0
)

@Serializable
data class NotificationActionDto(
    val id: String = "",
    val title: String = "",
    val destructive: Boolean = false
)

@Serializable
data class NotificationDto(
    val id: String = "",
    val title: String = "",
    val message: String = "",
    val image_url: String = "",
    val critical: Boolean = false,
    val actions: List<NotificationActionDto> = emptyList(),
    val created_at: String = "",
    val resolved_at: String? = null
)

class DomusApiClient(
    private val client: OkHttpClient = OkHttpClient(),
    private val json: Json = Json { ignoreUnknownKeys = true; isLenient = true }
) {
    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    /**
     * Builds API URLs without allowing caller-controlled identifiers to become
     * additional path segments or query components.
     */
    private fun apiUrl(baseUrl: String, vararg segments: String): HttpUrl {
        val base = baseUrl.toHttpUrlOrNull()
            ?: throw IOException("URL base non valida")
        return base.newBuilder().apply {
            segments.forEach { segment ->
                require(segment.isNotEmpty()) { "Segmento URL vuoto" }
                addPathSegment(segment)
            }
        }.build()
    }

    suspend fun login(baseUrl: String, username: String, password: String, twoFactorCode: String? = null): Result<AuthSession> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(LoginRequest.serializer(), LoginRequest(username, password, twoFactorCode))
            val request = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "auth", "login"))
                .post(reqBody.toRequestBody(jsonMediaType))
                .build()

            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Credenziali errate o server non raggiungibile: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: ""
                val resp = json.decodeFromString<LoginResponse>(bodyStr)
                if (resp.requiresTwoFactor || resp.token.isBlank()) {
                    return@withContext Result.failure(IOException("TWO_FACTOR_REQUIRED: inserire il codice di verifica"))
                }
                Result.success(
                    AuthSession(
                        token = resp.token,
                        username = resp.user?.username ?: username,
                        role = resp.user?.role ?: "user",
                        isAuthenticated = true
                    )
                )
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun validateSession(baseUrl: String, token: String): Result<AuthSession> = withContext(Dispatchers.IO) {
        try {
            val request = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "auth", "me"))
                .header("Authorization", "Bearer $token").get().build()
            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(ApiHttpException(response.code, "Sessione non valida: HTTP ${response.code}"))
                val user = json.decodeFromString<UserDto>(response.body?.string() ?: "{}")
                Result.success(AuthSession(token = token, username = user.username, role = user.role, isAuthenticated = true))
            }
        } catch (e: Exception) { Result.failure(e) }
    }

    suspend fun fetchEntities(baseUrl: String, token: String?): Result<List<Entity>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "entities"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore fetch entità: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "[]"
                val entities = json.decodeFromString<List<Entity>>(bodyStr)
                Result.success(entities)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchAreas(baseUrl: String, token: String?): Result<List<Area>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "areas"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(ApiHttpException(response.code, "Errore fetch aree: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "[]"
                val areas = json.decodeFromString<List<Area>>(bodyStr)
                Result.success(areas)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun callService(
        baseUrl: String,
        token: String?,
        domain: String,
        service: String,
        entityId: String,
        data: Map<String, Any> = emptyMap()
    ): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("entity_id", kotlinx.serialization.json.JsonPrimitive(entityId))
                if (data.isNotEmpty()) {
                    put("data", kotlinx.serialization.json.buildJsonObject {
                        data.forEach { (key, value) ->
                            put(key, when (value) {
                                is Boolean -> kotlinx.serialization.json.JsonPrimitive(value)
                                is Number -> kotlinx.serialization.json.JsonPrimitive(value)
                                else -> kotlinx.serialization.json.JsonPrimitive(value.toString())
                            })
                        }
                    })
                }
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "services", domain, service))
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }
            reqBuilder.addHeader("X-Idempotency-Key", java.util.UUID.randomUUID().toString())

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore esecuzione comando: HTTP ${response.code}"))
                }
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun sendAssistantPrompt(baseUrl: String, token: String?, prompt: String): Result<AssistantChatResponseDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(AssistantChatRequestDto.serializer(), AssistantChatRequestDto(prompt))
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "assistant", "chat"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore assistente: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val chatResp = json.decodeFromString<AssistantChatResponseDto>(bodyStr)
                Result.success(chatResp)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchChores(baseUrl: String, token: String?): Result<List<ChoreDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "chores"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore fetch faccende: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "[]"
                val chores = json.decodeFromString<List<ChoreDto>>(bodyStr)
                Result.success(chores)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createChore(baseUrl: String, token: String?, chore: CreateChoreRequestDto): Result<ChoreDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateChoreRequestDto.serializer(), chore)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "chores"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore creazione faccenda: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<ChoreDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun completeChore(baseUrl: String, token: String?, choreId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "chores", choreId, "complete"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore completamento faccenda: HTTP ${response.code}"))
                }
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteChore(baseUrl: String, token: String?, choreId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "chores", choreId))
                .delete()
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore eliminazione faccenda: HTTP ${response.code}"))
                }
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchShopping(baseUrl: String, token: String?): Result<List<ShoppingItemDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "shopping"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore fetch spesa: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "[]"
                val items = json.decodeFromString<List<ShoppingItemDto>>(bodyStr)
                Result.success(items)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createShopping(baseUrl: String, token: String?, item: CreateShoppingItemDto): Result<ShoppingItemDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateShoppingItemDto.serializer(), item)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "shopping"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore creazione articolo: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<ShoppingItemDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun toggleShopping(baseUrl: String, token: String?, itemId: String): Result<ShoppingItemDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "shopping", itemId, "toggle"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore toggle articolo spesa: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val updated = json.decodeFromString<ShoppingItemDto>(bodyStr)
                Result.success(updated)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteShopping(baseUrl: String, token: String?, itemId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "shopping", itemId))
                .delete()
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore eliminazione articolo spesa: HTTP ${response.code}"))
                }
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun clearCompletedShopping(baseUrl: String, token: String?): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "shopping", "clear-completed"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore pulizia spesa completata: HTTP ${response.code}"))
                }
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchEnergyFlow(baseUrl: String, token: String?): Result<EnergyFlowDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "energy", "flow"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Errore fetch flusso energia: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val flow = json.decodeFromString<EnergyFlowDto>(bodyStr)
                Result.success(flow)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchNotifications(baseUrl: String, token: String?): Result<List<NotificationDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "notifications"))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful && response.code != 404 && response.code != 405) {
                    return@withContext Result.failure(ApiHttpException(response.code, "Errore fetch notifiche: HTTP ${response.code}"))
                }
                if (!response.isSuccessful) {
                    // Older servers may expose only the actionable notifications endpoint.
                    val fallbackReq = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "notifications", "actionable"))
                    if (!token.isNullOrBlank()) {
                        fallbackReq.addHeader("Authorization", "Bearer $token")
                    }
                    client.newCall(fallbackReq.build()).execute().use { fbResponse ->
                        if (!fbResponse.isSuccessful) {
                            return@withContext Result.failure(ApiHttpException(fbResponse.code, "Errore fetch notifiche: HTTP ${fbResponse.code}"))
                        }
                        val bodyStr = fbResponse.body?.string() ?: "[]"
                        val notifs = json.decodeFromString<List<NotificationDto>>(bodyStr)
                        return@withContext Result.success(notifs)
                    }
                }
                val bodyStr = response.body?.string() ?: "[]"
                val notifs = json.decodeFromString<List<NotificationDto>>(bodyStr)
                Result.success(notifs)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchAutomations(baseUrl: String, token: String?): Result<List<AutomationRule>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "automations"))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.success(AutomationRule.sampleRules)
                }
                val bodyStr = response.body?.string() ?: "{}"
                val res = try {
                    json.decodeFromString<AutomationsResponseDto>(bodyStr).automations
                } catch (e: Exception) {
                    try {
                        json.decodeFromString<List<AutomationRule>>(bodyStr)
                    } catch (e2: Exception) {
                        AutomationRule.sampleRules
                    }
                }
                Result.success(if (res.isEmpty()) AutomationRule.sampleRules else res)
            }
        } catch (e: Exception) {
            Result.success(AutomationRule.sampleRules)
        }
    }

    suspend fun createAutomation(baseUrl: String, token: String?, rule: AutomationRule): Result<AutomationRule> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(AutomationRule.serializer(), rule)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "automations"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione regola: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<AutomationRule>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateAutomation(baseUrl: String, token: String?, ruleId: String, rule: AutomationRule): Result<AutomationRule> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(AutomationRule.serializer(), rule)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "automations", ruleId))
                .put(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore aggiornamento regola: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val updated = json.decodeFromString<AutomationRule>(bodyStr)
                Result.success(updated)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteAutomation(baseUrl: String, token: String?, ruleId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "automations", ruleId)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore cancellazione regola: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun triggerAutomation(baseUrl: String, token: String?, ruleId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "automations", ruleId, "trigger"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore test regola: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchLogbook(baseUrl: String, token: String?, limit: Int = 100): Result<List<TimelineEvent>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "logbook").newBuilder().addQueryParameter("limit", limit.toString()).build())
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.success(TimelineEvent.sampleTimeline)
                }
                val bodyStr = response.body?.string() ?: "{}"
                val entries = try {
                    json.decodeFromString<LogbookResponseDto>(bodyStr).entries
                } catch (e: Exception) {
                    try {
                        json.decodeFromString<List<TimelineEvent>>(bodyStr)
                    } catch (e2: Exception) {
                        TimelineEvent.sampleTimeline
                    }
                }
                Result.success(if (entries.isEmpty()) TimelineEvent.sampleTimeline else entries)
            }
        } catch (e: Exception) {
            Result.success(TimelineEvent.sampleTimeline)
        }
    }

    suspend fun fetchMeshTopology(baseUrl: String, token: String?, protocol: String? = null): Result<MeshTopologyDto> = withContext(Dispatchers.IO) {
        try {
            val meshUrl = apiUrl(baseUrl, "api", "v1", "mesh", "topology").newBuilder().apply {
                if (protocol != null) addQueryParameter("protocol", protocol)
            }.build()
            val reqBuilder = Request.Builder().url(meshUrl)
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.success(MeshTopologyDto())
                }
                val bodyStr = response.body?.string() ?: "{}"
                val top = json.decodeFromString<MeshTopologyDto>(bodyStr)
                Result.success(top)
            }
        } catch (e: Exception) {
            Result.success(MeshTopologyDto())
        }
    }

    suspend fun fetchSystemHealth(baseUrl: String, token: String?): Result<Pair<SystemHealthDto, Int>> = withContext(Dispatchers.IO) {
        try {
            val start = System.currentTimeMillis()
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "system", "health"))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                val latency = maxOf(1, (System.currentTimeMillis() - start).toInt())
                if (!response.isSuccessful) {
                    return@withContext Result.failure(ApiHttpException(response.code, "Errore health check: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val health = json.decodeFromString<SystemHealthDto>(bodyStr)
                Result.success(Pair(health, latency))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchFamilyMembers(baseUrl: String, token: String?): Result<List<FamilyMemberDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "family", "members"))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(ApiHttpException(response.code, "Errore fetch membri famiglia: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val res = try {
                    json.decodeFromString<FamilyMembersResponseDto>(bodyStr).members
                } catch (e: Exception) {
                    json.decodeFromString<List<FamilyMemberDto>>(bodyStr)
                }
                Result.success(res)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchFamilyInvites(baseUrl: String, token: String?): Result<List<FamilyInviteDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "family", "invites"))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(ApiHttpException(response.code, "Errore fetch inviti famiglia: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val res = try {
                    json.decodeFromString<FamilyInvitesResponseDto>(bodyStr).invites
                } catch (e: Exception) {
                    json.decodeFromString<List<FamilyInviteDto>>(bodyStr)
                }
                Result.success(res)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createFamilyInvite(baseUrl: String, token: String?, role: String, expiresInHours: Int): Result<FamilyInviteDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateInviteRequestDto.serializer(), CreateInviteRequestDto(role, expiresInHours))
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "family", "invites"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione invito: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val inv = json.decodeFromString<FamilyInviteDto>(bodyStr)
                Result.success(inv)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun revokeFamilyInvite(baseUrl: String, token: String?, inviteId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "family", "invites", inviteId)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore revoca invito: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun scanTag(baseUrl: String, token: String?, tagId: String, scannedBy: String = "Android_Companion"): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("scanned_by", kotlinx.serialization.json.JsonPrimitive(scannedBy))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "tags", tagId, "scan"))
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Tag scan error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun registerPushToken(baseUrl: String, token: String?, deviceToken: String, deviceName: String = "Android Device"): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("platform", kotlinx.serialization.json.JsonPrimitive("fcm"))
                put("device_token", kotlinx.serialization.json.JsonPrimitive(deviceToken))
                put("device_name", kotlinx.serialization.json.JsonPrimitive(deviceName))
                put("app_version", kotlinx.serialization.json.JsonPrimitive("1.0.0"))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "notifications", "push", "tokens"))
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Push token registration error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchFinancialSummary(baseUrl: String, token: String?, period: String = "month"): Result<com.domus.app.domain.model.FinancialSummaryDto> = withContext(Dispatchers.IO) {
        try {
            val summaryUrl = apiUrl(baseUrl, "api", "v1", "finance", "summary").newBuilder().addQueryParameter("period", period).build()
            val reqBuilder = Request.Builder().url(summaryUrl).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore riepilogo spese: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val sum = json.decodeFromString<com.domus.app.domain.model.FinancialSummaryDto>(bodyStr)
                Result.success(sum)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchExpenses(baseUrl: String, token: String?): Result<List<com.domus.app.domain.model.ExpenseDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "finance", "expenses")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore lista spese: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<com.domus.app.domain.model.ExpenseDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createExpense(baseUrl: String, token: String?, req: com.domus.app.domain.model.CreateExpenseRequestDto): Result<com.domus.app.domain.model.ExpenseDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(com.domus.app.domain.model.CreateExpenseRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "finance", "expenses"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione spesa: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<com.domus.app.domain.model.ExpenseDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun setBudget(baseUrl: String, token: String?, req: com.domus.app.domain.model.SetBudgetRequestDto): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(com.domus.app.domain.model.SetBudgetRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "finance", "budgets"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore impostazione budget: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ==========================================
    // PANTRY, RECIPES, MEAL PLAN & RECEIPT OCR
    // ==========================================

    suspend fun fetchPantryItems(baseUrl: String, token: String?): Result<List<PantryItemDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "pantry", "items")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore fetch dispensa: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<PantryItemDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createPantryItem(baseUrl: String, token: String?, req: CreatePantryItemRequestDto): Result<PantryItemDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreatePantryItemRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "items"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione alimento: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<PantryItemDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updatePantryStock(baseUrl: String, token: String?, id: String, delta: Double? = null, quantity: Double? = null): Result<PantryItemDto> = withContext(Dispatchers.IO) {
        try {
            val req = UpdatePantryStockRequestDto(delta = delta, quantity = quantity)
            val reqBody = json.encodeToString(UpdatePantryStockRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "items", id, "stock"))
                .patch(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore aggiornamento scorte: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val updated = json.decodeFromString<PantryItemDto>(bodyStr)
                Result.success(updated)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deletePantryItem(baseUrl: String, token: String?, id: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "pantry", "items", id)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchRecipes(baseUrl: String, token: String?): Result<List<RecipeDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "pantry", "recipes")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore fetch ricette: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<RecipeDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createRecipe(baseUrl: String, token: String?, req: CreateRecipeRequestDto): Result<RecipeDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateRecipeRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "recipes"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione ricetta: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<RecipeDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteRecipe(baseUrl: String, token: String?, id: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "pantry", "recipes", id)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchMealPlans(baseUrl: String, token: String?, startDate: String? = null, endDate: String? = null): Result<List<MealPlanDto>> = withContext(Dispatchers.IO) {
        try {
            var url = "$baseUrl/api/v1/pantry/meals"
            val params = mutableListOf<String>()
            if (!startDate.isNullOrBlank()) params.add("start_date=$startDate")
            if (!endDate.isNullOrBlank()) params.add("end_date=$endDate")
            if (params.isNotEmpty()) url += "?" + params.joinToString("&")

            val reqBuilder = Request.Builder().url(url).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore fetch pasti: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<MealPlanDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createMealPlan(baseUrl: String, token: String?, req: CreateMealPlanRequestDto): Result<MealPlanDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateMealPlanRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "meals"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione piano pasto: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val created = json.decodeFromString<MealPlanDto>(bodyStr)
                Result.success(created)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun consumeMealPlan(baseUrl: String, token: String?, planId: String, personsCount: Int? = null): Result<ConsumeMealPlanResponseDto> = withContext(Dispatchers.IO) {
        try {
            val req = ConsumeMealPlanRequestDto(persons_count = personsCount)
            val reqBody = json.encodeToString(ConsumeMealPlanRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "meals", planId, "consume"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore consumo pasto: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val res = json.decodeFromString<ConsumeMealPlanResponseDto>(bodyStr)
                Result.success(res)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteMealPlan(baseUrl: String, token: String?, id: String): Result<Boolean> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "pantry", "meals", id)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                Result.success(response.isSuccessful)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun parseReceipt(baseUrl: String, token: String?, rawText: String): Result<ReceiptDto> = withContext(Dispatchers.IO) {
        try {
            val req = ReceiptParseRequestDto(raw_text = rawText)
            val reqBody = json.encodeToString(ReceiptParseRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "receipts", "parse"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore analisi scontrino: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val parsed = json.decodeFromString<ReceiptDto>(bodyStr)
                Result.success(parsed)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun importReceipt(baseUrl: String, token: String?, req: ReceiptImportRequestDto): Result<ReceiptImportResultDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(ReceiptImportRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "pantry", "receipts", "import"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore importazione scontrino: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val result = json.decodeFromString<ReceiptImportResultDto>(bodyStr)
                Result.success(result)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ==========================================
    // MEDICINE CABINET & EXPIRATION APIS
    // ==========================================

    suspend fun getMedicines(
        baseUrl: String,
        token: String?,
        location: String? = null,
        status: String? = null,
        query: String? = null
    ): Result<List<MedicineDto>> = withContext(Dispatchers.IO) {
        try {
            val urlBuilder = "$baseUrl/api/v1/medicines".toHttpUrlOrNull()?.newBuilder()
                ?: return@withContext Result.failure(IOException("URL non valido"))
            if (!location.isNullOrBlank()) urlBuilder.addQueryParameter("location", location)
            if (!status.isNullOrBlank()) urlBuilder.addQueryParameter("status", status)
            if (!query.isNullOrBlank()) urlBuilder.addQueryParameter("q", query)

            val reqBuilder = Request.Builder().url(urlBuilder.build()).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<MedicineDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getMedicine(baseUrl: String, token: String?, id: String): Result<MedicineDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "medicines", id)).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val med = json.decodeFromString<MedicineDto>(bodyStr)
                Result.success(med)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createMedicine(baseUrl: String, token: String?, req: CreateMedicineRequestDto): Result<MedicineDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateMedicineRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "medicines"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione farmaco: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val med = json.decodeFromString<MedicineDto>(bodyStr)
                Result.success(med)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateMedicine(baseUrl: String, token: String?, id: String, req: UpdateMedicineRequestDto): Result<MedicineDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(UpdateMedicineRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "medicines", id))
                .put(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore modifica farmaco: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val med = json.decodeFromString<MedicineDto>(bodyStr)
                Result.success(med)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deleteMedicine(baseUrl: String, token: String?, id: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "medicines", id)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore eliminazione farmaco: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun openMedicine(baseUrl: String, token: String?, id: String, openedAt: String? = null): Result<MedicineDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(OpenMedicineRequestDto.serializer(), OpenMedicineRequestDto(opened_at = openedAt))
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "medicines", id, "open"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore apertura confezione: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val med = json.decodeFromString<MedicineDto>(bodyStr)
                Result.success(med)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun consumeMedicine(baseUrl: String, token: String?, id: String, amount: Double = 1.0): Result<MedicineDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(ConsumeMedicineRequestDto.serializer(), ConsumeMedicineRequestDto(amount = amount))
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "medicines", id, "consume"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore assunzione dose: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val med = json.decodeFromString<MedicineDto>(bodyStr)
                Result.success(med)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getMedicineAlerts(baseUrl: String, token: String?, fridgeTemp: Double? = null): Result<List<MedicineAlertDto>> = withContext(Dispatchers.IO) {
        try {
            var url = "$baseUrl/api/v1/medicines/alerts"
            if (fridgeTemp != null) {
                url += "?fridge_temp=$fridgeTemp"
            }
            val reqBuilder = Request.Builder().url(url).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore avvisi medicinali: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val alerts = json.decodeFromString<List<MedicineAlertDto>>(bodyStr)
                Result.success(alerts)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun parseMedicineCode(baseUrl: String, token: String?, code: String): Result<DatamatrixParseResultDto> = withContext(Dispatchers.IO) {
        try {
            @Serializable data class CodePayload(val code: String)
            val reqBody = json.encodeToString(CodePayload.serializer(), CodePayload(code = code))
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "medicines", "parse-code"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore parsing codice: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val result = json.decodeFromString<DatamatrixParseResultDto>(bodyStr)
                Result.success(result)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun lookupAIFA(baseUrl: String, token: String?, query: String? = null, aic: String? = null): Result<List<AIFARecordDto>> = withContext(Dispatchers.IO) {
        try {
            val urlBuilder = "$baseUrl/api/v1/medicines/lookup".toHttpUrlOrNull()?.newBuilder()
                ?: return@withContext Result.failure(IOException("URL non valido"))
            if (!query.isNullOrBlank()) urlBuilder.addQueryParameter("q", query)
            if (!aic.isNullOrBlank()) urlBuilder.addQueryParameter("aic", aic)

            val reqBuilder = Request.Builder().url(urlBuilder.build()).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore lookup AIFA: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val list = json.decodeFromString<List<AIFARecordDto>>(bodyStr)
                Result.success(list)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ==========================================
    // PREDICTIVE CLIMATE APIS
    // ==========================================

    suspend fun getPredictiveStatus(baseUrl: String, token: String?): Result<SystemPredictiveStatusDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "status")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore stato predittivo: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val status = json.decodeFromString<SystemPredictiveStatusDto>(bodyStr)
                Result.success(status)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getPredictiveZones(baseUrl: String, token: String?): Result<List<PredictiveZoneConfigDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore elenco zone: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val resp = json.decodeFromString<PredictiveZonesResponseDto>(bodyStr)
                Result.success(resp.zones)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getPredictiveZone(baseUrl: String, token: String?, id: String): Result<PredictiveZoneConfigDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones", id)).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore zona predittiva: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val zone = json.decodeFromString<PredictiveZoneConfigDto>(bodyStr)
                Result.success(zone)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createPredictiveZone(baseUrl: String, token: String?, req: CreatePredictiveZoneRequestDto): Result<PredictiveZoneConfigDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreatePredictiveZoneRequestDto.serializer(), req)
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones"))
                .post(reqBody.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore creazione zona: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val zone = json.decodeFromString<PredictiveZoneConfigDto>(bodyStr)
                Result.success(zone)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun deletePredictiveZone(baseUrl: String, token: String?, id: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones", id)).delete()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore eliminazione zona: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun evaluatePredictiveZone(baseUrl: String, token: String?, id: String): Result<PredictiveEvaluationDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones", id, "evaluate"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore valutazione zona: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val eval = json.decodeFromString<PredictiveEvaluationDto>(bodyStr)
                Result.success(eval)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun evaluateAllPredictiveZones(baseUrl: String, token: String?): Result<List<PredictiveEvaluationDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "evaluate-all"))
                .post("{}".toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore valutazione globale: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "[]"
                val evals = json.decodeFromString<List<PredictiveEvaluationDto>>(bodyStr)
                Result.success(evals)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getZoneThermalProfile(baseUrl: String, token: String?, id: String): Result<ThermalProfileDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "climate", "predictive", "zones", id, "profile")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore profilo termico: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val profile = json.decodeFromString<ThermalProfileDto>(bodyStr)
                Result.success(profile)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getOptimalStart(
        baseUrl: String,
        token: String?,
        id: String,
        targetTemp: Double? = null,
        targetTime: String? = null
    ): Result<PreheatRecommendationDto> = withContext(Dispatchers.IO) {
        try {
            val urlBuilder = "$baseUrl/api/v1/climate/predictive/zones/$id/optimal-start".toHttpUrlOrNull()?.newBuilder()
                ?: return@withContext Result.failure(IOException("URL non valido"))
            if (targetTemp != null) urlBuilder.addQueryParameter("target_temp", targetTemp.toString())
            if (!targetTime.isNullOrBlank()) urlBuilder.addQueryParameter("target_time", targetTime)

            val reqBuilder = Request.Builder().url(urlBuilder.build()).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore avvio ottimale: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val rec = json.decodeFromString<PreheatRecommendationDto>(bodyStr)
                Result.success(rec)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getOptimalStop(
        baseUrl: String,
        token: String?,
        id: String,
        departureTime: String? = null
    ): Result<CoastDownRecommendationDto> = withContext(Dispatchers.IO) {
        try {
            val urlBuilder = "$baseUrl/api/v1/climate/predictive/zones/$id/optimal-stop".toHttpUrlOrNull()?.newBuilder()
                ?: return@withContext Result.failure(IOException("URL non valido"))
            if (!departureTime.isNullOrBlank()) urlBuilder.addQueryParameter("departure_time", departureTime)

            val reqBuilder = Request.Builder().url(urlBuilder.build()).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Errore spegnimento anticipato: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                val rec = json.decodeFromString<CoastDownRecommendationDto>(bodyStr)
                Result.success(rec)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ──────────────────────────────────────────────────────────────────
    // EV Wallbox OCPP — Ricarica Solare Intelligente
    // ──────────────────────────────────────────────────────────────────

    /** Fetches live wallbox telemetry from the OCPP backend. */
    suspend fun getWallboxStatus(baseUrl: String, token: String?): Result<WallboxStatusDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "energy", "wallbox")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Wallbox status error: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                Result.success(json.decodeFromString<WallboxStatusDto>(bodyStr))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /** Sets the wallbox charging mode (SOLAR_ONLY, SOLAR_OPTIMIZED, FAST, NIGHT). */
    suspend fun setWallboxMode(baseUrl: String, token: String?, mode: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("mode", kotlinx.serialization.json.JsonPrimitive(mode))
            }.toString()
            val body = payload.toRequestBody("application/json".toMediaType())
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "energy", "wallbox", "mode")).post(body)
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Wallbox mode error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ──────────────────────────────────────────────────────────────────
    // Commute Tracker — ETA-based Automations
    // ──────────────────────────────────────────────────────────────────

    /**
     * Posts the user's current GPS coordinates to the Domus CommuteTracker endpoint.
     * The backend calculates ETA and triggers preheat/lighting automations accordingly.
     */
    suspend fun postCommuteLocation(
        baseUrl: String,
        token: String?,
        userId: String,
        latitude: Double,
        longitude: Double
    ): Result<CommuteStatusDto> = withContext(Dispatchers.IO) {
        try {
            if (!latitude.isFinite() || latitude !in -90.0..90.0 || !longitude.isFinite() || longitude !in -180.0..180.0) {
                return@withContext Result.failure(IllegalArgumentException("Coordinate fuori intervallo"))
            }
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("user_id", kotlinx.serialization.json.JsonPrimitive(userId))
                put("latitude", kotlinx.serialization.json.JsonPrimitive(latitude))
                put("longitude", kotlinx.serialization.json.JsonPrimitive(longitude))
            }.toString()
            val body = payload.toRequestBody("application/json".toMediaType())
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "presence", "commute")).post(body)
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Commute update error: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                Result.success(json.decodeFromString<CommuteStatusDto>(bodyStr))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    /** Gets the current commute status (ETA, distance, phase) for the given user. */
    suspend fun getCommuteStatus(baseUrl: String, token: String?, userId: String): Result<CommuteStatusDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "presence", "commute", userId)).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Commute status error: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                Result.success(json.decodeFromString<CommuteStatusDto>(bodyStr))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // Vacation Mode (Smart Presence Replay Simulator)
    // ──────────────────────────────────────────────────────────────────────────

    suspend fun fetchVacationMode(baseUrl: String, token: String?): Result<VacationModeResponse> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "presence", "vacation")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Vacation mode error: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                Result.success(json.decodeFromString<VacationModeResponse>(bodyStr))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun setVacationMode(baseUrl: String, token: String?, enabled: Boolean): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("enabled", kotlinx.serialization.json.JsonPrimitive(enabled))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "presence", "vacation"))
                .post(payload.toRequestBody(jsonMediaType))
                .addHeader("X-Idempotency-Key", java.util.UUID.randomUUID().toString())
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Set vacation mode error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun registerVacationAction(
        baseUrl: String,
        token: String?,
        entityId: String,
        command: String,
        typicalTime: String
    ): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("entity_id", kotlinx.serialization.json.JsonPrimitive(entityId))
                put("command", kotlinx.serialization.json.JsonPrimitive(command))
                put("typical_time", kotlinx.serialization.json.JsonPrimitive(typicalTime))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "presence", "vacation", "actions"))
                .post(payload.toRequestBody(jsonMediaType))
                .addHeader("X-Idempotency-Key", java.util.UUID.randomUUID().toString())
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Register vacation action error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ──────────────────────────────────────────────────────────────────────────
    // IoT Network Guardian & Device Quarantine
    // ──────────────────────────────────────────────────────────────────────────

    suspend fun fetchNetworkLeaks(baseUrl: String, token: String?): Result<NetworkLeaksResponse> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url(apiUrl(baseUrl, "api", "v1", "network", "leaks")).get()
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Network leaks error: HTTP ${response.code}"))
                val bodyStr = response.body?.string() ?: "{}"
                Result.success(json.decodeFromString<NetworkLeaksResponse>(bodyStr))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun quarantineDevice(baseUrl: String, token: String?, mac: String, ip: String, reason: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("mac", kotlinx.serialization.json.JsonPrimitive(mac))
                put("ip", kotlinx.serialization.json.JsonPrimitive(ip))
                put("reason", kotlinx.serialization.json.JsonPrimitive(reason))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "network", "quarantine"))
                .post(payload.toRequestBody(jsonMediaType))
                .addHeader("X-Idempotency-Key", java.util.UUID.randomUUID().toString())
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Quarantine error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun unblockDevice(baseUrl: String, token: String?, mac: String, ip: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = kotlinx.serialization.json.buildJsonObject {
                put("mac", kotlinx.serialization.json.JsonPrimitive(mac))
                put("ip", kotlinx.serialization.json.JsonPrimitive(ip))
            }.toString()
            val reqBuilder = Request.Builder()
                .url(apiUrl(baseUrl, "api", "v1", "network", "unblock"))
                .post(payload.toRequestBody(jsonMediaType))
                .addHeader("X-Idempotency-Key", java.util.UUID.randomUUID().toString())
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.failure(IOException("Unblock error: HTTP ${response.code}"))
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}

// ──────────────────────────────────────────────────────────────────────────────
// DTOs – Wallbox OCPP & Commute Tracker & Vacation & Leaks
// ──────────────────────────────────────────────────────────────────────────────

@Serializable
data class WallboxStatusDto(
    val connected: Boolean = false,
    val charge_box_id: String = "",
    val status: String = "Available", // Available | Preparing | Charging | SuspendedEV | Finishing
    val active_power_w: Double = 0.0,
    val current_amps: Double = 0.0,
    val vehicle_soc_percent: Int? = null,
    val session_energy_kwh: Double = 0.0,
    val charging_mode: String = "SOLAR_OPTIMIZED",
    val solar_surplus_w: Double = 0.0,
    val grid_import_w: Double = 0.0
)

@Serializable
data class CommuteStatusDto(
    val user_id: String = "",
    val phase: String = "idle", // idle | commuting | approaching | arrived
    val distance_km: Double = 0.0,
    val eta_minutes: Int = 0,
    val latitude: Double? = null,
    val longitude: Double? = null,
    val automations_triggered: List<String> = emptyList()
)

@Serializable
data class VacationModeResponse(
    val enabled: Boolean = false,
    val jitter_minutes: Int = 15
)

@Serializable
data class NetworkLeaksResponse(
    val count: Int = 0,
    val leaks: List<NetworkLeakItem> = emptyList(),
    val isolated_macs: List<String> = emptyList()
)

@Serializable
data class NetworkLeakItem(
    val id: String = "",
    val mac: String = "",
    val ip: String = "",
    val destination: String = "",
    val timestamp: String = "",
    val action: String = "quarantined"
)
