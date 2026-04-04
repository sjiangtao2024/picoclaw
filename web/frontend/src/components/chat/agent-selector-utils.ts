import type { AppConfig } from "@/api/channels"

export interface AgentOption {
  id: string
  name: string
  isDefault: boolean
}

type AgentRecord = Record<string, unknown>

function asRecord(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>
  }
  return {}
}

function asAgentRecord(value: unknown): AgentRecord | null {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as AgentRecord
  }
  return null
}

function getAgentList(config: AppConfig | undefined): AgentRecord[] {
  const agents = asRecord(config?.agents)
  const list = agents.list
  if (!Array.isArray(list)) {
    return []
  }
  return list
    .map((item) => asAgentRecord(item))
    .filter((item): item is AgentRecord => item !== null)
}

export function listAvailableAgents(config: AppConfig | undefined): AgentOption[] {
  return getAgentList(config)
    .map((agent) => {
      const id = typeof agent.id === "string" ? agent.id.trim() : ""
      if (!id) {
        return null
      }
      const name = typeof agent.name === "string" && agent.name.trim() ? agent.name : id
      return {
        id,
        name,
        isDefault: agent.default === true,
      }
    })
    .filter((agent): agent is AgentOption => agent !== null)
}

export function getCurrentDefaultAgentId(
  config: AppConfig | undefined,
): string | null {
  const agents = listAvailableAgents(config)
  if (agents.length === 0) {
    return null
  }
  return agents.find((agent) => agent.isDefault)?.id ?? agents[0].id
}

export function buildDefaultAgentPatch(
  config: AppConfig | undefined,
  targetAgentId: string,
): Record<string, unknown> {
  const agents = getAgentList(config)
  if (!agents.some((agent) => agent.id === targetAgentId)) {
    throw new Error(`Unknown agent: ${targetAgentId}`)
  }

  return {
    agents: {
      list: agents.map((agent) => ({
        ...agent,
        default: agent.id === targetAgentId,
      })),
    },
  }
}
