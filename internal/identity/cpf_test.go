package identity

import "testing"

func TestNormalizeCPF11_ok(t *testing.T) {
	s, err := NormalizeCPF11("  12345678901  ")
	if err != nil || s != "12345678901" {
		t.Fatalf("got %q %v", s, err)
	}
}

func TestNormalizeCPF11_invalid(t *testing.T) {
	_, err := NormalizeCPF11("123")
	if err == nil {
		t.Fatal("expected error")
	}
}
