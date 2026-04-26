package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

// VerifySignature checks X-Signature-256: sha256=<hex> against HMAC-SHA256(secret, body).
func VerifySignature(body []byte, secret []byte, header string) error {
	header = strings.TrimSpace(header)
	if header == "" {
		return ErrMissingSignature
	}
	hexPart, ok := stripSHA256Prefix(header)
	if !ok {
		return ErrInvalidSignature
	}
	gotMAC, err := hex.DecodeString(hexPart)
	if err != nil || len(gotMAC) != sha256.Size {
		return ErrInvalidSignature
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	wantMAC := mac.Sum(nil)
	if subtle.ConstantTimeCompare(gotMAC, wantMAC) != 1 {
		return ErrInvalidSignature
	}
	return nil
}

func stripSHA256Prefix(s string) (hexPart string, ok bool) {
	s = strings.TrimSpace(s)
	if len(s) < 7 {
		return "", false
	}
	if !strings.EqualFold(s[:7], "sha256=") {
		return "", false
	}
	return s[7:], true
}
