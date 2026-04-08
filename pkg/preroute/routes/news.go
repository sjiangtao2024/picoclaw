package routes

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/preroute"
)

type NewsExecutor interface {
	Execute(ctx context.Context, rc preroute.Context) (text string, handled bool, err error)
}

type newsRoute struct {
	executor NewsExecutor
}

func NewNews(executor NewsExecutor) preroute.Route {
	return newsRoute{executor: executor}
}

func (r newsRoute) ID() string { return "news" }

func (r newsRoute) Handle(ctx context.Context, rc preroute.Context) (preroute.Result, error) {
	if r.executor == nil {
		return preroute.Result{}, nil
	}
	text, handled, err := r.executor.Execute(ctx, rc)
	if err != nil {
		return preroute.Result{}, err
	}
	if !handled {
		return preroute.Result{}, nil
	}
	return preroute.Result{
		Handled: true,
		Text:    text,
		RouteID: r.ID(),
	}, nil
}
