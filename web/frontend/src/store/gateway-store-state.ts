export type GatewayState =
  | "running"
  | "starting"
  | "restarting"
  | "stopping"
  | "stopped"
  | "error"
  | "unknown"

export interface GatewayStoreState {
  status: GatewayState
  canStart: boolean
  restartRequired: boolean
  pid?: number
}

export interface GatewayStatusPatch {
  gateway_status?: GatewayState
  gateway_start_allowed?: boolean
  gateway_restart_required?: boolean
  pid?: number
}

export function applyGatewayStatusPatch(
  prev: GatewayStoreState,
  data: GatewayStatusPatch,
): GatewayStoreState {
  return {
    status:
      prev.status === "stopping" && data.gateway_status === "running"
        ? "stopping"
        : (data.gateway_status ?? prev.status),
    canStart:
      prev.status === "stopping" && data.gateway_status === "running"
        ? false
        : (data.gateway_start_allowed ?? prev.canStart),
    restartRequired:
      prev.status === "stopping" && data.gateway_status === "running"
        ? false
        : (data.gateway_restart_required ?? prev.restartRequired),
    pid: data.pid ?? prev.pid,
  }
}
