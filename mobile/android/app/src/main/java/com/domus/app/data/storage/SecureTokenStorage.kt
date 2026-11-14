package com.domus.app.data.storage

import android.content.Context
import android.content.SharedPreferences
import androidx.security.crypto.EncryptedSharedPreferences
import androidx.security.crypto.MasterKey

interface TokenStorage {
    fun getToken(): String?
    fun saveToken(token: String?)
    fun getLocalEndpoint(): String?
    fun saveLocalEndpoint(endpoint: String?)
    fun getRemoteEndpoint(): String?
    fun saveRemoteEndpoint(endpoint: String?)
    fun clear()
}

class InMemoryTokenStorage(
    private var token: String? = null,
    private var localEndpoint: String? = "http://192.168.1.100:8080",
    private var remoteEndpoint: String? = "https://domus.local.myhome.net"
) : TokenStorage {
    override fun getToken(): String? = token
    override fun saveToken(token: String?) { this.token = token }
    override fun getLocalEndpoint(): String? = localEndpoint
    override fun saveLocalEndpoint(endpoint: String?) { this.localEndpoint = endpoint }
    override fun getRemoteEndpoint(): String? = remoteEndpoint
    override fun saveRemoteEndpoint(endpoint: String?) { this.remoteEndpoint = endpoint }
    override fun clear() { token = null }
}

class SharedPreferencesTokenStorage(
    private val context: Context? = null,
    private val fallbackStorage: InMemoryTokenStorage = InMemoryTokenStorage()
) : TokenStorage {
    private val prefs: SharedPreferences? by lazy {
        context?.let { appContext ->
            appContext.getSharedPreferences("domus_secure_prefs", Context.MODE_PRIVATE)
                .edit().remove("jwt_token").apply()
            runCatching {
                val masterKey = MasterKey.Builder(appContext)
                    .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                    .build()
                EncryptedSharedPreferences.create(
                    appContext, "domus_secure_prefs_v2", masterKey,
                    EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                    EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
                )
            }.getOrNull()
        }
    }
    override fun getToken(): String? = prefs?.getString("jwt_token", null) ?: fallbackStorage.getToken()
    override fun saveToken(token: String?) {
        fallbackStorage.saveToken(token)
        prefs?.edit()?.apply {
            if (token != null) putString("jwt_token", token) else remove("jwt_token")
            commit()
        }
    }
    override fun getLocalEndpoint(): String? = prefs?.getString("local_endpoint", null) ?: fallbackStorage.getLocalEndpoint()
    override fun saveLocalEndpoint(endpoint: String?) {
        fallbackStorage.saveLocalEndpoint(endpoint)
        prefs?.edit()?.apply { if (endpoint != null) putString("local_endpoint", endpoint) else remove("local_endpoint"); apply() }
    }
    override fun getRemoteEndpoint(): String? = prefs?.getString("remote_endpoint", null) ?: fallbackStorage.getRemoteEndpoint()
    override fun saveRemoteEndpoint(endpoint: String?) {
        fallbackStorage.saveRemoteEndpoint(endpoint)
        prefs?.edit()?.apply { if (endpoint != null) putString("remote_endpoint", endpoint) else remove("remote_endpoint"); apply() }
    }
    override fun clear() {
        fallbackStorage.clear()
        prefs?.edit()?.remove("jwt_token")?.commit()
    }
}
