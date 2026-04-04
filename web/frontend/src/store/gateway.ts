import { atom, getDefaultStore } from "jotai"

import { type GatewayStatusResponse, getGatewayStatus } from "@/api/gateway"
import { getGatewayPollIntervalMs } from "./gateway-polling"
import {
  applyGatewayStatusPatch,
  type GatewayState,
  type GatewayStoreState,
} from "./gateway-store-state"

type GatewayStorePatch = Partial<GatewayStoreState>

const DEFAULT_GATEWAY_STATE: GatewayStoreState = {
  status: "unknown",
  canStart: true,
  restartRequired: false,
  pid: undefined,
}

const GATEWAY_STOPPING_TIMEOUT_MS = 5000

interface RefreshGatewayStateOptions {
  force?: boolean
}

// Global atom for gateway state
export const gatewayAtom = atom<GatewayStoreState>(DEFAULT_GATEWAY_STATE)

let gatewayPollingSubscribers = 0
let gatewayPollingTimer: ReturnType<typeof setTimeout> | null = null
let gatewayPollingRequest: Promise<void> | null = null
let gatewayStoppingTimer: ReturnType<typeof setTimeout> | null = null

function clearGatewayStoppingTimeout() {
  if (gatewayStoppingTimer !== null) {
    clearTimeout(gatewayStoppingTimer)
    gatewayStoppingTimer = null
  }
}

function normalizeGatewayStoreState(
  prev: GatewayStoreState,
  patch: GatewayStorePatch,
) {
  const next = { ...prev, ...patch }

  if (
    next.status === prev.status &&
    next.canStart === prev.canStart &&
    next.restartRequired === prev.restartRequired
  ) {
    return prev
  }

  return next
}

export function updateGatewayStore(
  patch:
    | GatewayStorePatch
    | ((prev: GatewayStoreState) => GatewayStorePatch | GatewayStoreState),
) {
  const store = getDefaultStore()
  store.set(gatewayAtom, (prev) => {
    const nextPatch = typeof patch === "function" ? patch(prev) : patch
    return normalizeGatewayStoreState(prev, nextPatch)
  })
  const nextState = store.get(gatewayAtom)
  if (nextState?.status !== "stopping") {
    clearGatewayStoppingTimeout()
  }
}

export function beginGatewayStoppingTransition() {
  clearGatewayStoppingTimeout()
  updateGatewayStore({
    status: "stopping",
    canStart: false,
    restartRequired: false,
  })
  gatewayStoppingTimer = setTimeout(() => {
    gatewayStoppingTimer = null
    updateGatewayStore((prev) =>
      prev.status === "stopping" ? { status: "running" } : prev,
    )
    void refreshGatewayState({ force: true })
  }, GATEWAY_STOPPING_TIMEOUT_MS)
}

export function cancelGatewayStoppingTransition() {
  clearGatewayStoppingTimeout()
  updateGatewayStore((prev) =>
    prev.status === "stopping" ? { status: "running" } : prev,
  )
}

export function applyGatewayStatusToStore(
  data: Partial<
    Pick<
      GatewayStatusResponse,
      | "gateway_status"
      | "gateway_start_allowed"
      | "gateway_restart_required"
      | "pid"
    >
  >,
) {
  updateGatewayStore((prev) => applyGatewayStatusPatch(prev, data))
}

function nextGatewayPollInterval() {
  const status = getDefaultStore().get(gatewayAtom).status
  return getGatewayPollIntervalMs(status)
}

function scheduleGatewayPoll(delay = nextGatewayPollInterval()) {
  if (gatewayPollingSubscribers === 0) {
    return
  }

  if (gatewayPollingTimer !== null) {
    clearTimeout(gatewayPollingTimer)
  }

  gatewayPollingTimer = setTimeout(() => {
    gatewayPollingTimer = null
    void refreshGatewayState()
  }, delay)
}

export async function refreshGatewayState(
  options: RefreshGatewayStateOptions = {},
) {
  if (gatewayPollingRequest) {
    await gatewayPollingRequest
    if (options.force) {
      return refreshGatewayState()
    }
    return
  }

  gatewayPollingRequest = (async () => {
    try {
      const status = await getGatewayStatus()
      applyGatewayStatusToStore(status)
    } catch {
      // Preserve the last known state when a poll fails.
    } finally {
      gatewayPollingRequest = null
      scheduleGatewayPoll()
    }
  })()

  try {
    await gatewayPollingRequest
  } finally {
    if (gatewayPollingSubscribers === 0 && gatewayPollingTimer !== null) {
      clearTimeout(gatewayPollingTimer)
      gatewayPollingTimer = null
    }
  }
}

export function subscribeGatewayPolling() {
  gatewayPollingSubscribers += 1
  if (gatewayPollingSubscribers === 1) {
    void refreshGatewayState()
  }

  return () => {
    gatewayPollingSubscribers = Math.max(0, gatewayPollingSubscribers - 1)
    if (gatewayPollingSubscribers === 0 && gatewayPollingTimer !== null) {
      clearTimeout(gatewayPollingTimer)
      gatewayPollingTimer = null
    }
  }
}
