import { ref, onMounted, onUnmounted, type Ref } from 'vue'
import type { WsMessage } from '../api/types'

export interface WebSocketOptions {
  url?: string
  autoConnect?: boolean
  reconnectInterval?: number
  maxReconnectInterval?: number
  heartbeatInterval?: number
}

export type MessageHandler<T = any> = (payload: T, raw: WsMessage<T>) => void

export function useWebSocket(options: WebSocketOptions = {}) {
  const {
    url,
    autoConnect = true,
    reconnectInterval = 2000,
    maxReconnectInterval = 30000,
    heartbeatInterval = 30000
  } = options

  const isConnected = ref(false)
  const isConnecting = ref(false)
  const error = ref<string | null>(null)

  let ws: WebSocket | null = null
  let reconnectTimer: any = null
  let heartbeatTimer: any = null
  let currentBackoff = reconnectInterval
  let explicitlyClosed = false

  const listeners = new Map<string, Set<MessageHandler>>()

  const getWsUrl = (): string => {
    if (url) return url
    if (typeof window !== 'undefined') {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      return `${protocol}//${window.location.host}/ws`
    }
    return 'ws://localhost:8080/ws'
  }

  const send = (message: WsMessage): boolean => {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      return false
    }
    try {
      ws.send(JSON.stringify(message))
      return true
    } catch (e: any) {
      error.value = e?.message || 'Send failed'
      return false
    }
  }

  const callService = (
    domain: string,
    service: string,
    entity_id?: string,
    data?: Record<string, any>
  ): boolean => {
    return send({
      type: 'call_service',
      domain,
      service,
      entity_id,
      data
    })
  }

  const subscribe = <T = any>(eventType: string, handler: MessageHandler<T>): (() => void) => {
    if (!listeners.has(eventType)) {
      listeners.set(eventType, new Set())
    }
    listeners.get(eventType)!.add(handler as MessageHandler)

    // Return un-subscriber
    return () => {
      unsubscribe(eventType, handler)
    }
  }

  const unsubscribe = <T = any>(eventType: string, handler: MessageHandler<T>) => {
    const handlers = listeners.get(eventType)
    if (handlers) {
      handlers.delete(handler as MessageHandler)
      if (handlers.size === 0) {
        listeners.delete(eventType)
      }
    }
  }

  const dispatchEvent = (eventType: string, payload: any, raw: WsMessage) => {
    const specificHandlers = listeners.get(eventType)
    if (specificHandlers) {
      specificHandlers.forEach((h) => {
        try {
          h(payload, raw)
        } catch (err) {
          console.error(`Error in WebSocket subscriber for [${eventType}]:`, err)
        }
      })
    }
    // Wildcard '*' receives every event, regardless of whether specific handlers exist
    const wildcards = listeners.get('*')
    if (wildcards) {
      wildcards.forEach((h) => {
        try {
          h(payload, raw)
        } catch (err) {
          console.error('Error in WebSocket wildcard subscriber:', err)
        }
      })
    }
  }

  const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (isConnected.value) {
        send({ type: 'ping' })
      }
    }, heartbeatInterval)
  }

  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  const scheduleReconnect = () => {
    if (explicitlyClosed) return
    if (reconnectTimer) return

    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      currentBackoff = Math.min(currentBackoff * 1.5, maxReconnectInterval)
      connect()
    }, currentBackoff)
  }

  const connect = () => {
    if (typeof window === 'undefined' && typeof WebSocket === 'undefined') return
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    explicitlyClosed = false
    isConnecting.value = true
    error.value = null

    try {
      ws = new WebSocket(getWsUrl())

      ws.onopen = () => {
        isConnected.value = true
        isConnecting.value = false
        currentBackoff = reconnectInterval
        error.value = null
        startHeartbeat()
      }

      ws.onclose = (event) => {
        isConnected.value = false
        isConnecting.value = false
        stopHeartbeat()
        // Normal closure codes (1000 = normal, 1001 = going away) from server side
        // are still treated as unexpected if the client did not explicitly close.
        if (!explicitlyClosed) {
          scheduleReconnect()
        }
      }

      ws.onerror = (e: any) => {
        error.value = e?.message || 'WebSocket error'
      }

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data) as WsMessage
          if (msg.type === 'event' && msg.event_type) {
            dispatchEvent(msg.event_type, msg.payload, msg)
          } else if (msg.type === 'pong') {
            // Heartbeat ACK
          } else {
            dispatchEvent(msg.type, msg.payload ?? msg, msg)
          }
        } catch (err) {
          console.error('Failed to parse WebSocket incoming JSON message:', err)
        }
      }
    } catch (err: any) {
      isConnecting.value = false
      error.value = err?.message || 'Connection failed'
      scheduleReconnect()
    }
  }

  const disconnect = () => {
    explicitlyClosed = true
    stopHeartbeat()
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
    isConnected.value = false
    isConnecting.value = false
  }

  if (autoConnect) {
    connect()
  }

  return {
    isConnected,
    isConnecting,
    error,
    connect,
    disconnect,
    send,
    callService,
    subscribe,
    unsubscribe
  }
}
