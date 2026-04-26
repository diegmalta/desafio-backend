package webhook

import "testing"

func TestIdempotencyKey_stable(t *testing.T) {
	p := &EventPayload{
		ChamadoID:  "CH-2024-001234",
		StatusNovo: "em_execucao",
		Timestamp:  "2024-11-15T14:30:00Z",
		Tipo:       "status_change",
	}
	a := IdempotencyKey(p)
	b := IdempotencyKey(p)
	if a != b || len(a) != 64 {
		t.Fatalf("unexpected key: %q len=%d", a, len(a))
	}
}

func TestIdempotencyKey_differsOnStatus(t *testing.T) {
	base := EventPayload{
		ChamadoID:  "CH-1",
		StatusNovo: "a",
		Timestamp:  "2024-11-15T14:30:00Z",
		Tipo:       "status_change",
	}
	other := base
	other.StatusNovo = "b"
	if IdempotencyKey(&base) == IdempotencyKey(&other) {
		t.Fatal("keys should differ")
	}
}
