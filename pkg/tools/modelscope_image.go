package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ModelScopeImageToolOptions struct {
	Enabled        bool
	BaseURL        string
	TimeoutSeconds int
	DefaultSize    string
	HTTPClient     *http.Client
}

type ModelScopeImageTool struct {
	baseURL     string
	defaultSize string
	httpClient  *http.Client
}

type modelScopeImageResponse struct {
	Created int64 `json:"created,omitempty"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
		LocalPath     string `json:"local_path,omitempty"`
		MetaPath      string `json:"meta_path,omitempty"`
	} `json:"data"`
}

func NewModelScopeImageTool(opts ModelScopeImageToolOptions) *ModelScopeImageTool {
	if !opts.Enabled || strings.TrimSpace(opts.BaseURL) == "" {
		return nil
	}

	timeout := time.Duration(opts.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}

	defaultSize := strings.TrimSpace(opts.DefaultSize)
	if defaultSize == "" {
		defaultSize = "1024x1024"
	}

	return &ModelScopeImageTool{
		baseURL:     strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		defaultSize: defaultSize,
		httpClient:  client,
	}
}

func (t *ModelScopeImageTool) Name() string {
	return "modelscope-image"
}

func (t *ModelScopeImageTool) Description() string {
	return "Generate images through the local ModelScope image service."
}

func (t *ModelScopeImageTool) Parameters() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties": map[string]any{
			"prompt": map[string]any{
				"type":        "string",
				"description": "Prompt used to generate the image.",
			},
			"n": map[string]any{
				"type":        "number",
				"description": "Number of images to generate. Defaults to 1.",
				"minimum":     1,
				"maximum":     4,
			},
			"size": map[string]any{
				"type":        "string",
				"description": "Requested image size, for example 1024x1024.",
			},
		},
		"required": []string{"prompt"},
	}
}

func (t *ModelScopeImageTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	prompt, _ := args["prompt"].(string)
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ErrorResult("prompt is required")
	}

	n := 1
	switch value := args["n"].(type) {
	case float64:
		if value > 0 {
			n = int(value)
		}
	case int:
		if value > 0 {
			n = value
		}
	}

	size, _ := args["size"].(string)
	size = strings.TrimSpace(size)
	if size == "" {
		size = t.defaultSize
	}

	payload, err := json.Marshal(map[string]any{
		"prompt":          prompt,
		"n":               n,
		"size":            size,
		"response_format": "url",
	})
	if err != nil {
		return ErrorResult("marshal request failed").WithError(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.baseURL+"/v1/images/generations", bytes.NewReader(payload))
	if err != nil {
		return ErrorResult("build request failed").WithError(err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return ErrorResult(fmt.Sprintf("modelscope-image request failed: %v", err)).WithError(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ErrorResult("read response failed").WithError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ErrorResult(fmt.Sprintf("modelscope-image api error (status %d): %s", resp.StatusCode, strings.TrimSpace(string(body))))
	}

	var parsed modelScopeImageResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return ErrorResult("parse response failed").WithError(err)
	}

	llmJSON, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return ErrorResult("marshal result failed").WithError(err)
	}

	userMsg := fmt.Sprintf("已生成 %d 张图片", len(parsed.Data))
	if size != "" {
		userMsg += fmt.Sprintf("，尺寸 %s", size)
	}

	return &ToolResult{
		ForLLM:  string(llmJSON),
		ForUser: userMsg,
	}
}
