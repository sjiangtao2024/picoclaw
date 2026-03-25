import test from "node:test"
import assert from "node:assert/strict"

import {
  buildEditConfig,
  buildSavePayload,
} from "./channel-config-utils.ts"

test("buildEditConfig preserves blank edit slots for known secret fields", () => {
  const config = {
    app_id: "cli_xxx",
    app_secret: "stored-secret",
  }

  const editConfig = buildEditConfig(config)

  assert.equal(editConfig.app_secret, "stored-secret")
  assert.equal(editConfig._app_secret, "")
})

test("buildSavePayload keeps typed app secret for first-time feishu setup", () => {
  const channel = {
    name: "feishu",
    config_key: "feishu",
    display_name: "Feishu",
  }
  const editConfig = {
    app_id: "cli_xxx",
    _app_secret: "new-secret",
  }

  const payload = buildSavePayload(channel, editConfig, true)

  assert.equal(payload.app_id, "cli_xxx")
  assert.equal(payload.app_secret, "new-secret")
  assert.equal(payload.enabled, true)
})
