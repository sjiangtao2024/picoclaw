package preroute

import "context"

type Context struct {
	Channel    string
	ChatID     string
	AgentID    string
	Workspace  string
	Query      string
	QuotedText string
	MediaRefs  []string
	Metadata   map[string]string
}

type Result struct {
	Handled   bool
	Text      string
	MediaRefs []string
	RouteID   string
}

type Route interface {
	ID() string
	Handle(ctx context.Context, rc Context) (Result, error)
}
