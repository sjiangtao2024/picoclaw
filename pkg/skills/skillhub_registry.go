package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/utils"
)

const (
	defaultSkillHubSearchURL                  = "https://api.skillhub.tencent.com/api/v1/search"
	defaultSkillHubPrimaryDownloadURLTemplate = "https://api.skillhub.tencent.com/api/v1/download?slug={slug}"
	defaultSkillHubDownloadURLTemplate        = "https://skillhub-1388575217.cos.ap-guangzhou.myqcloud.com/skills/{slug}.zip"
)

// SkillHubRegistry implements SkillRegistry for Tencent SkillHub.
type SkillHubRegistry struct {
	searchURL                  string
	primaryDownloadURLTemplate string
	downloadURLTemplate        string
	maxZipSize                 int
	maxResponseSize            int
	client                     *http.Client
}

type skillhubSearchResponse struct {
	Results []skillhubSearchResult `json:"results"`
}

type skillhubSearchResult struct {
	Score       float64 `json:"score"`
	Slug        string  `json:"slug"`
	DisplayName string  `json:"displayName"`
	Summary     string  `json:"summary"`
	Version     string  `json:"version"`
}

// NewSkillHubRegistry creates a new SkillHub registry client from config.
func NewSkillHubRegistry(cfg SkillHubConfig) *SkillHubRegistry {
	timeout := defaultClawHubTimeout
	if cfg.Timeout > 0 {
		timeout = time.Duration(cfg.Timeout) * time.Second
	}

	maxZip := defaultMaxZipSize
	if cfg.MaxZipSize > 0 {
		maxZip = cfg.MaxZipSize
	}

	maxResp := defaultMaxResponseSize
	if cfg.MaxResponseSize > 0 {
		maxResp = cfg.MaxResponseSize
	}

	proxyFunc := http.ProxyFromEnvironment
	if cfg.UseProxy != nil && !*cfg.UseProxy {
		proxyFunc = nil
	} else if proxyURL, err := parseProxyURL(cfg.Proxy); err == nil && proxyURL != nil {
		proxyFunc = http.ProxyURL(proxyURL)
	}

	searchURL := strings.TrimSpace(cfg.SearchURL)
	if searchURL == "" {
		searchURL = defaultSkillHubSearchURL
	}

	primaryDownloadURLTemplate := strings.TrimSpace(cfg.PrimaryDownloadURLTemplate)
	if primaryDownloadURLTemplate == "" {
		primaryDownloadURLTemplate = defaultSkillHubPrimaryDownloadURLTemplate
	}

	downloadURLTemplate := strings.TrimSpace(cfg.DownloadURLTemplate)
	if downloadURLTemplate == "" {
		downloadURLTemplate = defaultSkillHubDownloadURLTemplate
	}

	return &SkillHubRegistry{
		searchURL:                  searchURL,
		primaryDownloadURLTemplate: primaryDownloadURLTemplate,
		downloadURLTemplate:        downloadURLTemplate,
		maxZipSize:                 maxZip,
		maxResponseSize:            maxResp,
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				Proxy:               proxyFunc,
				MaxIdleConns:        5,
				IdleConnTimeout:     30 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

func parseProxyURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	return url.Parse(trimmed)
}

func (s *SkillHubRegistry) Name() string {
	return "skillhub"
}

func (s *SkillHubRegistry) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	u, err := url.Parse(s.searchURL)
	if err != nil {
		return nil, fmt.Errorf("invalid search URL: %w", err)
	}

	q := u.Query()
	q.Set("q", query)
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}
	u.RawQuery = q.Encode()

	body, err := s.doGet(ctx, u.String(), "application/json")
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	var resp skillhubSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	results := make([]SearchResult, 0, len(resp.Results))
	for _, r := range resp.Results {
		slug := strings.TrimSpace(r.Slug)
		if slug == "" {
			continue
		}

		displayName := strings.TrimSpace(r.DisplayName)
		if displayName == "" {
			displayName = slug
		}

		results = append(results, SearchResult{
			Score:        r.Score,
			Slug:         slug,
			DisplayName:  displayName,
			Summary:      strings.TrimSpace(r.Summary),
			Version:      strings.TrimSpace(r.Version),
			RegistryName: s.Name(),
		})
	}

	return results, nil
}

