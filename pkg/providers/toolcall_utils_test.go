package providers

import "testing"

func TestNormalizeToolCallFillsMissingIDAndType(t *testing.T) {
	tc := NormalizeToolCall(ToolCall{
		Function: &FunctionCall{
			Name:      "web_search",
			Arguments: `{"query":"picoclaw"}`,
		},
	})

	if tc.ID == "" {
		t.Fatal("expected NormalizeToolCall to populate missing ID")
	}
	if tc.Type != "function" {
		t.Fatalf("Type = %q, want function", tc.Type)
	}
	if tc.Name != "web_search" {
		t.Fatalf("Name = %q, want web_search", tc.Name)
	}
	if tc.Function == nil || tc.Function.Arguments == "" {
		t.Fatal("expected Function arguments to remain populated")
	}
}
