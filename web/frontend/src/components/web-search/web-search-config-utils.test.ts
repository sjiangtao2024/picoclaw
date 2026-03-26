import test from "node:test"
import assert from "node:assert/strict"

import {
  buildWebSearchEditConfig,
  buildWebSearchSavePayload,
  hasStoredAPIKey,
} from "./web-search-config-utils.ts"

test("buildWebSearchEditConfig preserves blank edit slot for api key", () => {
  const config = {
    enabled: true,
    max_results: 8,
    api_key: "stored-secret",
  }

  const editConfig = buildWebSearchEditConfig(config)

  assert.equal(editConfig.api_key, "stored-secret")
  assert.equal(editConfig._api_key, "")
})

test("buildWebSearchSavePayload keeps typed api key for first save", () => {
  const payload = buildWebSearchSavePayload({
    prefer_native: false,
    duckduckgo_enabled: false,
    baidu_enabled: true,
    max_results: "8",
    _api_key: "new-secret",
  })

  assert.equal(payload.prefer_native, false)
  assert.equal(payload.duckduckgo.enabled, false)
  assert.equal(payload.baidu_search.enabled, true)
  assert.equal(payload.baidu_search.max_results, 8)
  assert.equal(payload.baidu_search.api_key, "new-secret")
})

test("hasStoredAPIKey recognizes backend presence hint", () => {
  assert.equal(hasStoredAPIKey({ api_key_set: true }), true)
  assert.equal(hasStoredAPIKey({ api_key_set: false }), false)
})

test("buildWebSearchSavePayload ignores api_key_set hint", () => {
  const payload = buildWebSearchSavePayload({
    prefer_native: false,
    duckduckgo_enabled: true,
    baidu_enabled: true,
    max_results: "10",
    api_key_set: true,
  })

  assert.equal("api_key_set" in payload.baidu_search, false)
})
