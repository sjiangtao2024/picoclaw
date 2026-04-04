export const CHAT_SEND_CONNECT_TIMEOUT_MS = 3_000
export const CHAT_SOCKET_HEALTHCHECK_TIMEOUT_MS = 1_500

interface MinimalSocketLike extends EventTarget {
  readyState: number
}

interface MinimalMessageSocketLike extends MinimalSocketLike {
  send(message: string): void
}

export function waitForSocketOpen(
  socket: MinimalSocketLike | null,
  timeoutMs: number = CHAT_SEND_CONNECT_TIMEOUT_MS,
): Promise<boolean> {
  if (!socket) {
    return Promise.resolve(false)
  }

  if (socket.readyState === WebSocket.OPEN) {
    return Promise.resolve(true)
  }

  if (socket.readyState !== WebSocket.CONNECTING) {
    return Promise.resolve(false)
  }

  return new Promise((resolve) => {
    let settled = false

    const cleanup = () => {
      socket.removeEventListener("open", handleOpen)
      socket.removeEventListener("error", handleFailure)
      socket.removeEventListener("close", handleFailure)
      clearTimeout(timer)
    }

    const finish = (result: boolean) => {
      if (settled) {
        return
      }
      settled = true
      cleanup()
      resolve(result)
    }

    const handleOpen = () => finish(true)
    const handleFailure = () => finish(false)

    const timer = setTimeout(() => finish(false), timeoutMs)

    socket.addEventListener("open", handleOpen)
    socket.addEventListener("error", handleFailure)
    socket.addEventListener("close", handleFailure)
  })
}

export function verifySocketHealth(
  socket: MinimalMessageSocketLike | null,
  timeoutMs: number = CHAT_SOCKET_HEALTHCHECK_TIMEOUT_MS,
  createPingId: () => string = () => `ping-${Date.now()}`,
): Promise<boolean> {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    return Promise.resolve(false)
  }

  return new Promise((resolve) => {
    let settled = false
    const pingId = createPingId()

    const cleanup = () => {
      socket.removeEventListener("message", handleMessage)
      socket.removeEventListener("error", handleFailure)
      socket.removeEventListener("close", handleFailure)
      clearTimeout(timer)
    }

    const finish = (result: boolean) => {
      if (settled) {
        return
      }
      settled = true
      cleanup()
      resolve(result)
    }

    const handleFailure = () => finish(false)
    const handleMessage = (event: Event) => {
      if (!(event instanceof MessageEvent)) {
        return
      }

      try {
        const message = JSON.parse(event.data as string) as {
          type?: string
          id?: string
        }
        if (message.type === "pong" && message.id === pingId) {
          finish(true)
        }
      } catch {
        // Ignore non-JSON frames from other handlers.
      }
    }

    const timer = setTimeout(() => finish(false), timeoutMs)

    socket.addEventListener("message", handleMessage)
    socket.addEventListener("error", handleFailure)
    socket.addEventListener("close", handleFailure)

    try {
      socket.send(
        JSON.stringify({
          type: "ping",
          id: pingId,
        }),
      )
    } catch {
      finish(false)
    }
  })
}
