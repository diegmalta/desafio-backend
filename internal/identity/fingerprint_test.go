package identity

import (
	"bytes"
	"testing"
)

func TestCitizenFingerprint_deterministic(t *testing.T) {
	p := []byte("pepper-32-bytes-for-test________")
	a := CitizenFingerprint(p, "12345678901")
	b := CitizenFingerprint(p, "12345678901")
	if !bytes.Equal(a, b) || len(a) != 32 {
		t.Fatalf("fingerprint len=%d", len(a))
	}
}
