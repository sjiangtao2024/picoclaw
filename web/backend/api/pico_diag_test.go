package api

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

func TestSummarizePicoTokenForLog(t *testing.T) {
	if got := summarizePicoTokenForLog(""); got != "empty" {
		t.Fatalf("empty token summary = %q", got)
	}
	if got := summarizePicoTokenForLog("abcdef"); got != "len=6" {
		t.Fatalf("short token summary = %q", got)
	}
	if got := summarizePicoTokenForLog("0de1188561a0540077e4bc685b496110"); got != "len=32 suffix=6110" {
		t.Fatalf("long token summary = %q", got)
	}
}

func TestSummarizePicoProtocolsForLog(t *testing.T) {
	got := summarizePicoProtocolsForLog([]string{"token.", "token.abcdef", "chat"})
	want := []string{"token.(empty)", "token.(len=6)", "raw(len=4)"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("protocol summary = %#v, want %#v", got, want)
	}
}

func TestPicoRequestLogFields(t *testing.T) {
	req := httptest.NewRequest("GET", "http://launcher.local/pico/ws", nil)
	req.Host = "192.168.1.63:18800"
	req.RemoteAddr = "192.168.1.44:53082"
	req.Header.Set("Origin", "http://192.168.1.63:18800")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.AddCookie(&http.Cookie{
		Name:  middleware.LauncherDashboardCookieName,
		Value: "session",
	})

	fields := picoRequestLogFields(req)
	if fields["host"] != "192.168.1.63:18800" {
		t.Fatalf("host = %#v", fields["host"])
	}
	if fields["origin"] != "http://192.168.1.63:18800" {
		t.Fatalf("origin = %#v", fields["origin"])
	}
	if fields["dashboard_cookie"] != true {
		t.Fatalf("dashboard_cookie = %#v", fields["dashboard_cookie"])
	}
}
