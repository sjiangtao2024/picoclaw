package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelScopeImageToolReturnsNilWhenDisabled(t *testing.T) {
	tool := NewModelScopeImageTool(ModelScopeImageToolOptions{
		Enabled: false,
		BaseURL: "http://127.0.0.1:8010",
	})
	if tool != nil {
		t.Fatal("expected nil tool when disabled")
	}
}

func TestModelScopeImageToolRejectsMissingPrompt(t *testing.T) {
	tool := NewModelScopeImageTool(ModelScopeImageToolOptions{
		Enabled:        true,
		BaseURL:        "http://127.0.0.1:8010",
		TimeoutSeconds: 30,
	})
	if tool == nil {
		t.Fatal("expected tool")
	}

	result := tool.Execute(context.Background(), map[string]any{})
	if !result.IsError {
		t.Fatal("expected error result")
	}
	if !strings.Contains(result.ForLLM, "prompt is required") {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
}

func TestModelScopeImageToolExecuteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("path = %s, want /v1/images/generations", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["prompt"] != "画一只橘猫" {
			t.Fatalf("prompt = %v, want 画一只橘猫", req["prompt"])
		}
		if req["size"] != "1024x1024" {
			t.Fatalf("size = %v, want 1024x1024", req["size"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"created": 1712000000,
			"data": []map[string]any{
				{
					"url":            "http://127.0.0.1:8010/v1/images/files/cat.png",
					"local_path":     "/root/picoclaw-plugins/modelscope-image/data/images/cat.png",
					"meta_path":      "/root/picoclaw-plugins/modelscope-image/data/meta/images/cat.json",
					"revised_prompt": "一只橘猫",
				},
			},
		})
	}))
	defer server.Close()

	tool := NewModelScopeImageTool(ModelScopeImageToolOptions{
		Enabled:        true,
		BaseURL:        server.URL,
		TimeoutSeconds: 30,
		DefaultSize:    "1024x1024",
	})
	if tool == nil {
		t.Fatal("expected tool")
	}

	result := tool.Execute(context.Background(), map[string]any{
		"prompt": "画一只橘猫",
		"n":      1,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForLLM, "cat.png") {
		t.Fatalf("unexpected llm result: %s", result.ForLLM)
	}
	if !strings.Contains(result.ForUser, "已生成 1 张图片") {
		t.Fatalf("unexpected user result: %s", result.ForUser)
	}
}
