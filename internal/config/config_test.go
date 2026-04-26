package config

import "testing"

func TestGetDefault_usesDefaultWhenUnset(t *testing.T) {
	t.Setenv("HTTP_ADDR", "")
	c := getDefault("HTTP_ADDR", ":9999")
	if c != ":9999" {
		t.Fatalf("expected default :9999, got %q", c)
	}
}
