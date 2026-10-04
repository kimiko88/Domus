import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { useWebSocket } from './useWebSocket'
import type { WsMessage } from '../api/types'

// Mock WebSocket
class MockWebSocket {
  static instances: MockWebSocket[] = []
  url: string
  readyState: number = WebSocket.CONNECTING
  sentData: string[] = []

  onopen: ((event: any) => void) | null = null
  onclose: ((event: any) => void) | null = null
  onmessage: ((event: any) => void) | null = null
  onerror: ((event: any) => void) | null = null

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }

  send(data: string) {
    this.sentData.push(data)
  }

  close() {
    this.readyState = WebSocket.CLOSED
    if (this.onclose) {
      this.onclose({ code: 1000, reason: 'Normal Closure' })
    }
  }

  // Test helpers
  simulateOpen() {
    this.readyState = WebSocket.OPEN
    if (this.onopen) {
      this.onopen({})
    }
  }

  simulateMessage(msg: WsMessage) {
    if (this.onmessage) {
      this.onmessage({ data: JSON.stringify(msg) })
    }
  }

  simulateClose() {
    this.readyState = WebSocket.CLOSED
    if (this.onclose) {
      this.onclose({ code: 1006, reason: 'Abnormal Closure' })
    }
  }
}

describe('useWebSocket Composable', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    // @ts-ignore
    global.WebSocket = MockWebSocket
    // @ts-ignore
    global.WebSocket.OPEN = 1
    // @ts-ignore
    global.WebSocket.CONNECTING = 0
    // @ts-ignore
    global.WebSocket.CLOSED = 3
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('initializes with disconnected state and connects', () => {
    const ws = useWebSocket({ autoConnect: false, url: 'ws://localhost/ws' })
    expect(ws.isConnected.value).toBe(false)
    expect(MockWebSocket.instances.length).toBe(0)

    ws.connect()
    expect(MockWebSocket.instances.length).toBe(1)
    const instance = MockWebSocket.instances[0]

    instance.simulateOpen()
    expect(ws.isConnected.value).toBe(true)
  })

  it('receives events and dispatches to subscribers', () => {
    const ws = useWebSocket({ autoConnect: true, url: 'ws://localhost/ws' })
    const instance = MockWebSocket.instances[0]
    instance.simulateOpen()

    const eventReceived: any[] = []
    const unsub = ws.subscribe('state_changed', (payload) => {
      eventReceived.push(payload)
    })

    instance.simulateMessage({
      type: 'event',
      event_type: 'state_changed',
      payload: { entity_id: 'light.living_room', new_state: 'on' }
    })

    expect(eventReceived.length).toBe(1)
    expect(eventReceived[0]).toEqual({ entity_id: 'light.living_room', new_state: 'on' })

    // Unsubscribe
    unsub()
    instance.simulateMessage({
      type: 'event',
      event_type: 'state_changed',
      payload: { entity_id: 'light.living_room', new_state: 'off' }
    })
    expect(eventReceived.length).toBe(1)
  })

  it('sends call_service message when requested', () => {
    const ws = useWebSocket({ autoConnect: true, url: 'ws://localhost/ws' })
    const instance = MockWebSocket.instances[0]
    instance.simulateOpen()

    ws.callService('light', 'turn_on', 'light.kitchen', { brightness: 255 })
    expect(instance.sentData.length).toBe(1)
    const parsed = JSON.parse(instance.sentData[0])
    expect(parsed).toEqual({
      type: 'call_service',
      domain: 'light',
      service: 'turn_on',
      entity_id: 'light.kitchen',
      data: { brightness: 255 }
    })
  })

  it('attempts reconnection when disconnected abnormally', () => {
    const ws = useWebSocket({ autoConnect: true, url: 'ws://localhost/ws', reconnectInterval: 1000 })
    const instance = MockWebSocket.instances[0]
    instance.simulateOpen()
    expect(ws.isConnected.value).toBe(true)

    instance.simulateClose()
    expect(ws.isConnected.value).toBe(false)

    // Fast-forward timer to trigger reconnect
    vi.advanceTimersByTime(1100)
    expect(MockWebSocket.instances.length).toBe(2)
  })
})
