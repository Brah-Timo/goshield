import type { WSMessage } from '@/types'
import { getAccessToken } from './api'

type MessageHandler = (msg: WSMessage) => void

class WebSocketClient {
  private ws: WebSocket | null = null
  private handlers: Set<MessageHandler> = new Set()
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private reconnectDelay = 2000
  private maxReconnectDelay = 30_000
  private intentionalClose = false

  connect(companyId: string) {
    this.intentionalClose = false
    this.reconnectDelay = 2000
    this._open(companyId)
  }

  private _open(companyId: string) {
    const token = getAccessToken()
    if (!token) return

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
    const host = window.location.host
    const url = `${proto}://${host}/ws?token=${encodeURIComponent(token)}&company_id=${companyId}`

    this.ws = new WebSocket(url)

    this.ws.onmessage = (e) => {
      try {
        const msg: WSMessage = JSON.parse(e.data)
        this.handlers.forEach((h) => h(msg))
      } catch {
        // ignore malformed messages
      }
    }

    this.ws.onclose = () => {
      if (!this.intentionalClose) {
        this.reconnectTimer = setTimeout(() => {
          this.reconnectDelay = Math.min(this.reconnectDelay * 1.5, this.maxReconnectDelay)
          this._open(companyId)
        }, this.reconnectDelay)
      }
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  disconnect() {
    this.intentionalClose = true
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ws?.close()
    this.ws = null
  }

  subscribe(handler: MessageHandler): () => void {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler)
  }

  get isConnected() {
    return this.ws?.readyState === WebSocket.OPEN
  }
}

export const wsClient = new WebSocketClient()
