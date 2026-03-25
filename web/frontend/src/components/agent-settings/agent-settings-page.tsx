import { IconDeviceFloppy } from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { getAppConfig, patchAppConfig } from "@/api/channels"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Field,
  FieldContent,
  FieldDescription,
  FieldGroup,
  FieldLabel,
} from "@/components/ui/field"
import { Input } from "@/components/ui/input"

type JsonRecord = Record<string, unknown>

interface AgentSettingsItem {
  id: string
  name: string
  isDefault: boolean
  maxTokens: string
  temperature: string
  maxToolIterations: string
}

interface AgentDefaultsSnapshot {
  maxTokens: string
  temperature: string
  maxToolIterations: string
}

function asRecord(value: unknown): JsonRecord {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as JsonRecord
  }
  return {}
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

function asNumberString(value: unknown): string {
  if (typeof value === "number" && Number.isFinite(value)) {
    return String(value)
  }
  return ""
}

function parseAgentDefaults(config: unknown): AgentDefaultsSnapshot {
  const root = asRecord(config)
  const agents = asRecord(root.agents)
  const defaults = asRecord(agents.defaults)

  return {
    maxTokens: asNumberString(defaults.max_tokens),
    temperature: asNumberString(defaults.temperature),
    maxToolIterations: asNumberString(defaults.max_tool_iterations),
  }
}

function parseAgentSettings(config: unknown): AgentSettingsItem[] {
  const root = asRecord(config)
  const agents = asRecord(root.agents)
  const list = Array.isArray(agents.list) ? agents.list : []

  return list
    .map((item) => {
      const agent = asRecord(item)
      const id = asString(agent.id).trim()
      if (!id) return null

      return {
        id,
        name: asString(agent.name).trim(),
        isDefault: agent.default === true,
        maxTokens: asNumberString(agent.max_tokens),
        temperature: asNumberString(agent.temperature),
        maxToolIterations: asNumberString(agent.max_tool_iterations),
      }
    })
    .filter((item): item is AgentSettingsItem => item !== null)
}

function parseOptionalInt(raw: string, label: string): number | undefined {
  const trimmed = raw.trim()
  if (!trimmed) return undefined

  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed < 1) {
    throw new Error(`${label} must be an integer greater than 0.`)
  }
  return parsed
}

function parseOptionalFloat(raw: string, label: string): number | undefined {
  const trimmed = raw.trim()
  if (!trimmed) return undefined

  const parsed = Number.parseFloat(trimmed)
  if (!Number.isFinite(parsed)) {
    throw new Error(`${label} must be a valid number.`)
  }
  return parsed
}

function withOptionalNumberField(
  agent: JsonRecord,
  key: "max_tokens" | "temperature" | "max_tool_iterations",
  value: number | undefined,
): JsonRecord {
  const next = { ...agent }
  if (value === undefined) {
    delete next[key]
  } else {
    next[key] = value
  }
  return next
}

