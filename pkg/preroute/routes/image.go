package routes

import (
	"context"

	"github.com/sipeed/picoclaw/pkg/preroute"
)

type ImageExecutor interface {
	Execute(ctx context.Context, rc preroute.Context) (preroute.Result, error)
}

type imageRoute struct {
	executor ImageExecutor
}

func NewImage(executor ImageExecutor) preroute.Route {
	return imageRoute{executor: executor}
}

func (r imageRoute) ID() string { return "image" }

func (r imageRoute) Handle(ctx context.Context, rc preroute.Context) (preroute.Result, error) {
	if r.executor == nil {
		return preroute.Result{}, nil
	}
	result, err := r.executor.Execute(ctx, rc)
	if err != nil {
		return preroute.Result{}, err
	}
	if result.Handled && result.RouteID == "" {
		result.RouteID = r.ID()
	}
	return result, nil
}
