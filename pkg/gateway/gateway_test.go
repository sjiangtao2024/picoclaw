package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sipeed/picoclaw/pkg/health"
)

func TestMarkHealthReadyExposesReadyEndpoint(t *testing.T) {
	srv := health.NewServer("127.0.0.1", 18790)
	mux := http.NewServeMux()
	srv.RegisterOnMux(mux)

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("before markHealthReady status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}

	markHealthReady(srv)

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("after markHealthReady status = %d, want %d", rec.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if payload["status"] != "ready" {
		t.Fatalf("ready payload status = %v, want ready", payload["status"])
	}
}
