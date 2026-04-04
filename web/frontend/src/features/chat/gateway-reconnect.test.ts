import assert from "node:assert/strict"
import test from "node:test"

import { shouldReconnectForGatewayUpdate } from "./gateway-reconnect.ts"

test("shouldReconnectForGatewayUpdate returns true when running gateway pid changes", () => {
  assert.equal(
    shouldReconnectForGatewayUpdate(
      { status: "running", pid: 101 },
      { status: "running", pid: 202 },
    ),
    true,
  )
})

test("shouldReconnectForGatewayUpdate ignores updates without a running-to-running pid change", () => {
  assert.equal(
    shouldReconnectForGatewayUpdate(
      { status: "running", pid: 101 },
      { status: "running", pid: 101 },
    ),
    false,
  )
  assert.equal(
    shouldReconnectForGatewayUpdate(
      { status: "starting", pid: undefined },
      { status: "running", pid: 202 },
    ),
    false,
  )
  assert.equal(
    shouldReconnectForGatewayUpdate(
      { status: "running", pid: undefined },
      { status: "running", pid: 202 },
    ),
    false,
  )
})
