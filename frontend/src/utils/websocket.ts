import { ref } from 'vue'

export const useWebSocket = (url: string) => {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const messages = ref<any[]>([])

  const connect = () => {
    ws.value = new WebSocket(url)

    ws.value.onopen = () => {
      connected.value = true
    }

    ws.value.onmessage = (event) => {
      const data = JSON.parse(event.data)
      messages.value.push(data)
    }

    ws.value.onclose = () => {
      connected.value = false
      // 自动重连
      setTimeout(connect, 3000)
    }
  }

  const send = (data: any) => {
    if (ws.value && connected.value) {
      ws.value.send(JSON.stringify(data))
    }
  }

  const close = () => {
    ws.value?.close()
  }

  return { connected, messages, connect, send, close }
}