export function AgentSettingsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<AgentSettingsItem[]>([])
  const [baseline, setBaseline] = useState<AgentSettingsItem[]>([])
  const [saving, setSaving] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ["config"],
    queryFn: getAppConfig,
  })

  useEffect(() => {
    if (!data) return
    const parsed = parseAgentSettings(data)
    setForm(parsed)
    setBaseline(parsed)
  }, [data])

  const defaults = useMemo(() => parseAgentDefaults(data), [data])
  const isDirty = JSON.stringify(form) !== JSON.stringify(baseline)

  const updateField = (
    agentID: string,
    key: keyof Pick<
      AgentSettingsItem,
      "maxTokens" | "temperature" | "maxToolIterations"
    >,
    value: string,
  ) => {
    setForm((prev) =>
      prev.map((item) => (item.id === agentID ? { ...item, [key]: value } : item)),
    )
  }

  const handleReset = () => {
    setForm(baseline)
    toast.info(t("pages.agent.settings.reset_success"))
  }

  const handleSave = async () => {
    if (!data) return

    try {
      setSaving(true)

      const root = asRecord(data)
      const agents = asRecord(root.agents)
      const sourceList = Array.isArray(agents.list) ? agents.list : []
      const formMap = new Map(form.map((item) => [item.id, item]))

      const nextList = sourceList.map((item) => {
        const agent = asRecord(item)
        const id = asString(agent.id).trim()
        const formItem = formMap.get(id)
        if (!formItem) return item

        const maxTokens = parseOptionalInt(
          formItem.maxTokens,
          `${id} max_tokens`,
        )
        const temperature = parseOptionalFloat(
          formItem.temperature,
          `${id} temperature`,
        )
        const maxToolIterations = parseOptionalInt(
          formItem.maxToolIterations,
          `${id} max_tool_iterations`,
        )

        let nextAgent = withOptionalNumberField(agent, "max_tokens", maxTokens)
        nextAgent = withOptionalNumberField(nextAgent, "temperature", temperature)
        nextAgent = withOptionalNumberField(
          nextAgent,
          "max_tool_iterations",
          maxToolIterations,
        )
        return nextAgent
      })

      await patchAppConfig({
        agents: {
          list: nextList,
        },
      })

      setBaseline(form)
      await queryClient.invalidateQueries({ queryKey: ["config"] })
      toast.success(t("pages.agent.settings.save_success"))
    } catch (err) {
      toast.error(
        err instanceof Error
          ? err.message
          : t("pages.agent.settings.save_error"),
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.agent_settings")}
        children={
          <>
            {isDirty ? (
              <Button variant="outline" onClick={handleReset} disabled={saving}>
                {t("common.reset")}
              </Button>
            ) : null}
            <Button onClick={handleSave} disabled={!isDirty || saving || !data}>
              <IconDeviceFloppy className="size-4" />
              {saving ? t("common.saving") : t("common.save")}
            </Button>
          </>
        }
      />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-6xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.agent.load_error")}
            </div>
          ) : (
            <section className="space-y-5">
              <p className="text-muted-foreground text-sm">
                {t("pages.agent.settings.description")}
              </p>

              <Card className="border-border/60 bg-muted/20" size="sm">
                <CardHeader>
                  <CardTitle>{t("pages.agent.settings.defaults_title")}</CardTitle>
                  <CardDescription>
                    {t("pages.agent.settings.defaults_description")}
                  </CardDescription>
                </CardHeader>
                <CardContent className="grid gap-3 md:grid-cols-3">
                  <DefaultValueCard
                    label={t("pages.agent.settings.fields.max_tokens")}
                    value={defaults.maxTokens}
                    emptyLabel={t("pages.agent.settings.unset")}
                  />
                  <DefaultValueCard
                    label={t("pages.agent.settings.fields.temperature")}
                    value={defaults.temperature}
                    emptyLabel={t("pages.agent.settings.unset")}
                  />
                  <DefaultValueCard
                    label={t("pages.agent.settings.fields.max_tool_iterations")}
                    value={defaults.maxToolIterations}
                    emptyLabel={t("pages.agent.settings.unset")}
                  />
                </CardContent>
              </Card>

              {form.length ? (
                <div className="grid gap-4 xl:grid-cols-2">
                  {form.map((agent) => (
                    <Card key={agent.id} className="border-border/60 gap-4" size="sm">
                      <CardHeader>
                        <div className="flex items-start justify-between gap-3">
                          <div>
                            <CardTitle className="font-semibold">
                              {agent.name || agent.id}
                            </CardTitle>
                            <CardDescription className="mt-2">
                              {t("pages.agent.settings.agent_id", { id: agent.id })}
                            </CardDescription>
                          </div>
                          {agent.isDefault ? (
                            <span className="rounded-md bg-emerald-100 px-2 py-1 text-[11px] font-semibold text-emerald-700">
                              {t("pages.agent.settings.default_badge")}
                            </span>
                          ) : null}
                        </div>
                      </CardHeader>
                      <CardContent>
                        <FieldGroup>
                          <Field>
                            <FieldLabel>
                              {t("pages.agent.settings.fields.max_tokens")}
                            </FieldLabel>
                            <FieldContent>
                              <Input
                                inputMode="numeric"
                                placeholder={defaults.maxTokens || t("pages.agent.settings.inherit_placeholder")}
                                value={agent.maxTokens}
                                onChange={(e) =>
                                  updateField(agent.id, "maxTokens", e.target.value)
                                }
                              />
                              <FieldDescription>
                                {t("pages.agent.settings.inherit_hint")}
                              </FieldDescription>
                            </FieldContent>
                          </Field>

                          <Field>
                            <FieldLabel>
                              {t("pages.agent.settings.fields.temperature")}
                            </FieldLabel>
                            <FieldContent>
                              <Input
                                inputMode="decimal"
                                placeholder={defaults.temperature || t("pages.agent.settings.inherit_placeholder")}
                                value={agent.temperature}
                                onChange={(e) =>
                                  updateField(agent.id, "temperature", e.target.value)
                                }
                              />
                              <FieldDescription>
                                {t("pages.agent.settings.temperature_hint")}
                              </FieldDescription>
                            </FieldContent>
                          </Field>

                          <Field>
                            <FieldLabel>
                              {t("pages.agent.settings.fields.max_tool_iterations")}
                            </FieldLabel>
                            <FieldContent>
                              <Input
                                inputMode="numeric"
                                placeholder={
                                  defaults.maxToolIterations ||
                                  t("pages.agent.settings.inherit_placeholder")
                                }
                                value={agent.maxToolIterations}
                                onChange={(e) =>
                                  updateField(
                                    agent.id,
                                    "maxToolIterations",
                                    e.target.value,
                                  )
                                }
                              />
                              <FieldDescription>
                                {t("pages.agent.settings.inherit_hint")}
                              </FieldDescription>
                            </FieldContent>
                          </Field>
                        </FieldGroup>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <Card className="border-dashed">
                  <CardContent className="text-muted-foreground py-10 text-center text-sm">
                    {t("pages.agent.settings.empty")}
                  </CardContent>
                </Card>
              )}
            </section>
          )}
        </div>
      </div>
    </div>
  )
}

function DefaultValueCard({
  label,
  value,
  emptyLabel,
}: {
  label: string
  value: string
  emptyLabel: string
}) {
  return (
    <div className="bg-background rounded-lg border px-4 py-3">
      <div className="text-muted-foreground text-xs tracking-[0.12em] uppercase">
        {label}
      </div>
      <div className="mt-2 text-sm font-medium">{value || emptyLabel}</div>
    </div>
  )
}
