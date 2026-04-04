package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/sipeed/picoclaw/web/backend/middleware"
)

func summarizePicoTokenForLog(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return "empty"
	}
	if len(token) <= 6 {
		return fmt.Sprintf("len=%d", len(token))
	}
	return fmt.Sprintf("len=%d suffix=%s", len(token), token[len(token)-4:])
}

func summarizePicoProtocolsForLog(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			out = append(out, "empty")
			continue
		}
		if after, ok := strings.CutPrefix(trimmed, tokenPrefix); ok {
			out = append(out, "token.("+summarizePicoTokenForLog(after)+")")
			continue
		}
		out = append(out, fmt.Sprintf("raw(len=%d)", len(trimmed)))
	}
	return out
}

func picoRequestLogFields(r *http.Request) map[string]any {
	fields := map[string]any{
		"host":        r.Host,
		"origin":      r.Header.Get("Origin"),
		"remote_addr": r.RemoteAddr,
	}
	if ua := strings.TrimSpace(r.UserAgent()); ua != "" {
		fields["user_agent"] = ua
	}
	if _, err := r.Cookie(middleware.LauncherDashboardCookieName); err == nil {
		fields["dashboard_cookie"] = true
	}
	return fields
}