func (s *SkillHubRegistry) GetSkillMeta(ctx context.Context, slug string) (*SkillMeta, error) {
	if err := utils.ValidateSkillIdentifier(slug); err != nil {
		return nil, fmt.Errorf("invalid slug %q: error: %s", slug, err.Error())
	}

	results, err := s.Search(ctx, slug, 10)
	if err != nil {
		return nil, err
	}

	for _, result := range results {
		if result.Slug != slug {
			continue
		}
		return &SkillMeta{
			Slug:          result.Slug,
			DisplayName:   result.DisplayName,
			Summary:       result.Summary,
			LatestVersion: result.Version,
			RegistryName:  s.Name(),
		}, nil
	}

	return nil, fmt.Errorf("skill %q not found", slug)
}

func (s *SkillHubRegistry) DownloadAndInstall(ctx context.Context, slug, version, targetDir string) (*InstallResult, error) {
	if err := utils.ValidateSkillIdentifier(slug); err != nil {
		return nil, fmt.Errorf("invalid slug %q: error: %s", slug, err.Error())
	}

	result := &InstallResult{}
	meta, err := s.GetSkillMeta(ctx, slug)
	if err == nil && meta != nil {
		result.Summary = meta.Summary
		result.Version = meta.LatestVersion
	}
	if strings.TrimSpace(version) != "" {
		result.Version = version
	}
	if result.Version == "" {
		result.Version = "latest"
	}

	downloadURLs := make([]string, 0, 2)
	if u := fillSkillHubSlugTemplate(s.primaryDownloadURLTemplate, slug); u != "" {
		downloadURLs = append(downloadURLs, u)
	}
	if u := fillSkillHubSlugTemplate(s.downloadURLTemplate, slug); u != "" {
		downloadURLs = append(downloadURLs, u)
	}
	if len(downloadURLs) == 0 {
		return nil, fmt.Errorf("no download URL configured for skillhub")
	}

	tmpPath, err := s.downloadToTempFileWithFallback(ctx, downloadURLs)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpPath)

	if err := utils.ExtractZipFile(tmpPath, targetDir); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *SkillHubRegistry) doGet(ctx context.Context, urlStr, accept string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)

	resp, err := utils.DoRequestWithRetry(s.client, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(s.maxResponseSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

func (s *SkillHubRegistry) downloadToTempFileWithFallback(ctx context.Context, urlStrs []string) (string, error) {
	var lastErr error
	for _, urlStr := range urlStrs {
		tmpPath, err := s.downloadToTempFileWithRetry(ctx, urlStr)
		if err == nil {
			return tmpPath, nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no download URLs available")
	}
	return "", lastErr
}

func (s *SkillHubRegistry) downloadToTempFileWithRetry(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/zip,application/octet-stream,*/*")

	resp, err := utils.DoRequestWithRetry(s.client, req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody := make([]byte, 512)
		n, _ := io.ReadFull(resp.Body, errBody)
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(errBody[:n]))
	}

	tmpFile, err := os.CreateTemp("", "picoclaw-skillhub-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	src := io.LimitReader(resp.Body, int64(s.maxZipSize)+1)
	written, err := io.Copy(tmpFile, src)
	if err != nil {
		cleanup()
		return "", fmt.Errorf("download write failed: %w", err)
	}

	if written > int64(s.maxZipSize) {
		cleanup()
		return "", fmt.Errorf("download too large: %d bytes (max %d)", written, s.maxZipSize)
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return tmpPath, nil
}

func fillSkillHubSlugTemplate(template, slug string) string {
	raw := strings.TrimSpace(template)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "{slug}") {
		return raw
	}
	return strings.ReplaceAll(raw, "{slug}", url.QueryEscape(slug))
}
