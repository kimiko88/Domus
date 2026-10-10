import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAssistantStore } from './assistant'
import { apiClient } from '../api/client'

describe('useAssistantStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('initializes with default welcome message', () => {
    const store = useAssistantStore()
    expect(store.messages.length).toBe(1)
    expect(store.messages[0].sender).toBe('assistant')
    expect(store.isProcessing).toBe(false)
  })

  it('sends prompt and records user message and assistant reply with executed actions', async () => {
    const mockReply = {
      user_prompt: 'Spegni la luce in cucina',
      response_text: 'Ho spento la luce in cucina.',
      actions_executed: [
        {
          entity_key: 'light.cucina',
          service: 'turn_off',
          success: true,
          message: 'Luce spenta'
        }
      ],
      engine_used: 'local_heuristic',
      execution_time_ms: 12
    }

    const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(mockReply as any)

    const store = useAssistantStore()
    const res = await store.sendPrompt('Spegni la luce in cucina')

    expect(postSpy).toHaveBeenCalledWith('/api/v1/assistant/chat', { prompt: 'Spegni la luce in cucina' })
    expect(res?.actions_executed.length).toBe(1)
    expect(store.messages.length).toBe(3) // welcome + user + assistant
    expect(store.messages[1].sender).toBe('user')
    expect(store.messages[1].text).toBe('Spegni la luce in cucina')
    expect(store.messages[2].sender).toBe('assistant')
    expect(store.messages[2].text).toBe('Ho spento la luce in cucina.')
  })

  it('clears history back to welcome message', () => {
    const store = useAssistantStore()
    store.messages.push({
      id: 'custom_1',
      sender: 'user',
      text: 'Test',
      timestamp: new Date().toISOString()
    })
    expect(store.messages.length).toBe(2)

    store.clearHistory()
    expect(store.messages.length).toBe(1)
    expect(store.messages[0].text).toContain('Cronologia azzerata')
  })
})
