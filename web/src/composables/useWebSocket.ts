import { ref, onUnmounted } from 'vue'
import type { Message } from '../types/message'

type MessageHandler = (msg: Message) => void

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 10
  const handlers = new Map<string, MessageHandler[]>()
  const pendingQueue: Message[] = []

  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function connect(url: string) {
    disconnect()

    const socket = new WebSocket(url)

    socket.onopen = () => {
      connected.value = true
      reconnectAttempts.value = 0
      // Flush pending messages
      const queue = pendingQueue.splice(0)
      for (const msg of queue) {
        socket.send(JSON.stringify(msg))
      }
    }

    socket.onclose = () => {
      connected.value = false
      scheduleReconnect(url)
    }

    socket.onerror = () => {
      connected.value = false
    }

    socket.onmessage = (event) => {
      try {
        const msg: Message = JSON.parse(event.data)
        const list = handlers.get(msg.type)
        if (list) {
          list.forEach(h => h(msg))
        }
      } catch {
        // ignore invalid messages
      }
    }

    ws.value = socket
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    connected.value = false
    pendingQueue.splice(0)
  }

  function scheduleReconnect(url: string) {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      return
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 30000)
    reconnectAttempts.value++

    reconnectTimer = setTimeout(() => {
      connect(url)
    }, delay)
  }

  function send(msg: Message) {
    const socket = ws.value
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(msg))
    } else {
      pendingQueue.push(msg)
    }
  }

  function on(type: string, handler: MessageHandler) {
    if (!handlers.has(type)) {
      handlers.set(type, [])
    }
    handlers.get(type)!.push(handler)
  }

  function off(type: string, handler: MessageHandler) {
    const list = handlers.get(type)
    if (list) {
      const idx = list.indexOf(handler)
      if (idx >= 0) {
        list.splice(idx, 1)
      }
    }
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    connected,
    reconnectAttempts,
    connect,
    disconnect,
    send,
    on,
    off,
  }
}
