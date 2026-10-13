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
        } catch (_: Exception) {
            Result.success(emptyList())
        }
    }

    suspend fun callService(baseUrl: String, token: String?, domain: String, service: String, entityId: String): Result<Unit> = withContext(Dispatchers.IO) {
        try {
            val payload = """{"entity_id":"$entityId"}"""
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
}
