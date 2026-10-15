package com.domus.app.data.mdns

import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlin.coroutines.resume

interface MdnsScanner {
    suspend fun discoverService(serviceType: String, timeoutMs: Long): DiscoveredHub?
}

data class DiscoveredHub(
    val name: String,
    val host: String,
    val port: Int
) {
    val httpBaseUrl: String get() = "http://$host:$port"
    val wsBaseUrl: String get() = "ws://$host:$port/api/v1/ws"
}

class MdnsDiscoveryEngine(
    private val nsdManager: NsdManager? = null,
    private val customScanner: MdnsScanner? = null
) {
    companion object {
        const val SERVICE_TYPE = "_domus._tcp."
    }

    suspend fun discoverLocalHub(timeoutMs: Long = 1500L): DiscoveredHub? {
        if (customScanner != null) {
            return customScanner.discoverService(SERVICE_TYPE, timeoutMs)
        }

        if (nsdManager == null) {
            return null
        }

        return withTimeoutOrNull(timeoutMs) {
            suspendCancellableCoroutine { continuation ->
                var isResolved = false

                val discoveryListener = object : NsdManager.DiscoveryListener {
                    override fun onStartDiscoveryFailed(serviceType: String?, errorCode: Int) {
                        if (!isResolved && continuation.isActive) {
                            isResolved = true
                            continuation.resume(null)
                        }
                    }

                    override fun onStopDiscoveryFailed(serviceType: String?, errorCode: Int) {}

                    override fun onDiscoveryStarted(serviceType: String?) {}

                    override fun onDiscoveryStopped(serviceType: String?) {}

                    override fun onServiceFound(serviceInfo: NsdServiceInfo?) {
                        serviceInfo?.let { info ->
                            nsdManager.resolveService(info, object : NsdManager.ResolveListener {
                                override fun onResolveFailed(serviceInfo: NsdServiceInfo?, errorCode: Int) {}

                                override fun onServiceResolved(resolvedInfo: NsdServiceInfo?) {
                                    if (!isResolved && continuation.isActive && resolvedInfo != null) {
                                        isResolved = true
                                        val host = resolvedInfo.host?.hostAddress ?: "127.0.0.1"
                                        val port = resolvedInfo.port
                                        val hub = DiscoveredHub(
                                            name = resolvedInfo.serviceName ?: "Domus Hub",
                                            host = host,
                                            port = port
                                        )
                                        continuation.resume(hub)
                                    }
                                }
                            })
                        }
                    }

                    override fun onServiceLost(serviceInfo: NsdServiceInfo?) {}
                }

                try {
                    nsdManager.discoverServices(SERVICE_TYPE, NsdManager.PROTOCOL_DNS_SD, discoveryListener)
                } catch (e: Exception) {
                    if (continuation.isActive) continuation.resume(null)
                }

                continuation.invokeOnCancellation {
                    try {
                        nsdManager.stopServiceDiscovery(discoveryListener)
                    } catch (e: Exception) {}
                }
            }
        }
    }
}
