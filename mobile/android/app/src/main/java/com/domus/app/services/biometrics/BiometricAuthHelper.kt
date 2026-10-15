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
     * Generates standard localized prompt metadata for high-security actions.
     */
    fun createPromptTitle(actionType: String = "disarm"): Pair<String, String> {
        return when (actionType) {
            "disarm" -> Pair(
                "Conferma Sicurezza Domus",
                "Autenticati con impronta o volto per disarmare l'allarme"
            )
            "lock" -> Pair(
                "Sblocco Serratura Smart",
                "Autenticazione richiesta per aprire la porta d'ingresso"
            )
            else -> Pair(
                "Autenticazione Biometrica",
                "Conferma l'operazione di sicurezza"
            )
        }
    }
}
