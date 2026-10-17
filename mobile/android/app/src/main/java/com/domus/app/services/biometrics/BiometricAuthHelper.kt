package com.domus.app.services.biometrics

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey

object BiometricAuthHelper {

    private const val KEY_ALIAS = "DomusBiometricSecurityKey"
    private const val ANDROID_KEYSTORE = "AndroidKeyStore"

    /**
     * Ensures an AES-256 GCM key exists in the hardware-backed Android Keystore,
     * configured to require biometric user authentication for cryptographic operations.
     */
    fun getOrCreateSecretKey(): SecretKey {
        val keyStore = KeyStore.getInstance(ANDROID_KEYSTORE).apply { load(null) }

        if (!keyStore.containsAlias(KEY_ALIAS)) {
            val keyGenerator = KeyGenerator.getInstance(
                KeyProperties.KEY_ALGORITHM_AES,
                ANDROID_KEYSTORE
            )

            val keyGenSpec = KeyGenParameterSpec.Builder(
                KEY_ALIAS,
                KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT
            )
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setUserAuthenticationRequired(true)
                .apply {
                    if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.R) {
                        setUserAuthenticationParameters(
                            0, // Require auth for every single cryptographic access
                            KeyProperties.AUTH_BIOMETRIC_STRONG
                        )
                    }
                }
                .build()

            keyGenerator.init(keyGenSpec)
            return keyGenerator.generateKey()
        }

        return (keyStore.getEntry(KEY_ALIAS, null) as KeyStore.SecretKeyEntry).secretKey
    }

    /**
     * Initializes a cipher instance ready to be wrapped into a BiometricPrompt.CryptoObject.
     */
    fun createCryptoCipher(): Cipher {
        val cipher = Cipher.getInstance(
            "${KeyProperties.KEY_ALGORITHM_AES}/${KeyProperties.BLOCK_MODE_GCM}/${KeyProperties.ENCRYPTION_PADDING_NONE}"
        )
        val secretKey = getOrCreateSecretKey()
        cipher.init(Cipher.ENCRYPT_MODE, secretKey)
        return cipher
    }

    /**
     * Checks if an entity is sensitive and requires zero-trust biometric authorization
     * before executing open/unlock/disarm operations.
     */
    fun isSensitiveEntity(domain: String, entityId: String): Boolean {
        val id = entityId.lowercase()
        if (domain == "alarm_control_panel" || domain == "lock") return true
        if (domain == "cover" && (id.contains("gate") || id.contains("cancello") || 
            id.contains("garage") || id.contains("portone") || id.contains("basculante") || id.contains("door"))) {
            return true
        }
        return false
    }

    /**
     * Generates standard localized prompt metadata for high-security actions.
     */
    fun createPromptTitle(actionType: String = "disarm"): Pair<String, String> {
        return when (actionType) {
            "disarm" -> Pair(
                "Conferma Sicurezza Domus",
                "Autenticati con impronta o volto per disarmare l'allarme"
            )
            "lock", "gate", "garage" -> Pair(
                "Sblocco Accesso Sensibile",
                "Autenticazione richiesta per aprire portone, cancello o garage"
            )
            else -> Pair(
                "Autenticazione Biometrica Zero-Trust",
                "Conferma l'operazione di sicurezza per l'entità protetta"
            )
        }
    }

    /**
     * Shows native BiometricPrompt to authorize critical operations.
     */
    fun authenticateCriticalAction(
        activity: androidx.fragment.app.FragmentActivity,
        title: String = "Conferma Sicurezza Domus",
        subtitle: String = "Autenticazione richiesta per completare l'operazione",
        onSuccess: () -> Unit,
        onError: (String) -> Unit = {}
    ) {
        val executor = androidx.core.content.ContextCompat.getMainExecutor(activity)
        val promptInfo = androidx.biometric.BiometricPrompt.PromptInfo.Builder()
            .setTitle(title)
            .setSubtitle(subtitle)
            .setNegativeButtonText("Annulla")
            .setAllowedAuthenticators(
                androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_STRONG or
                androidx.biometric.BiometricManager.Authenticators.BIOMETRIC_WEAK or
                androidx.biometric.BiometricManager.Authenticators.DEVICE_CREDENTIAL
            )
            .build()

        val biometricPrompt = androidx.biometric.BiometricPrompt(
            activity,
            executor,
            object : androidx.biometric.BiometricPrompt.AuthenticationCallback() {
                override fun onAuthenticationSucceeded(result: androidx.biometric.BiometricPrompt.AuthenticationResult) {
                    super.onAuthenticationSucceeded(result)
                    onSuccess()
                }

                override fun onAuthenticationError(errorCode: Int, errString: CharSequence) {
                    super.onAuthenticationError(errorCode, errString)
                    onError(errString.toString())
                }
            }
        )

        biometricPrompt.authenticate(promptInfo)
    }
}

