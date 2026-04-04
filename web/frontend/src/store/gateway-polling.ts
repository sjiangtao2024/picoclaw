export type GatewayPollingState =
  | "running"
  | "starting"
  | "restarting"
  | "stopping"
  | "stopped"
  | "error"
  | "unknown"

export const GATEWAY_POLL_INTERVAL_MS = 60_000
export const GATEWAY_TRANSIENT_POLL_INTERVAL_MS = 3_000

export function getGatewayPollIntervalMs(status: GatewayPollingState) {
  if (
    status === "starting" ||
    status === "restarting" ||
    status === "stopping"
  ) {
    return GATEWAY_TRANSIENT_POLL_INTERVAL_MS
  }
  return GATEWAY_POLL_INTERVAL_MS
}
