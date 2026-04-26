package authjwt

import "testing"

func TestBearerToken_ok(t *testing.T) {
	tok, ok := bearerToken("Bearer abc.def.ghi")
	if !ok || tok != "abc.def.ghi" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}

func TestBearerToken_caseInsensitiveScheme(t *testing.T) {
	tok, ok := bearerToken("bearer secretvalue")
	if !ok || tok != "secretvalue" {
		t.Fatalf("got %q ok=%v", tok, ok)
	}
}

func TestBearerToken_missing(t *testing.T) {
	_, ok := bearerToken("")
	if ok {
		t.Fatal("expected false")
	}
	_, ok = bearerToken("Basic x")
	if ok {
		t.Fatal("expected false")
	}
}
