package executors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTencentNewsCLIPathPrefersWorkspaceSkillBinary(t *testing.T) {
	workspace := t.TempDir()
	cliPath := filepath.Join(workspace, "skills", "tencent-news", "tencent-news-cli")
	if err := os.MkdirAll(filepath.Dir(cliPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cliPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, ok := ResolveTencentNewsCLIPath(workspace, "skills/tencent-news/tencent-news-cli")
	if !ok {
		t.Fatal("expected CLI path to resolve")
	}
	if got != cliPath {
		t.Fatalf("ResolveTencentNewsCLIPath() = %q, want %q", got, cliPath)
	}
}

func TestResolveTencentNewsAPIKeyFromBashrc(t *testing.T) {
	home := t.TempDir()
	bashrc := filepath.Join(home, ".bashrc")
	if err := os.WriteFile(bashrc, []byte("export TENCENT_NEWS_APIKEY=\"abc-123\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := resolveTencentNewsAPIKey(func(string) string { return "" }, home)
	if got != "abc-123" {
		t.Fatalf("resolveTencentNewsAPIKey() = %q, want abc-123", got)
	}
}

func TestResolveTencentNewsAPIKeyFromProfile(t *testing.T) {
	home := t.TempDir()
	profile := filepath.Join(home, ".profile")
	if err := os.WriteFile(profile, []byte("export TENCENT_NEWS_APIKEY=\"xyz-789\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := resolveTencentNewsAPIKey(func(string) string { return "" }, home)
	if got != "xyz-789" {
		t.Fatalf("resolveTencentNewsAPIKey() = %q, want xyz-789", got)
	}
}
