type JsonRecord = Record<string, unknown>

function asRecord(value: unknown): JsonRecord {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as JsonRecord
  }
  return {}
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBool(value: unknown): boolean {
  return value === true
}

export interface WebSearchEditConfig {
  prefer_native: boolean
  duckduckgo_enabled: boolean
  baidu_enabled: boolean
  max_results: string
  api_key?: string
  _api_key?: string
  api_key_set?: boolean
}

export function hasStoredAPIKey(config: unknown): boolean {
  const record = asRecord(config)
  if (asString(record.api_key).trim() !== "") {
    return true
  }
  return asBool(record.api_key_set)
}

export function buildWebSearchEditConfig(config: unknown): WebSearchEditConfig {
  const root = asRecord(config)
  const baidu = Object.keys(asRecord(root.baidu_search)).length
    ? asRecord(root.baidu_search)
    : root
  const duckduckgo = asRecord(root.duckduckgo)

  return {
    prefer_native: asBool(root.prefer_native),
    duckduckgo_enabled: asBool(duckduckgo.enabled),
    baidu_enabled: asBool(baidu.enabled),
    max_results:
      typeof baidu.max_results === "number" && Number.isFinite(baidu.max_results)
        ? String(baidu.max_results)
        : "",
    api_key: asString(baidu.api_key),
    _api_key: "",
    api_key_set: asBool(baidu.api_key_set),
  }
}

export function buildWebSearchSavePayload(edit: WebSearchEditConfig) {
  const payload: {
    prefer_native: boolean
    duckduckgo: { enabled: boolean }
    baidu_search: {
      enabled: boolean
      max_results?: number
      api_key?: string
    }
  } = {
    prefer_native: edit.prefer_native,
    duckduckgo: {
      enabled: edit.duckduckgo_enabled,
    },
    baidu_search: {
      enabled: edit.baidu_enabled,
    },
  }

  const maxResults = edit.max_results.trim()
  if (maxResults !== "") {
    payload.baidu_search.max_results = Number.parseInt(maxResults, 10)
  }

  const incomingAPIKey = asString(edit._api_key).trim()
  if (incomingAPIKey !== "") {
    payload.baidu_search.api_key = incomingAPIKey
  } else if (asString(edit.api_key).trim() !== "") {
    payload.baidu_search.api_key = asString(edit.api_key)
  }

  return payload
}
