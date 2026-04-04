import assert from "node:assert/strict"
import test from "node:test"

import {
  CHAT_SEND_CONNECT_TIMEOUT_MS,
  CHAT_SOCKET_HEALTHCHECK_TIMEOUT_MS,
  verifySocketHealth,
  waitForSocketOpen,
} from "./socket-connection.ts"

class FakeSocket extends EventTarget {
  readyState: number
  sentMessages: string[] = []

  constructor(readyState: number) {
    super()
    this.readyState = readyState
  }

  open() {
    this.readyState = WebSocket.OPEN
    this.dispatchEvent(new Event("open"))
  }

  fail(type: "error" | "close") {
    this.readyState = WebSocket.CLOSED
    this.dispatchEvent(new Event(type))
  }

  send(message: string) {
    this.sentMessages.push(message)
  }

  receive(data: unknown) {
    this.dispatchEvent(
      new MessageEvent("message", {
        data: typeof data === "string" ? data : JSON.stringify(data),
      }),
    )
  }
}

test("waitForSocketOpen resolves immediately for open sockets", async () => {
  const socket = new FakeSocket(WebSocket.OPEN)

  await assert.doesNotReject(() => waitForSocketOpen(socket, 20))
  assert.equal(await waitForSocketOpen(socket, 20), true)
})

test("waitForSocketOpen resolves when a connecting socket opens", async () => {
  const socket = new FakeSocket(WebSocket.CONNECTING)
  const pending = waitForSocketOpen(socket, 50)

  setTimeout(() => socket.open(), 5)

  assert.equal(await pending, true)
})

test("waitForSocketOpen rejects when a socket closes before opening", async () => {
  const socket = new FakeSocket(WebSocket.CONNECTING)
  const pending = waitForSocketOpen(socket, 50)

  setTimeout(() => socket.fail("close"), 5)

  assert.equal(await pending, false)
})

test("waitForSocketOpen rejects after the timeout", async () => {
  const socket = new FakeSocket(WebSocket.CONNECTING)

  assert.equal(
    await waitForSocketOpen(socket, Math.min(20, CHAT_SEND_CONNECT_TIMEOUT_MS)),
    false,
  )
})

test("verifySocketHealth sends a ping and resolves on matching pong", async () => {
  const socket = new FakeSocket(WebSocket.OPEN)
  const pending = verifySocketHealth(socket, 50, () => "ping-1")

  setTimeout(() => {
    socket.receive({ type: "pong", id: "ping-1" })
  }, 5)

  assert.equal(await pending, true)
  assert.equal(socket.sentMessages.length, 1)
  assert.match(socket.sentMessages[0], /"type":"ping"/)
})

test("verifySocketHealth rejects when pong does not arrive", async () => {
  const socket = new FakeSocket(WebSocket.OPEN)

  assert.equal(
    await verifySocketHealth(
      socket,
      Math.min(20, CHAT_SOCKET_HEALTHCHECK_TIMEOUT_MS),
      () => "ping-timeout",
    ),
    false,
  )
})
