package runtime

import (
	"context"
	"testing"
)

func TestLifecycleRunsHooksInOrder(t *testing.T) {
	t.Parallel()

	order := make([]string, 0, 4)
	lifecycle := NewLifecycle(nil)
	lifecycle.Append(Hook{
		Name: "one",
		Start: func(context.Context) error {
			order = append(order, "start-one")
			return nil
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-one")
			return nil
		},
	})
	lifecycle.Append(Hook{
		Name: "two",
		Start: func(context.Context) error {
			order = append(order, "start-two")
			return nil
		},
		Stop: func(context.Context) error {
			order = append(order, "stop-two")
			return nil
		},
	})

	if err := lifecycle.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := lifecycle.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}

	expected := []string{"start-one", "start-two", "stop-two", "stop-one"}
	if len(order) != len(expected) {
		t.Fatalf("unexpected order length: %#v", order)
	}
	for index := range expected {
		if order[index] != expected[index] {
			t.Fatalf("unexpected order at %d: %#v", index, order)
		}
	}
}
