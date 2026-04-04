import assert from "node:assert/strict"
import test from "node:test"

import { getGatewayPollIntervalMs } from "./gateway-polling.ts"

test("getGatewayPollIntervalMs uses slow polling for stable states", () => {
  assert.equal(getGatewayPollIntervalMs("running"), 60_000)
  assert.equal(getGatewayPollIntervalMs("stopped"), 60_000)
  assert.equal(getGatewayPollIntervalMs("error"), 60_000)
  assert.equal(getGatewayPollIntervalMs("unknown"), 60_000)
})

test("getGatewayPollIntervalMs uses faster polling for transient states", () => {
  assert.equal(getGatewayPollIntervalMs("starting"), 3_000)
  assert.equal(getGatewayPollIntervalMs("restarting"), 3_000)
  assert.equal(getGatewayPollIntervalMs("stopping"), 3_000)
})
