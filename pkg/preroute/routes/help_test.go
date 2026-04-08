package routes

import (
	"context"
	"testing"

	"github.com/sipeed/picoclaw/pkg/preroute"
)

func TestHelpRouteHandlesGlobalHelpIntent(t *testing.T) {
	route := NewHelp("当前可用功能")

	got, err := route.Handle(context.Background(), preroute.Context{Query: "帮助"})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !got.Handled {
		t.Fatal("expected help route to handle query")
	}
	if got.RouteID != "help" {
		t.Fatalf("RouteID = %q, want help", got.RouteID)
	}
	if got.Text != "当前可用功能" {
		t.Fatalf("Text = %q, want configured help text", got.Text)
	}
}
