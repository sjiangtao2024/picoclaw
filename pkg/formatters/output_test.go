package formatters

import "testing"

func TestFormatPlainTextChannels(t *testing.T) {
	input := "### Title\n\n- **bold** item\n- `code`\n\n```go\nfmt.Println(\"x\")\n```"
	got := Format("dingtalk", input)
	want := "Title\n• bold item\n• code\n\nfmt.Println(\"x\")"
	if got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}

func TestFormatLeavesOtherChannelsUntouched(t *testing.T) {
	input := "  **keep markdown**  "
	got := Format("feishu", input)
	want := "**keep markdown**"
	if got != want {
		t.Fatalf("Format() = %q, want %q", got, want)
	}
}
