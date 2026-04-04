package skills

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSkillHubRegistry(searchURL, primaryDownloadURLTemplate, downloadURLTemplate string) *SkillHubRegistry {
	return NewSkillHubRegistry(SkillHubConfig{
		Enabled:                    true,
		SearchURL:                  searchURL,
		PrimaryDownloadURLTemplate: primaryDownloadURLTemplate,
		DownloadURLTemplate:        downloadURLTemplate,
	})
}

func TestSkillHubRegistrySearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search", r.URL.Path)
		assert.Equal(t, "github", r.URL.Query().Get("q"))
		assert.Equal(t, "5", r.URL.Query().Get("limit"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"slug":        "github",
					"displayName": "Github",
					"summary":     "GitHub skill",
					"version":     "1.0.0",
					"score":       0.91,
				},
			},
		})
	}))
	defer srv.Close()

	reg := newTestSkillHubRegistry(srv.URL+"/api/v1/search", srv.URL+"/api/v1/download?slug={slug}", "")
	results, err := reg.Search(context.Background(), "github", 5)

	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "github", results[0].Slug)
	assert.Equal(t, "Github", results[0].DisplayName)
	assert.Equal(t, "GitHub skill", results[0].Summary)
	assert.Equal(t, "1.0.0", results[0].Version)
	assert.Equal(t, "skillhub", results[0].RegistryName)
	assert.InDelta(t, 0.91, results[0].Score, 0.001)
}

func TestSkillHubRegistryGetSkillMetaUsesExactSlugMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/search", r.URL.Path)
		assert.Equal(t, "github", r.URL.Query().Get("q"))

		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"slug":        "github-api",
					"displayName": "GitHub API",
					"summary":     "Other skill",
					"version":     "1.0.3",
					"score":       0.99,
				},
				{
					"slug":        "github",
					"displayName": "Github",
					"summary":     "Exact skill",
					"version":     "1.0.0",
					"score":       0.91,
				},
			},
		})
	}))
	defer srv.Close()

	reg := newTestSkillHubRegistry(srv.URL+"/api/v1/search", srv.URL+"/api/v1/download?slug={slug}", "")
	meta, err := reg.GetSkillMeta(context.Background(), "github")

	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "github", meta.Slug)
	assert.Equal(t, "Github", meta.DisplayName)
	assert.Equal(t, "Exact skill", meta.Summary)
	assert.Equal(t, "1.0.0", meta.LatestVersion)
	assert.Equal(t, "skillhub", meta.RegistryName)
}

func TestSkillHubRegistryDownloadAndInstallFallsBackToMirror(t *testing.T) {
	zipBuf := createTestZip(t, map[string]string{
		"SKILL.md":  "---\nname: github\ndescription: GitHub skill\n---\nUse GitHub skill",
		"README.md": "# GitHub Skill\n",
	})

	searchSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"results": []map[string]any{
				{
					"slug":        "github",
					"displayName": "Github",
					"summary":     "GitHub skill",
					"version":     "1.0.0",
					"score":       0.91,
				},
			},
		})
	}))
	defer searchSrv.Close()

	primaryAttempts := 0
	downloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/download":
			primaryAttempts++
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("rate limited"))
		case "/skills/github.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(zipBuf)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer downloadSrv.Close()

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "github")

	reg := newTestSkillHubRegistry(
		searchSrv.URL+"/api/v1/search",
		downloadSrv.URL+"/api/v1/download?slug={slug}",
		downloadSrv.URL+"/skills/{slug}.zip",
	)
	result, err := reg.DownloadAndInstall(context.Background(), "github", "", targetDir)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "GitHub skill", result.Summary)
	assert.GreaterOrEqual(t, primaryAttempts, 1)

	skillContent, err := os.ReadFile(filepath.Join(targetDir, "SKILL.md"))
	require.NoError(t, err)
	assert.Contains(t, string(skillContent), "Use GitHub skill")
}

func TestNewRegistryManagerFromConfigAddsSkillHub(t *testing.T) {
	rm := NewRegistryManagerFromConfig(RegistryConfig{
		ClawHub: ClawHubConfig{Enabled: true},
		SkillHub: SkillHubConfig{
			Enabled:                    true,
			SearchURL:                  "https://lightmake.site/api/v1/search",
			PrimaryDownloadURLTemplate: "https://lightmake.site/api/v1/download?slug={slug}",
		},
	})

	require.NotNil(t, rm.GetRegistry("clawhub"))
	require.NotNil(t, rm.GetRegistry("skillhub"))
}
