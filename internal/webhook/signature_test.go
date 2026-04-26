package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifySignature_ok(t *testing.T) {
	secret := []byte("test-secret-key-for-hmac")
	body := []byte(`{"chamado_id":"CH-1","tipo":"status_change"}`)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	if err := VerifySignature(body, secret, "sha256="+sig); err != nil {
		t.Fatal(err)
	}
}

func TestVerifySignature_caseInsensitivePrefix(t *testing.T) {
	secret := []byte("s")
	body := []byte("x")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))
	if err := VerifySignature(body, secret, "SHA256="+sig); err != nil {
		t.Fatal(err)
	}
}

func TestVerifySignature_missing(t *testing.T) {
	err := VerifySignature([]byte("a"), []byte("s"), "")
	if err == nil || !isUnauthorizedSig(err) {
		t.Fatalf("expected missing signature error, got %v", err)
	}
}

func TestVerifySignature_badHex(t *testing.T) {
	err := VerifySignature([]byte("a"), []byte("s"), "sha256=zz")
	if err == nil || !isUnauthorizedSig(err) {
		t.Fatalf("expected invalid signature, got %v", err)
	}
}

func TestVerifySignature_wrongMAC(t *testing.T) {
	secret := []byte("s")
	body := []byte("payload")
	if err := VerifySignature(body, secret, "sha256="+hex.EncodeToString(make([]byte, 32))); err == nil {
		t.Fatal("expected error")
	}
}

func isUnauthorizedSig(err error) bool {
	return err == ErrMissingSignature || err == ErrInvalidSignature
}
