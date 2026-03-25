import type { ChannelConfig, SupportedChannel } from "@/api/channels"

const SECRET_FIELD_MAP: Record<string, string> = {
  token: "_token",
  app_secret: "_app_secret",
  client_secret: "_client_secret",
  corp_secret: "_corp_secret",
  channel_secret: "_channel_secret",
  channel_access_token: "_channel_access_token",
  access_token: "_access_token",
  bot_token: "_bot_token",
  app_token: "_app_token",
  encoding_aes_key: "_encoding_aes_key",
  encrypt_key: "_encrypt_key",
  verification_token: "_verification_token",
  password: "_password",
  nickserv_password: "_nickserv_password",
  sasl_password: "_sasl_password",
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asBool(value: unknown): boolean {
  return value === true
}

export function hasStoredSecret(config: ChannelConfig, key: string): boolean {
  if (asString(config[key]).trim() !== "") {
    return true
  }
  return asBool(config[`${key}_set`])
}

export function buildEditConfig(config: ChannelConfig): ChannelConfig {
  const edit: ChannelConfig = { ...config }
  for (const secretKey of Object.keys(SECRET_FIELD_MAP)) {
    if (secretKey in config) {
      edit[SECRET_FIELD_MAP[secretKey]] = ""
    }
  }
  return edit
}

export function buildSavePayload(
  channel: SupportedChannel,
  editConfig: ChannelConfig,
  enabled: boolean,
): ChannelConfig {
  const payload: ChannelConfig = { enabled }

  for (const [secretKey, editKey] of Object.entries(SECRET_FIELD_MAP)) {
    const incoming = asString(editConfig[editKey])
    const existing = editConfig[secretKey]
    if (incoming !== "") {
      payload[secretKey] = incoming
      continue
    }
    if (secretKey in editConfig) {
      payload[secretKey] = existing
    }
  }

  for (const [key, value] of Object.entries(editConfig)) {
    if (key.startsWith("_")) continue
    if (key === "enabled") continue
    if (key.endsWith("_set")) continue

    if (key in SECRET_FIELD_MAP) {
      continue
    }

    payload[key] = value
  }

  if (channel.name === "whatsapp_native") {
    payload.use_native = true
  }
  if (channel.name === "whatsapp") {
    payload.use_native = false
  }

  return payload
}

export { SECRET_FIELD_MAP }
