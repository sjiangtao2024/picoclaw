package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsRegistriesConfig_ParsesSkillHub(t *testing.T) {
	cfg := DefaultConfig()

	err := json.Unmarshal([]byte(`{
		"tools": {
			"skills": {
				"registries": {
					"skillhub": {
						"enabled": true,
						"search_url": "https://lightmake.site/api/v1/search",
						"primary_download_url_template": "https://lightmake.site/api/v1/download?slug={slug}",
						"download_url_template": "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip",
						"use_proxy": false,
						"proxy": "http://127.0.0.1:7890"
					}
				}
			}
		}
	}`), cfg)

	require.NoError(t, err)
	require.True(t, cfg.Tools.Skills.Registries.SkillHub.Enabled)
	assert.Equal(t, "https://lightmake.site/api/v1/search", cfg.Tools.Skills.Registries.SkillHub.SearchURL)
	assert.Equal(t, "https://lightmake.site/api/v1/download?slug={slug}", cfg.Tools.Skills.Registries.SkillHub.PrimaryDownloadURLTemplate)
	assert.Equal(t, "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip", cfg.Tools.Skills.Registries.SkillHub.DownloadURLTemplate)
	require.NotNil(t, cfg.Tools.Skills.Registries.SkillHub.UseProxy)
	assert.False(t, *cfg.Tools.Skills.Registries.SkillHub.UseProxy)
	assert.Equal(t, "http://127.0.0.1:7890", cfg.Tools.Skills.Registries.SkillHub.Proxy)
}

func TestDefaultConfigPrefersSkillHubOverClawHub(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Tools.Skills.Registries.ClawHub.Enabled)
	assert.True(t, cfg.Tools.Skills.Registries.SkillHub.Enabled)
	assert.Equal(t, "https://api.skillhub.tencent.com/api/v1/search", cfg.Tools.Skills.Registries.SkillHub.SearchURL)
	assert.Equal(t, "https://api.skillhub.tencent.com/api/v1/download?slug={slug}", cfg.Tools.Skills.Registries.SkillHub.PrimaryDownloadURLTemplate)
}
