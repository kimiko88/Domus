package com.domus.app.domain.usecase

import com.domus.app.domain.model.AuthSession
import com.domus.app.domain.model.Entity
import com.domus.app.domain.repository.AuthRepository
import com.domus.app.domain.repository.EntityRepository

class SyncEntitiesUseCase(
    private val entityRepository: EntityRepository
) {
    suspend operator fun invoke(): Result<List<Entity>> {
        return entityRepository.syncEntities()
    }
}

class ToggleEntityUseCase(
    private val entityRepository: EntityRepository
) {
    suspend operator fun invoke(entityId: String): Result<Entity> {
        return entityRepository.toggleEntity(entityId)
    }
}

class AuthenticateUseCase(
    private val authRepository: AuthRepository
) {
    suspend fun login(username: String, password: String): Result<AuthSession> {
        return authRepository.login(username, password)
    }

    suspend fun logout() {
        authRepository.logout()
    }

    suspend fun checkSession(): AuthSession {
        return authRepository.restoreSession()
    }
}
