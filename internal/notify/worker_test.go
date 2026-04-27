package notify

import (
	"testing"
	"time"
)

func TestWorker_nextBackoff_capsShift(t *testing.T) {
	w := &Worker{BackoffBase: time.Millisecond}
	if d := w.nextBackoff(1); d != time.Millisecond*2 {
		t.Fatalf("attempt 1: got %v", d)
	}
	if d := w.nextBackoff(20); d != time.Millisecond*(1<<10) {
		t.Fatalf("attempt 20 capped at 10: got %v", d)
	}
}
