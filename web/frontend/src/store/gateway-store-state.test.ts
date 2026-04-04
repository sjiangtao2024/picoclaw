import assert from "node:assert/strict"
import test from "node:test"

import { applyGatewayStatusPatch, type GatewayStoreState } from "./gateway-store-state.ts"

test("applyGatewayStatusPatch stores the gateway pid", () => {
  const prev: GatewayStoreState = {
    status: "unknown",
    canStart: true,
    restartRequired: false,
    pid: undefined,
  }

  const next = applyGatewayStatusPatch(prev, {
    gateway_status: "running",
    gateway_start_allowed: false,
    gateway_restart_required: false,
    pid: 4242,
  })

  assert.equal(next.status, "running")
  assert.equal(next.pid, 4242)
})
