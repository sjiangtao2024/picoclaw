//go:build whatsapp_native

package whatsapp

import (
	"testing"
	"time"
)

func TestPairingStateIncludesExpiryCountdown(t *testing.T) {
	c := &WhatsAppNativeChannel{}
	c.setPairingCode("dummy-code", 20*time.Second)

	state := c.getPairingStateMap()
	if state["code"] != "dummy-code" {
		t.Fatalf("expected code to be set")
	}
	if state["event"] != "code" {
		t.Fatalf("expected event=code, got %v", state["event"])
	}
	expiresAt, _ := state["expires_at"].(string)
	if expiresAt == "" {
		t.Fatalf("expected expires_at to be set")
	}
	left, ok := state["expires_in_seconds"].(int64)
	if !ok {
		t.Fatalf("expected expires_in_seconds to be int64, got %T", state["expires_in_seconds"])
	}
	if left <= 0 || left > 20 {
		t.Fatalf("expected expires_in_seconds in (0,20], got %d", left)
	}

	c.handlePairingEvent("timeout")
	state = c.getPairingStateMap()
	if state["code"] != "" {
		t.Fatalf("expected code to be cleared on timeout")
	}
	if state["expires_in_seconds"].(int64) != 0 {
		t.Fatalf("expected expires_in_seconds to be 0 on timeout")
	}
}

