package preroute

import "context"

type Router struct {
	routes []Route
}

func NewRouter(routes []Route) *Router {
	return &Router{routes: append([]Route(nil), routes...)}
}

func (r *Router) Route(ctx context.Context, rc Context) (Result, error) {
	if r == nil {
		return Result{}, nil
	}
	for _, route := range r.routes {
		result, err := route.Handle(ctx, rc)
		if err != nil {
			return Result{}, err
		}
		if result.Handled {
			if result.RouteID == "" {
				result.RouteID = route.ID()
			}
			return result, nil
		}
	}
	return Result{}, nil
}
