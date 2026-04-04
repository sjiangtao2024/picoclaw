export type GatewayConnectionSnapshot = {
  status: "running" | "starting" | "restarting" | "stopping" | "stopped" | "error" | "unknown"
  pid?: number
}

export function shouldReconnectForGatewayUpdate(
  previous: GatewayConnectionSnapshot | null,
  next: GatewayConnectionSnapshot,
) {
  return (
    previous?.status === "running" &&
    next.status === "running" &&
    previous.pid !== undefined &&
    next.pid !== undefined &&
    previous.pid !== next.pid
  )
}
