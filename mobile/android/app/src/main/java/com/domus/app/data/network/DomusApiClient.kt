package com.domus.app.data.network

import com.domus.app.domain.model.Area
import com.domus.app.domain.model.AuthSession
import com.domus.app.domain.model.Entity
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import okhttp3.*
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException

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
}
