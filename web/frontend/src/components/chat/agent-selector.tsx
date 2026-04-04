import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { getAppConfig, patchAppConfig } from "@/api/channels"
import {
  buildDefaultAgentPatch,
  getCurrentDefaultAgentId,
  listAvailableAgents,
} from "@/components/chat/agent-selector-utils"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

export function AgentSelector() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedAgentId, setSelectedAgentId] = useState("")
  const [saving, setSaving] = useState(false)

  const { data: configData } = useQuery({
    queryKey: ["config"],
    queryFn: getAppConfig,
  })

  const agents = listAvailableAgents(configData)

  useEffect(() => {
    const currentDefaultAgentId = getCurrentDefaultAgentId(configData)
    setSelectedAgentId(currentDefaultAgentId ?? "")
  }, [configData])

  if (agents.length === 0) {
    return null
  }

  const handleValueChange = async (nextAgentId: string) => {
    if (!configData || nextAgentId === selectedAgentId) {
      return
    }

    const previousAgentId = selectedAgentId
    setSelectedAgentId(nextAgentId)
    setSaving(true)

    try {
      await patchAppConfig(buildDefaultAgentPatch(configData, nextAgentId))
      await queryClient.invalidateQueries({ queryKey: ["config"] })

      const nextAgent = agents.find((agent) => agent.id === nextAgentId)
      toast.success(
        t("chat.defaultAgentChanged", {
          name: nextAgent?.name ?? nextAgentId,
        }),
      )
    } catch (error) {
      setSelectedAgentId(previousAgentId)
      toast.error(
        error instanceof Error
          ? error.message
          : t("chat.defaultAgentChangeFailed"),
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <Select
      value={selectedAgentId}
      onValueChange={handleValueChange}
      disabled={saving || agents.length < 2}
    >
      <SelectTrigger
        size="sm"
        className="text-muted-foreground hover:text-foreground focus-visible:border-input h-8 max-w-[160px] min-w-[88px] bg-transparent shadow-none focus-visible:ring-0 sm:max-w-[220px]"
        aria-label={t("chat.agent")}
      >
        <SelectValue placeholder={t("chat.agent")} />
      </SelectTrigger>
      <SelectContent position="popper" align="start">
        {agents.map((agent) => (
          <SelectItem key={agent.id} value={agent.id}>
            {agent.name}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
