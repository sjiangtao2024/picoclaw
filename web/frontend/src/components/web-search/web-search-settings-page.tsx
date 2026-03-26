import { IconDeviceFloppy } from "@tabler/icons-react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { getAppConfig, patchAppConfig } from "@/api/channels"
import { PageHeader } from "@/components/page-header"
import { maskedSecretPlaceholder } from "@/components/secret-placeholder"
import { Field, KeyInput, SwitchCardField } from "@/components/shared-form"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"

import {
  buildWebSearchEditConfig,
  buildWebSearchSavePayload,
  hasStoredAPIKey,
  type WebSearchEditConfig,
} from "./web-search-config-utils"

function parseRootWebConfig(config: unknown) {
  const root =
    config && typeof config === "object" && !Array.isArray(config)
      ? (config as Record<string, unknown>)
      : {}
  const tools =
    root.tools && typeof root.tools === "object" && !Array.isArray(root.tools)
      ? (root.tools as Record<string, unknown>)
      : {}
  const web =
    tools.web && typeof tools.web === "object" && !Array.isArray(tools.web)
      ? tools.web
      : {}

  return web
}

function validateMaxResults(value: string): number | undefined {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number.parseInt(trimmed, 10)
  if (!Number.isFinite(parsed) || parsed < 1) {
    throw new Error("Baidu Search max results must be an integer greater than 0.")
  }
  return parsed
}

export function WebSearchSettingsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<WebSearchEditConfig | null>(null)
  const [baseline, setBaseline] = useState<WebSearchEditConfig | null>(null)
  const [saving, setSaving] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ["config"],
    queryFn: getAppConfig,
  })

  useEffect(() => {
    if (!data) return
    const parsed = buildWebSearchEditConfig(parseRootWebConfig(data))
    setForm(parsed)
    setBaseline(parsed)
  }, [data])

  const isDirty = useMemo(() => {
    if (!form || !baseline) return false
    return JSON.stringify(form) !== JSON.stringify(baseline)
  }, [form, baseline])

  const storedAPIKey = hasStoredAPIKey(form ?? {})
  const typedAPIKey = form?._api_key ?? ""
  const existingAPIKey = form?.api_key ?? ""

  const updateField = <K extends keyof WebSearchEditConfig>(
    key: K,
    value: WebSearchEditConfig[K],
  ) => {
    setForm((prev) => (prev ? { ...prev, [key]: value } : prev))
  }

  const handleReset = () => {
    if (!baseline) return
    setForm(baseline)
    toast.info(t("pages.agent.web_search.reset_success"))
  }

  const handleSave = async () => {
    if (!form) return

    try {
      setSaving(true)
      validateMaxResults(form.max_results)

      await patchAppConfig({
        tools: {
          web: buildWebSearchSavePayload(form),
        },
      })

      const nextTypedAPIKey = typedAPIKey.trim()
      const nextBaseline = {
        ...form,
        api_key: nextTypedAPIKey !== "" ? nextTypedAPIKey : existingAPIKey,
        _api_key: "",
        api_key_set: nextTypedAPIKey !== "" ? true : form.api_key_set,
      }
      setForm(nextBaseline)
      setBaseline(nextBaseline)
      await queryClient.invalidateQueries({ queryKey: ["config"] })
      toast.success(t("pages.agent.web_search.save_success"))
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : t("pages.agent.web_search.save_error"),
      )
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex h-full flex-col">
      <PageHeader
        title={t("navigation.web_search")}
        children={
          <>
            {isDirty ? (
              <Button variant="outline" onClick={handleReset} disabled={saving}>
                {t("common.reset")}
              </Button>
            ) : null}
            <Button onClick={handleSave} disabled={!isDirty || saving || !form}>
              <IconDeviceFloppy className="size-4" />
              {saving ? t("common.saving") : t("common.save")}
            </Button>
          </>
        }
      />

      <div className="flex-1 overflow-auto px-6 py-3">
        <div className="w-full max-w-4xl space-y-6">
          {isLoading ? (
            <div className="text-muted-foreground py-6 text-sm">
              {t("labels.loading")}
            </div>
          ) : error ? (
            <div className="text-destructive py-6 text-sm">
              {t("pages.agent.web_search.load_error")}
            </div>
          ) : form ? (
            <Card>
              <CardHeader>
                <CardTitle>{t("pages.agent.web_search.baidu.title")}</CardTitle>
                <CardDescription>
                  {t("pages.agent.web_search.baidu.description")}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-1">
                <SwitchCardField
                  label={t("pages.agent.web_search.fields.prefer_native.label")}
                  hint={t("pages.agent.web_search.fields.prefer_native.hint")}
                  checked={form.prefer_native}
                  onCheckedChange={(checked) =>
                    updateField("prefer_native", checked)
                  }
                  layout="setting-row"
                />

                <SwitchCardField
                  label={t("pages.agent.web_search.fields.duckduckgo_enabled.label")}
                  hint={t("pages.agent.web_search.fields.duckduckgo_enabled.hint")}
                  checked={form.duckduckgo_enabled}
                  onCheckedChange={(checked) =>
                    updateField("duckduckgo_enabled", checked)
                  }
                  layout="setting-row"
                />

                <SwitchCardField
                  label={t("pages.agent.web_search.fields.baidu_enabled.label")}
                  hint={t("pages.agent.web_search.fields.baidu_enabled.hint")}
                  checked={form.baidu_enabled}
                  onCheckedChange={(checked) => updateField("baidu_enabled", checked)}
                  layout="setting-row"
                />

                <Field
                  label={t("pages.agent.web_search.fields.max_results.label")}
                  hint={t("pages.agent.web_search.fields.max_results.hint")}
                  layout="setting-row"
                >
                  <Input
                    value={form.max_results}
                    onChange={(e) => updateField("max_results", e.target.value)}
                    inputMode="numeric"
                    placeholder="10"
                  />
                </Field>

                <Field
                  label={t("pages.agent.web_search.fields.api_key.label")}
                  hint={
                    storedAPIKey
                      ? t("pages.agent.web_search.fields.api_key.hint_set")
                      : t("pages.agent.web_search.fields.api_key.hint_empty")
                  }
                  layout="setting-row"
                >
                  <KeyInput
                    value={typedAPIKey}
                    onChange={(value) => updateField("_api_key", value)}
                    placeholder={
                      storedAPIKey
                        ? maskedSecretPlaceholder(
                            existingAPIKey,
                            t("pages.agent.web_search.fields.api_key.placeholder_set"),
                          )
                        : t("pages.agent.web_search.fields.api_key.placeholder_empty")
                    }
                  />
                </Field>
              </CardContent>
            </Card>
          ) : null}
        </div>
      </div>
    </div>
  )
}
