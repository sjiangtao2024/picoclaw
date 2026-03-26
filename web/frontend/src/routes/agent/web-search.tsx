import { createFileRoute } from "@tanstack/react-router"

import { WebSearchSettingsPage } from "@/components/web-search/web-search-settings-page"

export const Route = createFileRoute("/agent/web-search")({
  component: AgentWebSearchRoute,
})

function AgentWebSearchRoute() {
  return <WebSearchSettingsPage />
}
