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
import java.io.IOException

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
    val password: String
)

@Serializable
data class LoginResponse(
    val token: String,
    val user: UserDto? = null
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

    suspend fun login(baseUrl: String, username: String, password: String): Result<AuthSession> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(LoginRequest.serializer(), LoginRequest(username, password))
            val request = Request.Builder()
                .url("$baseUrl/api/v1/auth/login")
                .post(reqBody.toRequestBody(jsonMediaType))
                .build()

            client.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.failure(IOException("Credenziali errate o server non raggiungibile: HTTP ${response.code}"))
                }
                val bodyStr = response.body?.string() ?: ""
                val resp = json.decodeFromString<LoginResponse>(bodyStr)
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

    suspend fun fetchEntities(baseUrl: String, token: String?): Result<List<Entity>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/entities")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/areas")
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext Result.success(emptyList())
                }
                val bodyStr = response.body?.string() ?: "[]"
                val areas = json.decodeFromString<List<Area>>(bodyStr)
                Result.success(areas)
            }
        } catch (e: Exception) {
            Result.success(emptyList())
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
            val payload = if (data.isEmpty()) {
                """{"entity_id":"$entityId"}"""
            } else {
                val dataPairs = data.entries.joinToString(",") { (k, v) ->
                    when (v) {
                        is Number -> "\"$k\":$v"
                        is Boolean -> "\"$k\":$v"
                        else -> "\"$k\":\"$v\""
                    }
                }
                """{"entity_id":"$entityId","data":{$dataPairs}}"""
            }
            val reqBuilder = Request.Builder()
                .url("$baseUrl/api/v1/services/$domain/$service")
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

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
                .url("$baseUrl/api/v1/assistant/chat")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/chores")
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
                .url("$baseUrl/api/v1/chores")
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
                .url("$baseUrl/api/v1/chores/$choreId/complete")
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
                .url("$baseUrl/api/v1/chores/$choreId")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/shopping")
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
                .url("$baseUrl/api/v1/shopping")
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
                .url("$baseUrl/api/v1/shopping/$itemId/toggle")
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
                .url("$baseUrl/api/v1/shopping/$itemId")
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
                .url("$baseUrl/api/v1/shopping/clear-completed")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/energy/flow")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/notifications")
            if (!token.isNullOrBlank()) {
                reqBuilder.addHeader("Authorization", "Bearer $token")
            }

            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) {
                    // Fallback to /api/v1/notifications/actionable
                    val fallbackReq = Request.Builder().url("$baseUrl/api/v1/notifications/actionable")
                    if (!token.isNullOrBlank()) {
                        fallbackReq.addHeader("Authorization", "Bearer $token")
                    }
                    client.newCall(fallbackReq.build()).execute().use { fbResponse ->
                        if (!fbResponse.isSuccessful) {
                            return@withContext Result.success(emptyList())
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
            Result.success(emptyList())
        }
    }

    suspend fun fetchAutomations(baseUrl: String, token: String?): Result<List<AutomationRule>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/automations")
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
                .url("$baseUrl/api/v1/automations")
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
                .url("$baseUrl/api/v1/automations/$ruleId")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/automations/$ruleId").delete()
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
                .url("$baseUrl/api/v1/automations/$ruleId/trigger")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/logbook?limit=$limit")
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
            val query = if (protocol != null) "?protocol=$protocol" else ""
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/mesh/topology$query")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/system/health")
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                val latency = maxOf(1, (System.currentTimeMillis() - start).toInt())
                if (!response.isSuccessful) {
                    return@withContext Result.success(Pair(SystemHealthDto(), latency))
                }
                val bodyStr = response.body?.string() ?: "{}"
                val health = json.decodeFromString<SystemHealthDto>(bodyStr)
                Result.success(Pair(health, latency))
            }
        } catch (e: Exception) {
            Result.success(Pair(SystemHealthDto(), 12))
        }
    }

    suspend fun fetchFamilyMembers(baseUrl: String, token: String?): Result<List<FamilyMemberDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/family/members")
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.success(emptyList())
                val bodyStr = response.body?.string() ?: "{}"
                val res = try {
                    json.decodeFromString<FamilyMembersResponseDto>(bodyStr).members
                } catch (e: Exception) {
                    json.decodeFromString<List<FamilyMemberDto>>(bodyStr)
                }
                Result.success(res)
            }
        } catch (e: Exception) {
            Result.success(emptyList())
        }
    }

    suspend fun fetchFamilyInvites(baseUrl: String, token: String?): Result<List<FamilyInviteDto>> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/family/invites")
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                if (!response.isSuccessful) return@withContext Result.success(emptyList())
                val bodyStr = response.body?.string() ?: "{}"
                val res = try {
                    json.decodeFromString<FamilyInvitesResponseDto>(bodyStr).invites
                } catch (e: Exception) {
                    json.decodeFromString<List<FamilyInviteDto>>(bodyStr)
                }
                Result.success(res)
            }
        } catch (e: Exception) {
            Result.success(emptyList())
        }
    }

    suspend fun createFamilyInvite(baseUrl: String, token: String?, role: String, expiresInHours: Int): Result<FamilyInviteDto> = withContext(Dispatchers.IO) {
        try {
            val reqBody = json.encodeToString(CreateInviteRequestDto.serializer(), CreateInviteRequestDto(role, expiresInHours))
            val reqBuilder = Request.Builder()
                .url("$baseUrl/api/v1/family/invites")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/family/invites/$inviteId").delete()
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
            val payload = """{"scanned_by":"$scannedBy"}"""
            val reqBuilder = Request.Builder()
                .url("$baseUrl/api/v1/tags/$tagId/scan")
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun registerPushToken(baseUrl: String, token: String?, deviceToken: String, deviceName: String = "Android Device"): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = """{"platform":"fcm","device_token":"$deviceToken","device_name":"$deviceName","app_version":"1.0.0"}"""
            val reqBuilder = Request.Builder()
                .url("$baseUrl/api/v1/notifications/push/tokens")
                .post(payload.toRequestBody(jsonMediaType))
            if (!token.isNullOrBlank()) reqBuilder.addHeader("Authorization", "Bearer $token")
            client.newCall(reqBuilder.build()).execute().use { response ->
                Result.success(Unit)
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchFinancialSummary(baseUrl: String, token: String?, period: String = "month"): Result<com.domus.app.domain.model.FinancialSummaryDto> = withContext(Dispatchers.IO) {
        try {
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/finance/summary?period=$period").get()
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/finance/expenses").get()
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
                .url("$baseUrl/api/v1/finance/expenses")
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
                .url("$baseUrl/api/v1/finance/budgets")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/pantry/items").get()
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
                .url("$baseUrl/api/v1/pantry/items")
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
                .url("$baseUrl/api/v1/pantry/items/$id/stock")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/pantry/items/$id").delete()
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/pantry/recipes").get()
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
                .url("$baseUrl/api/v1/pantry/recipes")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/pantry/recipes/$id").delete()
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
                .url("$baseUrl/api/v1/pantry/meals")
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
                .url("$baseUrl/api/v1/pantry/meals/$planId/consume")
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
            val reqBuilder = Request.Builder().url("$baseUrl/api/v1/pantry/meals/$id").delete()
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
                .url("$baseUrl/api/v1/pantry/receipts/parse")
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
                .url("$baseUrl/api/v1/pantry/receipts/import")
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
}

