package telemetry

import (
	"context"
	"testing"
)

func TestInitTracesExporterNone(t *testing.T) {
	shutdown, err := Init(context.Background(), "unit-test", "none")
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInitEmptyServiceNameUsesDefault(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown, err := Init(ctx, "", "none")
	if err != nil {
		t.Fatal(err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}
}
