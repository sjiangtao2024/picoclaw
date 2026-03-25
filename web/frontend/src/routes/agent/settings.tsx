import { createFileRoute } from "@tanstack/react-router"

import { AgentSettingsPage } from "@/components/agent-settings/agent-settings-page"

export const Route = createFileRoute("/agent/settings")({
  component: AgentSettingsRoute,
})

function AgentSettingsRoute() {
  return <AgentSettingsPage />
}
