package preroute

import (
	"context"
	"testing"
)

type testRoute struct {
	id     string
	result Result
}

func (r testRoute) ID() string { return r.id }

func (r testRoute) Handle(context.Context, Context) (Result, error) {
	return r.result, nil
}

func TestRouterReturnsFirstHandledResult(t *testing.T) {
	router := NewRouter([]Route{
		testRoute{id: "skip", result: Result{}},
		testRoute{id: "help", result: Result{Handled: true, Text: "forced help", RouteID: "help"}},
		testRoute{id: "later", result: Result{Handled: true, Text: "later", RouteID: "later"}},
	})

	got, err := router.Route(context.Background(), Context{Query: "帮助"})
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if !got.Handled {
		t.Fatal("expected handled result")
	}
	if got.RouteID != "help" {
		t.Fatalf("RouteID = %q, want help", got.RouteID)
	}
	if got.Text != "forced help" {
		t.Fatalf("Text = %q, want forced help", got.Text)
	}
}
