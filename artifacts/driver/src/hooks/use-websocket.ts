import { useEffect, useRef, useCallback, useState } from 'react'
import { getWebSocketURL, getAuthToken } from '@/lib/api'

interface WSMessage {
  id: string
  type: string
  channel: string
  data: any
  timestamp: string
}

interface UseWebSocketOptions {
  onMessage?: (msg: WSMessage) => void
  onOpen?: () => void
  onClose?: () => void
  onError?: (err: Event) => void
  reconnectInterval?: number // ms
  maxReconnectAttempts?: number
}

/**
 * Persistent WebSocket hook that auto-reconnects with exponential backoff.
 * 
 * The WebSocket stays connected as long as the user is authenticated.
 * If the connection drops, it automatically reconnects.
 * 
 * Usage:
 *   const { isConnected, send, subscribe, unsubscribe } = useWebSocket({
 *     onMessage: (msg) => console.log(msg),
 *   })
 */
export function useWebSocket(options: UseWebSocketOptions = {}) {
  const {
    onMessage,
    onOpen,
    onClose,
    onError,
    reconnectInterval = 1000,
    maxReconnectAttempts = Infinity,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const reconnectTimerRef = useRef<NodeJS.Timeout | null>(null)
  const subscribedChannelsRef = useRef<Set<string>>(new Set())
  const onMessageRef = useRef(onMessage)
  const onOpenRef = useRef(onOpen)
  const onCloseRef = useRef(onClose)
  const onErrorRef = useRef(onError)

  // Update refs without re-creating the WebSocket
  useEffect(() => {
    onMessageRef.current = onMessage
    onOpenRef.current = onOpen
    onCloseRef.current = onClose
    onErrorRef.current = onError
  }, [onMessage, onOpen, onClose, onError])

  const connect = useCallback(() => {
    const token = getAuthToken()
    if (!token) return

    // Close existing connection
    if (wsRef.current) {
      wsRef.current.close()
      wsRef.current = null
    }

    const ws = new WebSocket(getWebSocketURL(token))
    wsRef.current = ws

    ws.onopen = () => {
      setIsConnected(true)
      reconnectAttemptsRef.current = 0
      // Re-subscribe to all channels
      subscribedChannelsRef.current.forEach((channel) => {
        ws.send(JSON.stringify({ action: 'subscribe', channel }))
      })
      onOpenRef.current?.()
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        // Skip ack/error/system messages
        if (msg.type === 'ack' || msg.type === 'error') return
        onMessageRef.current?.(msg)
      } catch (err) {
        console.error('WS parse error:', err)
      }
    }

    ws.onclose = () => {
      setIsConnected(false)
      wsRef.current = null
      onCloseRef.current?.()

      // Auto-reconnect with exponential backoff
      if (reconnectAttemptsRef.current < maxReconnectAttempts) {
        const delay = Math.min(
          reconnectInterval * Math.pow(2, reconnectAttemptsRef.current),
          30000 // max 30s
        )
        reconnectTimerRef.current = setTimeout(() => {
          reconnectAttemptsRef.current++
          connect()
        }, delay)
      }
    }

    ws.onerror = (err) => {
      onErrorRef.current?.(err)
    }
  }, [reconnectInterval, maxReconnectAttempts])

  // Connect on mount if token exists
  useEffect(() => {
    const token = getAuthToken()
    if (token) {
      connect()
    }

    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
      }
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [connect])

  const send = useCallback((msg: any) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg))
    }
  }, [])

  const subscribe = useCallback((channel: string) => {
    subscribedChannelsRef.current.add(channel)
    send({ action: 'subscribe', channel })
  }, [send])

  const unsubscribe = useCallback((channel: string) => {
    subscribedChannelsRef.current.delete(channel)
    send({ action: 'unsubscribe', channel })
  }, [send])

  return { isConnected, send, subscribe, unsubscribe }
}
