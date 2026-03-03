package providers

import (
	"strings"
	"testing"
)

func TestNormalizeToolCall_FillsMissingIDAndType(t *testing.T) {
	in := ToolCall{
		Function: &FunctionCall{
			Name:      "get_weather",
			Arguments: `{"city":"SF"}`,
		},
	}

	out := NormalizeToolCall(in)

	if out.ID == "" {
		t.Fatalf("ID should be generated when missing")
	}
	if !strings.HasPrefix(out.ID, "call_") {
		t.Fatalf("generated ID = %q, want prefix call_", out.ID)
	}
	if out.Type != "function" {
		t.Fatalf("Type = %q, want function", out.Type)
	}
	if out.Name != "get_weather" {
		t.Fatalf("Name = %q, want get_weather", out.Name)
	}
	if out.Arguments["city"] != "SF" {
		t.Fatalf("Arguments[city] = %v, want SF", out.Arguments["city"])
	}
}

func TestNormalizeToolCall_PreservesExistingID(t *testing.T) {
	in := ToolCall{
		ID:   "call_existing",
		Type: "function",
		Name: "echo",
		Arguments: map[string]any{
			"text": "hello",
		},
	}

	out := NormalizeToolCall(in)
	if out.ID != "call_existing" {
		t.Fatalf("ID = %q, want call_existing", out.ID)
	}
	if out.Function == nil {
		t.Fatalf("Function should be populated")
	}
	if out.Function.Name != "echo" {
		t.Fatalf("Function.Name = %q, want echo", out.Function.Name)
	}
}
